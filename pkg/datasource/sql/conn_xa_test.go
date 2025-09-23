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
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/go-sql-driver/mysql"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/tm"
)

type mockSQLInterceptor struct {
	before func(ctx context.Context, execCtx *types.ExecContext)
	after  func(ctx context.Context, execCtx *types.ExecContext)
}

func (mi *mockSQLInterceptor) Type() types.SQLType {
	return types.SQLTypeUnknown
}

// Before
func (mi *mockSQLInterceptor) Before(ctx context.Context, execCtx *types.ExecContext) error {
	if mi.before != nil {
		mi.before(ctx, execCtx)
	}
	return nil
}

// After
func (mi *mockSQLInterceptor) After(ctx context.Context, execCtx *types.ExecContext) error {
	if mi.after != nil {
		mi.after(ctx, execCtx)
	}
	return nil
}

type mockTxHook struct {
	beforeCommit   func(tx *Tx)
	beforeRollback func(tx *Tx)
}

func (mi *mockTxHook) BeforeCommit(tx *Tx) {
	if mi.beforeCommit != nil {
		mi.beforeCommit(tx)
	}
}

func (mi *mockTxHook) BeforeRollback(tx *Tx) {
	if mi.beforeRollback != nil {
		mi.beforeRollback(tx)
	}
}

// simulateExecContextError allows tests to inject driver errors for certain SQL strings.
// When set, baseMockConn will call this hook for each ExecContext.
var simulateExecContextError func(query string) error

func baseMockConn(t *testing.T, mockConn *mock.MockTestDriverConn, config dbTestConfig) {
	if branchStatusCache == nil {
		branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()
	}

	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			if simulateExecContextError != nil {
				if err := simulateExecContextError(query); err != nil {
					return &driver.ResultNoRows, err
				}
			}
			return &driver.ResultNoRows, nil
		},
	)
	mockConn.EXPECT().Exec(gomock.Any(), gomock.Any()).AnyTimes().Return(&driver.ResultNoRows, nil)
	mockConn.EXPECT().ResetSession(gomock.Any()).AnyTimes().Return(nil)
	mockConn.EXPECT().Close().AnyTimes().Return(nil)
	mockConn.EXPECT().Ping(gomock.Any()).AnyTimes().Return(nil)

	mockConn.EXPECT().QueryContext(
		gomock.Any(),
		"SELECT VERSION()",
		gomock.Any(),
	).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			var rows driver.Rows
			switch config.dbType {
			case types.DBTypeMySQL:
				data := config.versionResult
				if len(data) == 0 {
					data = [][]interface{}{{"5.7.36-log"}}
				}
				rows = &mysqlMockRows{data: data}
			case types.DBTypePostgreSQL:
				data := config.versionResult
				if len(data) == 0 {
					data = [][]interface{}{{"PostgreSQL 14.5 on x86_64-pc-linux-gnu"}}
				}
				rows = &pgMockRows{data: data}
			default:
				t.Fatalf("unsupported db type: %s", config.dbType)
			}
			return rows, nil
		},
	)

	mockConn.EXPECT().QueryContext(gomock.Any(), config.versionQuery, gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			var rows driver.Rows
			switch config.dbType {
			case types.DBTypeMySQL:
				rows = &mysqlMockRows{data: config.versionResult}
			case types.DBTypePostgreSQL:
				rows = &pgMockRows{data: config.versionResult}
			default:
				t.Fatalf("unsupported db type: %s", config.dbType)
			}
			return rows, nil
		})

	if config.dbType == types.DBTypePostgreSQL {
		mockConn.EXPECT().QueryContext(
			gomock.Any(),
			"SELECT current_setting('transaction_isolation')",
			gomock.Any(),
		).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				return &pgMockRows{data: [][]interface{}{{"read committed"}}}, nil
			})

		mockConn.EXPECT().QueryContext(
			gomock.Any(),
			"SELECT 1 FROM pg_proc WHERE proname = 'pg_prepared_xacts'",
			gomock.Any(),
		).AnyTimes().DoAndReturn(
			func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
				return &pgMockRows{data: [][]interface{}{{1}}}, nil
			})
	}
}

type mockTxAdapter struct {
	baseTx *Tx
	mockTx *mock.MockTestDriverTx
}

