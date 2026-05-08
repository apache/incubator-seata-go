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
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

type countingDriverTx struct {
	commitCalls   int
	rollbackCalls int
	commitErr     error
	rollbackErr   error
}

func (t *countingDriverTx) Commit() error {
	t.commitCalls++
	return t.commitErr
}

func (t *countingDriverTx) Rollback() error {
	t.rollbackCalls++
	return t.rollbackErr
}

func (m *mysqlMockRows) Columns() []string {
	//TODO implement me
	panic("implement me")
}

func (m *mysqlMockRows) Close() error {
	//TODO implement me
	panic("implement me")
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

type resetCloseTrackingConn struct {
	resetCalls int
	closeCalls int
}

func (c *resetCloseTrackingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *resetCloseTrackingConn) Close() error {
	c.closeCalls++
	return nil
}

func (c *resetCloseTrackingConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

func (c *resetCloseTrackingConn) ResetSession(context.Context) error {
	c.resetCalls++
	return nil
}

type singleConnConnector struct {
	conn driver.Conn
}

func (c *singleConnConnector) Connect(context.Context) (driver.Conn, error) {
	return c.conn, nil
}

func (c *singleConnConnector) Driver() driver.Driver {
	return nil
}

type beginOnlyConn struct {
	tx driver.Tx
}

func (c *beginOnlyConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}

func (c *beginOnlyConn) Close() error {
	return nil
}

func (c *beginOnlyConn) Begin() (driver.Tx, error) {
	return c.tx, nil
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
	xaConnTimeout = time.Hour

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

func initXAConnTestDB(t *testing.T, ctrl *gomock.Controller) *sql.DB {
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

	return db
}

func initXAConnTestResource(t *testing.T) (*gomock.Controller, *sql.DB, *mockSQLInterceptor, *mockTxHook) {
	ctrl := gomock.NewController(t)

	mockMgr := initMockResourceManager(branch.BranchTypeXA, ctrl)
	_ = mockMgr
	db := initXAConnTestDB(t, ctrl)

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

func initXAConnTestResourceWithBranchReport(
	t *testing.T,
	branchID int64,
	reportErr error,
) (*gomock.Controller, *sql.DB, *int32) {
	ctrl := gomock.NewController(t)

	var phaseoneDoneReportCnt int32
	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeXA)
	mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).AnyTimes().Return(branchID, nil)
	mockMgr.EXPECT().BranchReport(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(_ context.Context, param rm.BranchReportParam) error {
			assert.Equal(t, branchID, param.BranchId)
			if param.Status == branch.BranchStatusPhaseoneDone {
				atomic.AddInt32(&phaseoneDoneReportCnt, 1)
				return reportErr
			}
			return nil
		},
	)
	mockMgr.EXPECT().RegisterResource(gomock.Any()).AnyTimes().Return(nil)
	mockMgr.EXPECT().CreateTableMetaCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	rm.GetRmCacheInstance().RegisterResourceManager(mockMgr)

	db := initXAConnTestDB(t, ctrl)

	exec.CleanCommonHook()
	CleanTxHooks()

	return ctrl, db, &phaseoneDoneReportCnt
}

func TestConn_BeginTx_XAModeOracleBeginsTargetTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTx := mock.NewMockTestDriverTx(ctrl)
	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)

	conn := &Conn{
		targetConn: mockConn,
		res:        &DBResource{dbType: types.DBTypeOracle},
		txCtx:      &types.TransactionContext{TransactionMode: types.XAMode},
		dbType:     types.DBTypeOracle,
	}

	tx, err := conn.BeginTx(context.Background(), driver.TxOptions{})
	require.NoError(t, err)
	require.Same(t, mockTx, tx.(*Tx).target)
}

func TestConn_BeginTx_XAModeOracleRestoresAutoCommitOnTargetBeginFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	beginErr := errors.New("begin failed")
	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(nil, beginErr)

	conn := &Conn{
		targetConn: mockConn,
		res:        &DBResource{dbType: types.DBTypeOracle},
		txCtx:      &types.TransactionContext{TransactionMode: types.XAMode},
		autoCommit: true,
		dbType:     types.DBTypeOracle,
	}

	tx, err := conn.BeginTx(context.Background(), driver.TxOptions{})
	require.ErrorIs(t, err, beginErr)
	require.Nil(t, tx)
	assert.True(t, conn.GetAutoCommit())
}

