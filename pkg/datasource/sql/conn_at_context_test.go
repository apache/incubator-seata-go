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
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	undoparser "seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/parser"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

const atContextTestResourceID = "at-context-resource"

func atContextWithXID(xid string) context.Context {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, xid)
	return ctx
}

func newATContextTestDB(t *testing.T, ctrl *gomock.Controller, dbType types.DBType) (
	*sql.DB, *mock.MockTestDriverConn, *mock.MockDataSourceManager, *mock.MockTableMetaCache,
) {
	t.Helper()

	targetConn := mock.NewMockTestDriverConn(ctrl)
	targetConn.EXPECT().Close().Times(1).Return(nil)
	targetConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
	target := mock.NewMockTestDriverConnector(ctrl)
	target.EXPECT().Connect(gomock.Any()).Times(1).Return(targetConn, nil)

	manager := mock.NewMockDataSourceManager(ctrl)
	manager.SetBranchType(branch.BranchTypeAT)
	registerResourceManagerForTest(t, manager)

	meta := mock.NewMockTableMetaCache(ctrl)
	db := sql.OpenDB(&seataATConnector{seataConnector: &seataConnector{
		target: target,
		res: &DBResource{
			dbType: dbType, resourceID: atContextTestResourceID, metaCache: meta,
		},
		dbType: dbType,
		dbName: "seata_client",
	}})
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db, targetConn, manager, meta
}

func TestATContextRejectsConflictingXIDBeforeSQL(t *testing.T) {
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		t.Run(dbType.String(), func(t *testing.T) {
			for _, tc := range []struct {
				name     string
				beginCtx context.Context
			}{
				{"local", context.Background()},
				{"AT", atContextWithXID("bound-xid")},
			} {
				t.Run(tc.name, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					db, targetConn, _, _ := newATContextTestDB(t, ctrl, dbType)
					targetTx := mock.NewMockTestDriverTx(ctrl)
					targetStmt := mock.NewMockTestDriverStmt(ctrl)

					const query = "SELECT 1"
					targetConn.EXPECT().PrepareContext(gomock.Any(), query).Times(1).Return(targetStmt, nil)
					targetStmt.EXPECT().NumInput().AnyTimes().Return(0)
					targetStmt.EXPECT().Close().Times(1).Return(nil)

					// Prepare before BeginTx so Tx.StmtContext reuses a statement
					// that was created with a different transaction context.
					stmt, err := db.PrepareContext(context.Background(), query)
					require.NoError(t, err)
					defer stmt.Close()

					targetConn.EXPECT().BeginTx(tc.beginCtx, gomock.Any()).Times(1).Return(targetTx, nil)
					targetTx.EXPECT().Rollback().Times(1).Return(nil)
					tx, err := db.BeginTx(tc.beginCtx, nil)
					require.NoError(t, err)
					defer tx.Rollback()

					conflictCtx := atContextWithXID("different-xid")
					_, err = tx.ExecContext(conflictCtx, "UPDATE account SET balance = 10 WHERE id = 1")
					require.ErrorIs(t, err, ErrATTransactionContextConflict)
					_, err = tx.QueryContext(conflictCtx, query)
					require.ErrorIs(t, err, ErrATTransactionContextConflict)

					boundStmt := tx.StmtContext(tc.beginCtx, stmt)
					_, err = boundStmt.ExecContext(conflictCtx)
					require.ErrorIs(t, err, ErrATTransactionContextConflict)
					_, err = boundStmt.QueryContext(conflictCtx)
					require.ErrorIs(t, err, ErrATTransactionContextConflict)

					const txQuery = "SELECT 2"
					txStmt := mock.NewMockTestDriverStmt(ctrl)
					targetConn.EXPECT().PrepareContext(tc.beginCtx, txQuery).Times(1).Return(txStmt, nil)
					txStmt.EXPECT().NumInput().AnyTimes().Return(0)
					txStmt.EXPECT().Close().Times(1).Return(nil)
					preparedInTx, err := tx.PrepareContext(tc.beginCtx, txQuery)
					require.NoError(t, err)
					_, err = preparedInTx.ExecContext(conflictCtx)
					require.ErrorIs(t, err, ErrATTransactionContextConflict)

					targetStmt.EXPECT().ExecContext(tc.beginCtx, gomock.Any()).Times(1).Return(driver.ResultNoRows, nil)
					_, err = boundStmt.ExecContext(tc.beginCtx)
					require.NoError(t, err)
					require.NoError(t, tx.Rollback())
				})
			}
		})
	}
}

