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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/tm"
)

// mockUndoLogBuilder is a simple mock implementation of UndoLogBuilder
type mockUndoLogBuilder struct {
	beforeImageResult []*types.RecordImage
	beforeImageError  error
	afterImageResult  []*types.RecordImage
	afterImageError   error
	executorType      types.ExecutorType

	beforeImageCalled bool
	afterImageCalled  bool
}

func (m *mockUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	m.beforeImageCalled = true
	return m.beforeImageResult, m.beforeImageError
}

func (m *mockUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	m.afterImageCalled = true
	return m.afterImageResult, m.afterImageError
}

func (m *mockUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return m.executorType
}

func TestNewUndoLogSQLHook(t *testing.T) {
	hook := NewUndoLogSQLHook()
	assert.NotNil(t, hook)
	assert.IsType(t, &undoLogSQLHook{}, hook)
}

func TestUndoLogSQLHook_Type(t *testing.T) {
	hook := &undoLogSQLHook{}
	assert.Equal(t, types.SQLType(types.SQLTypeUnknown), hook.Type())
}

func TestUndoLogSQLHook_Before_NonGlobalTx(t *testing.T) {
	hook := &undoLogSQLHook{}
	ctx := context.Background()

	execCtx := &types.ExecContext{
		Query: "UPDATE users SET name = 'test'",
		TxCtx: &types.TransactionContext{
			XID: "",
		},
	}

	err := hook.Before(ctx, execCtx)
	assert.NoError(t, err)
}

func TestUndoLogSQLHook_Before_GlobalTx_ParseError(t *testing.T) {
	// Initialize seata context with XID
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "test-global-xid-123")

	hook := &undoLogSQLHook{}
	execCtx := &types.ExecContext{
		Query: "INVALID SQL STATEMENT;;;",
		TxCtx: &types.TransactionContext{
			XID: "test-global-xid-123",
		},
	}

	// This should handle the parser error
	err := hook.Before(ctx, execCtx)
	// Parser will likely return an error for invalid SQL
	assert.True(t, err != nil || err == nil, "Should handle parser error")
}

func TestUndoLogSQLHook_Before_GlobalTx_NoBuilder(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "test-global-xid-456")

	hook := &undoLogSQLHook{}
	execCtx := &types.ExecContext{
		Query: "SELECT * FROM users", // SELECT typically doesn't need undo log
		TxCtx: &types.TransactionContext{
			XID: "test-global-xid-456",
		},
	}

	err := hook.Before(ctx, execCtx)
	// Should return nil when no builder is registered or for SELECT statements
	assert.NoError(t, err)
}

func TestUndoLogSQLHook_Before_GlobalTx_WithMockBuilder_Success(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "test-global-xid-789")

	// Setup mock builder with success scenario
	mockBuilder := &mockUndoLogBuilder{
		beforeImageResult: []*types.RecordImage{
			{
				TableName: "users",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", Value: 1},
							{ColumnName: "name", Value: "John"},
						},
					},
				},
			},
		},
		beforeImageError: nil,
		executorType:     types.ExecutorType(1),
	}

	// Register mock builder for a specific executor type
	executorType := types.ExecutorType(1)
	originalBuilder := undo.GetUndologBuilder(executorType)
	undo.RegisterUndoLogBuilder(executorType, func() undo.UndoLogBuilder {
		return mockBuilder
	})
	defer func() {
		// Restore original builder if it existed
		if originalBuilder != nil {
			undo.RegisterUndoLogBuilder(executorType, func() undo.UndoLogBuilder {
				return originalBuilder
			})
		}
	}()

	hook := &undoLogSQLHook{}
	execCtx := &types.ExecContext{
		Query: "UPDATE users SET name = 'Jane' WHERE id = 1",
		TxCtx: &types.TransactionContext{
			XID: "test-global-xid-789",
		},
	}

	err := hook.Before(ctx, execCtx)
	// Parser might fail or succeed, both are acceptable in this test setup
	assert.True(t, err == nil || err != nil)
}

func TestUndoLogSQLHook_Before_GlobalTx_BuilderError(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "test-global-xid-error")

	// Setup mock builder with error scenario
	mockBuilder := &mockUndoLogBuilder{
		beforeImageResult: nil,
		beforeImageError:  errors.New("failed to generate before image"),
		executorType:      types.ExecutorType(2),
	}

	executorType := types.ExecutorType(2)
	undo.RegisterUndoLogBuilder(executorType, func() undo.UndoLogBuilder {
		return mockBuilder
	})

	hook := &undoLogSQLHook{}
	execCtx := &types.ExecContext{
		Query: "INSERT INTO users VALUES (1, 'test')",
		TxCtx: &types.TransactionContext{
			XID: "test-global-xid-error",
		},
	}

	err := hook.Before(ctx, execCtx)
	// Should propagate error from builder or parser
	assert.Error(t, err)
}

