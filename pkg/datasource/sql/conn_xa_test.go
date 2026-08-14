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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluele/gcache"
	"github.com/go-sql-driver/mysql"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

type mysqlMockRows struct {
	idx  int
	data [][]interface{}
}

func (m *mysqlMockRows) Columns() []string {
	//TODO implement me
	panic("implement me")
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
// When set, baseMockConn will call this hook for each direct ExecContext.
var simulateExecContextError func(query string) error

// simulateQueryContextError injects driver errors for certain SQL strings on the
// direct QueryContext path (e.g. returning driver.ErrSkip for a parameterized
// SELECT, as the default go-sql-driver DSN does). When set, baseMockConn calls it
// for each direct QueryContext.
var simulateQueryContextError func(query string) error

// simulatePreparedExecError injects an error from the prepared statement's
// ExecContext, keyed by the query it was prepared with. It lets tests drive the
// case where the in-branch Prepare+Exec fallback itself fails with a real (non
// ErrSkip) error, so the branch must roll back and report phase-1 failure.
var simulatePreparedExecError func(query string) error

// fakePreparedStmt models a driver prepared statement. It is what the driver
// returns from PrepareContext, and its ExecContext/QueryContext succeed by default -
// mirroring the real go-sql-driver, where the direct Execer answers driver.ErrSkip
// for parameterized statements but the Prepare+Exec path executes fine. This lets
// tests exercise XAConn's in-branch ErrSkip fallback (execPreparedInBranch /
// queryPreparedInBranch). simulatePreparedExecError can force the prepared exec to
// fail for the fallback-error rollback path.
type fakePreparedStmt struct {
	query string
}

func (s *fakePreparedStmt) Close() error  { return nil }
func (s *fakePreparedStmt) NumInput() int { return -1 }

func (s *fakePreparedStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &driver.ResultNoRows, nil
}

func (s *fakePreparedStmt) Query(args []driver.Value) (driver.Rows, error) {
	rows := &mysqlMockRows{}
	rows.data = [][]interface{}{{"8.0.29"}}
	return rows, nil
}

func (s *fakePreparedStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if simulatePreparedExecError != nil {
		if err := simulatePreparedExecError(s.query); err != nil {
			return nil, err
		}
	}
	return &driver.ResultNoRows, nil
}

func (s *fakePreparedStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	rows := &mysqlMockRows{}
	rows.data = [][]interface{}{{"8.0.29"}}
	return rows, nil
}

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

	// The Prepare+Exec fallback path (used when the direct ExecContext answers
	// driver.ErrSkip) prepares on the same physical connection and runs the
	// statement through the prepared stmt, which succeeds by default. The prepared
	// stmt keeps the query so simulatePreparedExecError can target it.
	mockConn.EXPECT().PrepareContext(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string) (driver.Stmt, error) {
			return &fakePreparedStmt{query: query}, nil
		})

	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			if simulateQueryContextError != nil {
				if err := simulateQueryContextError(query); err != nil {
					return nil, err
				}
			}
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

