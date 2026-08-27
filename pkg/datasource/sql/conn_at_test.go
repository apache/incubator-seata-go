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
	"io"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource/postgres"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	atexec "seata.apache.org/seata-go/v2/pkg/datasource/sql/exec/at"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

func TestMain(m *testing.M) {
	Init()
	m.Run()
}

func TestATConnRejectsPreparedMultiSQL(t *testing.T) {
	ctrl := gomock.NewController(t)
	targetConn := mock.NewMockTestDriverConn(ctrl)

	conn := &ATConn{Conn: &Conn{targetConn: targetConn, dbType: types.DBTypeMySQL}}

	queries := []string{"INSERT INTO t_user(id) VALUES (?);" + "INSERT INTO t_user(id) VALUES (?)",
		"UPDATE t_user SET name = ? WHERE id = ?;" + "DELETE FROM t_user_log WHERE user_id = ?",
	}

	for _, query := range queries {
		stmt, err := conn.Prepare(query)
		assert.Nil(t, stmt)
		assert.ErrorIs(t, err, errATPreparedMultiSQLUnsupported)

		stmt, err = conn.PrepareContext(context.Background(), query)
		assert.Nil(t, stmt)
		assert.ErrorIs(t, err, errATPreparedMultiSQLUnsupported)
	}
}

func TestRejectATPreparedMultiSQLAllowsSingleStatement(t *testing.T) {
	err := rejectATPreparedMultiSQL(types.DBTypeMySQL, "UPDATE t_user SET name = ? WHERE id = ?")

	assert.NoError(t, err)
}

func TestRejectATPreparedMultiSQLSkipsParserWithoutSemicolon(t *testing.T) {
	originalParseATPreparedSQL := parseATPreparedSQL
	t.Cleanup(func() { parseATPreparedSQL = originalParseATPreparedSQL })

	parseCalls := 0
	parseATPreparedSQL = func(string) (*types.ParseContext, error) {
		parseCalls++
		return nil, errors.New("unexpected parser call")
	}

	err := rejectATPreparedMultiSQL(types.DBTypeMySQL, "UPDATE t_user SET name = ? WHERE id = ?")

	assert.NoError(t, err)
	assert.Zero(t, parseCalls)
}

func TestRejectATPreparedMultiSQLAllowsParserFailure(t *testing.T) {
	originalParseATPreparedSQL := parseATPreparedSQL
	t.Cleanup(func() { parseATPreparedSQL = originalParseATPreparedSQL })

	parserError := errors.New("unsupported SQL syntax")
	tests := []struct {
		name     string
		parseCtx *types.ParseContext
		err      error
	}{
		{
			name: "parser returns error",
			err:  parserError,
		},
		{
			name: "parser returns nil context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseCalls := 0
			parseATPreparedSQL = func(string) (*types.ParseContext, error) {
				parseCalls++
				return tt.parseCtx, tt.err
			}

			err := rejectATPreparedMultiSQL(types.DBTypeMySQL, "UPDATE t_user SET name = ?; unsupported syntax")

			assert.NoError(t, err)
			assert.Equal(t, 1, parseCalls)
		})
	}
}

func TestATConnAllowsPreparedMultiSQLForPostgreSQL(t *testing.T) {
	ctrl := gomock.NewController(t)
	targetConn := mock.NewMockTestDriverConn(ctrl)
	targetStmt := mock.NewMockTestDriverStmt(ctrl)
	query := "UPDATE t_user SET name = $1 WHERE id = $2;DELETE FROM t_user_log WHERE user_id = $3"
	conn := &ATConn{Conn: &Conn{targetConn: targetConn, dbType: types.DBTypePostgreSQL}}

	targetConn.EXPECT().Prepare(query).Return(targetStmt, nil)
	targetConn.EXPECT().PrepareContext(gomock.Any(), query).Return(targetStmt, nil)

	stmt, err := conn.Prepare(query)
	assert.NotNil(t, stmt)
	assert.NoError(t, err)

	stmt, err = conn.PrepareContext(context.Background(), query)
	assert.NotNil(t, stmt)
	assert.NoError(t, err)
}

