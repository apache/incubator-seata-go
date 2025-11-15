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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUndoExecutorHolderImpl struct {
	mock.Mock
}

func (m *mockUndoExecutorHolderImpl) GetInsertExecutor(sqlUndoLog SQLUndoLog) UndoExecutor {
	args := m.Called(sqlUndoLog)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(UndoExecutor)
}

func (m *mockUndoExecutorHolderImpl) GetDeleteExecutor(sqlUndoLog SQLUndoLog) UndoExecutor {
	args := m.Called(sqlUndoLog)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(UndoExecutor)
}

func (m *mockUndoExecutorHolderImpl) GetUpdateExecutor(sqlUndoLog SQLUndoLog) UndoExecutor {
	args := m.Called(sqlUndoLog)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(UndoExecutor)
}

func TestUndoExecutorHolder_Interface(t *testing.T) {
	holder := &mockUndoExecutorHolderImpl{}
	assert.Implements(t, (*UndoExecutorHolder)(nil), holder)
}

func TestUndoExecutorHolder_GetInsertExecutor(t *testing.T) {
	holder := &mockUndoExecutorHolderImpl{}
	executor := &mockUndoExecutorImpl{}
	sqlUndoLog := SQLUndoLog{TableName: "test_table"}

	holder.On("GetInsertExecutor", sqlUndoLog).Return(executor)

	result := holder.GetInsertExecutor(sqlUndoLog)
	assert.Equal(t, executor, result)

	holder.AssertExpectations(t)
}

func TestUndoExecutorHolder_GetDeleteExecutor(t *testing.T) {
	holder := &mockUndoExecutorHolderImpl{}
	executor := &mockUndoExecutorImpl{}
	sqlUndoLog := SQLUndoLog{TableName: "test_table"}

	holder.On("GetDeleteExecutor", sqlUndoLog).Return(executor)

	result := holder.GetDeleteExecutor(sqlUndoLog)
	assert.Equal(t, executor, result)

	holder.AssertExpectations(t)
}

func TestUndoExecutorHolder_GetUpdateExecutor(t *testing.T) {
	holder := &mockUndoExecutorHolderImpl{}
	executor := &mockUndoExecutorImpl{}
	sqlUndoLog := SQLUndoLog{TableName: "test_table"}

	holder.On("GetUpdateExecutor", sqlUndoLog).Return(executor)

	result := holder.GetUpdateExecutor(sqlUndoLog)
	assert.Equal(t, executor, result)

	holder.AssertExpectations(t)
}

func TestUndoExecutorHolder_ReturnsNil(t *testing.T) {
	holder := &mockUndoExecutorHolderImpl{}
	sqlUndoLog := SQLUndoLog{TableName: "test_table"}

	holder.On("GetInsertExecutor", sqlUndoLog).Return(nil)
	holder.On("GetDeleteExecutor", sqlUndoLog).Return(nil)
	holder.On("GetUpdateExecutor", sqlUndoLog).Return(nil)

	insertResult := holder.GetInsertExecutor(sqlUndoLog)
	assert.Nil(t, insertResult)

	deleteResult := holder.GetDeleteExecutor(sqlUndoLog)
	assert.Nil(t, deleteResult)

	updateResult := holder.GetUpdateExecutor(sqlUndoLog)
	assert.Nil(t, updateResult)

	holder.AssertExpectations(t)
}
