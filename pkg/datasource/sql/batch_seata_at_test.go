/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sql

import (
	"context"
	gosql "database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/factor"
	undoparser "seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/parser"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

type batchATRows struct {
	columns []string
	data    [][]driver.Value
	index   int
}

func (r *batchATRows) Columns() []string { return r.columns }
func (r *batchATRows) Close() error      { return nil }
func (r *batchATRows) Next(dest []driver.Value) error {
	if r.index == len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.index])
	r.index++
	return nil
}

type batchATTableCache struct {
	tableMeta *types.TableMeta
}

func (batchATTableCache) Init(context.Context, *gosql.DB) error { return nil }
func (batchATTableCache) Destroy() error                        { return nil }
func (c batchATTableCache) GetTableMeta(_ context.Context, _, table string) (*types.TableMeta, error) {
	if c.tableMeta != nil {
		return c.tableMeta, nil
	}
	idColumn := types.ColumnMeta{ColumnName: "id", DatabaseTypeString: "BIGINT"}
	return &types.TableMeta{
		TableName:   table,
		ColumnNames: []string{"id", "balance"},
		Columns: map[string]types.ColumnMeta{
			"id":      idColumn,
			"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{idColumn}},
		},
	}, nil
}

func newBatchSeataATTestDB(t *testing.T,
	ctrl *gomock.Controller,
) (*gosql.DB, *mock.MockTestDriverConn, *mock.MockTestDriverTx, *mock.MockDataSourceManager) {
	t.Helper()

	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeAT)
	mockMgr.EXPECT().RegisterResource(gomock.Any()).Times(1).Return(nil)
	registerResourceManagerForTest(t, mockMgr)

	mockTx := mock.NewMockTestDriverTx(ctrl)
	mockConn := mock.NewMockTestDriverConn(ctrl)

	mockConn.EXPECT().
		QueryContext(gomock.Any(), "SELECT VERSION()", gomock.Any()).
		AnyTimes().
		DoAndReturn(func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			rows := &mysqlMockRows{}
			rows.data = [][]interface{}{
				{"8.0.29"},
			}
			return rows, nil
		})

	mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
	mockConn.EXPECT().Close().AnyTimes().Return(nil)

	connector := mock.NewMockTestDriverConnector(ctrl)
	connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)

	targetDB := gosql.OpenDB(connector)
	t.Cleanup(func() {
		_ = targetDB.Close()
	})

	previousTableCache := datasource.GetTableCache(types.DBTypeMySQL)
	t.Cleanup(func() {
		datasource.RegisterTableCache(types.DBTypeMySQL, previousTableCache)
	})

	proxyConnector, err := (&seataDriver{
		branchType: branch.BranchTypeAT,
		transType:  types.ATMode,
		descriptor: mySQLDriverDescriptor,
		target:     mySQLDriverDescriptor.target,
		targetName: "mysql",
	}).getOpenConnectorProxy(
		connector,
		types.DBTypeMySQL,
		targetDB,
		"root:password@tcp(mock:3306)/seata_client?multiStatements=true",
	)
	require.NoError(t, err)

	baseConnector, ok := proxyConnector.(*seataConnector)
	require.True(t, ok)
	db := gosql.OpenDB(&seataATConnector{seataConnector: baseConnector})

	return db, mockConn, mockTx, mockMgr
}

func expectBatchATImageQueries(t *testing.T, mockConn *mock.MockTestDriverConn, snapshots [][]driver.Value) {
	t.Helper()

	callIndex := 0
	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Not("SELECT VERSION()"), gomock.Any()).
		Times(len(snapshots)).DoAndReturn(func(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
		require.Len(t, args, 1)
		require.Equal(t, snapshots[callIndex][0], args[0].Value)
		if callIndex%2 == 0 {
			require.Contains(t, query, "FOR UPDATE")
		} else {
			require.NotContains(t, query, "FOR UPDATE")
		}
		rows := &batchATRows{columns: []string{"id", "balance"}, data: [][]driver.Value{snapshots[callIndex]}}
		callIndex++
		return rows, nil
	})
}

type batchATUndoExpectation struct {
	statement string
	undoArgs  []driver.NamedValue
}