func TestATConnQueryContextPrepareFallbackPreservesCloseErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	targetConn := mock.NewMockTestDriverConn(ctrl)
	targetStmt := mock.NewMockTestDriverStmt(ctrl)
	targetRows := mock.NewMockTestDriverRows(ctrl)
	ctx := context.Background()
	query := "SELECT id FROM t_user WHERE id = ?"
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}
	rowsCloseErr := errors.New("rows close failed")
	stmtCloseErr := errors.New("statement close failed")

	targetConn.EXPECT().QueryContext(ctx, query, args).Return(nil, driver.ErrSkip)
	targetConn.EXPECT().Prepare(query).Return(targetStmt, nil)
	targetStmt.EXPECT().QueryContext(ctx, args).Return(targetRows, nil)
	targetRows.EXPECT().Close().Return(rowsCloseErr)
	targetStmt.EXPECT().Close().Return(stmtCloseErr)

	conn := &ATConn{Conn: &Conn{
		res:        &DBResource{dbType: types.DBTypeMySQL},
		txCtx:      types.NewTxCtx(),
		targetConn: targetConn,
		dbType:     types.DBTypeMySQL,
	}}

	rows, err := conn.QueryContext(ctx, query, args)
	if !assert.NoError(t, err) || !assert.NotNil(t, rows) {
		return
	}
	closeErr := rows.Close()
	assert.ErrorIs(t, closeErr, rowsCloseErr)
	assert.ErrorIs(t, closeErr, stmtCloseErr)
}

type postgresMockRows struct {
	columns []string
	data    [][]driver.Value
	idx     int
}

func (m *postgresMockRows) Columns() []string {
	return m.columns
}

func (m *postgresMockRows) Close() error {
	return nil
}

func (m *postgresMockRows) Next(dest []driver.Value) error {
	if m.idx >= len(m.data) {
		return io.EOF
	}

	row := m.data[m.idx]
	for i := 0; i < len(row) && i < len(dest); i++ {
		dest[i] = row[i]
	}
	m.idx++
	return nil
}

func patchPostgresTableMeta(t *testing.T) *gomonkey.Patches {
	t.Helper()

	datasource.RegisterTableCache(types.DBTypePostgreSQL, postgres.NewTableMetaInstance(nil, "public"))
	return gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypePostgreSQL)), "GetTableMeta",
		func(_ *postgres.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
			return &types.TableMeta{
				TableName:   tableName,
				ColumnNames: []string{"id", "name", "age"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName:         "id",
						DatabaseTypeString: "INTEGER",
					},
					"name": {
						ColumnName:         "name",
						DatabaseTypeString: "VARCHAR",
					},
					"age": {
						ColumnName:         "age",
						DatabaseTypeString: "INTEGER",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "id",
						Columns: []types.ColumnMeta{
							{ColumnName: "id"},
						},
					},
				},
			}, nil
		})
}

func initAtConnTestResource(t *testing.T) (*gomock.Controller, *sql.DB, *mockSQLInterceptor, *mockTxHook) {
	ctrl := gomock.NewController(t)

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Commit().AnyTimes().Return(nil)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				rows := &mysqlMockRows{}
				rows.data = [][]interface{}{
					{"8.0.29"},
				}
				return rows, nil
			})
		baseMockConn(mockConn)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	mi := &mockSQLInterceptor{}

	ti := &mockTxHook{}

	exec.CleanCommonHook()
	CleanTxHooks()
	exec.RegisterCommonHook(mi)
	RegisterTxHook(ti)

	return ctrl, db, mi, ti
}

func initPostgresAtConnTestResource(t *testing.T) (*gomock.Controller, *sql.DB) {
	ctrl := gomock.NewController(t)

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
	if err != nil {
		t.Fatal(err)
	}

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Commit().AnyTimes().Return(nil)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		baseMockConn(mockConn)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	return ctrl, db
}

func TestATConn_PostgreSQLLocalPassThrough(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var execCount int32

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
				atomic.AddInt32(&execCount, 1)
				return driver.ResultNoRows, nil
			},
		)
		mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
		mockConn.EXPECT().Close().AnyTimes().Return(nil)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	_, err = conn.ExecContext(context.Background(), "SELECT 1")
	assert.NoError(t, err)

	assert.Equal(t, int32(1), atomic.LoadInt32(&execCount))
}

