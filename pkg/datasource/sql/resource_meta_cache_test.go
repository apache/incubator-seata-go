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
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/go-sql-driver/mysql"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	atexec "seata.apache.org/seata-go/v2/pkg/datasource/sql/exec/at"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

type resourceTestMetaReader struct{ id string }

func (r *resourceTestMetaReader) ResolveTableMetaKey(context.Context, driver.Conn, types.TableRef) (types.TableMetaKey, error) {
	return types.TableMetaKey{DBName: r.id, TableName: "account"}, nil
}

func (r *resourceTestMetaReader) GetTableMeta(context.Context, types.TableMetaKey) (*types.TableMeta, error) {
	return &types.TableMeta{TableName: "account"}, nil
}

func (r *resourceTestMetaReader) Destroy() error { return nil }

type poolBoundTestReader struct {
	*resourceTestMetaReader
	db *sql.DB
}

func (r *poolBoundTestReader) GetTableMeta(ctx context.Context, _ types.TableMetaKey) (*types.TableMeta, error) {
	var source string
	if err := r.db.QueryRowContext(ctx, "SELECT metadata_source").Scan(&source); err != nil {
		return nil, err
	}
	return &types.TableMeta{TableName: source}, nil
}

func TestNewExecContextUsesItsResourceMetaReader(t *testing.T) {
	readerA := &resourceTestMetaReader{id: "source_a"}
	readerB := &resourceTestMetaReader{id: "source_b"}
	connA := &Conn{res: &DBResource{metaCache: readerA}}
	connB := &Conn{res: &DBResource{metaCache: readerB}}

	if got := connA.newExecContext(nil, "UPDATE account SET value = 1", nil, nil).TableMetaReader; got != readerA {
		t.Fatalf("source A used reader %v, want its own reader %v", got, readerA)
	}
	if got := connB.newExecContext(nil, "UPDATE account SET value = 2", nil, nil).TableMetaReader; got != readerB {
		t.Fatalf("source B used reader %v, want its own reader %v", got, readerB)
	}
	if got := connA.newExecContext(nil, "UPDATE account SET value = 3", nil, nil).TableMetaReader; got != readerA {
		t.Fatalf("source A changed reader after source B was used: got %v", got)
	}
}

type disconnectedTestConnector struct{}

func (disconnectedTestConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errors.New("test connector has no server")
}

func (disconnectedTestConnector) Driver() driver.Driver { return &mysql.MySQLDriver{} }