func expectBatchATUndoExecutions(t *testing.T, ctrl *gomock.Controller, mockConn *mock.MockTestDriverConn, expectations ...batchATUndoExpectation) {
	t.Helper()

	orderedCalls := make([]*gomock.Call, 0, len(expectations)*3)
	for _, expectation := range expectations {
		expectation := expectation
		stmt := mock.NewMockTestDriverStmt(ctrl)
		prepareCall := mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(1).
			DoAndReturn(func(_ context.Context, query string) (driver.Stmt, error) {
				require.Contains(t, query, expectation.statement)
				return stmt, nil
			})
		stmt.EXPECT().NumInput().AnyTimes().Return(-1)
		execCall := stmt.EXPECT().ExecContext(gomock.Any(), expectation.undoArgs).Times(1).Return(driver.RowsAffected(1), nil)
		closeCall := stmt.EXPECT().Close().Times(1).Return(nil)
		orderedCalls = append(orderedCalls, prepareCall, execCall, closeCall)
	}
	gomock.InOrder(orderedCalls...)
}

func executeBatchATUndo(t *testing.T, db *gosql.DB, branchUndoLog *undo.BranchUndoLog) {
	t.Helper()

	branchUndoLog.Reverse()
	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	for index := range branchUndoLog.Logs {
		log := &branchUndoLog.Logs[index]
		tableMeta, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(context.Background(), "", log.TableName)
		require.NoError(t, err)
		log.SetTableMeta(tableMeta)
		executor, err := factor.GetUndoExecutor(types.DBTypeMySQL, *log)
		require.NoError(t, err)
		require.NoError(t, executor.ExecuteOn(context.Background(), types.DBTypeMySQL, conn))
	}
}

