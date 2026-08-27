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

package util

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
)

// Mock implementations for testing
type mockConn struct {
	prepareFunc func(query string) (driver.Stmt, error)
}

func (m *mockConn) Prepare(query string) (driver.Stmt, error) {
	if m.prepareFunc != nil {
		return m.prepareFunc(query)
	}
	return nil, errors.New("not implemented")
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) Begin() (driver.Tx, error) {
	return nil, errors.New("not implemented")
}

type mockConnPrepareContext struct {
	mockConn
	prepareContextFunc func(ctx context.Context, query string) (driver.Stmt, error)
}

func (m *mockConnPrepareContext) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if m.prepareContextFunc != nil {
		return m.prepareContextFunc(ctx, query)
	}
	return nil, errors.New("not implemented")
}

type mockStmt struct {
	closeFunc    func() error
	numInputFunc func() int
	execFunc     func(args []driver.Value) (driver.Result, error)
	queryFunc    func(args []driver.Value) (driver.Rows, error)
}

func (m *mockStmt) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockStmt) NumInput() int {
	if m.numInputFunc != nil {
		return m.numInputFunc()
	}
	return 0
}

func (m *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(args)
	}
	return nil, errors.New("not implemented")
}

func (m *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(args)
	}
	return nil, errors.New("not implemented")
}

type mockStmtExecContext struct {
	mockStmt
	execContextFunc func(ctx context.Context, args []driver.NamedValue) (driver.Result, error)
}

func (m *mockStmtExecContext) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, args)
	}
	return nil, errors.New("not implemented")
}

type mockStmtQueryContext struct {
	mockStmt
	queryContextFunc func(ctx context.Context, args []driver.NamedValue) (driver.Rows, error)
}

func (m *mockStmtQueryContext) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if m.queryContextFunc != nil {
		return m.queryContextFunc(ctx, args)
	}
	return nil, errors.New("not implemented")
}

type mockExecer struct {
	execFunc func(query string, args []driver.Value) (driver.Result, error)
}

func (m *mockExecer) Exec(query string, args []driver.Value) (driver.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(query, args)
	}
	return nil, errors.New("not implemented")
}

type mockExecerContext struct {
	execContextFunc func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error)
}

func (m *mockExecerContext) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, query, args)
	}
	return nil, errors.New("not implemented")
}

type mockQueryer struct {
	queryFunc func(query string, args []driver.Value) (driver.Rows, error)
}

func (m *mockQueryer) Query(query string, args []driver.Value) (driver.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(query, args)
	}
	return nil, errors.New("not implemented")
}

type mockQueryerContext struct {
	queryContextFunc func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error)
}

func (m *mockQueryerContext) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if m.queryContextFunc != nil {
		return m.queryContextFunc(ctx, query, args)
	}
	return nil, errors.New("not implemented")
}

type mockResult struct {
	lastInsertId int64
	rowsAffected int64
}

func (m *mockResult) LastInsertId() (int64, error) {
	return m.lastInsertId, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

type mockRows struct {
	columns []string
}

func (m *mockRows) Columns() []string {
	return m.columns
}

func (m *mockRows) Close() error {
	return nil
}

func (m *mockRows) Next(dest []driver.Value) error {
	return errors.New("no more rows")
}

// Tests for ctxDriverPrepare
func TestCtxDriverPrepare_WithContext(t *testing.T) {
	mockStmt := &mockStmt{}
	mockConn := &mockConnPrepareContext{
		prepareContextFunc: func(ctx context.Context, query string) (driver.Stmt, error) {
			assert.Equal(t, "SELECT * FROM test", query)
			return mockStmt, nil
		},
	}
	ctx := context.Background()
	query := "SELECT * FROM test"

	result, err := ctxDriverPrepare(ctx, mockConn, query)
	assert.NoError(t, err)
	assert.Equal(t, mockStmt, result)
}

func TestCtxDriverPrepare_WithoutContext(t *testing.T) {
	mockStmt := &mockStmt{}
	mockConn := &mockConn{
		prepareFunc: func(query string) (driver.Stmt, error) {
			assert.Equal(t, "SELECT * FROM test", query)
			return mockStmt, nil
		},
	}
	ctx := context.Background()
	query := "SELECT * FROM test"

	result, err := ctxDriverPrepare(ctx, mockConn, query)
	assert.NoError(t, err)
	assert.Equal(t, mockStmt, result)
}

func TestCtxDriverPrepare_WithCancelledContext(t *testing.T) {
	closeCalled := false
	mockStmt := &mockStmt{
		closeFunc: func() error {
			closeCalled = true
			return nil
		},
	}
	mockConn := &mockConn{
		prepareFunc: func(query string) (driver.Stmt, error) {
			return mockStmt, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	query := "SELECT * FROM test"

	result, err := ctxDriverPrepare(ctx, mockConn, query)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)
	assert.True(t, closeCalled)
}

func TestCtxDriverPrepare_PrepareError(t *testing.T) {
	expectedErr := errors.New("prepare error")
	mockConn := &mockConn{
		prepareFunc: func(query string) (driver.Stmt, error) {
			return nil, expectedErr
		},
	}
	ctx := context.Background()
	query := "SELECT * FROM test"

	result, err := ctxDriverPrepare(ctx, mockConn, query)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, expectedErr, err)
}

// Tests for ctxDriverExec
func TestCtxDriverExec_WithExecerContext(t *testing.T) {
	mockResult := &mockResult{lastInsertId: 1, rowsAffected: 1}
	mockExecerCtx := &mockExecerContext{
		execContextFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
			assert.Equal(t, "INSERT INTO test VALUES (?)", query)
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0].Value)
			return mockResult, nil
		},
	}
	ctx := context.Background()
	query := "INSERT INTO test VALUES (?)"
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverExec(ctx, mockExecerCtx, nil, query, args)
	assert.NoError(t, err)
	assert.Equal(t, mockResult, result)
}