func TestATConn_PostgreSQLGlobalOnConflictReturnsUnsupportedError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Rollback().Times(1).Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(mockTx, nil)
		mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
		mockConn.EXPECT().Close().AnyTimes().Return(nil)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	_, err = conn.ExecContext(ctx, "INSERT INTO users(id, age) VALUES (1, 1) ON CONFLICT (id) DO NOTHING")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, atexec.ErrPostgreSQLATUnsupported))
}

func TestATConn_PostgreSQLGlobalInsertInTxUsesReturningExecutor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	metaPatch := patchPostgresTableMeta(t)
	defer metaPatch.Reset()

	var queryLog []string

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				queryLog = append(queryLog, query)
				switch {
				case query == "SELECT VERSION()":
					return &postgresMockRows{
						columns: []string{"version"},
						data:    [][]driver.Value{{"PostgreSQL 15.4"}},
					}, nil
				case strings.Contains(query, "INSERT INTO t_user(name, age) VALUES ($1, $2) RETURNING"):
					return &postgresMockRows{
						columns: []string{"id", "name", "age"},
						data:    [][]driver.Value{{int64(101), "alice", int64(18)}},
					}, nil
				default:
					return nil, errors.New("unexpected query: " + query)
				}
			},
		)
		mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
		mockConn.EXPECT().Close().AnyTimes().Return(nil)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	assert.NoError(t, err)

	res, err := tx.ExecContext(context.Background(), "INSERT INTO t_user(name, age) VALUES ($1, $2)", "alice", int64(18))
	assert.NoError(t, err)
	rowsAffected, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)
	assert.Contains(t, queryLog[len(queryLog)-1], `RETURNING "id", "name", "age"`)
	assert.NoError(t, tx.Rollback())
}

func TestATConn_PostgreSQLGlobalUpdateInTxUsesRealExecutor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	metaPatch := patchPostgresTableMeta(t)
	defer metaPatch.Reset()

	var (
		queryLog []string
		execLog  []string
	)

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Commit().AnyTimes().Return(nil)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
				execLog = append(execLog, query)
				return driver.ResultNoRows, nil
			},
		)
		mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				queryLog = append(queryLog, query)
				switch {
				case query == "SELECT VERSION()":
					return &postgresMockRows{
						columns: []string{"version"},
						data:    [][]driver.Value{{"PostgreSQL 15.4"}},
					}, nil
				case strings.HasPrefix(query, "SELECT * FROM t_user WHERE id=$1"):
					return &postgresMockRows{
						columns: []string{"name", "age", "id"},
						data:    [][]driver.Value{{"alice", int64(18), int64(1)}},
					}, nil
				case strings.Contains(query, `("id") IN (($1))`):
					return &postgresMockRows{
						columns: []string{"name", "age", "id"},
						data:    [][]driver.Value{{"bob", int64(19), int64(1)}},
					}, nil
				default:
					return nil, errors.New("unexpected query: " + query)
				}
			},
		)
		mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
		mockConn.EXPECT().Close().AnyTimes().Return(nil)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	assert.NoError(t, err)

	_, err = tx.ExecContext(context.Background(), "UPDATE t_user SET name = $1, age = $2 WHERE id = $3", "bob", int64(19), int64(1))
	assert.NoError(t, err)
	assert.Contains(t, queryLog, "SELECT * FROM t_user WHERE id=$1 FOR UPDATE")
	assert.Contains(t, queryLog[len(queryLog)-1], `("id") IN (($1))`)
	assert.Equal(t, []string{"UPDATE t_user SET name = $1, age = $2 WHERE id = $3"}, execLog)
	assert.NoError(t, tx.Rollback())
}