func expectATContextTable(t *testing.T, meta *mock.MockTableMetaCache, dbType types.DBType, targetConn driver.Conn) {
	t.Helper()

	key := types.TableMetaKey{DBName: "seata_client", TableName: "account"}
	if dbType == types.DBTypePostgreSQL {
		key.Schema = "public"
	}
	table, err := (batchATTableCache{}).GetTableMeta(context.Background(), key)
	require.NoError(t, err)
	meta.EXPECT().ResolveTableMetaKey(gomock.Any(), targetConn, types.TableRef{TableName: "account"}).
		Times(1).Return(key, nil)
	meta.EXPECT().GetTableMeta(gomock.Any(), key).MinTimes(1).Return(table, nil)
}

func TestATContextSelectForUpdateUsesBoundXID(t *testing.T) {
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		t.Run(dbType.String(), func(t *testing.T) {
			for _, scenario := range []string{"matched", "empty", "prepare fallback", "lock conflict"} {
				t.Run(scenario, func(t *testing.T) {
					ctrl := gomock.NewController(t)
					db, targetConn, manager, meta := newATContextTestDB(t, ctrl, dbType)
					targetTx := mock.NewMockTestDriverTx(ctrl)

					const xid = "bound-xid"
					beginCtx := atContextWithXID(xid)
					queryCtx := context.Background()
					targetConn.EXPECT().BeginTx(beginCtx, gomock.Any()).Times(1).Return(targetTx, nil)
					targetTx.EXPECT().Rollback().Times(1).Return(nil)
					expectATContextTable(t, meta, dbType, targetConn)
					savepoint := targetConn.EXPECT().ExecContext(queryCtx, gomock.Any(), gomock.Any()).DoAndReturn(
						func(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
							require.True(t, strings.HasPrefix(query, "SAVEPOINT "), query)
							return driver.ResultNoRows, nil
						},
					)

					placeholder := "?"
					if dbType == types.DBTypePostgreSQL {
						placeholder = "$1"
					}
					businessSQL := "SELECT balance FROM account WHERE id = " + placeholder + " FOR UPDATE"
					args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}
					var keyData, businessData [][]driver.Value
					var lockCall *gomock.Call
					if scenario != "empty" {
						keyData = [][]driver.Value{{int64(1)}}
						businessData = [][]driver.Value{{int64(100)}}
						lockCall = manager.EXPECT().LockQuery(queryCtx, rm.LockQueryParam{
							BranchType: branch.BranchTypeAT,
							ResourceId: atContextTestResourceID,
							Xid:        xid,
							LockKeys:   "account:1",
						}).Times(1).Return(scenario != "lock conflict", nil)
					}
					keyRows := &batchATRows{columns: []string{"id"}, data: keyData}
					keyQuery := targetConn.EXPECT().QueryContext(queryCtx, gomock.Not(businessSQL), args).After(savepoint)
					if scenario == "prepare fallback" {
						keyQuery.Return(nil, driver.ErrSkip)
						keyStmt := mock.NewMockTestDriverStmt(ctrl)
						prepare := targetConn.EXPECT().PrepareContext(queryCtx, gomock.Not(businessSQL)).After(keyQuery).DoAndReturn(
							func(_ context.Context, query string) (driver.Stmt, error) {
								require.Contains(t, query, "FOR UPDATE")
								require.Contains(t, query, placeholder)
								return keyStmt, nil
							},
						)
						query := keyStmt.EXPECT().QueryContext(queryCtx, args).After(prepare).Return(keyRows, nil)
						keyQuery = keyStmt.EXPECT().Close().After(query).Return(nil)
					} else {
						keyQuery.DoAndReturn(func(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
							require.Contains(t, query, "FOR UPDATE")
							require.Contains(t, query, placeholder)
							return keyRows, nil
						})
					}
					var businessRows driver.Rows = &batchATRows{columns: []string{"balance"}, data: businessData}
					var closeConflictRows *gomock.Call
					if scenario == "lock conflict" {
						conflictRows := mock.NewMockTestDriverRows(ctrl)
						closeConflictRows = conflictRows.EXPECT().Close().After(lockCall).Return(nil)
						businessRows = conflictRows
					}
					businessQuery := targetConn.EXPECT().QueryContext(queryCtx, businessSQL, args).After(keyQuery)
					if scenario == "prepare fallback" {
						businessQuery.Return(nil, driver.ErrSkip)
						stmt := mock.NewMockTestDriverStmt(ctrl)
						prepare := targetConn.EXPECT().PrepareContext(queryCtx, businessSQL).After(businessQuery).Return(stmt, nil)
						businessQuery = stmt.EXPECT().QueryContext(queryCtx, args).After(prepare).Return(businessRows, nil)
						stmt.EXPECT().Close().After(lockCall).Return(nil)
					} else {
						businessQuery.Return(businessRows, nil)
					}
					if lockCall != nil {
						lockCall.After(businessQuery)
					}
					if scenario == "lock conflict" {
						targetConn.EXPECT().ExecContext(queryCtx, gomock.Any(), gomock.Any()).After(closeConflictRows).DoAndReturn(
							func(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
								require.True(t, strings.HasPrefix(query, "ROLLBACK TO SAVEPOINT "), query)
								return driver.ResultNoRows, nil
							},
						)
					}

					tx, err := db.BeginTx(beginCtx, nil)
					require.NoError(t, err)
					defer tx.Rollback()
					rows, err := tx.QueryContext(queryCtx, businessSQL, int64(1))
					if scenario == "lock conflict" {
						require.ErrorContains(t, err, "lock conflict")
						require.Nil(t, rows)
						require.NoError(t, tx.Rollback())
						return
					}
					require.NoError(t, err)
					defer rows.Close()

					if scenario != "empty" {
						require.True(t, rows.Next())
						var balance int64
						require.NoError(t, rows.Scan(&balance))
						require.EqualValues(t, 100, balance)
					}
					require.False(t, rows.Next())
					require.NoError(t, rows.Err())
					require.NoError(t, rows.Close())
					require.NoError(t, tx.Rollback())
				})
			}
		})
	}
}