func TestCtxDriverExec_WithoutExecerContext(t *testing.T) {
	mockResult := &mockResult{lastInsertId: 1, rowsAffected: 1}
	mockExecer := &mockExecer{
		execFunc: func(query string, args []driver.Value) (driver.Result, error) {
			assert.Equal(t, "INSERT INTO test VALUES (?)", query)
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0])
			return mockResult, nil
		},
	}
	ctx := context.Background()
	query := "INSERT INTO test VALUES (?)"
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverExec(ctx, nil, mockExecer, query, args)
	assert.NoError(t, err)
	assert.Equal(t, mockResult, result)
}

func TestCtxDriverExec_WithCancelledContext(t *testing.T) {
	mockExecer := &mockExecer{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	query := "INSERT INTO test VALUES (?)"
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverExec(ctx, nil, mockExecer, query, args)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)
}

func TestCtxDriverExec_WithNamedParameters(t *testing.T) {
	mockExecer := &mockExecer{}
	ctx := context.Background()
	query := "INSERT INTO test VALUES (?)"
	args := []driver.NamedValue{{Name: "param1", Value: "test"}}

	result, err := ctxDriverExec(ctx, nil, mockExecer, query, args)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Named Parameters")
}

// Tests for CtxDriverQuery
func TestCtxDriverQuery_WithQueryerContext(t *testing.T) {
	mockRows := &mockRows{columns: []string{"id", "name"}}
	mockQueryerCtx := &mockQueryerContext{
		queryContextFunc: func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			assert.Equal(t, "SELECT * FROM test", query)
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0].Value)
			return mockRows, nil
		},
	}
	ctx := context.Background()
	query := "SELECT * FROM test"
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := CtxDriverQuery(ctx, mockQueryerCtx, nil, query, args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, result)
}

func TestCtxDriverQuery_WithoutQueryerContext(t *testing.T) {
	mockRows := &mockRows{columns: []string{"id", "name"}}
	mockQueryer := &mockQueryer{
		queryFunc: func(query string, args []driver.Value) (driver.Rows, error) {
			assert.Equal(t, "SELECT * FROM test", query)
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0])
			return mockRows, nil
		},
	}
	ctx := context.Background()
	query := "SELECT * FROM test"
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := CtxDriverQuery(ctx, nil, mockQueryer, query, args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, result)
}

func TestCtxDriverQuery_WithCancelledContext(t *testing.T) {
	mockQueryer := &mockQueryer{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	query := "SELECT * FROM test"
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := CtxDriverQuery(ctx, nil, mockQueryer, query, args)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)
}

// Tests for ctxDriverStmtExec
func TestCtxDriverStmtExec_WithContext(t *testing.T) {
	mockResult := &mockResult{lastInsertId: 1, rowsAffected: 1}
	mockStmt := &mockStmtExecContext{
		execContextFunc: func(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0].Value)
			return mockResult, nil
		},
	}
	ctx := context.Background()
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverStmtExec(ctx, mockStmt, args)
	assert.NoError(t, err)
	assert.Equal(t, mockResult, result)
}

func TestCtxDriverStmtExec_WithoutContext(t *testing.T) {
	mockResult := &mockResult{lastInsertId: 1, rowsAffected: 1}
	mockStmt := &mockStmt{
		execFunc: func(args []driver.Value) (driver.Result, error) {
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0])
			return mockResult, nil
		},
	}
	ctx := context.Background()
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverStmtExec(ctx, mockStmt, args)
	assert.NoError(t, err)
	assert.Equal(t, mockResult, result)
}

