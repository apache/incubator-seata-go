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

package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQLTypeConstants(t *testing.T) {
	// Test some key SQL type constants
	assert.Equal(t, int(SQLTypeSelect), 0)
	assert.Equal(t, int(SQLTypeInsert), 1)
	assert.Equal(t, int(SQLTypeUpdate), 2)
	assert.Equal(t, int(SQLTypeDelete), 3)
	assert.Equal(t, int(SQLTypeSelectForUpdate), 4)
	assert.Equal(t, int(SQLTypeReplace), 5)
	assert.Equal(t, int(SQLTypeTruncate), 6)
	assert.Equal(t, int(SQLTypeCreate), 7)
	assert.Equal(t, int(SQLTypeDrop), 8)
	assert.Equal(t, int(SQLTypeLoad), 9)
	assert.Equal(t, int(SQLTypeMerge), 10)
	assert.Equal(t, int(SQLTypeShow), 11)
	assert.Equal(t, int(SQLTypeAlter), 12)
	assert.Equal(t, int(SQLTypeRename), 13)
	assert.Equal(t, int(SQLTypeDump), 14)
	assert.Equal(t, int(SQLTypeDebug), 15)
	assert.Equal(t, int(SQLTypeExplain), 16)
	assert.Equal(t, int(SQLTypeProcedure), 17)
	assert.Equal(t, int(SQLTypeDesc), 18)
}

func TestSQLType_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		sqlType  SQLType
		expected []byte
	}{
		{"SELECT", SQLTypeSelect, []byte("SELECT")},
		{"INSERT", SQLTypeInsert, []byte("INSERT")},
		{"UPDATE", SQLTypeUpdate, []byte("UPDATE")},
		{"DELETE", SQLTypeDelete, []byte("DELETE")},
		{"SELECT_FOR_UPDATE", SQLTypeSelectForUpdate, []byte("SELECT_FOR_UPDATE")},
		{"INSERT_ON_UPDATE", SQLTypeInsertOnDuplicateUpdate, []byte("INSERT_ON_UPDATE")},
		{"REPLACE", SQLTypeReplace, []byte("REPLACE")},
		{"TRUNCATE", SQLTypeTruncate, []byte("TRUNCATE")},
		{"CREATE", SQLTypeCreate, []byte("CREATE")},
		{"DROP", SQLTypeDrop, []byte("DROP")},
		{"LOAD", SQLTypeLoad, []byte("LOAD")},
		{"MERGE", SQLTypeMerge, []byte("MERGE")},
		{"SHOW", SQLTypeShow, []byte("SHOW")},
		{"ALTER", SQLTypeAlter, []byte("ALTER")},
		{"RENAME", SQLTypeRename, []byte("RENAME")},
		{"DUMP", SQLTypeDump, []byte("DUMP")},
		{"DEBUG", SQLTypeDebug, []byte("DEBUG")},
		{"EXPLAIN", SQLTypeExplain, []byte("EXPLAIN")},
		{"DESC", SQLTypeDesc, []byte("DESC")},
		{"SET", SQLTypeSet, []byte("SET")},
		{"RELOAD", SQLTypeReload, []byte("RELOAD")},
		{"SELECT_UNION", SQLTypeSelectUnion, []byte("SELECT_UNION")},
		{"CREATE_TABLE", SQLTypeCreateTable, []byte("CREATE_TABLE")},
		{"DROP_TABLE", SQLTypeDropTable, []byte("DROP_TABLE")},
		{"ALTER_TABLE", SQLTypeAlterTable, []byte("ALTER_TABLE")},
		{"SELECT_FROM_UPDATE", SQLTypeSelectFromUpdate, []byte("SELECT_FROM_UPDATE")},
		{"MULTI_DELETE", SQLTypeMultiDelete, []byte("MULTI_DELETE")},
		{"MULTI_UPDATE", SQLTypeMultiUpdate, []byte("MULTI_UPDATE")},
		{"CREATE_INDEX", SQLTypeCreateIndex, []byte("CREATE_INDEX")},
		{"DROP_INDEX", SQLTypeDropIndex, []byte("DROP_INDEX")},
		{"MULTI", SQLTypeMulti, []byte("MULTI")},
		{"INVALID", SQLType(9999), []byte("INVALID_SQLTYPE")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.sqlType.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSQLType_UnmarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected SQLType
	}{
		{"SELECT", []byte("SELECT"), SQLTypeSelect},
		{"INSERT", []byte("INSERT"), SQLTypeInsert},
		{"UPDATE", []byte("UPDATE"), SQLTypeUpdate},
		{"DELETE", []byte("DELETE"), SQLTypeDelete},
		{"SELECT_FOR_UPDATE", []byte("SELECT_FOR_UPDATE"), SQLTypeSelectForUpdate},
		{"INSERT_ON_UPDATE", []byte("INSERT_ON_UPDATE"), SQLTypeInsertOnDuplicateUpdate},
		{"REPLACE", []byte("REPLACE"), SQLTypeReplace},
		{"TRUNCATE", []byte("TRUNCATE"), SQLTypeTruncate},
		{"CREATE", []byte("CREATE"), SQLTypeCreate},
		{"DROP", []byte("DROP"), SQLTypeDrop},
		{"LOAD", []byte("LOAD"), SQLTypeLoad},
		{"MERGE", []byte("MERGE"), SQLTypeMerge},
		{"SHOW", []byte("SHOW"), SQLTypeShow},
		{"ALTER", []byte("ALTER"), SQLTypeAlter},
		{"RENAME", []byte("RENAME"), SQLTypeRename},
		{"DUMP", []byte("DUMP"), SQLTypeDump},
		{"DEBUG", []byte("DEBUG"), SQLTypeDebug},
		{"EXPLAIN", []byte("EXPLAIN"), SQLTypeExplain},
		{"DESC", []byte("DESC"), SQLTypeDesc},
		{"SET", []byte("SET"), SQLTypeSet},
		{"RELOAD", []byte("RELOAD"), SQLTypeReload},
		{"SELECT_UNION", []byte("SELECT_UNION"), SQLTypeSelectUnion},
		{"CREATE_TABLE", []byte("CREATE_TABLE"), SQLTypeCreateTable},
		{"DROP_TABLE", []byte("DROP_TABLE"), SQLTypeDropTable},
		{"ALTER_TABLE", []byte("ALTER_TABLE"), SQLTypeAlterTable},
		{"SELECT_FROM_UPDATE", []byte("SELECT_FROM_UPDATE"), SQLTypeSelectFromUpdate},
		{"MULTI_DELETE", []byte("MULTI_DELETE"), SQLTypeMultiDelete},
		{"MULTI_UPDATE", []byte("MULTI_UPDATE"), SQLTypeMultiUpdate},
		{"CREATE_INDEX", []byte("CREATE_INDEX"), SQLTypeCreateIndex},
		{"DROP_INDEX", []byte("DROP_INDEX"), SQLTypeDropIndex},
		{"MULTI", []byte("MULTI"), SQLTypeMulti},
		{"Unknown", []byte("UNKNOWN"), SQLType(0)}, // defaults to 0
		{"Empty", []byte(""), SQLType(0)},          // defaults to 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sqlType SQLType
			err := sqlType.UnmarshalText(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, sqlType)
		})
	}
}