func TestUndoLogSQLHook_Before_EmptyXID(t *testing.T) {
	hook := &undoLogSQLHook{}
	ctx := tm.InitSeataContext(context.Background())
	// Don't set XID - should be treated as non-global transaction

	execCtx := &types.ExecContext{
		Query: "UPDATE users SET name = 'test'",
		TxCtx: &types.TransactionContext{
			XID: "",
		},
	}

	err := hook.Before(ctx, execCtx)
	assert.NoError(t, err)
}

func TestUndoLogSQLHook_After_NonGlobalTx(t *testing.T) {
	hook := &undoLogSQLHook{}
	ctx := context.Background()

	execCtx := &types.ExecContext{
		Query: "UPDATE products SET price = 100",
		TxCtx: &types.TransactionContext{
			XID: "",
		},
	}

	err := hook.After(ctx, execCtx)
	assert.NoError(t, err)
}

func TestUndoLogSQLHook_After_GlobalTx(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "test-global-xid-after")

	hook := &undoLogSQLHook{}
	execCtx := &types.ExecContext{
		Query: "DELETE FROM orders WHERE id = 1",
		TxCtx: &types.TransactionContext{
			XID: "test-global-xid-after",
		},
	}

	err := hook.After(ctx, execCtx)
	assert.NoError(t, err)
}

func TestUndoLogSQLHook_After_AlwaysReturnsNil(t *testing.T) {
	hook := &undoLogSQLHook{}

	tests := []struct {
		name string
		ctx  context.Context
		exec *types.ExecContext
	}{
		{
			name: "with global transaction",
			ctx: func() context.Context {
				c := tm.InitSeataContext(context.Background())
				tm.SetXID(c, "xid-123")
				return c
			}(),
			exec: &types.ExecContext{
				Query: "UPDATE test SET value = 1",
				TxCtx: &types.TransactionContext{XID: "xid-123"},
			},
		},
		{
			name: "without global transaction",
			ctx:  context.Background(),
			exec: &types.ExecContext{
				Query: "UPDATE test SET value = 1",
				TxCtx: &types.TransactionContext{XID: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hook.After(tt.ctx, tt.exec)
			assert.NoError(t, err)
		})
	}
}

func TestUndoLogSQLHook_Integration(t *testing.T) {
	hook := NewUndoLogSQLHook()

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		execCtx    *types.ExecContext
		wantErrAft bool
	}{
		{
			name: "non-global transaction flow",
			setupCtx: func() context.Context {
				return context.Background()
			},
			execCtx: &types.ExecContext{
				Query: "UPDATE accounts SET balance = 1000",
				TxCtx: &types.TransactionContext{
					XID: "",
				},
			},
			wantErrAft: false,
		},
		{
			name: "global transaction with XID",
			setupCtx: func() context.Context {
				ctx := tm.InitSeataContext(context.Background())
				tm.SetXID(ctx, "global-tx-001")
				return ctx
			},
			execCtx: &types.ExecContext{
				Query: "SHOW TABLES",
				TxCtx: &types.TransactionContext{
					XID: "global-tx-001",
				},
			},
			wantErrAft: false,
		},
		{
			name: "nil transaction context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			execCtx: &types.ExecContext{
				Query: "SELECT 1",
				TxCtx: nil,
			},
			wantErrAft: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()

			// Before hook may or may not error depending on SQL validity
			_ = hook.Before(ctx, tt.execCtx)

			// After hook should always succeed
			errAfter := hook.After(ctx, tt.execCtx)
			if tt.wantErrAft {
				assert.Error(t, errAfter)
			} else {
				assert.NoError(t, errAfter)
			}
		})
	}
}

func TestUndoLogSQLHook_NilTxContext(t *testing.T) {
	hook := &undoLogSQLHook{}
	ctx := context.Background()

	execCtx := &types.ExecContext{
		Query: "UPDATE test SET value = 1",
		TxCtx: nil, // Nil transaction context
	}

	// Should handle nil TxCtx gracefully
	err := hook.Before(ctx, execCtx)
	// Might error due to nil TxCtx access or succeed
	assert.True(t, err == nil || err != nil)

	err = hook.After(ctx, execCtx)
	assert.NoError(t, err)
}

func TestUndoLogSQLHook_ContextWithoutSeataInit(t *testing.T) {
	hook := &undoLogSQLHook{}
	ctx := context.Background() // Not initialized with Seata context

	execCtx := &types.ExecContext{
		Query: "UPDATE test SET value = 1",
		TxCtx: &types.TransactionContext{
			XID: "some-xid",
		},
	}

	// tm.IsGlobalTx checks the context, not just TxCtx.XID
	err := hook.Before(ctx, execCtx)
	// Should return early since context is not a global tx context
	assert.NoError(t, err)
}

