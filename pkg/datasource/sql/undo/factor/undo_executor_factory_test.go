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

package factor

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

type mockUndoExecutorHolder struct {
	mock.Mock
}

func (m *mockUndoExecutorHolder) GetInsertExecutor(sqlUndoLog undo.SQLUndoLog) undo.UndoExecutor {
	args := m.Called(sqlUndoLog)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(undo.UndoExecutor)
}

func (m *mockUndoExecutorHolder) GetDeleteExecutor(sqlUndoLog undo.SQLUndoLog) undo.UndoExecutor {
	args := m.Called(sqlUndoLog)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(undo.UndoExecutor)
}

func (m *mockUndoExecutorHolder) GetUpdateExecutor(sqlUndoLog undo.SQLUndoLog) undo.UndoExecutor {
	args := m.Called(sqlUndoLog)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(undo.UndoExecutor)
}

type mockUndoExecutor struct {
	mock.Mock
}

func (m *mockUndoExecutor) ExecuteOn(ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
	args := m.Called(ctx, dbType, conn)
	return args.Error(0)
}

func TestGetUndoExecutor(t *testing.T) {
	tests := []struct {
		name         string
		dbType       types.DBType
		sqlType      types.SQLType
		setupMock    func(*mockUndoExecutorHolder, *mockUndoExecutor)
		expectError  bool
		errorMessage string
	}{
		{
			name:    "insert executor - success",
			dbType:  types.DBTypeMySQL,
			sqlType: types.SQLTypeInsert,
			setupMock: func(holder *mockUndoExecutorHolder, executor *mockUndoExecutor) {
				holder.On("GetInsertExecutor", mock.AnythingOfType("undo.SQLUndoLog")).Return(executor)
			},
			expectError: false,
		},
		{
			name:    "delete executor - success",
			dbType:  types.DBTypeMySQL,
			sqlType: types.SQLTypeDelete,
			setupMock: func(holder *mockUndoExecutorHolder, executor *mockUndoExecutor) {
				holder.On("GetDeleteExecutor", mock.AnythingOfType("undo.SQLUndoLog")).Return(executor)
			},
			expectError: false,
		},
		{
			name:    "update executor - success",
			dbType:  types.DBTypeMySQL,
			sqlType: types.SQLTypeUpdate,
			setupMock: func(holder *mockUndoExecutorHolder, executor *mockUndoExecutor) {
				holder.On("GetUpdateExecutor", mock.AnythingOfType("undo.SQLUndoLog")).Return(executor)
			},
			expectError: false,
		},
		{
			name:         "unsupported sql type",
			dbType:       types.DBTypeMySQL,
			sqlType:      types.SQLType(999),
			setupMock:    func(holder *mockUndoExecutorHolder, executor *mockUndoExecutor) {},
			expectError:  true,
			errorMessage: "sql type: 999 not support",
		},
		{
			name:         "unsupported db type",
			dbType:       types.DBTypePostgreSQL,
			sqlType:      types.SQLTypeInsert,
			setupMock:    func(holder *mockUndoExecutorHolder, executor *mockUndoExecutor) {},
			expectError:  true,
			errorMessage: "db type executor not implement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMap := undoExecutorHolderMap
			defer func() {
				undoExecutorHolderMap = originalMap
			}()

			mockHolder := &mockUndoExecutorHolder{}
			mockExecutor := &mockUndoExecutor{}
			tt.setupMock(mockHolder, mockExecutor)

			if tt.dbType == types.DBTypeMySQL {
				undoExecutorHolderMap = map[types.DBType]undo.UndoExecutorHolder{
					types.DBTypeMySQL: mockHolder,
				}
			} else {
				undoExecutorHolderMap = map[types.DBType]undo.UndoExecutorHolder{}
			}

			sqlUndoLog := undo.SQLUndoLog{
				SQLType: tt.sqlType,
			}

			result, err := GetUndoExecutor(tt.dbType, sqlUndoLog)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, mockExecutor, result)
			}

			mockHolder.AssertExpectations(t)
		})
	}
}