func newMockXAConn(t *testing.T, ctrl *gomock.Controller, branchID int64) (*XAConn, *mock.MockDataSourceManager) {
	t.Helper()

	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeXA)
	registerResourceManagerForTest(t, mockMgr)
	mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).AnyTimes().Return(branchID, nil)

	mockConn := mock.NewMockTestDriverConn(ctrl)
	baseMockConn(mockConn)

	return &XAConn{
		Conn: &Conn{
			res: &DBResource{
				resourceID: "jdbc:mysql://test/resource",
				dbType:     types.DBTypeMySQL,
			},
			txCtx:      types.NewTxCtx(),
			targetConn: mockConn,
			autoCommit: true,
			dbType:     types.DBTypeMySQL,
		},
	}, mockMgr
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

		assert.Equal(t, int32(2), atomic.LoadInt32(&comitCnt))
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
			name: "matching XAER_RMFAIL error with PREPARED state",
			err: &mysql.MySQLError{
				Number:  1399,
				Message: "Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the PREPARED state",
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
			classifier := &xa.MysqlXAErrorClassifier{}
			if got := classifier.IsAlreadyEnded(tt.err); got != tt.want {
				t.Errorf("MysqlXAErrorClassifier.IsAlreadyEnded() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Covers the XA rollback flow when End() returns XAER_RMFAIL (IDLE/already ended)
func TestXAConn_Rollback_HandleXAERRMFAILAlreadyEnded(t *testing.T) {
	ctrl, db, _, _ := initXAConnTestResource(t)
	defer func() {
		simulateExecContextError = nil
		db.Close()
		ctrl.Finish()
		CleanTxHooks()
	}()

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.New().String())

	// Inject: XA END returns XAER_RMFAIL(IDLE), normal SQL returns an error to trigger rollback
	simulateExecContextError = func(query string) error {
		upper := strings.ToUpper(query)
		if strings.HasPrefix(upper, "XA END") {
			return &mysql.MySQLError{
				Number:  types.ErrCodeXAER_RMFAIL_IDLE,
				Message: "Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the IDLE state",
			}
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

// Reproduces the review scenario where the branch is already PREPARED when Rollback runs:
// during autoCommit Commit the DB executed XA END + XA PREPARE, but the phase-1 report to
// the TC failed, so the branch is left in the PREPARED state. The follow-up rollback issues
// XA END(TMFAIL), which MySQL rejects with XAER_RMFAIL "...PREPARED state". Before the fix
// IsAlreadyEnded only recognized the IDLE-state message, so Rollback bailed out via
// rollbackErrorHandle() BEFORE running XA ROLLBACK, leaving the branch holding locks forever.
// This asserts XA ROLLBACK is still issued so the prepared branch releases its locks.
func TestXAConn_Rollback_PreparedBranchStillRollsBack(t *testing.T) {
	ctrl, db, _, _ := initXAConnTestResource(t)
	defer func() {
		simulateExecContextError = nil
		db.Close()
		ctrl.Finish()
		CleanTxHooks()
	}()

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.New().String())

	var rollbackSeen int32
	// Inject: XA END returns XAER_RMFAIL with the PREPARED-state message; user SQL fails to
	// trigger the rollback path; record whether XA ROLLBACK is subsequently issued.
	simulateExecContextError = func(query string) error {
		upper := strings.ToUpper(strings.TrimSpace(query))
		switch {
		case strings.HasPrefix(upper, "XA END"):
			return &mysql.MySQLError{
				Number:  types.ErrCodeXAER_RMFAIL_IDLE,
				Message: "Error 1399 (XAE07): XAER_RMFAIL: The command cannot be executed when global transaction is in the PREPARED state",
			}
		case strings.HasPrefix(upper, "XA ROLLBACK"):
			atomic.StoreInt32(&rollbackSeen, 1)
			return nil
		case !strings.HasPrefix(upper, "XA "):
			return io.EOF
		}
		return nil
	}

	_, err := db.ExecContext(ctx, "UPDATE user SET age = 1 WHERE id = 1")
	assert.Error(t, err, "expected error to trigger rollback path")
	assert.Equal(t, int32(1), atomic.LoadInt32(&rollbackSeen),
		"XA ROLLBACK must run so a PREPARED branch releases its locks")
}

func TestXAConn_ExecContext_AutoCommitReportsPhaseOneDone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, param rm.BranchReportParam) error {
			assert.Equal(t, branch.BranchTypeXA, param.BranchType)
			assert.Equal(t, int64(123), param.BranchId)
			assert.EqualValues(t, branch.BranchStatusPhaseoneDone, param.Status)
			return nil
		},
	).Times(1)

	var commitCnt int32
	RegisterTxHook(&mockTxHook{
		beforeCommit: func(tx *Tx) error {
			atomic.AddInt32(&commitCnt, 1)
			return nil
		},
	})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	_, err := xaConn.ExecContext(ctx, "SELECT 1", nil)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&commitCnt))
}

