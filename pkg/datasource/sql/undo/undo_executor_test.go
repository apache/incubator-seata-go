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

package undo

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type mockUndoExecutorImpl struct {
	mock.Mock
}

func (m *mockUndoExecutorImpl) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
	args := m.Called(ctx, dbType, conn)
	return args.Error(0)
}

func TestUndoExecutor_Interface(t *testing.T) {
	executor := &mockUndoExecutorImpl{}
	assert.Implements(t, (*UndoExecutor)(nil), executor)
}

func TestUndoExecutor_ExecuteOn(t *testing.T) {
	executor := &mockUndoExecutorImpl{}
	ctx := context.Background()
	dbType := types.DBTypeMySQL
	var conn *sql.Conn

	executor.On("ExecuteOn", ctx, dbType, conn).Return(nil)

	err := executor.ExecuteOn(ctx, dbType, conn)
	assert.NoError(t, err)

	executor.AssertExpectations(t)
}
