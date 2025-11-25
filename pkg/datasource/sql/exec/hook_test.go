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

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

// TestRegisterCommonHook tests the RegisterCommonHook function
func TestRegisterCommonHook(t *testing.T) {
	// Save original state and restore after test
	originalCommonHook := commonHook
	defer func() { commonHook = originalCommonHook }()

	tests := []struct {
		name          string
		initialHooks  []SQLHook
		hooksToAdd    []SQLHook
		expectedCount int
	}{
		{
			name:         "register single common hook",
			initialHooks: []SQLHook{},
			hooksToAdd: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			expectedCount: 1,
		},
		{
			name: "register multiple common hooks",
			initialHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			hooksToAdd: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
			},
			expectedCount: 3,
		},
		{
			name:         "register hook with unknown type",
			initialHooks: []SQLHook{},
			hooksToAdd: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeUnknown},
			},
			expectedCount: 1, // Common hooks accept any type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup initial state
			commonHook = make([]SQLHook, len(tt.initialHooks))
			copy(commonHook, tt.initialHooks)

			// Register hooks
			for _, hook := range tt.hooksToAdd {
				RegisterCommonHook(hook)
			}

			// Verify
			assert.Equal(t, tt.expectedCount, len(commonHook), "common hook count should match")
			assert.Equal(t, cap(commonHook) >= len(commonHook), true, "capacity should be sufficient")
		})
	}
}

// TestCleanCommonHook tests the CleanCommonHook function
func TestCleanCommonHook(t *testing.T) {
	// Save original state and restore after test
	originalCommonHook := commonHook
	defer func() { commonHook = originalCommonHook }()

	tests := []struct {
		name         string
		initialHooks []SQLHook
	}{
		{
			name:         "clean empty common hooks",
			initialHooks: []SQLHook{},
		},
		{
			name: "clean single common hook",
			initialHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
		},
		{
			name: "clean multiple common hooks",
			initialHooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			commonHook = make([]SQLHook, len(tt.initialHooks))
			copy(commonHook, tt.initialHooks)

			// Execute
			CleanCommonHook()

			// Verify
			assert.Equal(t, 0, len(commonHook), "common hooks should be empty")
			assert.Equal(t, 4, cap(commonHook), "capacity should be reset to 4")
		})
	}
}

// TestRegisterHook tests the RegisterHook function
func TestRegisterHook(t *testing.T) {
	// Save original state and restore after test
	originalHookSolts := hookSolts
	defer func() { hookSolts = originalHookSolts }()

	tests := []struct {
		name             string
		hooksToRegister  []SQLHook
		expectedSQLTypes []types.SQLType
		expectedCounts   map[types.SQLType]int
		skipUnknownType  bool
	}{
		{
			name: "register single hook",
			hooksToRegister: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			expectedSQLTypes: []types.SQLType{types.SQLTypeInsert},
			expectedCounts: map[types.SQLType]int{
				types.SQLTypeInsert: 1,
			},
		},
		{
			name: "register multiple hooks of same type",
			hooksToRegister: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
			},
			expectedSQLTypes: []types.SQLType{types.SQLTypeUpdate},
			expectedCounts: map[types.SQLType]int{
				types.SQLTypeUpdate: 3,
			},
		},
		{
			name: "register hooks of different types",
			hooksToRegister: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
				&mockSQLHook{sqlType: types.SQLTypeSelect},
			},
			expectedSQLTypes: []types.SQLType{
				types.SQLTypeInsert,
				types.SQLTypeUpdate,
				types.SQLTypeDelete,
				types.SQLTypeSelect,
			},
			expectedCounts: map[types.SQLType]int{
				types.SQLTypeInsert: 1,
				types.SQLTypeUpdate: 1,
				types.SQLTypeDelete: 1,
				types.SQLTypeSelect: 1,
			},
		},
		{
			name: "register hook with unknown type should be skipped",
			hooksToRegister: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeUnknown},
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			expectedSQLTypes: []types.SQLType{types.SQLTypeInsert},
			expectedCounts: map[types.SQLType]int{
				types.SQLTypeInsert: 1,
			},
			skipUnknownType: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			hookSolts = map[types.SQLType][]SQLHook{}

			// Register hooks
			for _, hook := range tt.hooksToRegister {
				RegisterHook(hook)
			}

			// Verify expected SQL types are registered
			for _, sqlType := range tt.expectedSQLTypes {
				hooks, exists := hookSolts[sqlType]
				assert.True(t, exists, "hook slot should exist for SQL type %v", sqlType)
				expectedCount := tt.expectedCounts[sqlType]
				assert.Equal(t, expectedCount, len(hooks), "hook count should match for SQL type %v", sqlType)
			}

			// Verify unknown type is skipped
			if tt.skipUnknownType {
				_, exists := hookSolts[types.SQLTypeUnknown]
				assert.False(t, exists, "unknown type should not be registered")
			}
		})
	}
}