func TestATConn_PostgreSQLGlobalDeleteInTxUsesRealExecutor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	metaPatch := patchPostgresTableMeta(t)
	defer metaPatch.Reset()

	var (
		queryLog []string
		execLog  []string
	)

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
				execLog = append(execLog, query)
				return driver.ResultNoRows, nil
			},
		)
		mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				queryLog = append(queryLog, query)
				switch query {
				case "SELECT VERSION()":
					return &postgresMockRows{
						columns: []string{"version"},
						data:    [][]driver.Value{{"PostgreSQL 15.4"}},
					}, nil
				case "SELECT * FROM t_user WHERE id=$1 FOR UPDATE":
					return &postgresMockRows{
						columns: []string{"id", "name", "age"},
						data:    [][]driver.Value{{int64(1), "alice", int64(18)}},
					}, nil
				default:
					return nil, errors.New("unexpected query: " + query)
				}
			},
		)
		mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
		mockConn.EXPECT().Close().AnyTimes().Return(nil)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	assert.NoError(t, err)

	_, err = tx.ExecContext(context.Background(), "DELETE FROM t_user WHERE id = $1", int64(1))
	assert.NoError(t, err)
	assert.Contains(t, queryLog, "SELECT * FROM t_user WHERE id=$1 FOR UPDATE")
	assert.Equal(t, []string{"DELETE FROM t_user WHERE id = $1"}, execLog)
	assert.NoError(t, tx.Rollback())
}

func TestATConn_PostgreSQLSelectForUpdateInTxUsesRealExecutor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	mockMgr.EXPECT().LockQuery(gomock.Any(), gomock.Any()).AnyTimes().Return(true, nil)

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	metaPatch := patchPostgresTableMeta(t)
	defer metaPatch.Reset()

	var queryLog []string

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().Begin().AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).AnyTimes().Return(mockTx, nil)
		mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				queryLog = append(queryLog, query)
				switch {
				case query == "SELECT VERSION()":
					return &postgresMockRows{
						columns: []string{"version"},
						data:    [][]driver.Value{{"PostgreSQL 15.4"}},
					}, nil
				case strings.HasPrefix(query, "savepoint "):
					return nil, nil
				case query == "SELECT id FROM t_user WHERE id=$1 FOR UPDATE":
					return &postgresMockRows{
						columns: []string{"id"},
						data:    [][]driver.Value{{int64(1)}},
					}, nil
				case query == "SELECT id,name FROM t_user WHERE id = $1 FOR UPDATE":
					return &postgresMockRows{
						columns: []string{"id", "name"},
						data:    [][]driver.Value{{int64(1), "alice"}},
					}, nil
				default:
					return nil, errors.New("unexpected query: " + query)
				}
			},
		)
		mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
		mockConn.EXPECT().Close().AnyTimes().Return(nil)

		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
		return connector
	})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	assert.NoError(t, err)

	rows, err := tx.QueryContext(context.Background(), "SELECT id,name FROM t_user WHERE id = $1 FOR UPDATE", int64(1))
	assert.NoError(t, err)
	assert.NoError(t, rows.Close())
	assert.Contains(t, queryLog, "SELECT id,name FROM t_user WHERE id = $1 FOR UPDATE")
	assert.NoError(t, tx.Rollback())
}

func TestATConn_ExecContext(t *testing.T) {
	ctrl, db, mi, ti := initAtConnTestResource(t)
	defer func() {
		ctrl.Finish()
		db.Close()
		CleanTxHooks()
	}()

	t.Run("have xid", func(t *testing.T) {
		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.New().String())
		t.Logf("set xid=%s", tm.GetXID(ctx))

		beforeHook := func(_ context.Context, execCtx *types.ExecContext) {
			t.Logf("on exec xid=%s", execCtx.TxCtx.XID)
			assert.Equal(t, tm.GetXID(ctx), execCtx.TxCtx.XID)
			assert.Equal(t, types.ATMode, execCtx.TxCtx.TransactionMode)
		}
		mi.before = beforeHook

		var comitCnt int32
		beforeCommit := func(tx *Tx) error {
			atomic.AddInt32(&comitCnt, 1)
			assert.Equal(t, types.ATMode, tx.tranCtx.TransactionMode)
			return nil
		}
		ti.beforeCommit = beforeCommit

		conn, err := db.Conn(context.Background())
		assert.NoError(t, err)

		_, err = conn.ExecContext(ctx, "SELECT 1")
		assert.NoError(t, err)
		_, err = db.ExecContext(ctx, "SELECT 1")
		assert.NoError(t, err)

		assert.Equal(t, int32(2), atomic.LoadInt32(&comitCnt))
	})

	t.Run("not xid", func(t *testing.T) {
		mi.before = func(_ context.Context, execCtx *types.ExecContext) {
			assert.Equal(t, "", execCtx.TxCtx.XID)
			assert.Equal(t, types.Local, execCtx.TxCtx.TransactionMode)
		}

		var comitCnt int32
		ti.beforeCommit = func(tx *Tx) error {
			atomic.AddInt32(&comitCnt, 1)
			return nil
		}

		conn, err := db.Conn(context.Background())
		assert.NoError(t, err)

		_, err = conn.ExecContext(context.Background(), "SELECT 1")
		assert.NoError(t, err)
		_, err = db.ExecContext(context.Background(), "SELECT 1")
		assert.NoError(t, err)

		_, err = db.Exec("SELECT 1")
		assert.NoError(t, err)

		assert.Equal(t, int32(0), atomic.LoadInt32(&comitCnt))
	})
}