// Regression for the autoCommit branch-reuse bug: after a statement's XA branch
// completes phase-1 (XA END + XA PREPARE + report), the session must no longer be
// marked as having an active branch, otherwise the next autoCommit statement on the
// SAME physical connection (which database/sql reuses via the pool, calling
// ResetSession in between) trips BeginTx's "should NEVER happen: setAutoCommit from
// true to false while xa branch is active" guard. Before the fix, XAConn.Commit's
// success path never cleared xaActive (only the rollback/cleanup path did) and
// ResetSession - living on the embedded *Conn - could not reach it, so the second
// statement always failed.
func TestXAConn_ExecContext_ReuseAfterAutoCommitBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer CleanTxHooks()

	// XAConn.Commit -> checkTimeout compares branchRegisterTime against xaConnTimeout,
	// which is 0 unless InitXA runs. Give the branch a real budget so phase-1 prepares.
	prevTimeout := xaConnTimeout
	xaConnTimeout = time.Minute
	defer func() { xaConnTimeout = prevTimeout }()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	var commitCnt int32
	RegisterTxHook(&mockTxHook{
		beforeCommit: func(tx *Tx) error {
			atomic.AddInt32(&commitCnt, 1)
			return nil
		},
	})

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	// First autoCommit statement: opens and completes a full XA branch.
	_, err := xaConn.ExecContext(ctx, "SELECT 1", nil)
	assert.NoError(t, err)
	// The Commit success path must clear the session-active flag on its own, so the
	// fix holds even for paths where database/sql does not call ResetSession.
	assert.False(t, xaConn.xaActive, "xaActive must be cleared after phase-1 completes")

	// Simulate database/sql returning the connection to the pool and reusing it:
	// ResetSession restores autoCommit=true (and, via the XAConn override, clears the
	// XA session flag as a backstop).
	assert.NoError(t, xaConn.ResetSession(ctx))
	assert.True(t, xaConn.autoCommit, "ResetSession must restore autoCommit for pooled reuse")
	assert.False(t, xaConn.xaActive, "ResetSession must leave no active XA branch")

	// Second autoCommit statement on the SAME XAConn must open a fresh branch instead
	// of failing the "xa branch is active" guard.
	_, err = xaConn.ExecContext(ctx, "SELECT 2", nil)
	assert.NoError(t, err, "second autoCommit statement on a reused XAConn must succeed")

	assert.Equal(t, int32(2), atomic.LoadInt32(&commitCnt))
}

// Reproduces the #904 "busy buffer" scenario on the query path: a SELECT ... FOR
// UPDATE opens a result set that still occupies the connection's read buffer. If the
// autoCommit branch were committed inline (XA END + XA PREPARE) while those rows are
// open, go-sql-driver would reject the new command with a "busy buffer" /
// "commands out of sync" error surfacing as "driver: bad connection". This asserts the
// branch commit is deferred: XA END / XA PREPARE / the phase-1 report only run once the
// caller closes the rows, so the busy-buffer collision never happens.
func TestXAConn_QueryContext_DefersBranchCommitUntilRowsClose(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer func() {
		simulateExecContextError = nil
		CleanTxHooks()
	}()

	// checkTimeout compares against xaConnTimeout, which is only set by InitXA in a
	// running server. Give the branch a real budget so the deferred commit prepares
	// instead of aborting as timed-out.
	prevTimeout := xaConnTimeout
	xaConnTimeout = time.Minute
	defer func() { xaConnTimeout = prevTimeout }()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)

	var reported int32
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, param rm.BranchReportParam) error {
			assert.EqualValues(t, branch.BranchStatusPhaseoneDone, param.Status)
			atomic.StoreInt32(&reported, 1)
			return nil
		},
	).Times(1)

	// Record when the branch-commit statements run on the physical connection.
	var endSeen, prepareSeen int32
	simulateExecContextError = func(query string) error {
		upper := strings.ToUpper(strings.TrimSpace(query))
		switch {
		case strings.HasPrefix(upper, "XA END"):
			atomic.StoreInt32(&endSeen, 1)
		case strings.HasPrefix(upper, "XA PREPARE"):
			atomic.StoreInt32(&prepareSeen, 1)
		}
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	rows, err := xaConn.QueryContext(ctx, "SELECT * FROM user WHERE id = 1 FOR UPDATE", nil)
	assert.NoError(t, err)

	// While the result set is still open, the branch must NOT have been committed -
	// issuing XA END / XA PREPARE here is exactly the #904 busy-buffer trigger.
	assert.Equal(t, int32(0), atomic.LoadInt32(&endSeen), "XA END must be deferred until rows close")
	assert.Equal(t, int32(0), atomic.LoadInt32(&prepareSeen), "XA PREPARE must be deferred until rows close")
	assert.Equal(t, int32(0), atomic.LoadInt32(&reported), "phase-1 report must be deferred until rows close")

	// The returned rows must be the deferred-commit wrapper.
	_, ok := rows.(*RowsCommitOnClose)
	assert.True(t, ok, "XA query rows must be wrapped in RowsCommitOnClose to defer the branch commit")

	// Closing the rows drains the connection first, then runs XA END + XA PREPARE + report.
	assert.NoError(t, rows.Close())

	assert.Equal(t, int32(1), atomic.LoadInt32(&endSeen), "XA END must run once rows are closed")
	assert.Equal(t, int32(1), atomic.LoadInt32(&prepareSeen), "XA PREPARE must run once rows are closed")
	assert.Equal(t, int32(1), atomic.LoadInt32(&reported), "phase-1 report must run once rows are closed")
}

