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

package exec

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

// TestRegisterATExecutor tests the RegisterATExecutor function
func TestRegisterATExecutor(t *testing.T) {
	// Clean up before test
	originalExecutors := atExecutors
	defer func() { atExecutors = originalExecutors }()
	atExecutors = make(map[types.DBType]func() SQLExecutor)

	tests := []struct {
		name    string
		dbType  types.DBType
		builder func() SQLExecutor
	}{
		{
			name:   "register MySQL executor",
			dbType: types.DBTypeMySQL,
			builder: func() SQLExecutor {
				return &mockSQLExecutor{}
			},
		},
		{
			name:   "register PostgreSQL executor",
			dbType: types.DBTypePostgreSQL,
			builder: func() SQLExecutor {
				return &mockSQLExecutor{}
			},
		},
		{
			name:   "register unknown type executor",
			dbType: types.DBTypeUnknown,
			builder: func() SQLExecutor {
				return &mockSQLExecutor{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterATExecutor(tt.dbType, tt.builder)
			assert.NotNil(t, atExecutors[tt.dbType], "executor should be registered")
			executor := atExecutors[tt.dbType]()
			assert.NotNil(t, executor, "executor builder should return an executor")
		})
	}
}

// TestBuildExecutor tests the BuildExecutor function
func TestBuildExecutor(t *testing.T) {
	// Setup: register a mock executor
	originalExecutors := atExecutors
	originalCommonHook := commonHook
	originalHookSolts := hookSolts
	defer func() {
		atExecutors = originalExecutors
		commonHook = originalCommonHook
		hookSolts = originalHookSolts
	}()

	atExecutors = make(map[types.DBType]func() SQLExecutor)
	commonHook = make([]SQLHook, 0, 4)
	hookSolts = map[types.SQLType][]SQLHook{}

	mockExecutor := &mockSQLExecutor{}
	RegisterATExecutor(types.DBTypeMySQL, func() SQLExecutor {
		return mockExecutor
	})

	// Register test hooks
	RegisterCommonHook(&mockSQLHook{sqlType: types.SQLTypeSelect})
	RegisterHook(&mockSQLHook{sqlType: types.SQLTypeInsert})

	tests := []struct {
		name            string
		dbType          types.DBType
		transactionMode types.TransactionMode
		query           string
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "build executor for INSERT statement",
			dbType:          types.DBTypeMySQL,
			transactionMode: types.ATMode,
			query:           "INSERT INTO users (name, age) VALUES ('Alice', 30)",
			wantErr:         false,
		},
		{
			name:            "build executor for UPDATE statement",
			dbType:          types.DBTypeMySQL,
			transactionMode: types.ATMode,
			query:           "UPDATE users SET age = 31 WHERE name = 'Alice'",
			wantErr:         false,
		},
		{
			name:            "build executor for DELETE statement",
			dbType:          types.DBTypeMySQL,
			transactionMode: types.ATMode,
			query:           "DELETE FROM users WHERE name = 'Alice'",
			wantErr:         false,
		},
		{
			name:            "build executor for SELECT statement",
			dbType:          types.DBTypeMySQL,
			transactionMode: types.ATMode,
			query:           "SELECT * FROM users WHERE name = 'Alice'",
			wantErr:         false,
		},
		{
			name:            "invalid SQL query",
			dbType:          types.DBTypeMySQL,
			transactionMode: types.ATMode,
			query:           "INVALID SQL QUERY",
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := BuildExecutor(tt.dbType, tt.transactionMode, tt.query)
			if tt.wantErr {
				assert.Error(t, err, "should return error for invalid query")
				assert.Nil(t, executor, "executor should be nil on error")
			} else {
				assert.NoError(t, err, "should not return error for valid query")
				assert.NotNil(t, executor, "executor should not be nil")
				// Verify that the mock executor received the interceptors
				assert.True(t, mockExecutor.interceptorsCalled, "Interceptors should be called")
				assert.NotEmpty(t, mockExecutor.hooks, "hooks should be set")
			}
		})
	}
}