func TestConn_BeginTx_XAModeMySQLKeepsExistingBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Times(0)

	conn := &Conn{
		targetConn: mockConn,
		res:        &DBResource{dbType: types.DBTypeMySQL},
		txCtx:      &types.TransactionContext{TransactionMode: types.XAMode},
		dbType:     types.DBTypeMySQL,
	}

	tx, err := conn.BeginTx(context.Background(), driver.TxOptions{})
	require.NoError(t, err)
	require.Nil(t, tx.(*Tx).target)
}

func TestXAConn_BeginTx_RestoresStateOnBranchRegisterFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registerErr := errors.New("branch register failed")
	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeXA)
	mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Return(int64(0), registerErr)
	rm.GetRmCacheInstance().RegisterResourceManager(mockMgr)

	mockTx := mock.NewMockTestDriverTx(ctrl)
	mockTx.EXPECT().Rollback().Return(nil)

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)

	previousTxCtx := types.NewTxCtx()
	conn := &XAConn{
		Conn: &Conn{
			targetConn: mockConn,
			res:        &DBResource{dbType: types.DBTypeOracle, resourceID: "oracle-resource"},
			txCtx:      previousTxCtx,
			autoCommit: true,
			dbType:     types.DBTypeOracle,
		},
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := conn.BeginTx(ctx, driver.TxOptions{})
	require.ErrorIs(t, err, registerErr)
	require.Nil(t, tx)
	assert.True(t, conn.GetAutoCommit())
	assert.Same(t, previousTxCtx, conn.txCtx)
	assert.Nil(t, conn.tx)
}

func TestXAConn_BeginTx_CleansHeldConnectionOnXAStartFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := mock.NewMockDataSourceManager(ctrl)
	mockMgr.SetBranchType(branch.BranchTypeXA)
	mockMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).Return(int64(13), nil)
	rm.GetRmCacheInstance().RegisterResourceManager(mockMgr)

	startErr := errors.New("xa start failed")

	mockTx := mock.NewMockTestDriverTx(ctrl)
	mockTx.EXPECT().Rollback().Return(nil)

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().BeginTx(gomock.Any(), gomock.Any()).Return(mockTx, nil)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(driver.ResultNoRows, startErr)

	resource := &DBResource{dbType: types.DBTypeOracle, resourceID: "oracle-resource"}
	previousTxCtx := types.NewTxCtx()
	conn := &XAConn{
		Conn: &Conn{
			targetConn: mockConn,
			res:        resource,
			txCtx:      previousTxCtx,
			autoCommit: true,
			dbType:     types.DBTypeOracle,
		},
	}

	ctx := tm.InitSeataContext(context.Background())
	xid := uuid.NewString()
	tm.SetXID(ctx, xid)
	expectedBranchXID := XaIdBuild(xid, 13).String()

	tx, err := conn.BeginTx(ctx, driver.TxOptions{})
	require.ErrorIs(t, err, startErr)
	require.Nil(t, tx)

	_, ok := resource.Lookup(expectedBranchXID)
	assert.False(t, ok)
	assert.True(t, conn.GetAutoCommit())
	assert.Same(t, previousTxCtx, conn.txCtx)
	assert.Nil(t, conn.tx)
}

func TestXAConn_ResetSession_MovesHeldConnectionOutOfPool(t *testing.T) {
	xaID := XaIdBuild("prepared-held-xid", 88)
	replacementConn := &resetCloseTrackingConn{}
	resource := &DBResource{
		dbType:       types.DBTypeOracle,
		shouldBeHeld: true,
		connector:    &singleConnConnector{conn: replacementConn},
	}
	targetConn := &resetCloseTrackingConn{}
	conn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        resource,
			txCtx: &types.TransactionContext{
				XID:             xaID.GetGlobalXid(),
				BranchID:        xaID.GetBranchId(),
				TransactionMode: types.XAMode,
			},
			autoCommit: false,
			dbType:     types.DBTypeOracle,
		},
		xaResource:  &fakeXAResource{},
		xaBranchXid: xaID,
		xaActive:    true,
		prepareTime: time.Now(),
		isConnKept:  true,
	}
	require.NoError(t, resource.Hold(xaID.String(), conn))
	conn.storeHeldConnectionCopy()

	err := conn.ResetSession(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, targetConn.resetCalls)
	assert.Nil(t, conn.xaBranchXid)
	assert.False(t, conn.isConnKept)
	assert.Same(t, replacementConn, conn.Conn.targetConn)

	held, ok := resource.Lookup(xaID.String())
	require.True(t, ok)
	heldConn := held.(*XAConn)
	assert.NotSame(t, conn, heldConn)
	assert.Same(t, targetConn, heldConn.Conn.targetConn)

	require.NoError(t, heldConn.XaCommit(context.Background(), xaID))
	_, ok = resource.Lookup(xaID.String())
	assert.False(t, ok)

	require.NoError(t, heldConn.Close())
	assert.Equal(t, 1, targetConn.closeCalls)
	require.NoError(t, conn.Close())
	assert.Equal(t, 1, replacementConn.closeCalls)
}

