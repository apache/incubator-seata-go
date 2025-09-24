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
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type mockSQLExecutor struct {
	BaseExecutor
}

func (m *mockSQLExecutor) ExecWithNamedValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
	return m.BaseExecutor.ExecWithNamedValue(ctx, execCtx, f)
}

func (m *mockSQLExecutor) ExecWithValue(ctx context.Context, execCtx *types.ExecContext, f CallbackWithNamedValue) (types.ExecResult, error) {
	return m.BaseExecutor.ExecWithValue(ctx, execCtx, f)
}

type mockSQLHook struct {
	beforeCalled bool
	afterCalled  bool
	sqlType      types.SQLType
}

func (m *mockSQLHook) Type() types.SQLType {
	return m.sqlType
}

func (m *mockSQLHook) Before(ctx context.Context, execCtx *types.ExecContext) error {
	m.beforeCalled = true
	return nil
}

func (m *mockSQLHook) After(ctx context.Context, execCtx *types.ExecContext) error {
	m.afterCalled = true
	return nil
}

func TestRegisterATExecutor(t *testing.T) {
	RegisterATExecutor(types.DBTypeMySQL, func() SQLExecutor {
		return &mockSQLExecutor{}
	})
	
	// Check if the executor is registered
	assert.NotNil(t, atExecutors[types.DBTypeMySQL])
}

func TestBuildExecutor(t *testing.T) {
	// Register a mock executor
	RegisterATExecutor(types.DBTypeMySQL, func() SQLExecutor {
		return &mockSQLExecutor{}
	})
	
	// Test building an executor with a valid query
	executor, err := BuildExecutor(types.DBTypeMySQL, types.ATMode, "SELECT * FROM test_table")
	assert.NoError(t, err)
	assert.NotNil(t, executor)
	
	// Test building an executor with an invalid query
	executor, err = BuildExecutor(types.DBTypeMySQL, types.ATMode, "INVALID SQL")
	assert.Error(t, err)
	assert.Nil(t, executor)
}

func TestBaseExecutor_Interceptors(t *testing.T) {
	executor := &BaseExecutor{}
	hooks := []SQLHook{&mockSQLHook{}}
	
	executor.Interceptors(hooks)
	
	assert.Equal(t, len(hooks), len(executor.hooks))
}

func TestBaseExecutor_ExecWithNamedValue(t *testing.T) {
	executor := &BaseExecutor{}
	hook := &mockSQLHook{}
	executor.Interceptors([]SQLHook{hook})
	
	ctx := context.Background()
	execCtx := &types.ExecContext{
		Query: "SELECT * FROM test_table",
	}
	
	callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		return nil, nil
	}
	
	_, err := executor.ExecWithNamedValue(ctx, execCtx, callback)
	assert.NoError(t, err)
	assert.True(t, hook.beforeCalled)
	assert.True(t, hook.afterCalled)
}

func TestBaseExecutor_ExecWithValue(t *testing.T) {
	executor := &BaseExecutor{}
	hook := &mockSQLHook{}
	executor.Interceptors([]SQLHook{hook})
	
	ctx := context.Background()
	execCtx := &types.ExecContext{
		Query: "SELECT * FROM test_table",
	}
	
	callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		return nil, nil
	}
	
	_, err := executor.ExecWithValue(ctx, execCtx, callback)
	assert.NoError(t, err)
	assert.True(t, hook.beforeCalled)
	assert.True(t, hook.afterCalled)
}