// End-to-end regression for the exact #904 sequence: under an autoCommit global
// transaction, a "SELECT ... FOR UPDATE" is immediately followed by an "UPDATE" on the
// SAME physical connection. The SELECT's open result set occupies the connection's read
// buffer; the busy-buffer error struck because the first branch used to be committed
// inline (XA END + XA PREPARE) while those rows were still open, then the second
// statement could not open its own branch. This drives the full flow - query, drain,
// commit branch 1, pool reuse (ResetSession), then the UPDATE as branch 2 - and asserts
// each statement forms its own complete branch (two XA END + XA PREPARE + phase-1
// reports) with no error, so the busy-buffer collision cannot recur.
func TestXAConn_AutoCommit_SelectForUpdateThenUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer func() {
		simulateExecContextError = nil
		CleanTxHooks()
	}()

	prevTimeout := xaConnTimeout
	xaConnTimeout = time.Minute
	defer func() { xaConnTimeout = prevTimeout }()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)

	var reportCnt int32
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, param rm.BranchReportParam) error {
			assert.EqualValues(t, branch.BranchStatusPhaseoneDone, param.Status)
			atomic.AddInt32(&reportCnt, 1)
			return nil
		},
	).AnyTimes()

	var endCnt, prepareCnt int32
	simulateExecContextError = func(query string) error {
		upper := strings.ToUpper(strings.TrimSpace(query))
		switch {
		case strings.HasPrefix(upper, "XA END"):
			atomic.AddInt32(&endCnt, 1)
		case strings.HasPrefix(upper, "XA PREPARE"):
			atomic.AddInt32(&prepareCnt, 1)
		}
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	// Statement 1: SELECT ... FOR UPDATE. The branch commit is deferred while the rows
	// are open, so no XA END / XA PREPARE fires yet - that would be the busy-buffer bug.
	rows, err := xaConn.QueryContext(ctx, "SELECT * FROM user WHERE id = 1 FOR UPDATE", nil)
	assert.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&endCnt), "branch 1 must not commit while its rows are open")

	// Draining/closing the rows completes branch 1 (XA END + XA PREPARE + report).
	assert.NoError(t, rows.Close())
	assert.Equal(t, int32(1), atomic.LoadInt32(&endCnt), "branch 1 commits once its rows close")
	assert.Equal(t, int32(1), atomic.LoadInt32(&prepareCnt))
	assert.False(t, xaConn.xaActive, "branch 1 must leave no active branch on the session")

	// database/sql returns the connection to the pool and resets it before reuse.
	assert.NoError(t, xaConn.ResetSession(ctx))

	// Statement 2: the follow-up UPDATE on the SAME connection must form its own branch.
	_, err = xaConn.ExecContext(ctx, "UPDATE user SET age = age + 1 WHERE id = 1", nil)
	assert.NoError(t, err, "UPDATE after SELECT ... FOR UPDATE must succeed (no busy buffer)")

	assert.Equal(t, int32(2), atomic.LoadInt32(&endCnt), "each statement forms one complete XA branch")
	assert.Equal(t, int32(2), atomic.LoadInt32(&prepareCnt))
	assert.Equal(t, int32(2), atomic.LoadInt32(&reportCnt), "each branch reports phase-1 done to the TC")
}