func TestATConn_PostgresSlice1Behavior(t *testing.T) {
	ctrl, db := initPostgresAtConnTestResource(t)
	defer func() {
		ctrl.Finish()
		db.Close()
		CleanTxHooks()
	}()

	t.Run("local execution stays pass-through", func(t *testing.T) {
		_, err := db.ExecContext(context.Background(), "UPDATE user SET name = 'alice' WHERE id = 1")
		assert.NoError(t, err)
	})

	t.Run("global insert-on-conflict remains controlled unsupported error", func(t *testing.T) {
		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.NewString())

		_, err := db.ExecContext(ctx, "INSERT INTO user(id, name) VALUES (1, 'alice') ON CONFLICT (id) DO NOTHING")
		assert.ErrorIs(t, err, atexec.ErrPostgreSQLATUnsupported)
	})

	t.Run("global tx begun earlier still returns unsupported insert-on-conflict error", func(t *testing.T) {
		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.NewString())

		tx, err := db.BeginTx(ctx, &sql.TxOptions{})
		assert.NoError(t, err)

		_, err = tx.ExecContext(context.Background(), "INSERT INTO user(id, name) VALUES (1, 'alice') ON CONFLICT (id) DO NOTHING")
		assert.ErrorIs(t, err, atexec.ErrPostgreSQLATUnsupported)

		assert.NoError(t, tx.Rollback())
	})
}

func TestATConn_BeginTx(t *testing.T) {
	ctrl, db, mi, ti := initAtConnTestResource(t)
	defer func() {
		ctrl.Finish()
		db.Close()
		CleanTxHooks()
	}()

	t.Run("tx-local", func(t *testing.T) {
		tx, err := db.Begin()
		assert.NoError(t, err)

		mi.before = func(_ context.Context, execCtx *types.ExecContext) {
			assert.Equal(t, "", execCtx.TxCtx.XID)
			assert.Equal(t, types.Local, execCtx.TxCtx.TransactionMode)
		}

		var comitCnt int32
		ti.beforeCommit = func(tx *Tx) error {
			atomic.AddInt32(&comitCnt, 1)
			return nil
		}

		_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
		assert.NoError(t, err)

		_, err = tx.ExecContext(tm.InitSeataContext(context.Background()), "SELECT * FROM user")
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		assert.Equal(t, int32(1), atomic.LoadInt32(&comitCnt))
	})

	t.Run("tx-local-context", func(t *testing.T) {
		tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
		assert.NoError(t, err)

		mi.before = func(_ context.Context, execCtx *types.ExecContext) {
			assert.Equal(t, "", execCtx.TxCtx.XID)
			assert.Equal(t, types.Local, execCtx.TxCtx.TransactionMode)
		}

		var comitCnt int32
		ti.beforeCommit = func(tx *Tx) error {
			atomic.AddInt32(&comitCnt, 1)
			return nil
		}

		_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
		assert.NoError(t, err)

		_, err = tx.ExecContext(tm.InitSeataContext(context.Background()), "SELECT * FROM user")
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		assert.Equal(t, int32(1), atomic.LoadInt32(&comitCnt))
	})

	t.Run("tx-at-context", func(t *testing.T) {
		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.NewString())
		tx, err := db.BeginTx(ctx, &sql.TxOptions{})
		assert.NoError(t, err)

		mi.before = func(_ context.Context, execCtx *types.ExecContext) {
			assert.Equal(t, tm.GetXID(ctx), execCtx.TxCtx.XID)
			assert.Equal(t, types.ATMode, execCtx.TxCtx.TransactionMode)
		}

		var comitCnt int32
		ti.beforeCommit = func(tx *Tx) error {
			atomic.AddInt32(&comitCnt, 1)
			return nil
		}

		_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
		assert.NoError(t, err)

		_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		assert.Equal(t, int32(1), atomic.LoadInt32(&comitCnt))
	})
}