func TestATContextUpdateWithoutXIDRegistersAndFlushesBoundBranch(t *testing.T) {
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		t.Run(dbType.String(), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db, targetConn, manager, meta := newATContextTestDB(t, ctrl, dbType)
			targetTx := mock.NewMockTestDriverTx(ctrl)

			previousUndoConfig := undo.UndoConfig
			undo.UndoConfig = undo.Config{LogSerialization: "json", LogTable: "undo_log"}
			t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })

			const xid = "bound-xid"
			const branchID = int64(123)
			beginCtx := atContextWithXID(xid)
			execCtx := context.Background()
			const businessSQL = "UPDATE account SET balance = 110 WHERE id = 1"
			targetConn.EXPECT().BeginTx(beginCtx, gomock.Any()).Times(1).Return(targetTx, nil)
			expectATContextTable(t, meta, dbType, targetConn)

			beforeImageCall := targetConn.EXPECT().QueryContext(execCtx, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
				func(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
					require.Contains(t, query, "FOR UPDATE")
					return &batchATRows{columns: []string{"id", "balance"}, data: [][]driver.Value{{int64(1), int64(100)}}}, nil
				},
			)
			execCall := targetConn.EXPECT().ExecContext(execCtx, businessSQL, gomock.Any()).Times(1).Return(driver.RowsAffected(1), nil)
			afterImageCall := targetConn.EXPECT().QueryContext(execCtx, gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
				func(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
					require.NotContains(t, query, "FOR UPDATE")
					return &batchATRows{columns: []string{"id", "balance"}, data: [][]driver.Value{{int64(1), int64(110)}}}, nil
				},
			)
			registerCall := manager.EXPECT().BranchRegister(gomock.Any(), rm.BranchRegisterParam{
				BranchType: branch.BranchTypeAT,
				ResourceId: atContextTestResourceID,
				Xid:        xid,
				LockKeys:   "ACCOUNT:1;",
			}).Times(1).Return(branchID, nil)
			undoStmt := mock.NewMockTestDriverStmt(ctrl)
			prepareUndoCall := targetConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
				func(_ context.Context, query string) (driver.Stmt, error) {
					require.Contains(t, query, "INSERT INTO undo_log")
					return undoStmt, nil
				},
			)
			flushUndoCall := undoStmt.EXPECT().ExecContext(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
				func(_ context.Context, args []driver.NamedValue) (driver.Result, error) {
					require.Len(t, args, 5)
					require.EqualValues(t, branchID, args[0].Value)
					require.Equal(t, xid, args[1].Value)
					log, err := (&undoparser.JsonParser{}).Decode(args[3].Value.([]byte))
					require.NoError(t, err)
					require.Equal(t, xid, log.Xid)
					require.EqualValues(t, branchID, log.BranchID)
					require.Len(t, log.Logs, 1)
					require.EqualValues(t, 100, log.Logs[0].BeforeImage.Rows[0].GetColumnMap()["balance"].Value)
					require.EqualValues(t, 110, log.Logs[0].AfterImage.Rows[0].GetColumnMap()["balance"].Value)
					return driver.ResultNoRows, nil
				},
			)
			closeUndoCall := undoStmt.EXPECT().Close().Times(1).Return(nil)
			commitCall := targetTx.EXPECT().Commit().Times(1).Return(nil)
			reportCall := manager.EXPECT().BranchReport(gomock.Any(), rm.BranchReportParam{
				BranchType: branch.BranchTypeAT,
				Xid:        xid,
				BranchId:   branchID,
				Status:     branch.BranchStatusPhaseoneDone,
			}).Times(1).Return(nil)
			gomock.InOrder(beforeImageCall, execCall, afterImageCall, registerCall, prepareUndoCall, flushUndoCall, closeUndoCall, commitCall, reportCall)

			tx, err := db.BeginTx(beginCtx, nil)
			require.NoError(t, err)
			defer tx.Rollback()
			_, err = tx.ExecContext(execCtx, businessSQL)
			require.NoError(t, err)
			require.NoError(t, tx.Commit())
		})
	}
}