// ErrSkip in-branch fallback under XA autoCommit.
//
// go-sql-driver returns driver.ErrSkip from Exec/Query whenever a statement carries
// bind arguments and the DSN does NOT set interpolateParams=true (the default). See
// go-sql-driver/mysql@v1.6.0 connection.go: `if len(args) != 0 { if !cfg.InterpolateParams
// { return nil, driver.ErrSkip } }`. database/sql normally answers ErrSkip by retrying the
// statement through the Prepare+Exec path.
//
// Under XA autoCommit + a global transaction, createNewTxOnExecIfNeed opens the XA branch
// (XA START) BEFORE running the statement, so it cannot hand the retry back to database/sql
// (that retry would run on another connection, outside the branch). Instead XAConn runs the
// Prepare+Exec fallback ITSELF on the same physical connection - which still holds XA START
// open - so a perfectly ordinary parameterized statement (`UPDATE ... WHERE id = ?` with the
// default MySQL DSN) completes inside the branch and the branch commits normally.
//
// The mock models the driver faithfully: the direct ExecContext answers ErrSkip for the
// business UPDATE, while PrepareContext + the prepared stmt's ExecContext succeed.
func TestXAConn_AutoCommit_ParameterizedStmtErrSkipFallsBackInBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer func() {
		simulateExecContextError = nil
		CleanTxHooks()
	}()

	prevTimeout := xaConnTimeout
	xaConnTimeout = time.Minute
	defer func() { xaConnTimeout = prevTimeout }()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)
	// The branch prepares and reports phase-1 success once the in-branch fallback succeeds.
	var reportCnt int32
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, _ interface{}) error {
			atomic.AddInt32(&reportCnt, 1)
			return nil
		})

	// Model go-sql-driver's default behavior: a parameterized business statement answers
	// ErrSkip on the direct Execer path; the XA control statements (XA START/END/PREPARE)
	// succeed. PrepareContext + prepared ExecContext (wired in baseMockConn) succeed.
	simulateExecContextError = func(query string) error {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "UPDATE") {
			return driver.ErrSkip
		}
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	_, err := xaConn.ExecContext(ctx, "UPDATE user SET age = age + 1 WHERE id = ?",
		[]driver.NamedValue{{Ordinal: 1, Value: int64(1)}})

	// The fix: the parameterized UPDATE completes inside the branch via the in-branch
	// Prepare+Exec fallback, and the branch commits (phase-1 reported to the TC).
	assert.NoError(t, err, "parameterized UPDATE should complete via the in-branch Prepare+Exec fallback")
	assert.Equal(t, int32(1), atomic.LoadInt32(&reportCnt),
		"the branch prepares and reports phase-1 success after the in-branch fallback")
}

// The real #904 scenario is a PARAMETERIZED `SELECT ... FOR UPDATE WHERE id = ?`
// (the samples all bind parameters). Under the default MySQL DSN the direct Queryer
// answers driver.ErrSkip for it, so this exercises the query-path in-branch fallback
// (queryPreparedInBranch) AND the #904 busy-buffer guard together: the fallback rows
// must still be wrapped in RowsCommitOnClose so the branch commit (XA END + XA
// PREPARE) is deferred until the caller drains/closes the rows, never issued on top
// of the still-open result set. Closing the rows also closes the prepared stmt
// (rowsWithStmt) and then runs XA END + XA PREPARE + the phase-1 report exactly once.
func TestXAConn_AutoCommit_ParameterizedSelectForUpdateErrSkipDefersBranchCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer func() {
		simulateExecContextError = nil
		simulateQueryContextError = nil
		CleanTxHooks()
	}()

	prevTimeout := xaConnTimeout
	xaConnTimeout = time.Minute
	defer func() { xaConnTimeout = prevTimeout }()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)

	var reportCnt int32
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, param rm.BranchReportParam) error {
			assert.EqualValues(t, branch.BranchStatusPhaseoneDone, param.Status)
			atomic.AddInt32(&reportCnt, 1)
			return nil
		})

	// The XA control statements run on the direct Execer path and succeed; count when
	// the deferred branch commit fires.
	var endCnt, prepareCnt int32
	simulateExecContextError = func(query string) error {
		upper := strings.ToUpper(strings.TrimSpace(query))
		switch {
		case strings.HasPrefix(upper, "XA END"):
			atomic.AddInt32(&endCnt, 1)
		case strings.HasPrefix(upper, "XA PREPARE"):
			atomic.AddInt32(&prepareCnt, 1)
		}
		return nil
	}
	// Model the default go-sql-driver DSN: the direct Queryer answers ErrSkip for the
	// parameterized business SELECT, forcing the in-branch prepared-query fallback.
	simulateQueryContextError = func(query string) error {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") {
			return driver.ErrSkip
		}
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	rows, err := xaConn.QueryContext(ctx, "SELECT * FROM user WHERE id = ? FOR UPDATE",
		[]driver.NamedValue{{Ordinal: 1, Value: int64(1)}})
	assert.NoError(t, err, "parameterized SELECT ... FOR UPDATE must complete via the in-branch prepared-query fallback")

	// Even though we fell back to queryPreparedInBranch, the branch commit must still be
	// deferred while the rows are open - issuing XA END / XA PREPARE now is the #904 bug.
	_, ok := rows.(*RowsCommitOnClose)
	assert.True(t, ok, "the in-branch query fallback must still wrap rows in RowsCommitOnClose to defer the branch commit")
	assert.Equal(t, int32(0), atomic.LoadInt32(&endCnt), "XA END must be deferred until the fallback rows close")
	assert.Equal(t, int32(0), atomic.LoadInt32(&prepareCnt), "XA PREPARE must be deferred until the fallback rows close")
	assert.Equal(t, int32(0), atomic.LoadInt32(&reportCnt), "phase-1 report must be deferred until the fallback rows close")

	// Closing the rows closes both the driver rows and the prepared stmt (rowsWithStmt),
	// then runs the deferred XA END + XA PREPARE + phase-1 report exactly once.
	assert.NoError(t, rows.Close())
	assert.Equal(t, int32(1), atomic.LoadInt32(&endCnt), "XA END runs once the fallback rows close")
	assert.Equal(t, int32(1), atomic.LoadInt32(&prepareCnt), "XA PREPARE runs once the fallback rows close")
	assert.Equal(t, int32(1), atomic.LoadInt32(&reportCnt), "the branch reports phase-1 done once the fallback rows close")
}