func TestATConn_CreateTxHelpersRollbackOnError(t *testing.T) {
	t.Run("createTxAndExecIfNeeded rolls back created tx on error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		CleanTxHooks()
		defer CleanTxHooks()

		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Rollback().Return(nil).Times(1)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil).Times(1)

		atConn := &ATConn{
			Conn: &Conn{
				res: &DBResource{
					dbType:     types.DBTypeMySQL,
					resourceID: "resource-id",
				},
				txCtx: &types.TransactionContext{
					TransactionMode: types.ATMode,
				},
				targetConn: mockConn,
				autoCommit: true,
			},
		}

		var rollbackCnt int32
		RegisterTxHook(&mockTxHook{
			beforeRollback: func(tx *Tx) {
				atomic.AddInt32(&rollbackCnt, 1)
			},
		})

		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.NewString())
		expectedErr := errors.New("exec failed")

		ret, err := atConn.createTxAndExecIfNeeded(ctx, func() (types.ExecResult, error) {
			return nil, expectedErr
		})

		assert.Nil(t, ret)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int32(1), atomic.LoadInt32(&rollbackCnt))
	})

	t.Run("createTxAndQueryIfNeeded rolls back created tx on error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		CleanTxHooks()
		defer CleanTxHooks()

		mockTx := mock.NewMockTestDriverTx(ctrl)
		mockTx.EXPECT().Rollback().Return(nil).Times(1)

		mockConn := mock.NewMockTestDriverConn(ctrl)
		mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil).Times(1)

		atConn := &ATConn{
			Conn: &Conn{
				res: &DBResource{
					dbType:     types.DBTypeMySQL,
					resourceID: "resource-id",
				},
				txCtx: &types.TransactionContext{
					TransactionMode: types.ATMode,
				},
				targetConn: mockConn,
				autoCommit: true,
			},
		}

		var rollbackCnt int32
		RegisterTxHook(&mockTxHook{
			beforeRollback: func(tx *Tx) {
				atomic.AddInt32(&rollbackCnt, 1)
			},
		})

		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.NewString())
		expectedErr := errors.New("query failed")

		ret, err := atConn.createTxAndQueryIfNeeded(ctx, func() (types.ExecResult, error) {
			return nil, expectedErr
		})

		assert.Nil(t, ret)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int32(1), atomic.LoadInt32(&rollbackCnt))
	})
}

type mockTxHook struct {
	beforeCommit   func(tx *Tx) error
	beforeRollback func(tx *Tx)
}

// BeforeCommit
func (mi *mockTxHook) BeforeCommit(tx *Tx) error {
	if mi.beforeCommit != nil {
		return mi.beforeCommit(tx)
	}
	return nil
}

// BeforeRollback
func (mi *mockTxHook) BeforeRollback(tx *Tx) {
	if mi.beforeRollback != nil {
		mi.beforeRollback(tx)
	}
}