func TestATContextReusesConnectionAfterTransactionEnds(t *testing.T) {
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		t.Run(dbType.String(), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db, targetConn, _, _ := newATContextTestDB(t, ctrl, dbType)

			plainCtx := context.Background()
			firstCtx := atContextWithXID("first-xid")
			secondCtx := atContextWithXID("second-xid")
			beginErr := errors.New("begin failed")
			firstTx := mock.NewMockTestDriverTx(ctrl)
			secondTx := mock.NewMockTestDriverTx(ctrl)
			localTx := mock.NewMockTestDriverTx(ctrl)
			const query = "SELECT 1"
			gomock.InOrder(
				targetConn.EXPECT().BeginTx(firstCtx, gomock.Any()).Return(nil, beginErr),
				targetConn.EXPECT().BeginTx(secondCtx, gomock.Any()).Return(firstTx, nil),
				targetConn.EXPECT().ExecContext(secondCtx, query, gomock.Any()).Return(driver.ResultNoRows, nil),
				firstTx.EXPECT().Commit().Return(nil),
				targetConn.EXPECT().BeginTx(firstCtx, gomock.Any()).Return(secondTx, nil),
				targetConn.EXPECT().ExecContext(firstCtx, query, gomock.Any()).Return(driver.ResultNoRows, nil),
				secondTx.EXPECT().Rollback().Return(nil),
				targetConn.EXPECT().BeginTx(plainCtx, gomock.Any()).Return(localTx, nil),
				targetConn.EXPECT().ExecContext(plainCtx, query, gomock.Any()).Return(driver.ResultNoRows, nil),
				localTx.EXPECT().Commit().Return(nil),
			)

			conn, err := db.Conn(plainCtx)
			require.NoError(t, err)
			defer conn.Close()
			_, err = conn.BeginTx(firstCtx, nil)
			require.ErrorIs(t, err, beginErr)

			tx, err := conn.BeginTx(secondCtx, nil)
			require.NoError(t, err)
			_, err = tx.ExecContext(secondCtx, query)
			require.NoError(t, err)
			require.NoError(t, tx.Commit())

			tx, err = conn.BeginTx(firstCtx, nil)
			require.NoError(t, err)
			_, err = tx.ExecContext(secondCtx, query)
			require.ErrorIs(t, err, ErrATTransactionContextConflict)
			_, err = tx.ExecContext(firstCtx, query)
			require.NoError(t, err)
			require.NoError(t, tx.Rollback())

			tx, err = conn.BeginTx(plainCtx, nil)
			require.NoError(t, err)
			_, err = tx.ExecContext(firstCtx, query)
			require.ErrorIs(t, err, ErrATTransactionContextConflict)
			_, err = tx.ExecContext(plainCtx, query)
			require.NoError(t, err)
			require.NoError(t, tx.Commit())
		})
	}
}