// TestHookExecution tests the execution order and behavior of hooks
func TestHookExecution(t *testing.T) {
	tests := []struct {
		name            string
		hooks           []SQLHook
		wantBeforeCount int
		wantAfterCount  int
	}{
		{
			name: "hooks execute in order",
			hooks: []SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			wantBeforeCount: 3,
			wantAfterCount:  3,
		},
		{
			name: "before hook error does not prevent execution",
			hooks: []SQLHook{
				&mockSQLHook{
					sqlType:     types.SQLTypeInsert,
					beforeError: fmt.Errorf("before hook error"),
				},
			},
			wantBeforeCount: 1,
			wantAfterCount:  1, // After hooks run even if before fails (error is ignored)
		},
		{
			name: "after hook error is logged but doesn't fail",
			hooks: []SQLHook{
				&mockSQLHook{
					sqlType:    types.SQLTypeInsert,
					afterError: fmt.Errorf("after hook error"),
				},
			},
			wantBeforeCount: 1,
			wantAfterCount:  1, // After hooks always run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &BaseExecutor{
				hooks: tt.hooks,
			}

			execCtx := &types.ExecContext{
				Query:       "INSERT INTO users VALUES (?, ?)",
				NamedValues: []driver.NamedValue{{Value: "Alice"}, {Value: 30}},
			}

			callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return newMockExecResult(1, 1), nil
			}

			_, err := executor.ExecWithNamedValue(context.Background(), execCtx, callback)
			assert.NoError(t, err, "should not return error")

			// Verify hook call counts
			beforeCount := 0
			afterCount := 0
			for _, hook := range tt.hooks {
				mockHook := hook.(*mockSQLHook)
				beforeCount += mockHook.beforeCallCount
				afterCount += mockHook.afterCallCount
			}
			assert.Equal(t, tt.wantBeforeCount, beforeCount, "before hook call count should match")
			assert.Equal(t, tt.wantAfterCount, afterCount, "after hook call count should match")
		})
	}
}