func TestExecBatchContextWithSeataATDriverUsesSingleBranchLifecycle(t *testing.T) {
	CleanTxHooks()
	t.Cleanup(CleanTxHooks)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockConn, mockTx, mockMgr := newBatchSeataATTestDB(t, ctrl)
	defer db.Close()
	datasource.RegisterTableCache(types.DBTypeMySQL, batchATTableCache{})

	previousUndoConfig := undo.UndoConfig
	undo.UndoConfig = undo.Config{LogSerialization: "json", LogTable: "undo_log"}
	t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })

	ctx := tm.InitSeataContext(context.Background())
	xid := uuid.NewString()
	tm.SetXID(ctx, xid)

	query := "UPDATE account SET balance = ? WHERE id = ?"
	var txCtx *types.TransactionContext
	RegisterTxHook(&mockTxHook{beforeCommit: func(tx *Tx) error {
		txCtx = tx.tranCtx
		return nil
	}})

	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
	expectBatchATImageQueries(t, mockConn, [][]driver.Value{
		{int64(1), int64(100)}, {int64(1), int64(110)},
		{int64(1), int64(110)}, {int64(1), int64(120)},
	})
	mockConn.EXPECT().ExecContext(gomock.Any(), query, gomock.Any()).Times(2).DoAndReturn(
		func(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
			require.Len(t, args, 2)
			return driver.RowsAffected(1), nil
		},
	)

	const branchID = int64(123)
	var registeredLockKeys []string
	registerCall := mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, param rm.BranchRegisterParam) (int64, error) {
			require.Equal(t, xid, param.Xid)
			registeredLockKeys = strings.FieldsFunc(param.LockKeys, func(r rune) bool { return r == ';' })
			return branchID, nil
		},
	)

	undoStmt := mock.NewMockTestDriverStmt(ctrl)
	prepareUndoCall := mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, query string) (driver.Stmt, error) {
			require.Contains(t, query, "INSERT INTO undo_log")
			return undoStmt, nil
		},
	)
	undoStmt.EXPECT().Close().Times(1).Return(nil)
	var branchUndoLog *undo.BranchUndoLog
	flushUndoCall := undoStmt.EXPECT().ExecContext(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, args []driver.NamedValue) (driver.Result, error) {
			require.Len(t, args, 5)
			require.EqualValues(t, branchID, args[0].Value)
			require.Equal(t, xid, args[1].Value)
			rollbackInfo, ok := args[3].Value.([]byte)
			require.True(t, ok)
			var err error
			branchUndoLog, err = (&undoparser.JsonParser{}).Decode(rollbackInfo)
			require.NoError(t, err)
			return driver.ResultNoRows, nil
		},
	)

	commitCall := mockTx.EXPECT().Commit().Times(1).Return(nil)
	reportCall := mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, param rm.BranchReportParam) error {
			require.EqualValues(t, branchID, param.BranchId)
			require.EqualValues(t, branch.BranchStatusPhaseoneDone, param.Status)
			return nil
		},
	)
	gomock.InOrder(registerCall, prepareUndoCall, flushUndoCall, commitCall, reportCall)

	result, err := ExecBatchContext(ctx, db, query, [][]any{{int64(110), int64(1)}, {int64(120), int64(1)}})

	require.NoError(t, err)
	require.Equal(t, BatchTransactionCommitted, result.Outcome.TransactionState)
	require.NotNil(t, txCtx)
	require.Len(t, txCtx.RoundImages.BeofreImages(), 2)
	require.Len(t, txCtx.RoundImages.AfterImages(), 2)

	lockKeys := make([]string, 0, len(txCtx.LockKeys))
	for lockKey := range txCtx.LockKeys {
		lockKeys = append(lockKeys, lockKey)
	}
	require.Equal(t, []string{"ACCOUNT:1"}, lockKeys)
	require.ElementsMatch(t, lockKeys, registeredLockKeys)

	require.NotNil(t, branchUndoLog)
	require.Equal(t, xid, branchUndoLog.Xid)
	require.EqualValues(t, branchID, branchUndoLog.BranchID)
	require.Len(t, branchUndoLog.Logs, 2)
	for i, balances := range [][2]int64{{100, 110}, {110, 120}} {
		require.EqualValues(t, 1, branchUndoLog.Logs[i].BeforeImage.Rows[0].GetColumnMap()["id"].Value)
		require.EqualValues(t, balances[0], branchUndoLog.Logs[i].BeforeImage.Rows[0].GetColumnMap()["balance"].Value)
		require.EqualValues(t, balances[1], branchUndoLog.Logs[i].AfterImage.Rows[0].GetColumnMap()["balance"].Value)
	}

	expectBatchATUndoExecutions(t, ctrl, mockConn,
		batchATUndoExpectation{
			statement: "UPDATE account",
			undoArgs:  []driver.NamedValue{{Ordinal: 1, Value: int64(110)}, {Ordinal: 2, Value: int64(1)}},
		},
		batchATUndoExpectation{
			statement: "UPDATE account",
			undoArgs:  []driver.NamedValue{{Ordinal: 1, Value: int64(100)}, {Ordinal: 2, Value: int64(1)}},
		},
	)
	executeBatchATUndo(t, db, branchUndoLog)
}