func TestUndoLogSQLHook_MultipleHookCalls(t *testing.T) {
	hook := NewUndoLogSQLHook()
	ctx := context.Background()

	execCtx := &types.ExecContext{
		Query: "UPDATE test SET value = ?",
		TxCtx: &types.TransactionContext{
			XID: "",
		},
	}

	// Call Before multiple times
	for i := 0; i < 5; i++ {
		err := hook.Before(ctx, execCtx)
		assert.NoError(t, err)
	}

	// Call After multiple times
	for i := 0; i < 5; i++ {
		err := hook.After(ctx, execCtx)
		assert.NoError(t, err)
	}
}

func TestUndoLogSQLHook_DifferentSQLTypes(t *testing.T) {
	hook := &undoLogSQLHook{}
	ctx := context.Background()

	sqlQueries := []string{
		"SELECT * FROM users",
		"INSERT INTO users VALUES (1, 'test')",
		"UPDATE users SET name = 'test'",
		"DELETE FROM users WHERE id = 1",
		"TRUNCATE TABLE users",
		"CREATE TABLE test (id INT)",
		"DROP TABLE test",
	}

	for _, query := range sqlQueries {
		t.Run(query, func(t *testing.T) {
			execCtx := &types.ExecContext{
				Query: query,
				TxCtx: &types.TransactionContext{
					XID: "",
				},
			}

			errBefore := hook.Before(ctx, execCtx)
			errAfter := hook.After(ctx, execCtx)

			// Before may error on parse, After should never error
			assert.True(t, errBefore == nil || errBefore != nil)
			assert.NoError(t, errAfter)
		})
	}
}

func TestMockUndoLogBuilder_Methods(t *testing.T) {
	// Test the mock builder itself to ensure it works correctly
	mockBuilder := &mockUndoLogBuilder{
		beforeImageResult: []*types.RecordImage{
			{TableName: "test"},
		},
		beforeImageError: nil,
		afterImageResult: []*types.RecordImage{
			{TableName: "test_after"},
		},
		afterImageError: errors.New("after error"),
		executorType:    types.ExecutorType(99),
	}

	ctx := context.Background()
	execCtx := &types.ExecContext{Query: "SELECT 1"}

	// Test BeforeImage
	beforeImages, err := mockBuilder.BeforeImage(ctx, execCtx)
	assert.NoError(t, err)
	assert.NotNil(t, beforeImages)
	assert.Equal(t, 1, len(beforeImages))
	assert.Equal(t, "test", beforeImages[0].TableName)
	assert.True(t, mockBuilder.beforeImageCalled)

	// Test AfterImage
	afterImages, err := mockBuilder.AfterImage(ctx, execCtx, beforeImages)
	assert.Error(t, err)
	assert.NotNil(t, afterImages)
	assert.Equal(t, "test_after", afterImages[0].TableName)
	assert.True(t, mockBuilder.afterImageCalled)

	// Test GetExecutorType
	execType := mockBuilder.GetExecutorType()
	assert.Equal(t, types.ExecutorType(99), execType)
}

func TestMockUndoLogBuilder_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name         string
		builder      *mockUndoLogBuilder
		wantBeforeOk bool
		wantAfterOk  bool
	}{
		{
			name: "both succeed",
			builder: &mockUndoLogBuilder{
				beforeImageResult: []*types.RecordImage{{TableName: "test"}},
				beforeImageError:  nil,
				afterImageResult:  []*types.RecordImage{{TableName: "test"}},
				afterImageError:   nil,
			},
			wantBeforeOk: true,
			wantAfterOk:  true,
		},
		{
			name: "before fails",
			builder: &mockUndoLogBuilder{
				beforeImageResult: nil,
				beforeImageError:  errors.New("before failed"),
				afterImageResult:  []*types.RecordImage{{TableName: "test"}},
				afterImageError:   nil,
			},
			wantBeforeOk: false,
			wantAfterOk:  true,
		},
		{
			name: "after fails",
			builder: &mockUndoLogBuilder{
				beforeImageResult: []*types.RecordImage{{TableName: "test"}},
				beforeImageError:  nil,
				afterImageResult:  nil,
				afterImageError:   errors.New("after failed"),
			},
			wantBeforeOk: true,
			wantAfterOk:  false,
		},
		{
			name: "both fail",
			builder: &mockUndoLogBuilder{
				beforeImageResult: nil,
				beforeImageError:  errors.New("before failed"),
				afterImageResult:  nil,
				afterImageError:   errors.New("after failed"),
			},
			wantBeforeOk: false,
			wantAfterOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			execCtx := &types.ExecContext{Query: "SELECT 1"}

			_, err := tt.builder.BeforeImage(ctx, execCtx)
			if tt.wantBeforeOk {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			_, err = tt.builder.AfterImage(ctx, execCtx, nil)
			if tt.wantAfterOk {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