func TestATContextAutomaticTransactionsUseEachExecutionContext(t *testing.T) {
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		t.Run(dbType.String(), func(t *testing.T) {
			CleanTxHooks()
			t.Cleanup(CleanTxHooks)
			ctrl := gomock.NewController(t)
			db, targetConn, _, _ := newATContextTestDB(t, ctrl, dbType)
			plainCtx := context.Background()
			prepareCtx := atContextWithXID("prepare-xid")
			firstCtx := atContextWithXID("first-xid")
			secondCtx := atContextWithXID("second-xid")

			var committedXIDs []string
			RegisterTxHook(&mockTxHook{beforeCommit: func(tx *Tx) error {
				require.Equal(t, types.ATMode, tx.tranCtx.TransactionMode)
				committedXIDs = append(committedXIDs, tx.tranCtx.XID)
				return nil
			}})

			const query = "SELECT 1"
			targetStmt := mock.NewMockTestDriverStmt(ctrl)
			// Expose only the legacy driver.Stmt methods to the proxy.
			legacyStmt := struct{ driver.Stmt }{targetStmt}
			targetStmt.EXPECT().NumInput().AnyTimes().Return(0)
			targetStmt.EXPECT().Close().Times(1).Return(nil)
			firstTx := mock.NewMockTestDriverTx(ctrl)
			secondTx := mock.NewMockTestDriverTx(ctrl)
			gomock.InOrder(
				targetConn.EXPECT().PrepareContext(prepareCtx, query).Return(legacyStmt, nil),
				targetConn.EXPECT().BeginTx(firstCtx, gomock.Any()).Return(firstTx, nil),
				targetConn.EXPECT().ExecContext(firstCtx, query, gomock.Any()).Return(driver.ResultNoRows, nil),
				firstTx.EXPECT().Commit().Return(nil),
				targetStmt.EXPECT().Query(gomock.Any()).Return(&atContextTypedRows{batchATRows{columns: []string{"id"}}}, nil),
				targetConn.EXPECT().BeginTx(secondCtx, gomock.Any()).Return(secondTx, nil),
				targetConn.EXPECT().ExecContext(secondCtx, query, gomock.Any()).Return(driver.ResultNoRows, nil),
				secondTx.EXPECT().Commit().Return(nil),
				targetStmt.EXPECT().Exec(gomock.Any()).Return(driver.ResultNoRows, nil),
			)

			conn, err := db.Conn(plainCtx)
			require.NoError(t, err)
			defer conn.Close()
			stmt, err := conn.PrepareContext(prepareCtx, query)
			require.NoError(t, err)
			defer stmt.Close()

			_, err = conn.ExecContext(firstCtx, query)
			require.NoError(t, err)
			require.Equal(t, []string{"first-xid"}, committedXIDs)

			rows, err := stmt.QueryContext(secondCtx)
			require.NoError(t, err)
			defer rows.Close()
			require.Equal(t, []string{"first-xid"}, committedXIDs)
			columnTypes, err := rows.ColumnTypes()
			require.NoError(t, err)
			require.Len(t, columnTypes, 1)
			require.Equal(t, "BIGINT", columnTypes[0].DatabaseTypeName())
			require.NoError(t, rows.Close())
			require.Equal(t, []string{"first-xid"}, committedXIDs)

			_, err = conn.ExecContext(secondCtx, query)
			require.NoError(t, err)
			_, err = stmt.Exec()
			require.NoError(t, err)
			require.Equal(t, []string{"first-xid", "second-xid"}, committedXIDs)
		})
	}
}

type atContextTypedRows struct{ batchATRows }

func (*atContextTypedRows) ColumnTypeDatabaseTypeName(int) string { return "BIGINT" }

