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

package fence

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"flag"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

func TestInitFenceConfig(t *testing.T) {
	log.Init()

	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "config with enable=false",
			config: Config{
				Enable:       false,
				Url:          "test-url",
				LogTableName: "test_table",
				CleanPeriod:  10 * time.Minute,
			},
		},
		{
			name: "config with enable=true",
			config: Config{
				Enable:       true,
				Url:          "",
				LogTableName: "tcc_fence_log",
				CleanPeriod:  5 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitFenceConfig(tt.config)
			assert.Equal(t, tt.config, FenceConfig)
		})
	}
}

func TestConfig_RegisterFlagsWithPrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "register with prefix 'tcc'",
			prefix: "tcc",
		},
		{
			name:   "register with prefix 'fence'",
			prefix: "fence",
		},
		{
			name:   "register with empty prefix",
			prefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)

			cfg.RegisterFlagsWithPrefix(tt.prefix, fs)

			// Verify flags are registered
			assert.NotNil(t, fs.Lookup(tt.prefix+".enable"))
			assert.NotNil(t, fs.Lookup(tt.prefix+".url"))
			assert.NotNil(t, fs.Lookup(tt.prefix+".log-table-name"))
			assert.NotNil(t, fs.Lookup(tt.prefix+".clean-period"))
		})
	}
}

func TestDoFence_AllPhases(t *testing.T) {
	log.Init()

	tests := []struct {
		name       string
		fencePhase enum.FencePhase
		xid        string
		txName     string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "fence phase not exist",
			fencePhase: enum.FencePhaseNotExist,
			xid:        "test-xid-001",
			txName:     "test-tx",
			wantErr:    true,
			errMsg:     "xid test-xid-001, tx name test-tx, fence phase not exist",
		},
		{
			name:       "illegal fence phase",
			fencePhase: enum.FencePhase(99),
			xid:        "test-xid-002",
			txName:     "test-tx",
			wantErr:    true,
			errMsg:     "fence phase: 99 illegal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			mock.ExpectBegin()
			tx, err := db.Begin()
			assert.NoError(t, err)

			ctx := context.Background()
			ctx = tm.InitSeataContext(ctx)
			tm.SetXID(ctx, tt.xid)
			tm.SetTxName(ctx, tt.txName)
			tm.SetFencePhase(ctx, tt.fencePhase)

			err = DoFence(ctx, tx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWithFence_CallbackError(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhaseNotExist)

	callbackErr := errors.New("business logic error")
	callback := func() error {
		return callbackErr
	}

	err = WithFence(ctx, tx, callback)
	assert.Error(t, err)
}

// Mock implementations for driver interfaces
type mockConn struct {
	driver.Conn
	prepareFunc      func(query string) (driver.Stmt, error)
	beginTxFunc      func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)
	closeFunc        func() error
	resetSessionFunc func(ctx context.Context) error
}

func (m *mockConn) Prepare(query string) (driver.Stmt, error) {
	if m.prepareFunc != nil {
		return m.prepareFunc(query)
	}
	return nil, nil
}

func (m *mockConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if m.beginTxFunc != nil {
		return m.beginTxFunc(ctx, opts)
	}
	return nil, errors.New("not implemented")
}

func (m *mockConn) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockConn) ResetSession(ctx context.Context) error {
	if m.resetSessionFunc != nil {
		return m.resetSessionFunc(ctx)
	}
	return driver.ErrSkip
}

// mockConnWithExec implements driver.Execer
type mockConnWithExec struct {
	*mockConn
	execFunc func(query string, args []driver.Value) (driver.Result, error)
}

func (m *mockConnWithExec) Exec(query string, args []driver.Value) (driver.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(query, args)
	}
	return nil, driver.ErrSkip
}

// mockConnWithExecContext implements driver.ExecerContext
type mockConnWithExecContext struct {
	*mockConn
	execContextFunc func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error)
}

func (m *mockConnWithExecContext) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

// mockConnWithQuery implements driver.Queryer
type mockConnWithQuery struct {
	*mockConn
	queryFunc func(query string, args []driver.Value) (driver.Rows, error)
}

func (m *mockConnWithQuery) Query(query string, args []driver.Value) (driver.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(query, args)
	}
	return nil, driver.ErrSkip
}

// mockConnWithQueryContext implements driver.QueryerContext
type mockConnWithQueryContext struct {
	*mockConn
	queryContextFunc func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error)
}

