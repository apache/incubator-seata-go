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
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

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

func TestBegin(t *testing.T) {
	tx, err := (&FenceConn{}).Begin()
	assert.NotNil(t, err)
	assert.Nil(t, tx)
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