func TestSQLType_MarshalUnmarshalRoundTrip(t *testing.T) {
	sqlTypes := []SQLType{
		SQLTypeSelect,
		SQLTypeInsert,
		SQLTypeUpdate,
		SQLTypeDelete,
		SQLTypeSelectForUpdate,
		SQLTypeInsertOnDuplicateUpdate,
		SQLTypeReplace,
		SQLTypeTruncate,
		SQLTypeCreate,
		SQLTypeDrop,
		SQLTypeLoad,
		SQLTypeMerge,
		SQLTypeShow,
		SQLTypeAlter,
		SQLTypeRename,
		SQLTypeDump,
		SQLTypeDebug,
		SQLTypeExplain,
		SQLTypeDesc,
		SQLTypeSet,
		SQLTypeReload,
		SQLTypeSelectUnion,
		SQLTypeCreateTable,
		SQLTypeDropTable,
		SQLTypeAlterTable,
		SQLTypeSelectFromUpdate,
		SQLTypeMultiDelete,
		SQLTypeMultiUpdate,
		SQLTypeCreateIndex,
		SQLTypeDropIndex,
		SQLTypeMulti,
	}

	for i, originalType := range sqlTypes {
		t.Run(fmt.Sprintf("SQLType_%d", i), func(t *testing.T) {
			// Marshal
			marshaled, err := originalType.MarshalText()
			assert.NoError(t, err)

			// Unmarshal
			var unmarshaled SQLType
			err = unmarshaled.UnmarshalText(marshaled)
			assert.NoError(t, err)

			// Should be equal
			assert.Equal(t, originalType, unmarshaled)
		})
	}
}

func TestSQLTypeSpecialConstants(t *testing.T) {
	// Test the special constants that have gaps
	// SQLTypeInsertIgnore = iota + 57, where iota was around 69 before this line
	assert.Equal(t, int(SQLTypeInsertIgnore), 101)            // roughly 44 + 57
	assert.Equal(t, int(SQLTypeInsertOnDuplicateUpdate), 102) // next in sequence
	assert.True(t, int(SQLTypeMulti) > 1000)                  // iota + 999
	assert.Equal(t, SQLTypeMulti+1, SQLTypeUnknown)           // Should be one more than multi
}

func TestSQLType_StringRepresentation(t *testing.T) {
	// Test that SQLType can be represented as an integer
	assert.Equal(t, int(SQLTypeSelect), 0)
	assert.Equal(t, int(SQLTypeInsert), 1)
	assert.Equal(t, int(SQLTypeUpdate), 2)
}