func TestATTxCommitFailuresBeforeLocalCommitRollback(t *testing.T) {
	for _, failurePoint := range []string{"before commit", "branch registration", "undo manager lookup", "undo log flush"} {
		t.Run(failurePoint, func(t *testing.T) {
			CleanTxHooks()
			t.Cleanup(CleanTxHooks)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			primaryErr := errors.New(failurePoint + " failed")
			localTx := mock.NewMockTestDriverTx(ctrl)
			rollbackCall := localTx.EXPECT().Rollback().Times(1).Return(nil)
			targetConn := mock.NewMockTestDriverConn(ctrl)
			txCtx := newATCommitFailureContext(types.DBTypeMySQL)
			expectedOutcome := atCommitOutcomeRolledBack
			var expectedRollbackErr, expectedReportErr error

			switch failurePoint {
			case "before commit":
				RegisterTxHook(&mockTxHook{beforeCommit: func(*Tx) error { return primaryErr }})
			case "branch registration":
				manager := newATCommitTestManager(t, ctrl)
				manager.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), primaryErr)
			case "undo manager lookup":
				expectedOutcome = atCommitOutcomeRollbackFailed
				expectedRollbackErr = errors.New("rollback cleanup failed")
				expectedReportErr = errors.New("branch failure report failed")
				rollbackCall.Return(expectedRollbackErr)
				txCtx.DBType = types.DBTypeOracle
				manager := newATCommitTestManager(t, ctrl)
				manager.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(1).Return(int64(123), nil)
				gomock.InOrder(rollbackCall, expectATCommitFailureReport(t, manager, expectedReportErr))
			case "undo log flush":
				previousUndoConfig := undo.UndoConfig
				undo.UndoConfig.LogSerialization = "json"
				t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })

				manager := newATCommitTestManager(t, ctrl)
				manager.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Times(1).Return(int64(123), nil)
				targetConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).Times(1).Return(nil, primaryErr)
				gomock.InOrder(rollbackCall, expectATCommitFailureReport(t, manager, nil))
			}

			err := (&ATTx{tx: &Tx{
				conn:    &Conn{targetConn: targetConn},
				tranCtx: txCtx,
				target:  localTx,
			}}).Commit()

			if assert.Error(t, err) {
				var commitErr *atCommitError
				if assert.ErrorAs(t, err, &commitErr) {
					assert.Equal(t, expectedOutcome, commitErr.outcome)
					assert.Equal(t, expectedRollbackErr, commitErr.rollbackErr)
					assert.Equal(t, expectedReportErr, commitErr.reportErr)
				}
				if failurePoint == "undo manager lookup" {
					assert.Contains(t, err.Error(), "not found UndoLogManager")
				} else {
					assert.ErrorIs(t, err, primaryErr)
				}
			}
		})
	}
}