func TestExecBatchContextWithSeataATDriverSkipsNoOpDeleteItem(t *testing.T) {
	CleanTxHooks()
	t.Cleanup(CleanTxHooks)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockConn, mockTx, mockMgr := newBatchSeataATTestDB(t, ctrl)
	defer db.Close()
	datasource.RegisterTableCache(types.DBTypeMySQL, batchATTableCache{})

	previousUndoConfig := undo.UndoConfig
	undo.UndoConfig = undo.Config{LogSerialization: "json", LogTable: "undo_log"}
	t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })

	ctx := tm.InitSeataContext(context.Background())
	xid := uuid.NewString()
	tm.SetXID(ctx, xid)

	const query = "DELETE FROM account WHERE id = ?"
	var txCtx *types.TransactionContext
	RegisterTxHook(&mockTxHook{beforeCommit: func(tx *Tx) error {
		txCtx = tx.tranCtx
		return nil
	}})

	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
	queryCount := 0
	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Not("SELECT VERSION()"), gomock.Any()).
		Times(2).DoAndReturn(func(_ context.Context, imageQuery string, args []driver.NamedValue) (driver.Rows, error) {
		require.Contains(t, imageQuery, "FOR UPDATE")
		require.Equal(t, []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}, args)
		queryCount++
		if queryCount == 1 {
			return &batchATRows{
				columns: []string{"id", "balance"},
				data:    [][]driver.Value{{int64(1), int64(100)}},
			}, nil
		}
		return &batchATRows{columns: []string{"id", "balance"}}, nil
	})

	execCount := 0
	mockConn.EXPECT().ExecContext(gomock.Any(), query, gomock.Any()).Times(2).DoAndReturn(
		func(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
			require.Equal(t, []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}, args)
			execCount++
			return driver.RowsAffected(2 - execCount), nil
		},
	)

	const branchID = int64(124)
	registerCall := mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, param rm.BranchRegisterParam) (int64, error) {
			require.Equal(t, xid, param.Xid)
			require.Equal(t, "ACCOUNT:1;", param.LockKeys)
			return branchID, nil
		},
	)

	undoStmt := mock.NewMockTestDriverStmt(ctrl)
	prepareUndoCall := mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, undoQuery string) (driver.Stmt, error) {
			require.Contains(t, undoQuery, "INSERT INTO undo_log")
			return undoStmt, nil
		},
	)
	undoStmt.EXPECT().Close().Times(1).Return(nil)
	var branchUndoLog *undo.BranchUndoLog
	flushUndoCall := undoStmt.EXPECT().ExecContext(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, args []driver.NamedValue) (driver.Result, error) {
			require.Len(t, args, 5)
			rollbackInfo, ok := args[3].Value.([]byte)
			require.True(t, ok)
			var err error
			branchUndoLog, err = (&undoparser.JsonParser{}).Decode(rollbackInfo)
			require.NoError(t, err)
			return driver.ResultNoRows, nil
		},
	)

	commitCall := mockTx.EXPECT().Commit().Times(1).Return(nil)
	reportCall := mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	gomock.InOrder(registerCall, prepareUndoCall, flushUndoCall, commitCall, reportCall)

	result, err := ExecBatchContext(ctx, db, query, [][]any{{int64(1)}, {int64(1)}})

	require.NoError(t, err)
	require.Equal(t, BatchTransactionCommitted, result.Outcome.TransactionState)
	require.Len(t, result.Items, 2)
	require.Equal(t, BatchItemExecuted, result.Items[0].State)
	require.Equal(t, BatchItemExecuted, result.Items[1].State)
	require.NotNil(t, txCtx)
	require.Len(t, txCtx.RoundImages.BeofreImages(), 1)
	require.Len(t, txCtx.RoundImages.AfterImages(), 1)
	require.Equal(t, map[string]struct{}{"ACCOUNT:1": {}}, txCtx.LockKeys)
	require.NotNil(t, branchUndoLog)
	require.Len(t, branchUndoLog.Logs, 1)
	require.EqualValues(t, types.SQLTypeDelete, branchUndoLog.Logs[0].SQLType)
	require.Len(t, branchUndoLog.Logs[0].BeforeImage.Rows, 1)
	require.Empty(t, branchUndoLog.Logs[0].AfterImage.Rows)

	expectBatchATUndoExecutions(t, ctrl, mockConn, batchATUndoExpectation{
		statement: "INSERT INTO account",
		undoArgs:  []driver.NamedValue{{Ordinal: 1, Value: int64(100)}, {Ordinal: 2, Value: int64(1)}},
	})
	executeBatchATUndo(t, db, branchUndoLog)
}

