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

package at

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/tm"
)

func TestATExecutor_Interceptors(t *testing.T) {
	tests := []struct {
		name         string
		interceptors []exec.SQLHook
	}{
		{
			name:         "set empty interceptors",
			interceptors: []exec.SQLHook{},
		},
		{
			name: "set single interceptor",
			interceptors: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
		},
		{
			name: "set multiple interceptors",
			interceptors: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &ATExecutor{}
			executor.Interceptors(tt.interceptors)
			assert.Equal(t, len(tt.interceptors), len(executor.hooks), "hooks count should match")
		})
	}
}

func TestATExecutor_ExecWithNamedValue_NonGlobalTx(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		sqlType   types.SQLType
		wantErr   bool
		callCount int
	}{
		{
			name:      "non-global transaction - INSERT",
			query:     "INSERT INTO users (name, age) VALUES (?, ?)",
			sqlType:   types.SQLTypeInsert,
			wantErr:   false,
			callCount: 1,
		},
		{
			name:      "non-global transaction - UPDATE",
			query:     "UPDATE users SET age = ? WHERE name = ?",
			sqlType:   types.SQLTypeUpdate,
			wantErr:   false,
			callCount: 1,
		},
		{
			name:      "non-global transaction - DELETE",
			query:     "DELETE FROM users WHERE name = ?",
			sqlType:   types.SQLTypeDelete,
			wantErr:   false,
			callCount: 1,
		},
		{
			name:      "non-global transaction - SELECT",
			query:     "SELECT * FROM users WHERE name = ?",
			sqlType:   types.SQLTypeSelect,
			wantErr:   false,
			callCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(tm.IsGlobalTx, func(ctx context.Context) bool {
				return false
			})
			defer patches.Reset()

			patchesParser := gomonkey.ApplyFunc(parser.DoParser, func(query string) (*types.ParseContext, error) {
				return &types.ParseContext{
					SQLType: tt.sqlType,
				}, nil
			})
			defer patchesParser.Reset()

			executor := &ATExecutor{}
			execCtx := &types.ExecContext{
				Query:       tt.query,
				NamedValues: []driver.NamedValue{{Value: "test"}},
			}

			callCount := 0
			callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				callCount++
				return &mockExecResult{rowsAffected: 1}, nil
			}

			result, err := executor.ExecWithNamedValue(context.Background(), execCtx, callback)

			if tt.wantErr {
				assert.Error(t, err, "should return error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.NotNil(t, result, "result should not be nil")
				assert.Equal(t, tt.callCount, callCount, "callback should be called once")
			}
		})
	}
}

func TestATExecutor_ExecWithNamedValue_GlobalTx(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		sqlType types.SQLType
		wantErr bool
	}{
		{
			name:    "global transaction - INSERT",
			query:   "INSERT INTO users (name, age) VALUES (?, ?)",
			sqlType: types.SQLTypeInsert,
			wantErr: false,
		},
		{
			name:    "global transaction - UPDATE",
			query:   "UPDATE users SET age = ? WHERE name = ?",
			sqlType: types.SQLTypeUpdate,
			wantErr: false,
		},
		{
			name:    "global transaction - DELETE",
			query:   "DELETE FROM users WHERE name = ?",
			sqlType: types.SQLTypeDelete,
			wantErr: false,
		},
		{
			name:    "global transaction - SELECT FOR UPDATE",
			query:   "SELECT * FROM users WHERE name = ? FOR UPDATE",
			sqlType: types.SQLTypeSelectForUpdate,
			wantErr: false,
		},
		{
			name:    "global transaction - INSERT ON DUPLICATE UPDATE",
			query:   "INSERT INTO users (id, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE name = ?",
			sqlType: types.SQLTypeInsertOnDuplicateUpdate,
			wantErr: false,
		},
		{
			name:    "global transaction - MULTI",
			query:   "UPDATE users SET age = ? WHERE id IN (?, ?)",
			sqlType: types.SQLTypeMulti,
			wantErr: false,
		},
		{
			name:    "global transaction - plain SQL (SELECT without FOR UPDATE)",
			query:   "SELECT * FROM users WHERE name = ?",
			sqlType: types.SQLTypeSelect,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(tm.IsGlobalTx, func(ctx context.Context) bool {
				return true
			})
			defer patches.Reset()

			patchesParser := gomonkey.ApplyFunc(parser.DoParser, func(query string) (*types.ParseContext, error) {
				return &types.ParseContext{
					SQLType: tt.sqlType,
				}, nil
			})
			defer patchesParser.Reset()

			executor := &ATExecutor{
				hooks: []exec.SQLHook{
					&mockSQLHook{sqlType: tt.sqlType},
				},
			}
			execCtx := &types.ExecContext{
				Query:       tt.query,
				NamedValues: []driver.NamedValue{{Value: "test"}},
			}

			callCount := 0
			callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				callCount++
				return &mockExecResult{rowsAffected: 1}, nil
			}

			result, err := executor.ExecWithNamedValue(context.Background(), execCtx, callback)

			if tt.wantErr {
				assert.Error(t, err, "should return error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.NotNil(t, result, "result should not be nil")
			}
		})
	}
}