func TestCtxDriverStmtExec_WithCancelledContext(t *testing.T) {
	mockStmt := &mockStmt{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverStmtExec(ctx, mockStmt, args)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)
}

// Tests for ctxDriverStmtQuery
func TestCtxDriverStmtQuery_WithContext(t *testing.T) {
	mockRows := &mockRows{columns: []string{"id", "name"}}
	mockStmt := &mockStmtQueryContext{
		queryContextFunc: func(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0].Value)
			return mockRows, nil
		},
	}
	ctx := context.Background()
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverStmtQuery(ctx, mockStmt, args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, result)
}

func TestCtxDriverStmtQuery_WithoutContext(t *testing.T) {
	mockRows := &mockRows{columns: []string{"id", "name"}}
	mockStmt := &mockStmt{
		queryFunc: func(args []driver.Value) (driver.Rows, error) {
			assert.Equal(t, 1, len(args))
			assert.Equal(t, "test", args[0])
			return mockRows, nil
		},
	}
	ctx := context.Background()
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverStmtQuery(ctx, mockStmt, args)
	assert.NoError(t, err)
	assert.Equal(t, mockRows, result)
}

func TestCtxDriverStmtQuery_WithCancelledContext(t *testing.T) {
	mockStmt := &mockStmt{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	args := []driver.NamedValue{{Ordinal: 1, Value: "test"}}

	result, err := ctxDriverStmtQuery(ctx, mockStmt, args)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.Canceled, err)
}

// Tests for namedValueToValue
func TestNamedValueToValue_Success(t *testing.T) {
	input := []driver.NamedValue{
		{Ordinal: 1, Value: "test"},
		{Ordinal: 2, Value: int64(123)},
	}

	result, err := namedValueToValue(input)
	assert.NoError(t, err)
	assert.Equal(t, []driver.Value{"test", int64(123)}, result)
}

func TestNamedValueToValue_WithNamedParameter(t *testing.T) {
	input := []driver.NamedValue{
		{Name: "param1", Value: "test"},
	}

	result, err := namedValueToValue(input)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Named Parameters")
}

func TestNamedValueToValue_Empty(t *testing.T) {
	input := []driver.NamedValue{}

	result, err := namedValueToValue(input)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestCtxDriverExecWithPrepareFallback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	query := "INSERT INTO t_user(id, name) VALUES (?, ?)"
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}, {Ordinal: 2, Value: "user1"}}

	targetConn := mock.NewMockTestDriverConn(ctrl)
	targetStmt := mock.NewMockTestDriverStmt(ctrl)

	targetConn.EXPECT().ExecContext(ctx, query, args).Return(nil, driver.ErrSkip)
	targetConn.EXPECT().PrepareContext(ctx, query).Return(targetStmt, nil)
	targetStmt.EXPECT().ExecContext(ctx, args).Return(driver.RowsAffected(1), nil)
	targetStmt.EXPECT().Close().Return(nil)

	result, err := CtxDriverExecWithPrepareFallback(ctx, targetConn, query, args)

	if !assert.NoError(t, err) || !assert.NotNil(t, result) {
		return
	}

	affected, err := result.RowsAffected()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), affected)
}

func TestNewRowsWithStmt(t *testing.T) {
	t.Run("close preserves rows and statement errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetRows := mock.NewMockTestDriverRows(ctrl)
		targetStmt := mock.NewMockTestDriverStmt(ctrl)
		rowsCloseErr := errors.New("rows close failed")
		stmtCloseErr := errors.New("statement close failed")

		targetRows.EXPECT().Close().Times(1).Return(rowsCloseErr)
		targetStmt.EXPECT().Close().Times(1).Return(stmtCloseErr)

		closeErr := NewRowsWithStmt(targetRows, targetStmt).Close()
		assert.ErrorIs(t, closeErr, rowsCloseErr)
		assert.ErrorIs(t, closeErr, stmtCloseErr)
	})

	t.Run("nil rows", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetStmt := mock.NewMockTestDriverStmt(ctrl)
		targetStmt.EXPECT().Close().Times(1).Return(nil)

		assert.NoError(t, NewRowsWithStmt(nil, targetStmt).Close())
	})

	t.Run("nil statement", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetRows := mock.NewMockTestDriverRows(ctrl)
		targetRows.EXPECT().Close().Times(1).Return(nil)

		assert.NoError(t, NewRowsWithStmt(targetRows, nil).Close())
	})

	t.Run("nil rows and statement", func(t *testing.T) {
		assert.NoError(t, NewRowsWithStmt(nil, nil).Close())
	})
}