// TestBaseExecutor_Interceptors tests the Interceptors method
func TestBaseExecutor_Interceptors(t *testing.T) {
	tests := []struct {
		name         string
		interceptors []SQLHook
	}{
		{
			name:         "set empty interceptors",
			interceptors: []SQLHook{},
		},
		{
			name: "set single interceptor",
			interceptors: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
		},
		{
			name: "set multiple interceptors",
			interceptors: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &BaseExecutor{}
			executor.Interceptors(tt.interceptors)
			assert.Equal(t, len(tt.interceptors), len(executor.hooks), "hooks count should match")
		})
	}
}

// TestBaseExecutor_ExecWithNamedValue tests the ExecWithNamedValue method
func TestBaseExecutor_ExecWithNamedValue(t *testing.T) {
	tests := []struct {
		name            string
		setupHooks      []SQLHook
		innerExecutor   SQLExecutor
		callback        CallbackWithNamedValue
		execCtx         *types.ExecContext
		wantErr         bool
		wantBeforeCount int
		wantAfterCount  int
	}{
		{
			name: "execute without hooks and without inner executor",
			callback: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return newMockExecResult(1, 1), nil
			},
			execCtx: &types.ExecContext{
				Query:       "INSERT INTO users VALUES (?, ?)",
				NamedValues: []driver.NamedValue{{Value: "Alice"}, {Value: 30}},
			},
			wantErr:         false,
			wantBeforeCount: 0,
			wantAfterCount:  0,
		},
		{
			name: "execute with hooks",
			setupHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			callback: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return newMockExecResult(1, 1), nil
			},
			execCtx: &types.ExecContext{
				Query:       "INSERT INTO users VALUES (?, ?)",
				NamedValues: []driver.NamedValue{{Value: "Alice"}, {Value: 30}},
			},
			wantErr:         false,
			wantBeforeCount: 2,
			wantAfterCount:  2,
		},
		{
			name: "callback returns error",
			setupHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			callback: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return nil, fmt.Errorf("execution error")
			},
			execCtx: &types.ExecContext{
				Query:       "INSERT INTO users VALUES (?, ?)",
				NamedValues: []driver.NamedValue{{Value: "Alice"}, {Value: 30}},
			},
			wantErr:         true,
			wantBeforeCount: 1,
			wantAfterCount:  1, // After hooks are called even on error (defer)
		},
		{
			name: "execute with inner executor",
			setupHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			innerExecutor: &mockSQLExecutor{
				execWithNamedValueFunc: func(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
					return newMockExecResult(2, 2), nil
				},
			},
			callback: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				// This should not be called when inner executor is set
				panic("callback should not be called")
			},
			execCtx: &types.ExecContext{
				Query:       "INSERT INTO users VALUES (?, ?)",
				NamedValues: []driver.NamedValue{{Value: "Bob"}, {Value: 25}},
			},
			wantErr:         false,
			wantBeforeCount: 1,
			wantAfterCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &BaseExecutor{
				hooks: tt.setupHooks,
				ex:    tt.innerExecutor,
			}

			result, err := executor.ExecWithNamedValue(context.Background(), tt.execCtx, tt.callback)

			if tt.wantErr {
				assert.Error(t, err, "should return error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.NotNil(t, result, "result should not be nil")
			}

			// Verify hooks were called
			beforeCount := 0
			afterCount := 0
			for _, hook := range tt.setupHooks {
				mockHook := hook.(*mockSQLHook)
				beforeCount += mockHook.beforeCallCount
				afterCount += mockHook.afterCallCount
			}
			assert.Equal(t, tt.wantBeforeCount, beforeCount, "before hook call count should match")
			assert.Equal(t, tt.wantAfterCount, afterCount, "after hook call count should match")
		})
	}
}