func TestExecBatchContextWithSeataATDriverReversesCompositePrimaryKeyInsert(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockConn, mockTx, mockMgr := newBatchSeataATTestDB(t, ctrl)
	defer db.Close()
	pk1 := types.ColumnMeta{ColumnName: "pk1", DatabaseTypeString: "BIGINT"}
	pk2 := types.ColumnMeta{ColumnName: "pk2", DatabaseTypeString: "BIGINT"}
	datasource.RegisterTableCache(types.DBTypeMySQL, batchATTableCache{tableMeta: &types.TableMeta{
		TableName: "account", ColumnNames: []string{"pk1", "balance", "pk2"},
		Columns: map[string]types.ColumnMeta{
			"pk1": pk1, "balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"}, "pk2": pk2,
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{pk2, pk1}},
		},
	}})

	previousUndoConfig := undo.UndoConfig
	undo.UndoConfig = undo.Config{LogSerialization: "json", LogTable: "undo_log"}
	t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })
	ctx := tm.InitSeataContext(context.Background())
	xid := uuid.NewString()
	tm.SetXID(ctx, xid)
	const query = "INSERT INTO account (pk2, balance, pk1) VALUES (?, ?, ?)"

	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
	mockConn.EXPECT().ExecContext(gomock.Any(), query, []driver.NamedValue{
		{Ordinal: 1, Value: int64(22)}, {Ordinal: 2, Value: int64(300)}, {Ordinal: 3, Value: int64(11)},
	}).Times(1).Return(driver.RowsAffected(1), nil)
	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), []driver.NamedValue{
		{Ordinal: 1, Value: int64(11)}, {Ordinal: 2, Value: int64(22)},
	}).Times(1).DoAndReturn(func(_ context.Context, imageQuery string, _ []driver.NamedValue) (driver.Rows, error) {
		require.NotContains(t, imageQuery, "FOR UPDATE")
		return &batchATRows{
			columns: []string{"pk1", "balance", "pk2"},
			data:    [][]driver.Value{{int64(11), int64(300), int64(22)}},
		}, nil
	})

	const branchID = int64(125)
	registerCall := mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, param rm.BranchRegisterParam) (int64, error) {
			require.Equal(t, xid, param.Xid)
			require.Equal(t, "ACCOUNT:11_22;", param.LockKeys)
			return branchID, nil
		},
	)
	undoStmt := mock.NewMockTestDriverStmt(ctrl)
	prepareUndoCall := mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(1).Return(undoStmt, nil)
	undoStmt.EXPECT().Close().Times(1).Return(nil)
	var branchUndoLog *undo.BranchUndoLog
	flushUndoCall := undoStmt.EXPECT().ExecContext(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, args []driver.NamedValue) (driver.Result, error) {
			rollbackInfo, ok := args[3].Value.([]byte)
			require.True(t, ok)
			var err error
			branchUndoLog, err = (&undoparser.JsonParser{}).Decode(rollbackInfo)
			require.NoError(t, err)
			return driver.ResultNoRows, nil
		},
	)
	commitCall := mockTx.EXPECT().Commit().Times(1).Return(nil)
	reportCall := mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	gomock.InOrder(registerCall, prepareUndoCall, flushUndoCall, commitCall, reportCall)

	result, err := ExecBatchContext(ctx, db, query, [][]any{{int64(22), int64(300), int64(11)}})
	require.NoError(t, err)
	require.Equal(t, BatchTransactionCommitted, result.Outcome.TransactionState)
	require.NotNil(t, branchUndoLog)
	require.Len(t, branchUndoLog.Logs, 1)
	require.EqualValues(t, types.SQLTypeInsert, branchUndoLog.Logs[0].SQLType)

	expectBatchATUndoExecutions(t, ctrl, mockConn, batchATUndoExpectation{
		statement: "WHERE pk1 = ?  and pk2 = ?", undoArgs: []driver.NamedValue{
			{Ordinal: 1, Value: int64(11)}, {Ordinal: 2, Value: int64(22)},
		},
	})
	executeBatchATUndo(t, db, branchUndoLog)
}

func TestExecBatchContextWithSeataATDriverRollsBackOwnedTransactionOnFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockConn, mockTx, mockMgr := newBatchSeataATTestDB(t, ctrl)
	defer db.Close()
	datasource.RegisterTableCache(types.DBTypeMySQL, batchATTableCache{})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	query := "UPDATE account SET balance = ? WHERE id = ?"
	execErr := errors.New("execute failed")

	var execCount int32
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
	expectBatchATImageQueries(t, mockConn, [][]driver.Value{
		{int64(1), int64(100)}, {int64(1), int64(110)}, {int64(2), int64(200)},
	})
	mockConn.EXPECT().ExecContext(gomock.Any(), query, gomock.Any()).
		Times(2).DoAndReturn(func(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
		count := atomic.AddInt32(&execCount, 1)
		if count == 2 {
			return nil, execErr
		}
		return driver.RowsAffected(1), nil
	})

	mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(0)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).Times(0)
	mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(0)
	mockTx.EXPECT().Commit().Times(0)
	mockTx.EXPECT().Rollback().Times(1).Return(nil)
	result, err := ExecBatchContext(ctx, db, query, [][]any{
		{int64(110), int64(1)}, {int64(220), int64(2)}, {int64(330), int64(3)},
	})

	require.Error(t, err)
	require.ErrorIs(t, err, execErr)
	require.Contains(t, err.Error(), "batch item 1")
	require.Equal(t, BatchItemExecuted, result.Items[0].State)
	require.Equal(t, BatchItemFailed, result.Items[1].State)
	require.Equal(t, BatchItemNotExecuted, result.Items[2].State)

	// item2 must never execute
	require.Equal(t, int32(2), atomic.LoadInt32(&execCount))
}