func (m *mockConnWithQueryContext) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if m.queryContextFunc != nil {
		return m.queryContextFunc(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

// mockConnWithResetSession implements driver.SessionResetter
type mockConnWithResetSession struct {
	*mockConn
	resetFunc func(ctx context.Context) error
}

func (m *mockConnWithResetSession) ResetSession(ctx context.Context) error {
	if m.resetFunc != nil {
		return m.resetFunc(ctx)
	}
	return nil
}

type mockTx struct {
	driver.Tx
	commitFunc   func() error
	rollbackFunc func() error
}

func (m *mockTx) Commit() error {
	if m.commitFunc != nil {
		return m.commitFunc()
	}
	return nil
}

func (m *mockTx) Rollback() error {
	if m.rollbackFunc != nil {
		return m.rollbackFunc()
	}
	return nil
}

type mockResult struct {
	driver.Result
}

func (m *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

func TestFenceConn_Prepare(t *testing.T) {
	mockConn := &mockConn{
		prepareFunc: func(query string) (driver.Stmt, error) {
			return nil, nil
		},
	}

	fenceConn := &FenceConn{TargetConn: mockConn}
	stmt, err := fenceConn.Prepare("SELECT 1")
	assert.NoError(t, err)
	assert.Nil(t, stmt)
}

func TestFenceConn_PrepareContext(t *testing.T) {
	mockConn := &mockConn{
		prepareFunc: func(query string) (driver.Stmt, error) {
			return nil, nil
		},
	}

	fenceConn := &FenceConn{TargetConn: mockConn}
	stmt, err := fenceConn.PrepareContext(context.Background(), "SELECT 1")
	assert.NoError(t, err)
	assert.Nil(t, stmt)
}

func TestFenceConn_Exec(t *testing.T) {
	tests := []struct {
		name    string
		conn    driver.Conn
		wantErr bool
	}{
		{
			name: "exec supported",
			conn: &mockConnWithExec{
				mockConn: &mockConn{},
				execFunc: func(query string, args []driver.Value) (driver.Result, error) {
					return &mockResult{}, nil
				},
			},
			wantErr: false,
		},
		{
			name:    "exec not supported",
			conn:    &mockConn{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fenceConn := &FenceConn{TargetConn: tt.conn}
			result, err := fenceConn.Exec("INSERT INTO test VALUES (?)", []driver.Value{1})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, driver.ErrSkip, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestFenceConn_ExecContext(t *testing.T) {
	tests := []struct {
		name    string
		conn    driver.Conn
		wantErr bool
		errType error
	}{
		{
			name: "exec context supported",
			conn: &mockConnWithExecContext{
				mockConn: &mockConn{},
				execContextFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
					return &mockResult{}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "fallback to exec - exec supported",
			conn: &mockConnWithExec{
				mockConn: &mockConn{},
				execFunc: func(query string, args []driver.Value) (driver.Result, error) {
					return &mockResult{}, nil
				},
			},
			wantErr: false,
		},
		{
			name:    "fallback to exec - exec not supported",
			conn:    &mockConn{},
			wantErr: true,
			errType: driver.ErrSkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fenceConn := &FenceConn{TargetConn: tt.conn}
			args := []driver.NamedValue{{Ordinal: 1, Value: 1}}
			result, err := fenceConn.ExecContext(context.Background(), "INSERT INTO test VALUES (?)", args)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestFenceConn_Query(t *testing.T) {
	tests := []struct {
		name    string
		conn    driver.Conn
		wantErr bool
	}{
		{
			name: "query supported",
			conn: &mockConnWithQuery{
				mockConn: &mockConn{},
				queryFunc: func(query string, args []driver.Value) (driver.Rows, error) {
					return nil, nil
				},
			},
			wantErr: false,
		},
		{
			name:    "query not supported",
			conn:    &mockConn{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fenceConn := &FenceConn{TargetConn: tt.conn}
			rows, err := fenceConn.Query("SELECT * FROM test", []driver.Value{})

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, driver.ErrSkip, err)
			} else {
				assert.NoError(t, err)
				assert.Nil(t, rows)
			}
		})
	}
}

func TestFenceConn_QueryContext(t *testing.T) {
	tests := []struct {
		name    string
		conn    driver.Conn
		wantErr bool
		errType error
	}{
		{
			name: "query context supported",
			conn: &mockConnWithQueryContext{
				mockConn: &mockConn{},
				queryContextFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
					return nil, nil
				},
			},
			wantErr: false,
		},
		{
			name: "fallback to query - query supported",
			conn: &mockConnWithQuery{
				mockConn: &mockConn{},
				queryFunc: func(query string, args []driver.Value) (driver.Rows, error) {
					return nil, nil
				},
			},
			wantErr: false,
		},
		{
			name:    "fallback to query - query not supported",
			conn:    &mockConn{},
			wantErr: true,
			errType: driver.ErrSkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fenceConn := &FenceConn{TargetConn: tt.conn}
			args := []driver.NamedValue{{Ordinal: 1, Value: 1}}
			rows, err := fenceConn.QueryContext(context.Background(), "SELECT * FROM test WHERE id = ?", args)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
				assert.Nil(t, rows)
			}
		})
	}
}

func TestFenceConn_ResetSession(t *testing.T) {
	tests := []struct {
		name    string
		conn    driver.Conn
		wantErr bool
		errType error
	}{
		{
			name: "reset session supported",
			conn: &mockConnWithResetSession{
				mockConn: &mockConn{},
				resetFunc: func(ctx context.Context) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name:    "reset session not supported",
			conn:    &mockConn{},
			wantErr: true,
			errType: driver.ErrSkip,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fenceConn := &FenceConn{TargetConn: tt.conn}
			err := fenceConn.ResetSession(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.Equal(t, tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFenceConn_Close(t *testing.T) {
	tests := []struct {
		name    string
		conn    *mockConn
		wantErr bool
	}{
		{
			name: "close success",
			conn: &mockConn{
				closeFunc: func() error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "close with error",
			conn: &mockConn{
				closeFunc: func() error {
					return errors.New("close error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fenceConn := &FenceConn{TargetConn: tt.conn}
			err := fenceConn.Close()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFenceTx_Commit(t *testing.T) {
	log.Init()

	tests := []struct {
		name         string
		targetTx     *mockTx
		wantErr      bool
		setupContext func() context.Context
	}{
		{
			name: "commit success",
			targetTx: &mockTx{
				commitFunc: func() error {
					return nil
				},
			},
			wantErr: false,
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = tm.InitSeataContext(ctx)
				tm.SetFenceTxBeginedFlag(ctx, true)
				return ctx
			},
		},
		{
			name: "commit with target tx error",
			targetTx: &mockTx{
				commitFunc: func() error {
					return errors.New("commit failed")
				},
			},
			wantErr: true,
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = tm.InitSeataContext(ctx)
				return ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			mock.ExpectBegin()
			fenceTx, err := db.Begin()
			assert.NoError(t, err)

			if !tt.wantErr {
				mock.ExpectCommit()
			}

			ctx := tt.setupContext()
			tx := &FenceTx{
				Ctx:           ctx,
				TargetTx:      tt.targetTx,
				TargetFenceTx: fenceTx,
			}

			err = tx.Commit()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify fence tx flag is cleared
				assert.False(t, tm.IsFenceTxBegin(ctx))
			}
		})
	}
}

func TestFenceTx_Rollback(t *testing.T) {
	log.Init()

	tests := []struct {
		name         string
		targetTx     *mockTx
		wantErr      bool
		setupContext func() context.Context
	}{
		{
			name: "rollback success",
			targetTx: &mockTx{
				rollbackFunc: func() error {
					return nil
				},
			},
			wantErr: false,
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = tm.InitSeataContext(ctx)
				tm.SetFenceTxBeginedFlag(ctx, true)
				return ctx
			},
		},
		{
			name: "rollback with target tx error",
			targetTx: &mockTx{
				rollbackFunc: func() error {
					return errors.New("rollback failed")
				},
			},
			wantErr: true,
			setupContext: func() context.Context {
				ctx := context.Background()
				ctx = tm.InitSeataContext(ctx)
				return ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			mock.ExpectBegin()
			fenceTx, err := db.Begin()
			assert.NoError(t, err)

			if !tt.wantErr {
				mock.ExpectRollback()
			}

			ctx := tt.setupContext()
			tx := &FenceTx{
				Ctx:           ctx,
				TargetTx:      tt.targetTx,
				TargetFenceTx: fenceTx,
			}

			err = tx.Rollback()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify fence tx flag is cleared
				assert.False(t, tm.IsFenceTxBegin(ctx))
			}
		})
	}
}

func TestSeataFenceConnector_Connect(t *testing.T) {
	mockConn := &mockConn{}
	mockConnector := &mockConnector{
		connectFunc: func(ctx context.Context) (driver.Conn, error) {
			return mockConn, nil
		},
	}

	connector := &SeataFenceConnector{
		TargetConnector: mockConnector,
		TargetDB:        &sql.DB{},
	}

	conn, err := connector.Connect(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	fenceConn, ok := conn.(*FenceConn)
	assert.True(t, ok)
	assert.Equal(t, mockConn, fenceConn.TargetConn)
}

func TestSeataFenceConnector_Driver(t *testing.T) {
	mockDriver := &FenceDriver{}
	mockConnector := &mockConnector{
		driverFunc: func() driver.Driver {
			return mockDriver
		},
	}

	connector := &SeataFenceConnector{
		TargetConnector: mockConnector,
	}

	driver := connector.Driver()
	assert.Equal(t, mockDriver, driver)
}

type mockConnector struct {
	connectFunc func(ctx context.Context) (driver.Conn, error)
	driverFunc  func() driver.Driver
}

func (m *mockConnector) Connect(ctx context.Context) (driver.Conn, error) {
	if m.connectFunc != nil {
		return m.connectFunc(ctx)
	}
	return nil, nil
}

func (m *mockConnector) Driver() driver.Driver {
	if m.driverFunc != nil {
		return m.driverFunc()
	}
	return nil
}

// Mock connection without BeginTx support
type mockConnNoBeginTx struct {
	driver.Conn
}

func (m *mockConnNoBeginTx) Prepare(query string) (driver.Stmt, error) {
	return nil, errors.New("not used")
}

func (m *mockConnNoBeginTx) Close() error {
	return nil
}

// Additional tests for BeginTx
func TestFenceConn_BeginTx_NotSupported(t *testing.T) {
	log.Init()

	mockConn := &mockConnNoBeginTx{}
	fenceConn := &FenceConn{TargetConn: mockConn}

	tx, err := fenceConn.BeginTx(context.Background(), driver.TxOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation unsupported")
	assert.Nil(t, tx)
}

func TestFenceConn_BeginTx_NoSeataContext(t *testing.T) {
	log.Init()

	mockConnWithBegin := &mockConnWithBeginTx{
		mockConn: &mockConn{},
		beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
			return &mockTx{}, nil
		},
	}

	fenceConn := &FenceConn{
		TargetConn: mockConnWithBegin,
		TargetDB:   &sql.DB{},
	}

	tx, err := fenceConn.BeginTx(context.Background(), driver.TxOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "there is not seata context")
	assert.Nil(t, tx)
}

// mockConnWithBeginTx implements driver.ConnBeginTx
type mockConnWithBeginTx struct {
	*mockConn
	beginTxFunc func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)
}

func (m *mockConnWithBeginTx) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if m.beginTxFunc != nil {
		return m.beginTxFunc(ctx, opts)
	}
	return nil, errors.New("not implemented")
}

func TestDoFence_PreparePhase(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectPrepare("insert into  tcc_fence_log  (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	assert.NoError(t, err)

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid-prepare")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhasePrepare)
	bac := &tm.BusinessActionContext{
		Xid:        "test-xid-prepare",
		BranchId:   123,
		ActionName: "test-action",
	}
	tm.SetBusinessActionContext(ctx, bac)

	err = DoFence(ctx, tx)
	assert.NoError(t, err)

	tx.Commit()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDoFence_CommitPhase(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectPrepare("select xid, branch_id, action_name, status, gmt_create, gmt_modified from  tcc_fence_log  where xid = ? and branch_id = ? for update").
		ExpectQuery().
		WithArgs("test-xid-commit", int64(456)).
		WillReturnRows(sqlmock.NewRows([]string{"xid", "branch_id", "action_name", "status", "gmt_create", "gmt_modified"}).
			AddRow("test-xid-commit", int64(456), "test-action", enum.StatusTried, now, now))
	mock.ExpectPrepare("update  tcc_fence_log  set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? ").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	assert.NoError(t, err)

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid-commit")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhaseCommit)
	bac := &tm.BusinessActionContext{
		Xid:        "test-xid-commit",
		BranchId:   456,
		ActionName: "test-action",
	}
	tm.SetBusinessActionContext(ctx, bac)

	err = DoFence(ctx, tx)
	assert.NoError(t, err)

	tx.Commit()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDoFence_RollbackPhase(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectPrepare("select xid, branch_id, action_name, status, gmt_create, gmt_modified from  tcc_fence_log  where xid = ? and branch_id = ? for update").
		ExpectQuery().
		WithArgs("test-xid-rollback", int64(789)).
		WillReturnRows(sqlmock.NewRows([]string{"xid", "branch_id", "action_name", "status", "gmt_create", "gmt_modified"}).
			AddRow("test-xid-rollback", int64(789), "test-action", enum.StatusTried, now, now))
	mock.ExpectPrepare("update  tcc_fence_log  set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? ").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	assert.NoError(t, err)

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid-rollback")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhaseRollback)
	bac := &tm.BusinessActionContext{
		Xid:        "test-xid-rollback",
		BranchId:   789,
		ActionName: "test-action",
	}
	tm.SetBusinessActionContext(ctx, bac)

	err = DoFence(ctx, tx)
	assert.NoError(t, err)

	tx.Commit()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithFence_CallbackSuccess(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectPrepare("insert into  tcc_fence_log  (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	assert.NoError(t, err)

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid-success")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhasePrepare)
	bac := &tm.BusinessActionContext{
		Xid:        "test-xid-success",
		BranchId:   999,
		ActionName: "test-action",
	}
	tm.SetBusinessActionContext(ctx, bac)

	callbackExecuted := false
	callback := func() error {
		callbackExecuted = true
		return nil
	}

	err = WithFence(ctx, tx, callback)
	assert.NoError(t, err)
	assert.True(t, callbackExecuted)

	tx.Commit()
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Test BeginTx with BeginTx call failure
func TestFenceConn_BeginTx_BeginTxError(t *testing.T) {
	log.Init()

	expectedErr := errors.New("begin tx failed")
	mockConnWithBegin := &mockConnWithBeginTx{
		mockConn: &mockConn{},
		beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
			return nil, expectedErr
		},
	}

	fenceConn := &FenceConn{
		TargetConn: mockConnWithBegin,
	}

	tx, err := fenceConn.BeginTx(context.Background(), driver.TxOptions{})
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, tx)
}

// Test BeginTx with fence transaction already begun
func TestFenceConn_BeginTx_FenceTxAlreadyBegun(t *testing.T) {
	log.Init()

	mockTx := &mockTx{}
	mockConnWithBegin := &mockConnWithBeginTx{
		mockConn: &mockConn{},
		beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
			return mockTx, nil
		},
	}

	fenceConn := &FenceConn{
		TargetConn: mockConnWithBegin,
	}

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid")
	tm.SetFenceTxBeginedFlag(ctx, true) // Mark fence tx as already begun

	tx, err := fenceConn.BeginTx(ctx, driver.TxOptions{})
	assert.NoError(t, err)
	assert.Equal(t, mockTx, tx) // Should return the original tx directly
}

// Test BeginTx with TargetDB.BeginTx failure
func TestFenceConn_BeginTx_TargetDBBeginTxError(t *testing.T) {
	log.Init()

	mockTx := &mockTx{
		rollbackFunc: func() error {
			return nil
		},
	}
	mockConnWithBegin := &mockConnWithBeginTx{
		mockConn: &mockConn{},
		beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
			return mockTx, nil
		},
	}

	// Create a closed database to trigger BeginTx error
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	db.Close() // Close it to cause BeginTx to fail

	fenceConn := &FenceConn{
		TargetConn: mockConnWithBegin,
		TargetDB:   db,
	}

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid")
	tm.SetFenceTxBeginedFlag(ctx, false)

	tx, err := fenceConn.BeginTx(ctx, driver.TxOptions{})
	assert.Error(t, err)
	assert.Nil(t, tx)
}

// Test BeginTx with WithFence failure
func TestFenceConn_BeginTx_WithFenceError(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mockTx := &mockTx{
		rollbackFunc: func() error {
			return nil
		},
	}
	mockConnWithBegin := &mockConnWithBeginTx{
		mockConn: &mockConn{},
		beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
			return mockTx, nil
		},
	}

	// Expect both transactions to begin
	mock.ExpectBegin()

	fenceConn := &FenceConn{
		TargetConn: mockConnWithBegin,
		TargetDB:   db,
	}

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhaseNotExist) // This will cause WithFence to fail
	tm.SetFenceTxBeginedFlag(ctx, false)

	tx, err := fenceConn.BeginTx(ctx, driver.TxOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fence phase not exist")
	assert.Nil(t, tx)
}

// Test BeginTx successful path
func TestFenceConn_BeginTx_Success(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	mockTx := &mockTx{
		rollbackFunc: func() error {
			return nil
		},
	}
	mockConnWithBegin := &mockConnWithBeginTx{
		mockConn: &mockConn{},
		beginTxFunc: func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
			return mockTx, nil
		},
	}

	// Expect fence transaction operations
	mock.ExpectBegin()
	mock.ExpectPrepare("insert into  tcc_fence_log  (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Note: No ExpectCommit here - the transaction is kept open in FenceTx

	fenceConn := &FenceConn{
		TargetConn: mockConnWithBegin,
		TargetDB:   db,
	}

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhasePrepare)
	tm.SetFenceTxBeginedFlag(ctx, false)

	bac := &tm.BusinessActionContext{
		Xid:           "test-xid",
		BranchId:      1001,
		ActionName:    "test-action",
		ActionContext: make(map[string]interface{}),
	}
	tm.SetBusinessActionContext(ctx, bac)

	tx, err := fenceConn.BeginTx(ctx, driver.TxOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	fenceTx, ok := tx.(*FenceTx)
	assert.True(t, ok)
	assert.Equal(t, mockTx, fenceTx.TargetTx)
	assert.NotNil(t, fenceTx.TargetFenceTx)
	assert.Equal(t, ctx, fenceTx.Ctx)

	// Clean up - the fence tx needs to be rolled back or committed
	// For this test, we just verify the expectations were met so far
	// In real usage, the FenceTx would be committed/rolled back by the caller
}

// Test ResetSession with error
func TestFenceConn_ResetSession_Error(t *testing.T) {
	log.Init()

	expectedErr := errors.New("reset session failed")
	mockConnWithReset := &mockConnWithResetSession{
		mockConn: &mockConn{},
		resetFunc: func(ctx context.Context) error {
			return expectedErr
		},
	}

	fenceConn := &FenceConn{TargetConn: mockConnWithReset}
	err := fenceConn.ResetSession(context.Background())
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

// Test WithFence callback error formatting
func TestWithFence_CallbackErrorFormatting(t *testing.T) {
	log.Init()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectPrepare("insert into  tcc_fence_log  (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	assert.NoError(t, err)

	ctx := context.Background()
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, "test-xid")
	tm.SetTxName(ctx, "test-tx")
	tm.SetFencePhase(ctx, enum.FencePhasePrepare)

	bac := &tm.BusinessActionContext{
		Xid:           "test-xid",
		BranchId:      1001,
		ActionName:    "test-action",
		ActionContext: make(map[string]interface{}),
	}
	tm.SetBusinessActionContext(ctx, bac)

	expectedCallbackErr := errors.New("specific business error")
	callback := func() error {
		return expectedCallbackErr
	}

	err = WithFence(ctx, tx, callback)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the business method error msg of:")
	assert.ErrorIs(t, err, expectedCallbackErr)
}

// Test OpenConnector with DriverContext.OpenConnector error
func TestFenceDriver_OpenConnector_DriverContextError(t *testing.T) {
	log.Init()

	expectedErr := errors.New("open connector failed")
	mockDriverCtx := &mockDriverContext{
		openConnectorFunc: func(name string) (driver.Connector, error) {
			return nil, expectedErr
		},
	}

	fd := &FenceDriver{
		TargetDriver: mockDriverCtx,
	}

	connector, err := fd.OpenConnector("test-dsn")
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, connector)
}

// Test SeataFenceConnector.Connect with error
func TestSeataFenceConnector_Connect_Error(t *testing.T) {
	log.Init()

	expectedErr := errors.New("connect failed")
	mockConnector := &mockConnector{
		connectFunc: func(ctx context.Context) (driver.Conn, error) {
			return nil, expectedErr
		},
	}

	connector := &SeataFenceConnector{
		TargetConnector: mockConnector,
		TargetDB:        &sql.DB{},
	}

	conn, err := connector.Connect(context.Background())
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, conn)
}

// Mock DriverContext for testing
type mockDriverContext struct {
	driver.Driver
	openConnectorFunc func(name string) (driver.Connector, error)
}

func (m *mockDriverContext) OpenConnector(name string) (driver.Connector, error) {
	if m.openConnectorFunc != nil {
		return m.openConnectorFunc(name)
	}
	return nil, errors.New("not implemented")
}

func (m *mockDriverContext) Open(name string) (driver.Conn, error) {
	return nil, errors.New("not used")
}