func TestATContextIdlePreparedWriteDoesNotDropATIntent(t *testing.T) {
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		t.Run(dbType.String(), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db, targetConn, _, _ := newATContextTestDB(t, ctrl, dbType)
			xidCtx := atContextWithXID("bound-xid")
			prepareCtx := xidCtx
			execCtx := context.Background()
			if dbType == types.DBTypePostgreSQL {
				prepareCtx, execCtx = execCtx, prepareCtx
			}

			const query = "UPDATE account SET balance = 10 WHERE id = 1"
			targetStmt := mock.NewMockTestDriverStmt(ctrl)
			targetConn.EXPECT().PrepareContext(prepareCtx, query).Return(targetStmt, nil)
			targetStmt.EXPECT().NumInput().Return(0)
			targetStmt.EXPECT().Close().Return(nil)

			stmt, err := db.PrepareContext(prepareCtx, query)
			require.NoError(t, err)
			defer stmt.Close()
			_, err = stmt.ExecContext(execCtx)
			require.ErrorContains(t, err, "transaction started with BeginTx")
		})
	}
}

func TestATContextAutomaticExecRollsBackCommitPanic(t *testing.T) {
	CleanTxHooks()
	t.Cleanup(CleanTxHooks)
	ctrl := gomock.NewController(t)
	db, targetConn, _, _ := newATContextTestDB(t, ctrl, types.DBTypeMySQL)
	ctx := atContextWithXID("bound-xid")
	panicValue := errors.New("before commit panic")
	RegisterTxHook(&mockTxHook{beforeCommit: func(*Tx) error { panic(panicValue) }})

	targetTx := mock.NewMockTestDriverTx(ctrl)
	nextTx := mock.NewMockTestDriverTx(ctrl)
	gomock.InOrder(
		targetConn.EXPECT().BeginTx(ctx, gomock.Any()).Return(targetTx, nil),
		targetConn.EXPECT().ExecContext(ctx, "SELECT 1", gomock.Any()).Return(driver.ResultNoRows, nil),
		targetTx.EXPECT().Rollback().Return(nil),
		targetConn.EXPECT().BeginTx(context.Background(), gomock.Any()).Return(nextTx, nil),
		nextTx.EXPECT().Rollback().Return(nil),
	)

	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	require.PanicsWithValue(t, panicValue, func() { _, _ = conn.ExecContext(ctx, "SELECT 1") })

	tx, err := conn.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
}

func TestATContextLegacyBeginReleasesConnection(t *testing.T) {
	for _, finish := range []string{"commit", "rollback", "begin error"} {
		t.Run(finish, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db, targetConn, _, _ := newATContextTestDB(t, ctrl, types.DBTypePostgreSQL)
			ctx := atContextWithXID("bound-xid")
			legacyTx := mock.NewMockTestDriverTx(ctrl)
			nextTx := mock.NewMockTestDriverTx(ctrl)
			beginErr := errors.New("legacy begin failed")

			switch finish {
			case "commit":
				targetConn.EXPECT().Begin().Return(legacyTx, nil)
				commit := legacyTx.EXPECT().Commit().Return(nil)
				legacyTx.EXPECT().Rollback().After(commit).Return(sql.ErrTxDone)
			case "rollback":
				targetConn.EXPECT().Begin().Return(legacyTx, nil)
				legacyTx.EXPECT().Rollback().Return(nil)
			default:
				targetConn.EXPECT().Begin().Return(nil, beginErr)
			}
			targetConn.EXPECT().BeginTx(ctx, gomock.Any()).Return(nextTx, nil)
			nextTx.EXPECT().Rollback().Return(nil)

			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			defer conn.Close()
			require.NoError(t, conn.Raw(func(raw any) error {
				tx, err := raw.(driver.Conn).Begin()
				if finish == "begin error" {
					require.ErrorIs(t, err, beginErr)
					return nil
				}
				require.NoError(t, err)
				if finish == "commit" {
					defer func() { require.ErrorIs(t, tx.Rollback(), sql.ErrTxDone) }()
					return tx.Commit()
				}
				return tx.Rollback()
			}))

			tx, err := conn.BeginTx(ctx, nil)
			require.NoError(t, err)
			require.NoError(t, tx.Rollback())
		})
	}
}