func TestCtxDriverQueryWithPrepareFallback(t *testing.T) {
	ctx := context.Background()
	query := "SELECT id, name FROM t_user WHERE id IN (?, ?)"
	args := []driver.NamedValue{
		{
			Ordinal: 1,
			Value:   int64(1),
		},
		{
			Ordinal: 2,
			Value:   int64(2),
		},
	}

	t.Run("direct query success does not prepare", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetConn := mock.NewMockTestDriverConn(ctrl)
		targetRows := mock.NewMockTestDriverRows(ctrl)

		targetConn.EXPECT().QueryContext(ctx, query, args).Times(1).Return(targetRows, nil)
		targetRows.EXPECT().Close().Times(1).Return(nil)

		rows, err := CtxDriverQueryWithPrepareFallback(ctx, targetConn, query, args)

		if !assert.NoError(t, err) || !assert.NotNil(t, rows) {
			return
		}

		assert.Same(t, targetRows, rows)
		assert.NoError(t, rows.Close())
	})

	t.Run("ErrSkip falls back to prepared query", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetConn := mock.NewMockTestDriverConn(ctrl)
		targetStmt := mock.NewMockTestDriverStmt(ctrl)
		targetRows := mock.NewMockTestDriverRows(ctrl)

		targetConn.EXPECT().QueryContext(ctx, query, args).Times(1).Return(nil, driver.ErrSkip)
		targetConn.EXPECT().PrepareContext(ctx, query).Times(1).Return(targetStmt, nil)
		targetStmt.EXPECT().QueryContext(ctx, args).Times(1).Return(targetRows, nil)
		targetRows.EXPECT().Close().Times(1).Return(nil)
		targetStmt.EXPECT().Close().Times(1).Return(nil)

		rows, err := CtxDriverQueryWithPrepareFallback(ctx, targetConn, query, args)

		if !assert.NoError(t, err) || !assert.NotNil(t, rows) {
			return
		}

		assert.NotSame(t, targetRows, rows)
		assert.NoError(t, rows.Close())
	})

	t.Run("non ErrSkip error does not prepare", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetConn := mock.NewMockTestDriverConn(ctrl)
		expectedErr := errors.New("direct query failed")

		targetConn.EXPECT().QueryContext(ctx, query, args).Times(1).Return(nil, expectedErr)
		rows, err := CtxDriverQueryWithPrepareFallback(ctx, targetConn, query, args)

		assert.Nil(t, rows)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("prepare error is returned", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetConn := mock.NewMockTestDriverConn(ctrl)
		expectedErr := errors.New("prepare query failed")

		targetConn.EXPECT().QueryContext(ctx, query, args).Times(1).Return(nil, driver.ErrSkip)
		targetConn.EXPECT().PrepareContext(ctx, query).Times(1).Return(nil, expectedErr)
		rows, err := CtxDriverQueryWithPrepareFallback(ctx, targetConn, query, args)

		assert.Nil(t, rows)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("prepared query error closes statement", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetConn := mock.NewMockTestDriverConn(ctrl)
		targetStmt := mock.NewMockTestDriverStmt(ctrl)
		expectedErr := errors.New("prepared query failed")

		targetConn.EXPECT().QueryContext(ctx, query, args).Times(1).Return(nil, driver.ErrSkip)
		targetConn.EXPECT().PrepareContext(ctx, query).Times(1).Return(targetStmt, nil)
		targetStmt.EXPECT().QueryContext(ctx, args).Times(1).Return(nil, expectedErr)
		targetStmt.EXPECT().Close().Times(1).Return(nil)

		rows, err := CtxDriverQueryWithPrepareFallback(ctx, targetConn, query, args)
		assert.Nil(t, rows)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("close preserves rows and statement errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		targetConn := mock.NewMockTestDriverConn(ctrl)
		targetStmt := mock.NewMockTestDriverStmt(ctrl)
		targetRows := mock.NewMockTestDriverRows(ctrl)

		rowsCloseErr := errors.New("rows close failed")
		stmtCloseErr := errors.New("statement close failed")

		targetConn.EXPECT().QueryContext(ctx, query, args).Times(1).Return(nil, driver.ErrSkip)
		targetConn.EXPECT().PrepareContext(ctx, query).Times(1).Return(targetStmt, nil)
		targetStmt.EXPECT().QueryContext(ctx, args).Times(1).Return(targetRows, nil)
		targetRows.EXPECT().Close().Times(1).Return(rowsCloseErr)
		targetStmt.EXPECT().Close().Times(1).Return(stmtCloseErr)

		rows, err := CtxDriverQueryWithPrepareFallback(ctx, targetConn, query, args)

		if !assert.NoError(t, err) || !assert.NotNil(t, rows) {
			return
		}

		closeErr := rows.Close()

		assert.ErrorIs(t, closeErr, rowsCloseErr)
		assert.ErrorIs(t, closeErr, stmtCloseErr)
	})
}