// TestBaseExecutor_ExecWithValue tests the ExecWithValue method
func TestBaseExecutor_ExecWithValue(t *testing.T) {
	tests := []struct {
		name            string
		setupHooks      []SQLHook
		callback        CallbackWithNamedValue
		execCtx         *types.ExecContext
		wantErr         bool
		wantBeforeCount int
		wantAfterCount  int
	}{
		{
			name: "execute with values - converts to NamedValues",
			callback: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				// Verify that values were converted to NamedValues
				assert.NotEmpty(t, args, "args should not be empty")
				return newMockExecResult(1, 1), nil
			},
			execCtx: &types.ExecContext{
				Query:  "UPDATE users SET age = ? WHERE name = ?",
				Values: []driver.Value{31, "Alice"},
			},
			wantErr:         false,
			wantBeforeCount: 0,
			wantAfterCount:  0,
		},
		{
			name: "execute with values and hooks",
			setupHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
			},
			callback: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return newMockExecResult(1, 1), nil
			},
			execCtx: &types.ExecContext{
				Query:  "UPDATE users SET age = ? WHERE name = ?",
				Values: []driver.Value{31, "Alice"},
			},
			wantErr:         false,
			wantBeforeCount: 1,
			wantAfterCount:  1,
		},
		{
			name: "callback returns error with values",
			setupHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
			},
			callback: func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return nil, fmt.Errorf("execution error")
			},
			execCtx: &types.ExecContext{
				Query:  "UPDATE users SET age = ? WHERE name = ?",
				Values: []driver.Value{31, "Alice"},
			},
			wantErr:         true,
			wantBeforeCount: 1,
			wantAfterCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &BaseExecutor{
				hooks: tt.setupHooks,
			}

			result, err := executor.ExecWithValue(context.Background(), tt.execCtx, tt.callback)

			if tt.wantErr {
				assert.Error(t, err, "should return error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.NotNil(t, result, "result should not be nil")
			}

			// Verify hooks were called
			beforeCount := 0
			afterCount := 0
			for _, hook := range tt.setupHooks {
				mockHook := hook.(*mockSQLHook)
				beforeCount += mockHook.beforeCallCount
				afterCount += mockHook.afterCallCount
			}
			assert.Equal(t, tt.wantBeforeCount, beforeCount, "before hook call count should match")
			assert.Equal(t, tt.wantAfterCount, afterCount, "after hook call count should match")
		})
	}
}

// Mock implementations for testing

// mockSQLExecutor is a mock implementation of SQLExecutor
type mockSQLExecutor struct {
	hooks                  []SQLHook
	interceptorsCalled     bool
	execWithNamedValueFunc func(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error)
	execWithValueFunc      func(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error)
}

func (m *mockSQLExecutor) Interceptors(interceptors []SQLHook) {
	m.hooks = interceptors
	m.interceptorsCalled = true
}

func (m *mockSQLExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
	if m.execWithNamedValueFunc != nil {
		return m.execWithNamedValueFunc(ctx, execCtx, f)
	}
	return f(ctx, execCtx.Query, execCtx.NamedValues)
}

func (m *mockSQLExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
	if m.execWithValueFunc != nil {
		return m.execWithValueFunc(ctx, execCtx, f)
	}
	return f(ctx, execCtx.Query, execCtx.NamedValues)
}

// mockSQLHook is a mock implementation of SQLHook
type mockSQLHook struct {
	sqlType         types.SQLType
	beforeCallCount int
	afterCallCount  int
	beforeError     error
	afterError      error
}

func (m *mockSQLHook) Type() types.SQLType {
	return m.sqlType
}

func (m *mockSQLHook) Before(ctx context.Context, execCtx *types.ExecContext) error {
	m.beforeCallCount++
	return m.beforeError
}

func (m *mockSQLHook) After(ctx context.Context, execCtx *types.ExecContext) error {
	m.afterCallCount++
	return m.afterError
}

// mockExecResult is a mock implementation of types.ExecResult
type mockExecResult struct {
	lastInsertID int64
	rowsAffected int64
}

func newMockExecResult(lastInsertID, rowsAffected int64) types.ExecResult {
	return &mockExecResult{
		lastInsertID: lastInsertID,
		rowsAffected: rowsAffected,
	}
}

func (m *mockExecResult) GetRows() driver.Rows {
	panic("not implemented for write result")
}

func (m *mockExecResult) GetResult() driver.Result {
	return sqlmock.NewResult(m.lastInsertID, m.rowsAffected)
}