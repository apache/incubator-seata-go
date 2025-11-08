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

type mysqlMockRows struct {
	idx  int
	data [][]interface{}
}

func (m *mysqlMockRows) Columns() []string {
	return []string{"value"}
}

func (m *mysqlMockRows) Close() error {
	return nil
}

func (m *mysqlMockRows) Next(dest []driver.Value) error {
	if m.idx == len(m.data) {
		return io.EOF
	}

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	cnt := min(len(m.data[0]), len(dest))

	for i := 0; i < cnt; i++ {
		dest[i] = m.data[m.idx][i]
	}
	m.idx++
	return nil
}

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

// simulateExecContextError allows tests to inject driver errors for certain SQL strings.
// When set, baseMockConn will call this hook for each ExecContext.
var simulateExecContextError func(query string) error

func baseMockConn(mockConn *mock.MockTestDriverConn) {
	branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()

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

	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			rows := &mysqlMockRows{}
			rows.data = [][]interface{}{
				{"8.0.29"},
			}
			return rows, nil
		})
}

func initXAConnTestResource(t *testing.T) (*gomock.Controller, *sql.DB, *mockSQLInterceptor, *mockTxHook) {
	ctrl := gomock.NewController(t)

	mockMgr := initMockResourceManager(branch.BranchTypeXA, ctrl)
	_ = mockMgr
	//db, err := sql.Open("seata-xa-mysql", "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	db, err := sql.Open("seata-xa-mysql", "root:12345678@tcp(127.0.0.1:3306)/seata_client?multiStatements=true&interpolateParams=true")
	if err != nil {
		t.Fatal(err)
	}

	_ = initMockXaConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
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

	mi := &mockSQLInterceptor{}
	ti := &mockTxHook{}

	exec.CleanCommonHook()
	CleanTxHooks()
	exec.RegisterCommonHook(mi)
	RegisterTxHook(ti)

	return ctrl, db, mi, ti
}