func TestExecBatchWithSeataATDriverStopsOnInsertAfterImageValidationFailure(t *testing.T) {
	for _, test := range []struct {
		name  string
		owned bool
		state BatchTransactionState
	}{
		{name: "owned", owned: true, state: BatchTransactionRolledBack},
		{name: "caller owned", state: BatchTransactionPending},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			db, mockConn, mockTx, mockMgr := newBatchSeataATTestDB(t, ctrl)
			defer db.Close()
			datasource.RegisterTableCache(types.DBTypeMySQL, batchATTableCache{})

			ctx := tm.InitSeataContext(context.Background())
			tm.SetXID(ctx, uuid.NewString())
			const query = "INSERT INTO account (id, balance) VALUES (?, ?)"
			batchArgs := [][]any{{int64(1), int64(100)}, {int64(2), int64(200)}, {int64(3), int64(300)}}

			mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
			execCount := 0
			mockConn.EXPECT().ExecContext(gomock.Any(), query, gomock.Any()).Times(2).
				DoAndReturn(func(context.Context, string, []driver.NamedValue) (driver.Result, error) {
					execCount++
					return driver.RowsAffected(1), nil
				})
			queryCount := 0
			mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Not("SELECT VERSION()"), gomock.Any()).Times(2).
				DoAndReturn(func(_ context.Context, imageQuery string, args []driver.NamedValue) (driver.Rows, error) {
					require.NotContains(t, imageQuery, "FOR UPDATE")
					queryCount++
					id := args[0].Value
					if queryCount == 2 {
						id = int64(99)
					}
					return &batchATRows{columns: []string{"id", "balance"}, data: [][]driver.Value{{id, int64(queryCount * 100)}}}, nil
				})
			mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(0)
			mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).Times(0)
			mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(0)
			mockTx.EXPECT().Commit().Times(0)
			mockTx.EXPECT().Rollback().Times(1).Return(nil)

			var result BatchResult
			var err error
			if test.owned {
				result, err = ExecBatchContext(ctx, db, query, batchArgs)
			} else {
				tx, beginErr := db.BeginTx(ctx, nil)
				require.NoError(t, beginErr)
				result, err = ExecBatchInTxContext(ctx, tx, query, batchArgs)
				require.NoError(t, tx.Rollback())
			}

			require.ErrorContains(t, err, "unexpected primary key [99]")
			require.Equal(t, test.state, result.Outcome.TransactionState)
			require.Equal(t, []BatchItemState{BatchItemExecuted, BatchItemFailed, BatchItemNotExecuted}, []BatchItemState{
				result.Items[0].State, result.Items[1].State, result.Items[2].State,
			})
			require.Equal(t, 2, execCount)
			require.Equal(t, 2, queryCount)
		})
	}
}

func TestExecBatchContextMapsATPreCommitOutcome(t *testing.T) {
	for _, test := range []struct {
		name          string
		rollbackErr   error
		expectedState BatchTransactionState
	}{
		{name: "rolled back", expectedState: BatchTransactionRolledBack},
		{name: "rollback failed", rollbackErr: errors.New("rollback failed"), expectedState: BatchTransactionRollbackFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			CleanTxHooks()
			t.Cleanup(CleanTxHooks)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			db, mockConn, mockTx, _ := newBatchSeataATTestDB(t, ctrl)
			defer db.Close()

			commitErr := errors.New("before commit failed")
			RegisterTxHook(&mockTxHook{beforeCommit: func(*Tx) error { return commitErr }})

			mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
			mockConn.EXPECT().ExecContext(gomock.Any(), "SELECT ?", gomock.Any()).Times(1).Return(driver.ResultNoRows, nil)
			mockTx.EXPECT().Rollback().Times(1).Return(test.rollbackErr)

			ctx := tm.InitSeataContext(context.Background())
			tm.SetXID(ctx, uuid.NewString())
			result, err := ExecBatchContext(ctx, db, "SELECT ?", [][]any{{"item0"}})

			require.ErrorIs(t, err, commitErr)
			require.Equal(t, BatchPhaseCommit, result.Outcome.FailurePhase)
			require.Equal(t, NoFailedBatchItem, result.Outcome.FailedIndex)
			require.Equal(t, test.expectedState, result.Outcome.TransactionState)
			require.Equal(t, BatchItemExecuted, result.Items[0].State)

			var batchErr *BatchError
			require.ErrorAs(t, err, &batchErr)
			if test.rollbackErr == nil {
				require.NoError(t, batchErr.RollbackErr)
			} else {
				require.ErrorIs(t, err, test.rollbackErr)
				require.ErrorIs(t, batchErr.RollbackErr, test.rollbackErr)
			}
		})
	}
}