// When the in-branch Prepare+Exec fallback itself fails with a real (non-ErrSkip)
// error, the branch must not leak: createNewTxOnExecIfNeed rolls it back and reports
// phase-1 FAILED to the TC, and surfaces the concrete error (never driver.ErrSkip) to
// the caller. This guards the post-fallback error path added with the fix.
func TestXAConn_AutoCommit_InBranchFallbackErrorRollsBackBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	CleanTxHooks()
	defer func() {
		simulateExecContextError = nil
		simulatePreparedExecError = nil
		CleanTxHooks()
	}()

	prevTimeout := xaConnTimeout
	xaConnTimeout = time.Minute
	defer func() { xaConnTimeout = prevTimeout }()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)

	var failedReportCnt int32
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, param rm.BranchReportParam) error {
			if param.Status == branch.BranchStatusPhaseoneFailed {
				atomic.AddInt32(&failedReportCnt, 1)
			}
			return nil
		})

	// Direct Execer answers ErrSkip for the business UPDATE (default DSN behavior); the
	// XA control statements succeed.
	simulateExecContextError = func(query string) error {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "UPDATE") {
			return driver.ErrSkip
		}
		return nil
	}
	// The in-branch prepared exec then fails with a real error (e.g. a constraint
	// violation) - this is NOT ErrSkip, so it must abort and roll back the branch.
	prepErr := errors.New("Error 1062: Duplicate entry for key 'PRIMARY'")
	simulatePreparedExecError = func(query string) error {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "UPDATE") {
			return prepErr
		}
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	_, err := xaConn.ExecContext(ctx, "UPDATE user SET age = age + 1 WHERE id = ?",
		[]driver.NamedValue{{Ordinal: 1, Value: int64(1)}})

	assert.Error(t, err, "a real error from the in-branch fallback must surface")
	assert.False(t, errors.Is(err, driver.ErrSkip), "the caller must never see raw driver.ErrSkip - the fix converts it into a concrete result or error")
	assert.ErrorIs(t, err, prepErr, "the concrete fallback error must be surfaced to the caller")
	assert.Equal(t, int32(1), atomic.LoadInt32(&failedReportCnt),
		"the failed branch must report phase-1 FAILED to the TC so it does not leak")
	assert.False(t, xaConn.xaActive, "the rolled-back branch must leave no active branch on the session")
}

func TestXAConn_BeginTx_DoesNotStartPhysicalTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	xaConn, mockMgr := newMockXAConn(t, ctrl, 123)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, param rm.BranchReportParam) error {
			assert.Equal(t, branch.BranchTypeXA, param.BranchType)
			assert.Equal(t, int64(123), param.BranchId)
			assert.EqualValues(t, branch.BranchStatusPhaseoneFailed, param.Status)
			return nil
		},
	).Times(1)

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := xaConn.BeginTx(ctx, driver.TxOptions{})
	assert.NoError(t, err)

	xaTx, ok := tx.(*XATx)
	if assert.True(t, ok) {
		_, noop := xaTx.tx.target.(xaBranchTx)
		assert.True(t, noop)
	}

	err = tx.Rollback()
	assert.NoError(t, err)
}

func TestXABranchTx_CommitRollbackFailFast(t *testing.T) {
	branchTx := xaBranchTx{}

	err := branchTx.Commit()
	assert.ErrorIs(t, err, errXABranchLifecycleManaged)

	err = branchTx.Rollback()
	assert.ErrorIs(t, err, errXABranchLifecycleManaged)
}