func (m *mockTxAdapter) Commit() error {
	err := m.mockTx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (m *mockTxAdapter) Rollback() error {
	return m.mockTx.Rollback()
}

func (m *mockTxAdapter) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return m.mockTx.ExecContext(ctx, query, args)
}

func (m *mockTxAdapter) register(ctx *types.TransactionContext) error {
	return m.baseTx.register(ctx)
}

func initXAConnTestResource(t *testing.T, ctrl *gomock.Controller, config dbTestConfig) (*sql.DB, *mockSQLInterceptor, *mockTxHook) {
	mockMgr := initMockResourceManager(branch.BranchTypeXA, ctrl)
	_ = mockMgr

	db, err := sql.Open(config.xaDriverName, config.dsn)
	if err != nil {
		t.Fatalf("failed to open %s XA conn: %v", config.name, err)
	}

	_ = initMockXaConnector(
		t,
		ctrl,
		db,
		config,
		func(t *testing.T, ctrl *gomock.Controller, config dbTestConfig) driver.Connector {
			mockTx := mock.NewMockTestDriverTx(ctrl)
			mockTx.EXPECT().Commit().AnyTimes().Return(nil)
			mockTx.EXPECT().Rollback().AnyTimes().Return(nil)

			if config.dbType == types.DBTypePostgreSQL {
				mockTx.EXPECT().ExecContext(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).AnyTimes().DoAndReturn(
					func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
						return &driver.ResultNoRows, nil
					},
				)
			}

			mockConn := mock.NewMockTestDriverConn(ctrl)

			mockDriverConn := &Conn{
				targetConn: mockConn,
				txCtx:      types.NewTxCtx(),
				isTestMode: true,
			}

			txIface, err := newTx(
				withDriverConn(mockDriverConn),
				withTxCtx(&types.TransactionContext{
					TransactionMode: types.XAMode,
					DBType:          config.dbType,
					ResourceID:      "test-resource",
				}),
				withOriginTx(mockTx),
			)
			if err != nil {
				t.Fatalf("newTx failed: %v", err)
			}
			baseTx, ok := txIface.(*Tx)
			if !ok {
				t.Fatalf("tx type assert failed: expect *Tx, got %T", txIface)
			}

			if baseTx.target != mockTx {
				t.Fatalf("baseTx.target binding error! Expected mockTx (address: %p), got %T (address: %p)",
					mockTx, baseTx.target, baseTx.target)
			}

			txAdapter := &mockTxAdapter{
				baseTx: baseTx,
				mockTx: mockTx,
			}
			mockConn.EXPECT().Begin().AnyTimes().DoAndReturn(
				func() (driver.Tx, error) {
					return txAdapter, nil
				},
			)

			mockConn.EXPECT().
				BeginTx(
					gomock.Any(),
					gomock.AssignableToTypeOf(driver.TxOptions{}),
				).
				AnyTimes().
				DoAndReturn(
					func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
						return txAdapter, nil
					},
				)


			baseMockConn(t, mockConn, config)

			connector := mock.NewMockTestDriverConnector(ctrl)
			connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
			//return connector

			return mockDriverConn
		},
	)

	mi := &mockSQLInterceptor{}
	ti := &mockTxHook{}
	exec.CleanCommonHook()
	CleanTxHooks()
	exec.RegisterCommonHook(mi)

	return db, mi, ti
}

