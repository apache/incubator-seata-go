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

package hook

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestNewLoggerSQLHook(t *testing.T) {
	hook := NewLoggerSQLHook()
	assert.NotNil(t, hook)
	assert.IsType(t, &loggerSQLHook{}, hook)
}

func TestLoggerSQLHook_Type(t *testing.T) {
	hook := &loggerSQLHook{}
	assert.Equal(t, types.SQLType(types.SQLTypeUnknown), hook.Type())
}

func TestLoggerSQLHook_Before(t *testing.T) {
	tests := []struct {
		name    string
		execCtx *types.ExecContext
		wantErr bool
	}{
		{
			name: "basic query with empty tx context",
			execCtx: &types.ExecContext{
				Query:  "SELECT * FROM users",
				TxCtx:  &types.TransactionContext{},
				Values: nil,
			},
			wantErr: false,
		},
		{
			name: "query with tx context",
			execCtx: &types.ExecContext{
				Query: "INSERT INTO users VALUES (?, ?)",
				TxCtx: &types.TransactionContext{
					XID:          "test-xid-123",
					LocalTransID: "local-tx-456",
				},
				Values: []driver.Value{1, "test"},
			},
			wantErr: false,
		},
		{
			name: "query with named values",
			execCtx: &types.ExecContext{
				Query: "UPDATE users SET name = :name WHERE id = :id",
				TxCtx: &types.TransactionContext{
					XID:          "test-xid-789",
					LocalTransID: "local-tx-012",
				},
				NamedValues: []driver.NamedValue{
					{Name: "name", Value: "John"},
					{Name: "id", Value: 1},
				},
			},
			wantErr: false,
		},
		{
			name: "query with both named values and values",
			execCtx: &types.ExecContext{
				Query: "DELETE FROM users WHERE id = ?",
				TxCtx: &types.TransactionContext{
					XID:          "test-xid-345",
					LocalTransID: "local-tx-678",
				},
				NamedValues: []driver.NamedValue{
					{Name: "id", Value: 1},
				},
				Values: []driver.Value{1},
			},
			wantErr: false,
		},
		{
			name: "empty query",
			execCtx: &types.ExecContext{
				Query: "",
				TxCtx: &types.TransactionContext{
					XID:          "test-xid-empty",
					LocalTransID: "local-tx-empty",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &loggerSQLHook{}
			ctx := context.Background()
			err := hook.Before(ctx, tt.execCtx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoggerSQLHook_After(t *testing.T) {
	tests := []struct {
		name    string
		execCtx *types.ExecContext
		wantErr bool
	}{
		{
			name: "after execution - success",
			execCtx: &types.ExecContext{
				Query: "SELECT * FROM users",
				TxCtx: &types.TransactionContext{
					XID:          "test-xid-after",
					LocalTransID: "local-tx-after",
				},
			},
			wantErr: false,
		},
		{
			name: "after execution - empty tx context",
			execCtx: &types.ExecContext{
				Query: "SELECT * FROM products",
				TxCtx: &types.TransactionContext{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &loggerSQLHook{}
			ctx := context.Background()
			err := hook.After(ctx, tt.execCtx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoggerSQLHook_Integration(t *testing.T) {
	hook := NewLoggerSQLHook()
	ctx := context.Background()

	execCtx := &types.ExecContext{
		Query: "UPDATE accounts SET balance = balance - ? WHERE id = ?",
		TxCtx: &types.TransactionContext{
			XID:          "integration-test-xid",
			LocalTransID: "integration-test-local",
		},
		Values: []driver.Value{100.50, 42},
	}

	// Test Before hook
	err := hook.Before(ctx, execCtx)
	assert.NoError(t, err)

	// Test After hook
	err = hook.After(ctx, execCtx)
	assert.NoError(t, err)
}