func TestXAConn_ExecContext(t *testing.T) {

	ctrl, db, mi, ti := initXAConnTestResource(t)
	defer func() {
		ctrl.Finish()
		db.Close()
		CleanTxHooks()
	}()

	t.Run("have xid", func(t *testing.T) {
		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.New().String())

		before := func(_ context.Context, execCtx *types.ExecContext) {
			t.Logf("on exec xid=%s", execCtx.TxCtx.XID)
			assert.Equal(t, tm.GetXID(ctx), execCtx.TxCtx.XID)
			assert.Equal(t, types.XAMode, execCtx.TxCtx.TransactionMode)
		}
		mi.before = before

		var comitCnt int32
		beforeCommit := func(tx *Tx) error {
			atomic.AddInt32(&comitCnt, 1)
			assert.Equal(t, tx.tranCtx.TransactionMode, types.XAMode)
			return nil
		}
		ti.beforeCommit = beforeCommit

		conn, err := db.Conn(context.Background())
		assert.NoError(t, err)

		_, err = conn.ExecContext(ctx, "SELECT 1")
		assert.NoError(t, err)
		_, err = db.ExecContext(ctx, "SELECT 1")
		assert.NoError(t, err)

		// todo fix
		assert.Equal(t, int32(0), atomic.LoadInt32(&comitCnt))
	})

	t.Run("not xid", func(t *testing.T) {
		before := func(_ context.Context, execCtx *types.ExecContext) {
			assert.Equal(t, "", execCtx.TxCtx.XID)
			assert.Equal(t, types.Local, execCtx.TxCtx.TransactionMode)
		}
		mi.before = before

		var comitCnt int32
		beforeCommit := func(tx *Tx) error {
			atomic.AddInt32(&comitCnt, 1)
			return nil
		}
		ti.beforeCommit = beforeCommit

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

func TestXAConn_BeginTx(t *testing.T) {
	ctrl, db, mi, ti := initXAConnTestResource(t)
	defer func() {
		CleanTxHooks()
		db.Close()
		ctrl.Finish()
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

	t.Run("tx-xa-context", func(t *testing.T) {
		ctx := tm.InitSeataContext(context.Background())
		tm.SetXID(ctx, uuid.NewString())
		tx, err := db.BeginTx(ctx, &sql.TxOptions{})
		assert.NoError(t, err)

		mi.before = func(_ context.Context, execCtx *types.ExecContext) {
			assert.Equal(t, tm.GetXID(ctx), execCtx.TxCtx.XID)
			assert.Equal(t, types.XAMode, execCtx.TxCtx.TransactionMode)
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
	ctrl, db, _, ti := initXAConnTestResource(t)
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

// Test for issue #904: QueryContext in autoCommit mode should bypass XA transaction logic
// to avoid "busy buffer" errors when executing Query followed by Exec
func TestXAConn_QueryContext_AutoCommit_BypassXA(t *testing.T) {
	ctrl, db, _, _ := initXAConnTestResource(t)
	defer func() {
		db.Close()
		ctrl.Finish()
		CleanTxHooks()
		simulateExecContextError = nil
	}()

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tests := []struct {
		name             string
		setupTx          func() (*sql.Tx, error)
		executeQuery     func(context.Context) (*sql.Rows, error)
		executeExec      func(context.Context) (sql.Result, error)
		wantQueryXAStart bool
		wantQueryXAEnd   bool
		wantExecXAStart  bool
		wantExecXAEnd    bool
		cleanupTx        func(*sql.Tx) error
		description      string
	}{
		{
			name:    "QueryContext should bypass XA in autoCommit mode",
			setupTx: func() (*sql.Tx, error) { return nil, nil },
			executeQuery: func(ctx context.Context) (*sql.Rows, error) {
				return db.QueryContext(ctx, "SELECT balance FROM account WHERE id = ?", 1)
			},
			executeExec: func(ctx context.Context) (sql.Result, error) {
				return db.ExecContext(ctx, "UPDATE account SET balance = ? WHERE id = ?", 100, 1)
			},
			wantQueryXAStart: false,
			wantQueryXAEnd:   false,
			wantExecXAStart:  true,
			wantExecXAEnd:    true,
			cleanupTx:        func(tx *sql.Tx) error { return nil },
			description:      "In autoCommit mode, Query should bypass XA but Exec should use XA",
		},
		{
			name: "QueryContext should use XA in explicit transaction",
			setupTx: func() (*sql.Tx, error) {
				return db.BeginTx(ctx, &sql.TxOptions{})
			},
			executeQuery: func(ctx context.Context) (*sql.Rows, error) {
				// Will be overridden to use tx.QueryContext in test
				return nil, nil
			},
			executeExec:      nil, // Not testing Exec in this case
			wantQueryXAStart: true,
			wantQueryXAEnd:   false, // Query doesn't trigger END in explicit tx
			wantExecXAStart:  false,
			wantExecXAEnd:    false,
			cleanupTx: func(tx *sql.Tx) error {
				return tx.Commit()
			},
			description: "In explicit transaction, Query should use XA transaction logic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var xaStartExecuted, xaEndExecuted bool

			// Track which SQL commands are executed
			simulateExecContextError = func(query string) error {
				upper := strings.ToUpper(query)
				if strings.HasPrefix(upper, "XA START") {
					xaStartExecuted = true
				} else if strings.HasPrefix(upper, "XA END") {
					xaEndExecuted = true
				}
				return nil
			}

			// Setup transaction if needed
			tx, err := tt.setupTx()
			assert.NoError(t, err)

			// Execute Query
			var rows *sql.Rows
			if tx != nil {
				rows, err = tx.QueryContext(ctx, "SELECT balance FROM account WHERE id = ?", 1)
			} else {
				rows, err = tt.executeQuery(ctx)
			}
			assert.NoError(t, err)
			if rows != nil {
				_ = rows.Close()
			}

			// Verify Query XA execution
			assert.Equal(t, tt.wantQueryXAStart, xaStartExecuted,
				"Query XA START execution mismatch: %s", tt.description)
			assert.Equal(t, tt.wantQueryXAEnd, xaEndExecuted,
				"Query XA END execution mismatch: %s", tt.description)

			// Reset flags for Exec phase
			xaStartExecuted, xaEndExecuted = false, false

			// Execute Exec if defined
			if tt.executeExec != nil {
				_, err = tt.executeExec(ctx)
				assert.NoError(t, err)

				// Verify Exec XA execution
				assert.Equal(t, tt.wantExecXAStart, xaStartExecuted,
					"Exec XA START execution mismatch: %s", tt.description)
				assert.Equal(t, tt.wantExecXAEnd, xaEndExecuted,
					"Exec XA END execution mismatch: %s", tt.description)
			}

			// Cleanup transaction if needed
			if tx != nil {
				err = tt.cleanupTx(tx)
				assert.NoError(t, err)
			}

			// Reset for next test
			simulateExecContextError = nil
		})
	}
}