func TestXAConn_ExecContext(t *testing.T) {
	for _, config := range getAllDBTestConfigs() {
		t.Run(config.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			var db *sql.DB
			defer func() {
				ctrl.Finish()
				CleanTxHooks()
				if db != nil {
					db.Close()
				}
			}()

			var err error
			db, _, _ = initXAConnTestResource(t, ctrl, config)
			if err != nil {
				t.Fatalf("init test resource failed: %v", err)
			}

			t.Run("have xid", func(t *testing.T) {
				ctx := tm.InitSeataContext(context.Background())
				expectedXID := uuid.New().String()
				tm.SetXID(ctx, expectedXID)

				mi := &mockSQLInterceptor{}
				mi.before = func(_ context.Context, execCtx *types.ExecContext) {
					assert.Equal(t, expectedXID, execCtx.TxCtx.XID, "XID mismatch")
					assert.Equal(t, types.XAMode, execCtx.TxCtx.TransactionMode, "transaction mode should be XA")
				}

				ti := &mockTxHook{}
				var commitCnt int32
				ti.beforeCommit = func(tx *Tx) {
					atomic.AddInt32(&commitCnt, 1)
					assert.Equal(t, types.XAMode, tx.tranCtx.TransactionMode, "tx mode should be XA")
				}

				conn, err := db.Conn(context.Background())
				assert.NoError(t, err, "failed to get db conn")

				_, err = conn.ExecContext(ctx, "SELECT 1")
				assert.NoError(t, err, "conn.ExecContext failed")
				_, err = db.ExecContext(ctx, "SELECT 1")
				assert.NoError(t, err, "db.ExecContext failed")

				assert.Equal(t, int32(0), atomic.LoadInt32(&commitCnt), "commit count should be 0")
			})

			t.Run("not xid", func(t *testing.T) {
				mi := &mockSQLInterceptor{}
				mi.before = func(_ context.Context, execCtx *types.ExecContext) {
					assert.Equal(t, "", execCtx.TxCtx.XID, "XID should be empty")
					assert.Equal(t, types.Local, execCtx.TxCtx.TransactionMode, "transaction mode should be Local")
				}

				ti := &mockTxHook{}
				var commitCnt int32
				ti.beforeCommit = func(tx *Tx) {
					atomic.AddInt32(&commitCnt, 1)
				}

				conn, err := db.Conn(context.Background())
				assert.NoError(t, err, "failed to get db conn")

				_, err = conn.ExecContext(context.Background(), "SELECT 1")
				assert.NoError(t, err, "conn.ExecContext failed")
				_, err = db.ExecContext(context.Background(), "SELECT 1")
				assert.NoError(t, err, "db.ExecContext failed")
				_, err = db.Exec("SELECT 1")
				assert.NoError(t, err, "db.Exec failed")

				assert.Equal(t, int32(0), atomic.LoadInt32(&commitCnt), "commit count should be 0")
			})
		})
	}
}

func TestXAConn_BeginTx(t *testing.T) {
	for _, config := range getAllDBTestConfigs() {
		t.Run(config.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			var db *sql.DB
			defer func() {
				ctrl.Finish()
				CleanTxHooks()
				if db != nil {
					db.Close()
				}
			}()

			var err error
			db, _, _ = initXAConnTestResource(t, ctrl, config)
			if err != nil {
				t.Fatalf("init test resource failed: %v", err)
			}

			t.Run("tx-local", func(t *testing.T) {
				CleanTxHooks()
				tx, err := db.Begin()
				assert.NoError(t, err, "failed to begin local tx")

				mi := &mockSQLInterceptor{}
				mi.before = func(_ context.Context, execCtx *types.ExecContext) {
					assert.Equal(t, "", execCtx.TxCtx.XID, "XID should be empty")
					assert.Equal(t, types.Local, execCtx.TxCtx.TransactionMode, "mode should be Local")
				}

				ti := &mockTxHook{}
				var commitCnt int32
				ti.beforeCommit = func(tx *Tx) {
					atomic.AddInt32(&commitCnt, 1)
				}
				RegisterTxHook(ti)

				_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
				assert.NoError(t, err, "tx.ExecContext failed")
				_, err = tx.ExecContext(tm.InitSeataContext(context.Background()), "SELECT * FROM user")
				assert.NoError(t, err, "tx.ExecContext with seata ctx failed")

				err = tx.Commit()
				assert.NoError(t, err, "failed to commit local tx")

				assert.Equal(t, int32(1), atomic.LoadInt32(&commitCnt), "commit count should be 1")
			})

			t.Run("tx-local-context", func(t *testing.T) {
				CleanTxHooks()
				tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
				assert.NoError(t, err, "failed to begin local tx with ctx")

				mi := &mockSQLInterceptor{}
				mi.before = func(_ context.Context, execCtx *types.ExecContext) {
					assert.Equal(t, "", execCtx.TxCtx.XID, "XID should be empty")
					assert.Equal(t, types.Local, execCtx.TxCtx.TransactionMode, "mode should be Local")
				}

				ti := &mockTxHook{}
				var commitCnt int32
				ti.beforeCommit = func(tx *Tx) {
					atomic.AddInt32(&commitCnt, 1)
				}
				RegisterTxHook(ti)

				_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
				assert.NoError(t, err, "tx.ExecContext failed")
				_, err = tx.ExecContext(tm.InitSeataContext(context.Background()), "SELECT * FROM user")
				assert.NoError(t, err, "tx.ExecContext with seata ctx failed")

				err = tx.Commit()
				assert.NoError(t, err, "failed to commit local tx")

				assert.Equal(t, int32(1), atomic.LoadInt32(&commitCnt), "commit count should be 1")
			})

			t.Run("tx-xa-context", func(t *testing.T) {
				CleanTxHooks()
				ctx := tm.InitSeataContext(context.Background())
				expectedXID := uuid.NewString()
				tm.SetXID(ctx, expectedXID)

				tx, err := db.BeginTx(ctx, &sql.TxOptions{})
				assert.NoError(t, err, "failed to begin XA tx")

				mi := &mockSQLInterceptor{}
				mi.before = func(_ context.Context, execCtx *types.ExecContext) {
					assert.Equal(t, expectedXID, execCtx.TxCtx.XID, "XID mismatch")
					assert.Equal(t, types.XAMode, execCtx.TxCtx.TransactionMode, "mode should be XA")
				}

				ti := &mockTxHook{}
				var commitCnt int32
				ti.beforeCommit = func(tx *Tx) {
					atomic.AddInt32(&commitCnt, 1)
				}
				RegisterTxHook(ti)

				_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
				assert.NoError(t, err, "tx.ExecContext failed")
				_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
				assert.NoError(t, err, "tx.ExecContext again failed")

				err = tx.Commit()
				assert.NoError(t, err, "failed to commit XA tx")

				assert.Equal(t, int32(1), atomic.LoadInt32(&commitCnt), "commit count should be 1")
			})
		})
	}
}