func TestATExecutor_ExecWithNamedValue_ParserError(t *testing.T) {
	patches := gomonkey.ApplyFunc(parser.DoParser, func(query string) (*types.ParseContext, error) {
		return nil, fmt.Errorf("parser error")
	})
	defer patches.Reset()

	executor := &ATExecutor{}
	execCtx := &types.ExecContext{
		Query:       "INVALID SQL",
		NamedValues: []driver.NamedValue{},
	}

	callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		return &mockExecResult{rowsAffected: 1}, nil
	}

	result, err := executor.ExecWithNamedValue(context.Background(), execCtx, callback)

	assert.Error(t, err, "should return parser error")
	assert.Nil(t, result, "result should be nil on error")
	assert.Contains(t, err.Error(), "parser error", "error message should contain parser error")
}

func TestATExecutor_ExecWithValue(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		values    []driver.Value
		sqlType   types.SQLType
		wantErr   bool
		callCount int
	}{
		{
			name:      "execute with values - converts to NamedValues",
			query:     "INSERT INTO users (name, age) VALUES (?, ?)",
			values:    []driver.Value{"Alice", 30},
			sqlType:   types.SQLTypeInsert,
			wantErr:   false,
			callCount: 1,
		},
		{
			name:      "execute with empty values",
			query:     "INSERT INTO users (name, age) VALUES ('Bob', 25)",
			values:    []driver.Value{},
			sqlType:   types.SQLTypeInsert,
			wantErr:   false,
			callCount: 1,
		},
		{
			name:      "execute with multiple values",
			query:     "UPDATE users SET name = ?, age = ?, city = ? WHERE id = ?",
			values:    []driver.Value{"Charlie", 35, "NYC", 1},
			sqlType:   types.SQLTypeUpdate,
			wantErr:   false,
			callCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(tm.IsGlobalTx, func(ctx context.Context) bool {
				return false
			})
			defer patches.Reset()

			patchesParser := gomonkey.ApplyFunc(parser.DoParser, func(query string) (*types.ParseContext, error) {
				return &types.ParseContext{
					SQLType: tt.sqlType,
				}, nil
			})
			defer patchesParser.Reset()

			executor := &ATExecutor{}
			execCtx := &types.ExecContext{
				Query:  tt.query,
				Values: tt.values,
			}

			callCount := 0
			callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				callCount++
				return &mockExecResult{rowsAffected: 1}, nil
			}

			result, err := executor.ExecWithValue(context.Background(), execCtx, callback)

			if tt.wantErr {
				assert.Error(t, err, "should return error")
			} else {
				assert.NoError(t, err, "should not return error")
				assert.NotNil(t, result, "result should not be nil")
				assert.Equal(t, tt.callCount, callCount, "callback should be called")
				// Verify that NamedValues were populated in execCtx
				assert.NotNil(t, execCtx.NamedValues, "NamedValues should be populated")
				assert.Equal(t, len(tt.values), len(execCtx.NamedValues), "NamedValues count should match values count")
			}
		})
	}
}

func TestATExecutor_ExecWithValue_ParserError(t *testing.T) {
	patches := gomonkey.ApplyFunc(parser.DoParser, func(query string) (*types.ParseContext, error) {
		return nil, fmt.Errorf("parser error")
	})
	defer patches.Reset()

	executor := &ATExecutor{}
	execCtx := &types.ExecContext{
		Query:  "INVALID SQL",
		Values: []driver.Value{"test"},
	}

	callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		return &mockExecResult{rowsAffected: 1}, nil
	}

	result, err := executor.ExecWithValue(context.Background(), execCtx, callback)

	assert.Error(t, err, "should return parser error")
	assert.Nil(t, result, "result should be nil on error")
}

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

type mockExecutor struct {
	execContextFunc func(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error)
}

func (m *mockExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if m.execContextFunc != nil {
		return m.execContextFunc(ctx, f)
	}
	return &mockExecResult{rowsAffected: 1}, nil
}

type mockExecResult struct {
	lastInsertID int64
	rowsAffected int64
	rows         driver.Rows
}

func (m *mockExecResult) GetRows() driver.Rows {
	return m.rows
}

func (m *mockExecResult) GetResult() driver.Result {
	return &mockResult{
		lastInsertID: m.lastInsertID,
		rowsAffected: m.rowsAffected,
	}
}

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m *mockResult) LastInsertId() (int64, error) {
	return m.lastInsertID, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}