func TestXAConn_ResetSession_DoesNotDetachNonHeldPreparedConnection(t *testing.T) {
	xaID := XaIdBuild("prepared-nonheld-xid", 90)
	resource := &DBResource{dbType: types.DBTypeMySQL}
	targetConn := &resetCloseTrackingConn{}
	conn := &XAConn{
		Conn: &Conn{
			targetConn: targetConn,
			res:        resource,
			txCtx: &types.TransactionContext{
				XID:             xaID.GetGlobalXid(),
				BranchID:        xaID.GetBranchId(),
				TransactionMode: types.XAMode,
			},
			autoCommit: false,
			dbType:     types.DBTypeMySQL,
		},
		xaResource:  &fakeXAResource{},
		xaBranchXid: xaID,
		xaActive:    true,
		prepareTime: time.Now(),
		isConnKept:  true,
	}

	err := conn.ResetSession(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, targetConn.resetCalls)
	assert.True(t, conn.GetAutoCommit())
	assert.Nil(t, conn.xaBranchXid)
}

func TestConn_BeginTxFallbackPreservesWrapperState(t *testing.T) {
	targetTx := &countingDriverTx{}
	conn := &Conn{
		targetConn: &beginOnlyConn{tx: targetTx},
		res:        &DBResource{dbType: types.DBTypeMySQL},
		autoCommit: true,
	}

	tx, err := conn.BeginTx(context.Background(), driver.TxOptions{})
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.False(t, conn.GetAutoCommit())
	assert.Equal(t, types.DBTypeMySQL, conn.txCtx.DBType)
	assert.Equal(t, driver.TxOptions{}, conn.txCtx.TxOpt)
}

func TestConn_ExecContext_NormalizesNilDriverResult(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConn := mock.NewMockTestDriverConn(ctrl)
	mockConn.EXPECT().ExecContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(driver.Result(nil), nil)

	conn := &Conn{
		targetConn: mockConn,
		res:        &DBResource{dbType: types.DBTypeOracle},
		txCtx:      types.NewTxCtx(),
		dbType:     types.DBTypeOracle,
	}

	result, err := conn.ExecContext(context.Background(), "INSERT INTO t VALUES (1)", nil)
	require.NoError(t, err)
	require.Equal(t, driver.ResultNoRows, result)
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
		_, sentinel := xaTx.tx.target.(xaBranchTx)
		assert.True(t, sentinel)
	}

	err = tx.Rollback()
	assert.NoError(t, err)
}

