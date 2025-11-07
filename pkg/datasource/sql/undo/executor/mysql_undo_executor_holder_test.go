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

package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestNewMySQLUndoExecutorHolder(t *testing.T) {
	holder := NewMySQLUndoExecutorHolder()
	assert.NotNil(t, holder)
	assert.IsType(t, &MySQLUndoExecutorHolder{}, holder)
}

func TestMySQLUndoExecutorHolder_GetInsertExecutor(t *testing.T) {
	holder := NewMySQLUndoExecutorHolder()
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLTypeInsert,
		AfterImage: &types.RecordImage{
			TableName: "test_table",
		},
	}

	executor := holder.GetInsertExecutor(sqlUndoLog)
	assert.NotNil(t, executor)
	assert.IsType(t, &mySQLUndoInsertExecutor{}, executor)
}

func TestMySQLUndoExecutorHolder_GetUpdateExecutor(t *testing.T) {
	holder := NewMySQLUndoExecutorHolder()
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLTypeUpdate,
		BeforeImage: &types.RecordImage{
			TableName: "test_table",
		},
		AfterImage: &types.RecordImage{
			TableName: "test_table",
		},
	}

	executor := holder.GetUpdateExecutor(sqlUndoLog)
	assert.NotNil(t, executor)
	assert.IsType(t, &mySQLUndoUpdateExecutor{}, executor)
}

func TestMySQLUndoExecutorHolder_GetDeleteExecutor(t *testing.T) {
	holder := NewMySQLUndoExecutorHolder()
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLTypeDelete,
		BeforeImage: &types.RecordImage{
			TableName: "test_table",
		},
	}

	executor := holder.GetDeleteExecutor(sqlUndoLog)
	assert.NotNil(t, executor)
	assert.IsType(t, &mySQLUndoDeleteExecutor{}, executor)
}

func TestMySQLUndoExecutorHolder_InterfaceImplementation(t *testing.T) {
	var holder undo.UndoExecutorHolder = NewMySQLUndoExecutorHolder()
	assert.NotNil(t, holder)
	assert.Implements(t, (*undo.UndoExecutorHolder)(nil), holder)
}

func TestMySQLUndoExecutorHolder_AllExecutorTypes(t *testing.T) {
	holder := NewMySQLUndoExecutorHolder()

	testCases := []struct {
		name     string
		sqlType  types.SQLType
		getFunc  func(undo.SQLUndoLog) undo.UndoExecutor
		expected interface{}
	}{
		{
			name:     "Insert Executor",
			sqlType:  types.SQLTypeInsert,
			getFunc:  holder.GetInsertExecutor,
			expected: &mySQLUndoInsertExecutor{},
		},
		{
			name:     "Update Executor",
			sqlType:  types.SQLTypeUpdate,
			getFunc:  holder.GetUpdateExecutor,
			expected: &mySQLUndoUpdateExecutor{},
		},
		{
			name:     "Delete Executor",
			sqlType:  types.SQLTypeDelete,
			getFunc:  holder.GetDeleteExecutor,
			expected: &mySQLUndoDeleteExecutor{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqlUndoLog := undo.SQLUndoLog{
				TableName: "test_table",
				SQLType:   tc.sqlType,
			}

			executor := tc.getFunc(sqlUndoLog)
			assert.NotNil(t, executor)
			assert.IsType(t, tc.expected, executor)
		})
	}
}
