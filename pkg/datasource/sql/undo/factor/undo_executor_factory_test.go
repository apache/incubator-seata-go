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
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestGetUndoExecutor_Insert(t *testing.T) {
	// Test getting INSERT undo executor for MySQL
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeInsert,
		TableName: "test_table",
	}

	executor, err := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)

	assert.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestGetUndoExecutor_Update(t *testing.T) {
	// Test getting UPDATE undo executor for MySQL
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeUpdate,
		TableName: "test_table",
	}

	executor, err := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)

	assert.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestGetUndoExecutor_Delete(t *testing.T) {
	// Test getting DELETE undo executor for MySQL
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeDelete,
		TableName: "test_table",
	}

	executor, err := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)

	assert.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestGetUndoExecutor_UnsupportedSQLType(t *testing.T) {
	// Test unsupported SQL types
	testCases := []struct {
		name    string
		sqlType types.SQLType
	}{
		{"SELECT", types.SQLTypeSelect},
		{"CREATE", types.SQLTypeCreate},
		{"DROP", types.SQLTypeDrop},
		{"ALTER", types.SQLTypeAlter},
		{"TRUNCATE", types.SQLTypeTruncate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqlUndoLog := undo.SQLUndoLog{
				SQLType:   tc.sqlType,
				TableName: "test_table",
			}

			executor, err := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)

			assert.Error(t, err)
			assert.Nil(t, executor)
			assert.Contains(t, err.Error(), "not support")
		})
	}
}

func TestGetUndoExecutor_UnsupportedDBType(t *testing.T) {
	// Test with unsupported database type
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeInsert,
		TableName: "test_table",
	}

	executor, err := GetUndoExecutor(types.DBTypePostgreSQL, sqlUndoLog)

	assert.Error(t, err)
	assert.Nil(t, executor)
	assert.Equal(t, ErrNotImplDBType, err)
}

func TestGetUndoExecutor_AllSupportedSQLTypes(t *testing.T) {
	// Test all supported SQL types with MySQL
	testCases := []struct {
		name    string
		sqlType types.SQLType
	}{
		{"Insert", types.SQLTypeInsert},
		{"Update", types.SQLTypeUpdate},
		{"Delete", types.SQLTypeDelete},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqlUndoLog := undo.SQLUndoLog{
				SQLType:   tc.sqlType,
				TableName: "test_table",
			}

			executor, err := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)

			assert.NoError(t, err)
			assert.NotNil(t, executor)
		})
	}
}

func TestGetUndoExecutor_WithBeforeAndAfterImages(t *testing.T) {
	// Test with complete undo log including before and after images
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeUpdate,
		TableName: "users",
		BeforeImage: &types.RecordImage{
			TableName: "users",
			Rows:      []types.RowImage{},
		},
		AfterImage: &types.RecordImage{
			TableName: "users",
			Rows:      []types.RowImage{},
		},
	}

	executor, err := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)

	assert.NoError(t, err)
	assert.NotNil(t, executor)
}

func TestGetUndoExecutor_InvalidDBTypeValue(t *testing.T) {
	// Test with invalid/custom DB type value
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeInsert,
		TableName: "test_table",
	}

	invalidDBType := types.DBType(999)
	executor, err := GetUndoExecutor(invalidDBType, sqlUndoLog)

	assert.Error(t, err)
	assert.Nil(t, executor)
}

func TestGetUndoExecutor_ErrorPropagation(t *testing.T) {
	// Test that errors from GetUndoExecutorHolder are properly propagated
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeInsert,
		TableName: "test_table",
	}

	executor, err := GetUndoExecutor(types.DBTypeOracle, sqlUndoLog)

	assert.Error(t, err)
	assert.Nil(t, executor)
	assert.Equal(t, ErrNotImplDBType, err)
}

func TestGetUndoExecutor_MultipleCallsSameType(t *testing.T) {
	// Test multiple calls with the same SQL type return valid executors
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeInsert,
		TableName: "test_table",
	}

	executor1, err1 := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)
	assert.NoError(t, err1)
	assert.NotNil(t, executor1)

	executor2, err2 := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)
	assert.NoError(t, err2)
	assert.NotNil(t, executor2)

	assert.IsType(t, executor1, executor2)
}

func TestGetUndoExecutor_EmptyTableName(t *testing.T) {
	// Test with empty table name (should still work as validation happens elsewhere)
	sqlUndoLog := undo.SQLUndoLog{
		SQLType:   types.SQLTypeInsert,
		TableName: "",
	}

	executor, err := GetUndoExecutor(types.DBTypeMySQL, sqlUndoLog)

	assert.NoError(t, err)
	assert.NotNil(t, executor)
}