func TestXATx_Commit_DrivesXAFirstPhaseLifecycle(t *testing.T) {
	ctrl, db, _, _ := initXAConnTestResource(t)
	defer func() {
		simulateExecContextError = nil
		xaConnTimeout = 0
		CleanTxHooks()
		_ = db.Close()
		ctrl.Finish()
	}()

	xaConnTimeout = time.Hour

	var queries []string
	simulateExecContextError = func(query string) error {
		queries = append(queries, query)
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	require.NoError(t, err)

	_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	assert.Contains(t, strings.Join(queries, "\n"), "XA START")
	assert.Contains(t, strings.Join(queries, "\n"), "XA END")
	assert.Contains(t, strings.Join(queries, "\n"), "XA PREPARE")
}

func TestXATx_Commit_ReturnsBranchReportErrorAfterSuccessfulPrepare(t *testing.T) {
	reportErr := errors.New("branch report failed")
	ctrl, db, reportCnt := initXAConnTestResourceWithBranchReport(t, 11, reportErr)
	defer func() {
		simulateExecContextError = nil
		xaConnTimeout = 0
		CleanTxHooks()
		_ = db.Close()
		ctrl.Finish()
	}()

	xaConnTimeout = time.Hour

	var queries []string
	simulateExecContextError = func(query string) error {
		queries = append(queries, query)
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	require.NoError(t, err)

	_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
	require.NoError(t, err)

	err = tx.Commit()
	require.ErrorIs(t, err, reportErr)

	queryLog := strings.Join(queries, "\n")
	assert.Contains(t, queryLog, "XA START")
	assert.Contains(t, queryLog, "XA END")
	assert.Contains(t, queryLog, "XA PREPARE")
	assert.NotContains(t, queryLog, "XA ROLLBACK")
	assert.Greater(t, atomic.LoadInt32(reportCnt), int32(0))
}

func TestXAConn_ExecContext_ReturnsXAFirstPhaseCommitError(t *testing.T) {
	ctrl, db, _, _ := initXAConnTestResource(t)
	defer func() {
		simulateExecContextError = nil
		xaConnTimeout = 0
		CleanTxHooks()
		_ = db.Close()
		ctrl.Finish()
	}()

	xaConnTimeout = time.Hour

	prepareErr := errors.New("prepare failed")
	var queries []string
	simulateExecContextError = func(query string) error {
		queries = append(queries, query)
		if strings.HasPrefix(strings.ToUpper(query), "XA PREPARE") {
			return prepareErr
		}
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	_, err := db.ExecContext(ctx, "SELECT 1")
	require.ErrorIs(t, err, prepareErr)
	assert.Contains(t, strings.Join(queries, "\n"), "XA PREPARE")
	assert.Contains(t, strings.Join(queries, "\n"), "XA ROLLBACK")
}

func TestXAConn_Commit_SetsPrepareTime(t *testing.T) {
	branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()
	xaConnTimeout = time.Hour
	defer func() {
		xaConnTimeout = 0
	}()

	xaID := XaIdBuild("prepare-time-xid", 1)
	conn := &XAConn{
		Conn: &Conn{
			res: &DBResource{dbType: types.DBTypeOracle},
			txCtx: &types.TransactionContext{
				XID:             xaID.GetGlobalXid(),
				BranchID:        xaID.GetBranchId(),
				TransactionMode: types.XAMode,
			},
		},
		xaResource:         &fakeXAResource{},
		xaBranchXid:        xaID,
		xaActive:           true,
		branchRegisterTime: time.Now(),
	}

	beforeCommit := time.Now()
	require.NoError(t, conn.Commit(context.Background()))

	assert.False(t, conn.prepareTime.IsZero())
	assert.False(t, conn.prepareTime.Before(beforeCommit))
	assert.WithinDuration(t, time.Now(), conn.prepareTime, time.Second)
}

func TestXAConn_Commit_ReadOnlyMarksBranchCommitted(t *testing.T) {
	branchStatusCache = gcache.New(1024).LRU().Expiration(time.Minute * 10).Build()
	xaConnTimeout = time.Hour
	defer func() {
		branchStatusCache = nil
		xaConnTimeout = 0
	}()

	xaID := XaIdBuild("readonly-xid", 1)
	targetTx := &countingDriverTx{}
	dbResource := &DBResource{
		dbType:       types.DBTypeOracle,
		shouldBeHeld: true,
	}
	fakeResource := &fakeXAResource{prepareErr: xa.ErrXAReadOnly}
	conn := &XAConn{
		Conn: &Conn{
			res:    dbResource,
			dbType: types.DBTypeOracle,
			txCtx: &types.TransactionContext{
				XID:             xaID.GetGlobalXid(),
				BranchID:        xaID.GetBranchId(),
				TransactionMode: types.XAMode,
			},
		},
		tx:                 &Tx{target: targetTx},
		xaResource:         fakeResource,
		xaBranchXid:        xaID,
		xaActive:           true,
		branchRegisterTime: time.Now(),
		isConnKept:         true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), conn))

	require.NoError(t, conn.Commit(context.Background()))

	status, err := branchStatus(xaID.String())
	require.NoError(t, err)
	assert.EqualValues(t, branch.BranchStatusPhasetwoCommitted, status)
	assert.Equal(t, []int{xa.TMSuccess}, fakeResource.endFlags)
	assert.Empty(t, fakeResource.rollbackXIDs)
	assert.Equal(t, 1, targetTx.commitCalls)
	assert.Equal(t, 0, targetTx.rollbackCalls)
	_, ok := dbResource.Lookup(xaID.String())
	assert.False(t, ok)
	assert.False(t, conn.xaActive)
	assert.Nil(t, conn.xaBranchXid)
}

func TestXAConn_Rollback_UsesTMSuccessForOracle(t *testing.T) {
	xaID := XaIdBuild("oracle-rollback-xid", 1)
	targetTx := &countingDriverTx{}
	fakeResource := &fakeXAResource{}
	conn := &XAConn{
		Conn: &Conn{
			res:    &DBResource{dbType: types.DBTypeOracle},
			dbType: types.DBTypeOracle,
			txCtx: &types.TransactionContext{
				XID:      xaID.GetGlobalXid(),
				BranchID: xaID.GetBranchId(),
			},
		},
		tx:          &Tx{target: targetTx},
		xaResource:  fakeResource,
		xaBranchXid: xaID,
		xaActive:    true,
	}

	require.NoError(t, conn.Rollback(context.Background()))
	assert.Equal(t, []int{xa.TMSuccess}, fakeResource.endFlags)
	assert.Equal(t, []string{xaID.String()}, fakeResource.rollbackXIDs)
	assert.Equal(t, 0, targetTx.commitCalls)
	assert.Equal(t, 1, targetTx.rollbackCalls)
}

func TestXAConn_Rollback_RollsBackOracleTargetTxWhenEndFails(t *testing.T) {
	xaID := XaIdBuild("oracle-rollback-end-error-xid", 1)
	endErr := errors.New("xa end failed")
	targetTx := &countingDriverTx{}
	dbResource := &DBResource{
		dbType:       types.DBTypeOracle,
		shouldBeHeld: true,
	}
	fakeResource := &fakeXAResource{endErr: endErr}
	conn := &XAConn{
		Conn: &Conn{
			res:    dbResource,
			dbType: types.DBTypeOracle,
			txCtx: &types.TransactionContext{
				XID:      xaID.GetGlobalXid(),
				BranchID: xaID.GetBranchId(),
			},
		},
		tx:                &Tx{target: targetTx},
		xaResource:        fakeResource,
		xaErrorClassifier: &xa.OracleXAErrorClassifier{},
		xaBranchXid:       xaID,
		xaActive:          true,
		isConnKept:        true,
	}
	require.NoError(t, dbResource.Hold(xaID.String(), conn))

	err := conn.Rollback(context.Background())

	require.ErrorIs(t, err, endErr)
	assert.Equal(t, []int{xa.TMSuccess}, fakeResource.endFlags)
	assert.Empty(t, fakeResource.rollbackXIDs)
	assert.Equal(t, 0, targetTx.commitCalls)
	assert.Equal(t, 1, targetTx.rollbackCalls)
	_, ok := dbResource.Lookup(xaID.String())
	assert.False(t, ok)
	assert.Nil(t, conn.xaBranchXid)
	assert.True(t, conn.autoCommit)
}

func TestXAConn_Rollback_UsesTMFailForNonOracle(t *testing.T) {
	xaID := XaIdBuild("mysql-rollback-xid", 1)
	targetTx := &countingDriverTx{}
	fakeResource := &fakeXAResource{}
	conn := &XAConn{
		Conn: &Conn{
			res:    &DBResource{dbType: types.DBTypeMySQL},
			dbType: types.DBTypeMySQL,
			txCtx: &types.TransactionContext{
				XID:      xaID.GetGlobalXid(),
				BranchID: xaID.GetBranchId(),
			},
		},
		tx:          &Tx{target: targetTx},
		xaResource:  fakeResource,
		xaBranchXid: xaID,
		xaActive:    true,
	}

	require.NoError(t, conn.Rollback(context.Background()))
	assert.Equal(t, []int{xa.TMFail}, fakeResource.endFlags)
	assert.Equal(t, []string{xaID.String()}, fakeResource.rollbackXIDs)
	assert.Equal(t, 0, targetTx.commitCalls)
	assert.Equal(t, 0, targetTx.rollbackCalls)
}

func TestXATx_Rollback_DrivesXAEndAndRollback(t *testing.T) {
	ctrl, db, _, _ := initXAConnTestResource(t)
	defer func() {
		simulateExecContextError = nil
		xaConnTimeout = 0
		CleanTxHooks()
		_ = db.Close()
		ctrl.Finish()
	}()

	var queries []string
	simulateExecContextError = func(query string) error {
		queries = append(queries, query)
		return nil
	}

	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, uuid.NewString())

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	require.NoError(t, err)

	_, err = tx.ExecContext(context.Background(), "SELECT * FROM user")
	require.NoError(t, err)

	err = tx.Rollback()
	require.NoError(t, err)

	assert.Contains(t, strings.Join(queries, "\n"), "XA START")
	assert.Contains(t, strings.Join(queries, "\n"), "XA END")
	assert.Contains(t, strings.Join(queries, "\n"), "XA ROLLBACK")
}