func TestATTxCommitFailureDiscardsConnection(t *testing.T) {
	for _, rollbackFailure := range []bool{true, false} {
		name := "local commit failure"
		if rollbackFailure {
			name = "rollback failure"
		}
		t.Run(name, func(t *testing.T) {
			CleanTxHooks()
			t.Cleanup(CleanTxHooks)
			exec.CleanCommonHook()
			t.Cleanup(exec.CleanCommonHook)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var primaryErr error = errors.New(name)
			localTx := mock.NewMockTestDriverTx(ctrl)
			var cleanupErr error
			expectedOutcome := atCommitOutcomeCommitUnknown
			var commitCall *gomock.Call
			if rollbackFailure {
				cleanupErr = errors.New("rollback cleanup failed")
				expectedOutcome = atCommitOutcomeRollbackFailed
				RegisterTxHook(&mockTxHook{beforeCommit: func(*Tx) error { return primaryErr }})
				localTx.EXPECT().Rollback().Times(1).Return(cleanupErr)
			} else {
				primaryErr = driver.ErrBadConn
				RegisterTxHook(&mockTxHook{beforeCommit: func(tx *Tx) error {
					tx.tranCtx.BranchID = 123
					return nil
				}})
				commitCall = localTx.EXPECT().Commit().Times(1).Return(primaryErr)
				localTx.EXPECT().Rollback().Times(0)
			}

			targetConn := mock.NewMockTestDriverConn(ctrl)
			targetConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(1).Return(localTx, nil)
			query := "SELECT 1"
			var args []any
			if rollbackFailure {
				previousTableCache := datasource.GetTableCache(types.DBTypeMySQL)
				t.Cleanup(func() {
					datasource.RegisterTableCache(types.DBTypeMySQL, previousTableCache)
				})
				datasource.RegisterTableCache(types.DBTypeMySQL, batchATTableCache{})

				query = "UPDATE account SET balance = ? WHERE id = ?"
				args = []any{int64(110), int64(1)}
				expectBatchATImageQueries(t, targetConn, [][]driver.Value{
					{int64(1), int64(100)}, {int64(1), int64(110)},
				})
			}
			targetConn.EXPECT().ExecContext(gomock.Any(), query, gomock.Any()).Times(1).Return(driver.ResultNoRows, nil)
			var closeCount atomic.Int32
			targetConn.EXPECT().Close().AnyTimes().DoAndReturn(func() error {
				closeCount.Add(1)
				return nil
			})

			txCtx := types.NewTxCtx()
			txCtx.XID = "test-xid"
			txCtx.TransactionMode = types.ATMode
			txCtx.DBType = types.DBTypeMySQL
			if !rollbackFailure {
				manager := newATCommitTestManager(t, ctrl)
				gomock.InOrder(commitCall, expectATCommitFailureReport(t, manager, nil))
			}
			conn := &Conn{
				res:        &DBResource{dbType: types.DBTypeMySQL, resourceID: "test-resource"},
				txCtx:      txCtx,
				targetConn: targetConn,
				autoCommit: true,
			}
			connector := mock.NewMockTestDriverConnector(ctrl)
			connector.EXPECT().Connect(gomock.Any()).Times(1).Return(&ATConn{Conn: conn}, nil)
			db := sql.OpenDB(connector)
			defer db.Close()

			ctx := tm.InitSeataContext(context.Background())
			tm.SetXID(ctx, "test-xid")
			_, err := db.ExecContext(ctx, query, args...)

			assert.Contains(t, err.Error(), primaryErr.Error())
			assert.NotErrorIs(t, err, driver.ErrBadConn)
			var commitErr *atCommitError
			if assert.ErrorAs(t, err, &commitErr) {
				assert.Equal(t, primaryErr, commitErr.cause)
				assert.Equal(t, expectedOutcome, commitErr.outcome)
				assert.Equal(t, cleanupErr, commitErr.rollbackErr)
			}
			if cleanupErr != nil {
				assert.Contains(t, err.Error(), cleanupErr.Error())
			}
			validator, ok := any(conn).(driver.Validator)
			assert.True(t, ok)
			if ok {
				assert.False(t, validator.IsValid())
			}
			assert.Equal(t, int32(1), closeCount.Load())
		})
	}
}

func newATCommitFailureContext(dbType types.DBType) *types.TransactionContext {
	txCtx := types.NewTxCtx()
	txCtx.XID = "test-xid"
	txCtx.ResourceID = "test-resource"
	txCtx.TransactionMode = types.ATMode
	txCtx.DBType = dbType
	txCtx.LockKeys["test_table:1"] = struct{}{}
	txCtx.RoundImages.AppendBeofreImage(&types.RecordImage{
		TableName: "test_table",
		SQLType:   types.SQLTypeUpdate,
		Rows:      []types.RowImage{{}},
	})
	txCtx.RoundImages.AppendAfterImage(&types.RecordImage{
		TableName: "test_table",
		SQLType:   types.SQLTypeUpdate,
		Rows:      []types.RowImage{{}},
	})
	return txCtx
}

func newATCommitTestManager(t *testing.T, ctrl *gomock.Controller) *mock.MockDataSourceManager {
	manager := mock.NewMockDataSourceManager(ctrl)
	manager.SetBranchType(branch.BranchTypeAT)
	registerResourceManagerForTest(t, manager)
	return manager
}

func expectATCommitFailureReport(t *testing.T, manager *mock.MockDataSourceManager, reportErr error) *gomock.Call {
	times := 1
	if reportErr != nil {
		times = 5
	}
	return manager.EXPECT().BranchReport(gomock.Any(), gomock.Any()).Times(times).DoAndReturn(
		func(_ context.Context, param rm.BranchReportParam) error {
			assert.EqualValues(t, branch.BranchStatusPhaseoneFailed, param.Status)
			return reportErr
		},
	)
}