func TestXAConn_Rollback_XAER_RMFAIL(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no error case",
			err:  nil,
			want: false,
		},
		{
			name: "matching XAER_RMFAIL error with IDLE state",
			err: &mysql.MySQLError{
				Number:  1399,
				Message: "Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the IDLE state",
			},
			want: true,
		},
		{
			name: "matching XAER_RMFAIL error with already ended",
			err: &mysql.MySQLError{
				Number:  1399,
				Message: "Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction has already ended",
			},
			want: true,
		},
		{
			name: "matching error code but mismatched message",
			err: &mysql.MySQLError{
				Number:  1399,
				Message: "Error 1399 (XAE07): XAER_RMFAIL: Other error message",
			},
			want: false,
		},
		{
			name: "mismatched error code but matching message",
			err: &mysql.MySQLError{
				Number:  1234,
				Message: "The command cannot be executed when global transaction is in the IDLE state",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isXAER_RMFAILAlreadyEnded(tt.err); got != tt.want {
				t.Errorf("isXAER_RMFAILAlreadyEnded() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Covers the XA rollback flow when End() returns XAER_RMFAIL (IDLE/already ended)
func TestXAConn_Rollback_HandleXAERRMFAILAlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	config := getAllDBTestConfigs()[0] // Use first config for this test
	db, _, ti := initXAConnTestResource(t, ctrl, config)
	defer func() {
		simulateExecContextError = nil
		db.Close()
		ctrl.Finish()
		CleanTxHooks()
	}()

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.New().String())

	// Ensure Tx.Rollback has a non-nil underlying target to avoid nil-deref when test triggers rollback
	ti.beforeRollback = func(tx *Tx) {
		mtx := mock.NewMockTestDriverTx(ctrl)
		mtx.EXPECT().Rollback().AnyTimes().Return(nil)
		tx.target = mtx
	}

	// Inject: XA END returns XAER_RMFAIL(IDLE), normal SQL returns an error to trigger rollback
	simulateExecContextError = func(query string) error {
		upper := strings.ToUpper(query)
		if strings.HasPrefix(upper, "XA END") {
			return &mysql.MySQLError{Number: types.ErrCodeXAER_RMFAIL_IDLE, Message: "Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the IDLE state"}
		}
		if !strings.HasPrefix(upper, "XA ") {
			return io.EOF
		}
		return nil
	}

	// Execute to enter XA flow; the user SQL fails, but rollback should proceed without panicking
	_, err := db.ExecContext(ctx, "SELECT 1")
	if err == nil {
		t.Fatalf("expected error to trigger rollback path")
	}
}