func TestExecBatchInTxContextWithSeataATDriverAllowsFollowingExecInSameTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockConn, mockTx, _ := newBatchSeataATTestDB(t, ctrl)
	defer db.Close()

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	batchQuery := "SELECT ?"
	normalQuery := "SELECT ?"

	var executedArgs []any

	// Caller creates one transaction.
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)

	// 2 batch items + 1 ordinary SQL .
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(3).
		DoAndReturn(func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			executedArgs = append(executedArgs, args[0].Value)
			return driver.ResultNoRows, nil
		})

	mockTx.EXPECT().Commit().Times(1).Return(nil)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = ExecBatchInTxContext(ctx, tx, batchQuery, [][]any{{"item0"}, {"item1"}})
	require.NoError(t, err)

	// Batch execution must not close caller-owned transaction
	_, err = tx.ExecContext(ctx, normalQuery, "normal")
	require.NoError(t, err)

	require.NoError(t, tx.Commit())
	require.Equal(t, []any{"item0", "item1", "normal"}, executedArgs)
}

func TestExecBatchInTxContextKeepsATModeSelectedAtBegin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockConn, mockTx, _ := newBatchSeataATTestDB(t, ctrl)
	defer db.Close()

	beginCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(beginCtx, uuid.NewString())
	const query = "REPLACE INTO account (id, balance) VALUES (?, ?)"

	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
	mockConn.EXPECT().ExecContext(gomock.Any(), query, gomock.Any()).Times(0)
	mockTx.EXPECT().Rollback().Times(1).Return(nil)

	tx, err := db.BeginTx(beginCtx, nil)
	require.NoError(t, err)
	result, err := ExecBatchInTxContext(context.Background(), tx, query, [][]any{{int64(1), int64(100)}})

	require.Error(t, err)
	require.Contains(t, err.Error(), "AT mode currently does not support REPLACE INTO statement")
	require.Equal(t, BatchItemFailed, result.Items[0].State)
	require.Equal(t, BatchTransactionPending, result.Outcome.TransactionState)
	require.NoError(t, tx.Rollback())
}

func TestExecBatchInTxContextKeepsLocalModeSelectedAtBegin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db, mockConn, mockTx, _ := newBatchSeataATTestDB(t, ctrl)
	defer db.Close()

	execCtx := tm.InitSeataContext(context.Background())
	tm.SetXID(execCtx, uuid.NewString())
	const query = "REPLACE INTO account (id, balance) VALUES (?, ?)"

	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
	mockConn.EXPECT().ExecContext(gomock.Any(), query, gomock.Any()).Times(1).DoAndReturn(
		func(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
			require.Equal(t, []driver.NamedValue{
				{Ordinal: 1, Value: int64(1)},
				{Ordinal: 2, Value: int64(100)},
			}, args)
			return driver.RowsAffected(1), nil
		},
	)
	mockTx.EXPECT().Rollback().Times(1).Return(nil)

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	result, err := ExecBatchInTxContext(execCtx, tx, query, [][]any{{int64(1), int64(100)}})

	require.NoError(t, err)
	require.Equal(t, BatchItemExecuted, result.Items[0].State)
	require.Equal(t, BatchTransactionPending, result.Outcome.TransactionState)
	require.NoError(t, tx.Rollback())
}