func TestOpenConnectorKeepsSeparateCachesForSameDBType(t *testing.T) {
	ctrl := gomock.NewController(t)
	initMockResourceManager(branch.BranchTypeAT, ctrl)
	descriptor := mySQLDriverDescriptor
	descriptor.newTableMetaCache = func(db *sql.DB, dbName string) datasource.TableMetaCache {
		return &poolBoundTestReader{resourceTestMetaReader: &resourceTestMetaReader{id: dbName}, db: db}
	}
	d := &seataDriver{branchType: branch.BranchTypeAT, transType: types.ATMode, descriptor: descriptor}
	connector := disconnectedTestConnector{}
	open := func(dsn string) (*DBResource, sqlmock.Sqlmock) {
		t.Helper()
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		proxy, err := d.getOpenConnectorProxy(connector, types.DBTypeMySQL, db, dsn)
		if err != nil {
			t.Fatal(err)
		}
		return proxy.(*seataConnector).res, mock
	}

	resourceA, mockA := open("root:pass@tcp(127.0.0.1:3306)/source_a")
	resourceB, mockB := open("root:pass@tcp(127.0.0.1:3306)/source_b")
	if resourceA.metaCache == nil || resourceB.metaCache == nil {
		t.Fatal("each resource must own a table metadata cache")
	}
	if resourceA.metaCache == resourceB.metaCache {
		t.Fatal("same-type resources share a metadata cache")
	}
	mockA.ExpectQuery("SELECT metadata_source").WillReturnRows(sqlmock.NewRows([]string{"source"}).AddRow("source_a"))
	mockA.ExpectQuery("SELECT metadata_source").WillReturnRows(sqlmock.NewRows([]string{"source"}).AddRow("source_a"))
	mockB.ExpectQuery("SELECT metadata_source").WillReturnRows(sqlmock.NewRows([]string{"source"}).AddRow("source_b"))
	lookup := func(resource *DBResource) string {
		t.Helper()
		execCtx := (&Conn{res: resource}).newExecContext(nil, "UPDATE account SET value = 1", nil, nil)
		key, err := execCtx.TableMetaReader.ResolveTableMetaKey(context.Background(), execCtx.Conn, types.TableRef{TableName: "account"})
		if err != nil {
			t.Fatal(err)
		}
		meta, err := execCtx.TableMetaReader.GetTableMeta(context.Background(), key)
		if err != nil {
			t.Fatal(err)
		}
		return meta.TableName
	}
	for _, tc := range []struct {
		resource *DBResource
		want     string
	}{{resourceA, "source_a"}, {resourceB, "source_b"}, {resourceA, "source_a"}} {
		if got := lookup(tc.resource); got != tc.want {
			t.Fatalf("metadata came from %q, want %q", got, tc.want)
		}
	}
	if err := mockA.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if err := mockB.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type recordingUndoManager struct {
	undo.UndoLogManager
	readers []types.TableMetaReader
}

func (m *recordingUndoManager) RunUndo(_ context.Context, _ string, _ int64, _ *sql.DB, _ string, reader types.TableMetaReader) error {
	m.readers = append(m.readers, reader)
	return nil
}

func TestBranchRollbackUsesItsResourceMetaReader(t *testing.T) {
	readerA := &resourceTestMetaReader{id: "source_a"}
	readerB := &resourceTestMetaReader{id: "source_b"}
	manager := &ATSourceManager{}
	manager.resourceCache.Store("source_a", &DBResource{dbType: types.DBTypeMySQL, metaCache: readerA})
	manager.resourceCache.Store("source_b", &DBResource{dbType: types.DBTypeMySQL, metaCache: readerB})
	undoManager := &recordingUndoManager{}
	patch := gomonkey.ApplyFunc(undo.GetUndoLogManager, func(types.DBType) (undo.UndoLogManager, error) {
		return undoManager, nil
	})
	defer patch.Reset()

	for _, resourceID := range []string{"source_a", "source_b", "source_a"} {
		status, err := manager.BranchRollback(context.Background(), rm.BranchResource{ResourceId: resourceID})
		if err != nil || status != branch.BranchStatusPhasetwoRollbacked {
			t.Fatalf("rollback %s returned status %v, error %v", resourceID, status, err)
		}
	}
	if len(undoManager.readers) != 3 || undoManager.readers[0] != readerA || undoManager.readers[1] != readerB || undoManager.readers[2] != readerA {
		t.Fatalf("rollback readers = %v, want A, B, A", undoManager.readers)
	}
}

type capturedSQLBytes struct{ value *[]byte }

func (c capturedSQLBytes) Match(value driver.Value) bool {
	data, ok := value.([]byte)
	if ok {
		*c.value = append([]byte(nil), data...)
	}
	return ok
}

func TestATExecutionAndRollbackKeepProductionCachesOnTheirResources(t *testing.T) {
	previousUndoConfig := undo.UndoConfig
	undo.UndoConfig = undo.Config{LogSerialization: "json", LogTable: "undo_log", OnlyCareUpdateColumns: true}
	t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })

	ctrl := gomock.NewController(t)
	resourceManager := mock.NewMockDataSourceManager(ctrl)
	resourceManager.SetBranchType(branch.BranchTypeAT)
	resourceManager.EXPECT().RegisterResource(gomock.Any()).Times(2).Return(nil)
	registerResourceManagerForTest(t, resourceManager)

	d := &seataDriver{branchType: branch.BranchTypeAT, transType: types.ATMode, descriptor: mySQLDriverDescriptor}
	type testResource struct {
		name string
		db   *sql.DB
		mock sqlmock.Sqlmock
		res  *DBResource
	}
	open := func(name string) *testResource {
		t.Helper()
		db, sqlMock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		proxy, err := d.getOpenConnectorProxy(disconnectedTestConnector{}, types.DBTypeMySQL, db,
			"root:pass@tcp(127.0.0.1:3306)/"+name)
		require.NoError(t, err)
		return &testResource{name: name, db: db, mock: sqlMock, res: proxy.(*seataConnector).res}
	}
	a, b := open("source_a"), open("source_b")
	require.NotSame(t, a.res.metaCache, b.res.metaCache)

	manager := &ATSourceManager{}
	manager.resourceCache.Store(a.res.GetResourceId(), a.res)
	manager.resourceCache.Store(b.res.GetResourceId(), b.res)
	undoManager, err := undo.GetUndoLogManager(types.DBTypeMySQL)
	require.NoError(t, err)

	const query = "UPDATE account SET balance = ? WHERE id = ?"
	for i, step := range []struct {
		resource *testResource
		oldValue any
		newValue any
		dataType string
	}{
		{a, int64(100), int64(110), "BIGINT"},
		{b, "old", "new", "VARCHAR"},
		{a, int64(110), int64(120), "BIGINT"},
	} {
		state := step.resource
		id := int64(1)
		xid := fmt.Sprintf("%s-xid-%d", state.name, i)
		var logContext, rollbackInfo []byte

		state.mock.ExpectQuery(regexp.QuoteMeta("SELECT DATABASE()")).
			WillReturnRows(sqlmock.NewRows([]string{"DATABASE()"}).AddRow(state.name))
		if i < 2 {
			state.mock.ExpectPrepare("FROM INFORMATION_SCHEMA.COLUMNS").ExpectQuery().
				WithArgs(state.name, "account").WillReturnRows(sqlmock.NewRows([]string{
				"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE", "COLUMN_TYPE",
				"COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
			}).AddRow("account", state.name, "id", "BIGINT", "BIGINT(20)", "PRI", "NO", nil, "").
				AddRow("account", state.name, "balance", step.dataType, step.dataType, "", "NO", nil, ""))
			state.mock.ExpectPrepare("FROM `INFORMATION_SCHEMA`.`STATISTICS`").ExpectQuery().
				WithArgs(state.name, "account").WillReturnRows(sqlmock.NewRows(
				[]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).AddRow("PRIMARY", "id", 0))
		}
		state.mock.ExpectQuery("SELECT SQL_NO_CACHE .* FOR UPDATE").
			WillReturnRows(sqlmock.NewRows([]string{"balance", "id"}).AddRow(step.oldValue, id))
		state.mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(step.newValue, id).
			WillReturnResult(sqlmock.NewResult(0, 1))
		state.mock.ExpectQuery("SELECT balance,id FROM `" + state.name + "`\\.`account`").WithArgs(id).
			WillReturnRows(sqlmock.NewRows([]string{"balance", "id"}).AddRow(step.newValue, id))
		state.mock.ExpectPrepare("INSERT INTO undo_log").WillBeClosed().ExpectExec().
			WithArgs(sqlmock.AnyArg(), xid, capturedSQLBytes{&logContext}, capturedSQLBytes{&rollbackInfo}, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		txCtx := types.NewTxCtx()
		txCtx.TransactionMode = types.ATMode
		txCtx.XID, txCtx.BranchID = xid, 1
		sqlConn, err := state.db.Conn(context.Background())
		require.NoError(t, err)
		err = sqlConn.Raw(func(raw any) error {
			driverConn := raw.(driver.Conn)
			conn := &Conn{res: state.res, targetConn: driverConn, dbType: types.DBTypeMySQL, dbName: state.name}
			execCtx := conn.newExecContext(txCtx, query, nil, []driver.NamedValue{
				{Ordinal: 1, Value: step.newValue}, {Ordinal: 2, Value: id},
			})
			_, err := (&atexec.ATExecutor{}).ExecWithNamedValue(context.Background(), execCtx,
				func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
					result, err := driverConn.(driver.ExecerContext).ExecContext(ctx, query, args)
					if err != nil {
						return nil, err
					}
					return types.NewResult(types.WithResult(result)), nil
				})
			if err != nil {
				return err
			}
			wantKey := types.TableMetaKey{DBName: state.name, TableName: "account"}
			if execCtx.TableMetaKey == nil || *execCtx.TableMetaKey != wantKey {
				return fmt.Errorf("AT execution table metadata key = %v, want %v", execCtx.TableMetaKey, wantKey)
			}
			return undoManager.FlushUndoLog(txCtx, driverConn)
		})
		require.NoError(t, err)
		require.NoError(t, sqlConn.Close())
		require.NotEmpty(t, logContext)
		require.NotEmpty(t, rollbackInfo)

		state.mock.ExpectBegin()
		state.mock.ExpectPrepare("SELECT .* FROM undo_log .* FOR UPDATE").WillBeClosed().
			ExpectQuery().WithArgs(int64(1), xid).WillReturnRows(sqlmock.NewRows(
			[]string{"branch_id", "xid", "context", "rollback_info", "log_status"}).
			AddRow(1, xid, logContext, rollbackInfo, 0))
		state.mock.ExpectPrepare("UPDATE `"+state.name+"`\\.`account`").WillBeClosed().
			ExpectExec().WithArgs(step.oldValue, id).WillReturnResult(sqlmock.NewResult(0, 1))
		state.mock.ExpectPrepare("DELETE FROM undo_log").WillBeClosed().
			ExpectExec().WithArgs(int64(1), xid).WillReturnResult(sqlmock.NewResult(0, 1))
		state.mock.ExpectCommit()
		status, err := manager.BranchRollback(context.Background(), rm.BranchResource{
			ResourceId: state.res.GetResourceId(), Xid: xid, BranchId: 1,
		})
		require.NoError(t, err)
		require.EqualValues(t, branch.BranchStatusPhasetwoRollbacked, status)
	}
	require.NoError(t, a.mock.ExpectationsWereMet())
	require.NoError(t, b.mock.ExpectationsWereMet())
}