// TestHookIntegration tests the integration between hook registration and execution
func TestHookIntegration(t *testing.T) {
	// Save original state and restore after test
	originalCommonHook := commonHook
	originalHookSolts := hookSolts
	defer func() {
		commonHook = originalCommonHook
		hookSolts = originalHookSolts
	}()

	// Clean state
	commonHook = make([]SQLHook, 0, 4)
	hookSolts = map[types.SQLType][]SQLHook{}

	// Register common hooks
	commonHook1 := &mockSQLHook{sqlType: types.SQLTypeSelect}
	commonHook2 := &mockSQLHook{sqlType: types.SQLTypeUpdate}
	RegisterCommonHook(commonHook1)
	RegisterCommonHook(commonHook2)

	// Register type-specific hooks
	insertHook1 := &mockSQLHook{sqlType: types.SQLTypeInsert}
	insertHook2 := &mockSQLHook{sqlType: types.SQLTypeInsert}
	RegisterHook(insertHook1)
	RegisterHook(insertHook2)

	updateHook := &mockSQLHook{sqlType: types.SQLTypeUpdate}
	RegisterHook(updateHook)

	// Verify registration
	assert.Equal(t, 2, len(commonHook), "should have 2 common hooks")
	assert.Equal(t, 2, len(hookSolts[types.SQLTypeInsert]), "should have 2 insert hooks")
	assert.Equal(t, 1, len(hookSolts[types.SQLTypeUpdate]), "should have 1 update hook")

	// Test hook execution with BaseExecutor
	executor := &BaseExecutor{
		hooks: []SQLHook{commonHook1, insertHook1, insertHook2},
	}

	execCtx := &types.ExecContext{
		Query:       "INSERT INTO users VALUES (?, ?)",
		NamedValues: []driver.NamedValue{{Value: "Alice"}, {Value: 30}},
	}

	callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		return newMockExecResult(1, 1), nil
	}

	result, err := executor.ExecWithNamedValue(context.Background(), execCtx, callback)

	assert.NoError(t, err, "execution should succeed")
	assert.NotNil(t, result, "result should not be nil")

	// Verify all hooks were called
	assert.Equal(t, 1, commonHook1.beforeCallCount, "common hook 1 before should be called")
	assert.Equal(t, 1, commonHook1.afterCallCount, "common hook 1 after should be called")
	assert.Equal(t, 1, insertHook1.beforeCallCount, "insert hook 1 before should be called")
	assert.Equal(t, 1, insertHook1.afterCallCount, "insert hook 1 after should be called")
	assert.Equal(t, 1, insertHook2.beforeCallCount, "insert hook 2 before should be called")
	assert.Equal(t, 1, insertHook2.afterCallCount, "insert hook 2 after should be called")

	// Clean hooks and verify
	CleanCommonHook()
	assert.Equal(t, 0, len(commonHook), "common hooks should be cleaned")
}

// TestHookChainExecution tests that hooks execute in a chain and respect the execution order
func TestHookChainExecution(t *testing.T) {
	executionOrder := []string{}

	hook1 := &trackingHook{
		sqlType: types.SQLTypeInsert,
		onBefore: func() {
			executionOrder = append(executionOrder, "hook1-before")
		},
		onAfter: func() {
			executionOrder = append(executionOrder, "hook1-after")
		},
	}

	hook2 := &trackingHook{
		sqlType: types.SQLTypeInsert,
		onBefore: func() {
			executionOrder = append(executionOrder, "hook2-before")
		},
		onAfter: func() {
			executionOrder = append(executionOrder, "hook2-after")
		},
	}

	hook3 := &trackingHook{
		sqlType: types.SQLTypeInsert,
		onBefore: func() {
			executionOrder = append(executionOrder, "hook3-before")
		},
		onAfter: func() {
			executionOrder = append(executionOrder, "hook3-after")
		},
	}

	executor := &BaseExecutor{
		hooks: []SQLHook{hook1, hook2, hook3},
	}

	execCtx := &types.ExecContext{
		Query:       "INSERT INTO users VALUES (?, ?)",
		NamedValues: []driver.NamedValue{{Value: "Alice"}, {Value: 30}},
	}

	callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		executionOrder = append(executionOrder, "callback")
		return newMockExecResult(1, 1), nil
	}

	_, err := executor.ExecWithNamedValue(context.Background(), execCtx, callback)
	assert.NoError(t, err, "execution should succeed")

	// Verify execution order: all before hooks, then callback, then all after hooks (in same order)
	expectedOrder := []string{
		"hook1-before",
		"hook2-before",
		"hook3-before",
		"callback",
		"hook1-after",
		"hook2-after",
		"hook3-after",
	}
	assert.Equal(t, expectedOrder, executionOrder, "hooks should execute in correct order")
}

// trackingHook is a helper hook that tracks execution order
type trackingHook struct {
	sqlType  types.SQLType
	onBefore func()
	onAfter  func()
}

func (h *trackingHook) Type() types.SQLType {
	return h.sqlType
}

func (h *trackingHook) Before(ctx context.Context, execCtx *types.ExecContext) error {
	if h.onBefore != nil {
		h.onBefore()
	}
	return nil
}

func (h *trackingHook) After(ctx context.Context, execCtx *types.ExecContext) error {
	if h.onAfter != nil {
		h.onAfter()
	}
	return nil
}