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

package at

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/opcode"
	"github.com/arana-db/parser/test_driver"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestBaseExecBuildLockKey(t *testing.T) {
	var exec baseExecutor

	columnID := types.ColumnMeta{
		ColumnName: "id",
	}
	columnUserId := types.ColumnMeta{
		ColumnName: "userId",
	}
	columnName := types.ColumnMeta{
		ColumnName: "name",
	}
	columnAge := types.ColumnMeta{
		ColumnName: "age",
	}
	columnNonExistent := types.ColumnMeta{
		ColumnName: "non_existent",
	}

	columnsTwoPk := []types.ColumnMeta{columnID, columnUserId}
	columnsThreePk := []types.ColumnMeta{columnID, columnUserId, columnAge}
	columnsMixPk := []types.ColumnMeta{columnName, columnAge}

	getColumnImage := func(columnName string, value interface{}) types.ColumnImage {
		return types.ColumnImage{KeyType: types.IndexTypePrimaryKey, ColumnName: columnName, Value: value}
	}

	tests := []struct {
		name     string
		metaData types.TableMeta
		records  types.RecordImage
		expected string
	}{
		{
			"Two Primary Keys",
			types.TableMeta{
				TableName: "test_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsTwoPk},
				},
			},
			types.RecordImage{
				TableName: "test_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "user1")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "user2")}},
				},
			},
			"TEST_NAME:1_user1,2_user2",
		},
		{
			"Three Primary Keys",
			types.TableMeta{
				TableName: "test2_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsThreePk},
				},
			},
			types.RecordImage{
				TableName: "test2_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "one"), getColumnImage("age", "11")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "two"), getColumnImage("age", "22")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 3), getColumnImage("userId", "three"), getColumnImage("age", "33")}},
				},
			},
			"TEST2_NAME:1_one_11,2_two_22,3_three_33",
		},
		{
			name: "Single Primary Key",
			metaData: types.TableMeta{
				TableName: "single_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "single_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 100)}},
				},
			},
			expected: "SINGLE_KEY:100",
		},
		{
			name: "Mixed Type Keys",
			metaData: types.TableMeta{
				TableName: "mixed_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsMixPk},
				},
			},
			records: types.RecordImage{
				TableName: "mixed_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("name", "mike"), getColumnImage("age", 25)}},
				},
			},
			expected: "MIXED_KEY:mike_25",
		},
		{
			name: "Empty Records",
			metaData: types.TableMeta{
				TableName: "empty",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records:  types.RecordImage{TableName: "empty"},
			expected: "EMPTY:",
		},
		{
			name: "Special Characters",
			metaData: types.TableMeta{
				TableName: "special",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "special",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", "A,b_c")}},
				},
			},
			expected: "SPECIAL:A,b_c",
		},
		{
			name: "Non-existent Key Name",
			metaData: types.TableMeta{
				TableName: "error_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnNonExistent}},
				},
			},
			records: types.RecordImage{
				TableName: "error_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1)}},
				},
			},
			expected: "ERROR_KEY:",
		},
		{
			name: "Multiple Rows With Nil PK Value",
			metaData: types.TableMeta{
				TableName: "nil_pk",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "nil_pk",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", nil)}},
					{Columns: []types.ColumnImage{getColumnImage("id", 123)}},
					{Columns: []types.ColumnImage{getColumnImage("id", nil)}},
				},
			},
			expected: "NIL_PK:,123,",
		},
		{
			name: "PK As Bool And Float",
			metaData: types.TableMeta{
				TableName: "type_pk",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnName, columnAge}},
				},
			},
			records: types.RecordImage{
				TableName: "type_pk",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("name", true), getColumnImage("age", 3.14)}},
					{Columns: []types.ColumnImage{getColumnImage("name", false), getColumnImage("age", 0.0)}},
				},
			},
			expected: "TYPE_PK:true_3.14,false_0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := exec.buildLockKey(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}

// TestBaseExecutor_BeforeHooks tests the beforeHooks method
func TestBaseExecutor_BeforeHooks(t *testing.T) {
	tests := []struct {
		name       string
		hooks      []exec.SQLHook
		wantErr    bool
		expectErr  string
		hookCalled int
	}{
		{
			name:       "no hooks",
			hooks:      []exec.SQLHook{},
			wantErr:    false,
			hookCalled: 0,
		},
		{
			name: "single hook - success",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			wantErr:    false,
			hookCalled: 1,
		},
		{
			name: "multiple hooks - all success",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
			},
			wantErr:    false,
			hookCalled: 3,
		},
		{
			name: "hook returns error",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert, beforeError: fmt.Errorf("hook error")},
			},
			wantErr:    true,
			expectErr:  "hook error",
			hookCalled: 1,
		},
		{
			name: "second hook returns error",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeUpdate, beforeError: fmt.Errorf("second hook error")},
			},
			wantErr:    true,
			expectErr:  "second hook error",
			hookCalled: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &baseExecutor{
				hooks: tt.hooks,
			}

			execCtx := &types.ExecContext{
				Query:       "INSERT INTO users VALUES (?, ?)",
				NamedValues: []driver.NamedValue{{Value: "test"}},
			}

			err := executor.beforeHooks(context.Background(), execCtx)

			if tt.wantErr {
				assert.Error(t, err, "should return error")
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr, "error message should match")
				}
			} else {
				assert.NoError(t, err, "should not return error")
			}

			// Verify hooks were called
			callCount := 0
			for _, hook := range tt.hooks {
				mockHook := hook.(*mockSQLHook)
				callCount += mockHook.beforeCallCount
			}
			assert.Equal(t, tt.hookCalled, callCount, "hook call count should match")
		})
	}
}

// TestBaseExecutor_AfterHooks tests the afterHooks method
func TestBaseExecutor_AfterHooks(t *testing.T) {
	tests := []struct {
		name       string
		hooks      []exec.SQLHook
		hookCalled int
	}{
		{
			name:       "no hooks",
			hooks:      []exec.SQLHook{},
			hookCalled: 0,
		},
		{
			name: "single hook - success",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
			},
			hookCalled: 1,
		},
		{
			name: "multiple hooks - all success",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert},
				&mockSQLHook{sqlType: types.SQLTypeUpdate},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
			},
			hookCalled: 3,
		},
		{
			name: "hook returns error - should not fail",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert, afterError: fmt.Errorf("hook error")},
			},
			hookCalled: 1,
		},
		{
			name: "multiple hooks with errors - should call all",
			hooks: []exec.SQLHook{
				&mockSQLHook{sqlType: types.SQLTypeInsert, afterError: fmt.Errorf("first error")},
				&mockSQLHook{sqlType: types.SQLTypeUpdate, afterError: fmt.Errorf("second error")},
				&mockSQLHook{sqlType: types.SQLTypeDelete},
			},
			hookCalled: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &baseExecutor{
				hooks: tt.hooks,
			}

			execCtx := &types.ExecContext{
				Query:       "INSERT INTO users VALUES (?, ?)",
				NamedValues: []driver.NamedValue{{Value: "test"}},
			}

			err := executor.afterHooks(context.Background(), execCtx)

			// afterHooks always returns nil, even if hooks return errors
			assert.NoError(t, err, "afterHooks should always return nil")

			// Verify hooks were called
			callCount := 0
			for _, hook := range tt.hooks {
				mockHook := hook.(*mockSQLHook)
				callCount += mockHook.afterCallCount
			}
			assert.Equal(t, tt.hookCalled, callCount, "hook call count should match")
		})
	}
}

// TestBaseExecutor_GetScanSlice tests the GetScanSlice method
func TestBaseExecutor_GetScanSlice(t *testing.T) {
	tests := []struct {
		name        string
		columnNames []string
		tableMeta   *types.TableMeta
		expectedLen int
	}{
		{
			name:        "empty columns",
			columnNames: []string{},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{},
			},
			expectedLen: 0,
		},
		{
			name:        "VARCHAR column",
			columnNames: []string{"name"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"name": {DatabaseTypeString: "VARCHAR"},
				},
			},
			expectedLen: 1,
		},
		{
			name:        "INT column - not nullable",
			columnNames: []string{"age"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"age": {DatabaseTypeString: "INT", IsNullable: 0},
				},
			},
			expectedLen: 1,
		},
		{
			name:        "INT column - nullable",
			columnNames: []string{"age"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"age": {DatabaseTypeString: "INT", IsNullable: 1},
				},
			},
			expectedLen: 1,
		},
		{
			name:        "DATETIME column",
			columnNames: []string{"created_at"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"created_at": {DatabaseTypeString: "DATETIME"},
				},
			},
			expectedLen: 1,
		},
		{
			name:        "DECIMAL column - not nullable",
			columnNames: []string{"price"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"price": {DatabaseTypeString: "DECIMAL", IsNullable: 0},
				},
			},
			expectedLen: 1,
		},
		{
			name:        "DECIMAL column - nullable",
			columnNames: []string{"price"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"price": {DatabaseTypeString: "DECIMAL", IsNullable: 1},
				},
			},
			expectedLen: 1,
		},
		{
			name:        "BLOB column (default case)",
			columnNames: []string{"data"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"data": {DatabaseTypeString: "BLOB"},
				},
			},
			expectedLen: 1,
		},
		{
			name:        "multiple columns of different types",
			columnNames: []string{"id", "name", "age", "created_at", "price"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"id":         {DatabaseTypeString: "BIGINT", IsNullable: 0},
					"name":       {DatabaseTypeString: "VARCHAR"},
					"age":        {DatabaseTypeString: "INT", IsNullable: 1},
					"created_at": {DatabaseTypeString: "TIMESTAMP"},
					"price":      {DatabaseTypeString: "DOUBLE", IsNullable: 1},
				},
			},
			expectedLen: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &baseExecutor{}
			scanSlice := executor.GetScanSlice(tt.columnNames, tt.tableMeta)

			assert.Equal(t, tt.expectedLen, len(scanSlice), "scan slice length should match")

			// Verify that each element is a pointer to the correct type
			for i, colName := range tt.columnNames {
				colMeta := tt.tableMeta.Columns[colName]
				switch colMeta.DatabaseTypeString {
				case "VARCHAR", "NVARCHAR", "VARCHAR2", "CHAR", "TEXT", "JSON", "TINYTEXT":
					_, ok := scanSlice[i].(*sql.NullString)
					assert.True(t, ok, "should be *sql.NullString for %s", colName)
				case "BIT", "INT", "LONGBLOB", "SMALLINT", "TINYINT", "BIGINT", "MEDIUMINT":
					if colMeta.IsNullable == 0 {
						_, ok := scanSlice[i].(*int64)
						assert.True(t, ok, "should be *int64 for non-nullable %s", colName)
					} else {
						_, ok := scanSlice[i].(*sql.NullInt64)
						assert.True(t, ok, "should be *sql.NullInt64 for nullable %s", colName)
					}
				case "DATE", "DATETIME", "TIME", "TIMESTAMP", "YEAR":
					_, ok := scanSlice[i].(*sql.NullTime)
					assert.True(t, ok, "should be *sql.NullTime for %s", colName)
				case "DECIMAL", "DOUBLE", "FLOAT":
					if colMeta.IsNullable == 0 {
						_, ok := scanSlice[i].(*float64)
						assert.True(t, ok, "should be *float64 for non-nullable %s", colName)
					} else {
						_, ok := scanSlice[i].(*sql.NullFloat64)
						assert.True(t, ok, "should be *sql.NullFloat64 for nullable %s", colName)
					}
				default:
					_, ok := scanSlice[i].(*sql.RawBytes)
					assert.True(t, ok, "should be *sql.RawBytes for %s", colName)
				}
			}
		})
	}
}

// TestBaseExecutor_ContainsPKByName tests the containsPKByName method
func TestBaseExecutor_ContainsPKByName(t *testing.T) {
	tests := []struct {
		name      string
		tableMeta *types.TableMeta
		columns   []string
		expected  bool
	}{
		{
			name: "contains single PK",
			tableMeta: &types.TableMeta{
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "id"},
						},
					},
				},
			},
			columns:  []string{"id", "name"},
			expected: true,
		},
		{
			name: "does not contain PK",
			tableMeta: &types.TableMeta{
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "id"},
						},
					},
				},
			},
			columns:  []string{"name", "age"},
			expected: false,
		},
		{
			name: "contains composite PK",
			tableMeta: &types.TableMeta{
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "tenant_id"},
							{ColumnName: "user_id"},
						},
					},
				},
			},
			columns:  []string{"tenant_id", "user_id", "name"},
			expected: true,
		},
		{
			name: "partial composite PK",
			tableMeta: &types.TableMeta{
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "tenant_id"},
							{ColumnName: "user_id"},
						},
					},
				},
			},
			columns:  []string{"tenant_id", "name"},
			expected: false,
		},
		{
			name: "no primary key",
			tableMeta: &types.TableMeta{
				Indexs: map[string]types.IndexMeta{},
			},
			columns:  []string{"id", "name"},
			expected: false,
		},
		{
			name: "case insensitive match",
			tableMeta: &types.TableMeta{
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "ID"},
						},
					},
				},
			},
			columns:  []string{"id", "name"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &baseExecutor{}
			result := executor.containsPKByName(tt.tableMeta, tt.columns)
			assert.Equal(t, tt.expected, result, "containsPKByName result should match")
		})
	}
}

// TestGetSqlNullValue tests the getSqlNullValue function
func TestGetSqlNullValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "valid NullString",
			input:    sql.NullString{String: "test", Valid: true},
			expected: "test",
		},
		{
			name:     "invalid NullString",
			input:    sql.NullString{String: "test", Valid: false},
			expected: nil,
		},
		{
			name:     "valid NullInt64",
			input:    sql.NullInt64{Int64: 42, Valid: true},
			expected: int64(42),
		},
		{
			name:     "invalid NullInt64",
			input:    sql.NullInt64{Int64: 42, Valid: false},
			expected: nil,
		},
		{
			name:     "valid NullFloat64",
			input:    sql.NullFloat64{Float64: 3.14, Valid: true},
			expected: 3.14,
		},
		{
			name:     "invalid NullFloat64",
			input:    sql.NullFloat64{Float64: 3.14, Valid: false},
			expected: nil,
		},
		{
			name:     "valid NullBool",
			input:    sql.NullBool{Bool: true, Valid: true},
			expected: true,
		},
		{
			name:     "invalid NullBool",
			input:    sql.NullBool{Bool: true, Valid: false},
			expected: nil,
		},
		{
			name:     "regular string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "regular int",
			input:    42,
			expected: 42,
		},
		{
			name:     "regular float",
			input:    3.14,
			expected: 3.14,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSqlNullValue(tt.input)
			assert.Equal(t, tt.expected, result, "getSqlNullValue result should match")
		})
	}
}

// TestBaseExecutor_BuildWhereConditionByPKs tests the buildWhereConditionByPKs method
func TestBaseExecutor_BuildWhereConditionByPKs(t *testing.T) {
	tests := []struct {
		name       string
		pkNameList []string
		rowSize    int
		dbType     string
		maxInSize  int
		expected   string
	}{
		{
			name:       "single PK, single row",
			pkNameList: []string{"id"},
			rowSize:    1,
			dbType:     "mysql",
			maxInSize:  1000,
			expected:   "(`id`) IN ((?))",
		},
		{
			name:       "single PK, multiple rows within maxInSize",
			pkNameList: []string{"id"},
			rowSize:    3,
			dbType:     "mysql",
			maxInSize:  1000,
			expected:   "(`id`) IN ((?),(?),(?))",
		},
		{
			name:       "composite PK, single row",
			pkNameList: []string{"tenant_id", "user_id"},
			rowSize:    1,
			dbType:     "mysql",
			maxInSize:  1000,
			expected:   "(`tenant_id`,`user_id`) IN ((?,?))",
		},
		{
			name:       "composite PK, multiple rows",
			pkNameList: []string{"tenant_id", "user_id"},
			rowSize:    2,
			dbType:     "mysql",
			maxInSize:  1000,
			expected:   "(`tenant_id`,`user_id`) IN ((?,?),(?,?))",
		},
		{
			name:       "single PK, exceeds maxInSize",
			pkNameList: []string{"id"},
			rowSize:    5,
			dbType:     "mysql",
			maxInSize:  2,
			expected:   "(`id`) IN ((?),(?)) OR (`id`) IN ((?),(?)) OR (`id`) IN ((?))",
		},
		{
			name:       "composite PK, exceeds maxInSize",
			pkNameList: []string{"tenant_id", "user_id"},
			rowSize:    5,
			dbType:     "mysql",
			maxInSize:  3,
			expected:   "(`tenant_id`,`user_id`) IN ((?,?),(?,?),(?,?)) OR (`tenant_id`,`user_id`) IN ((?,?),(?,?))",
		},
		{
			name:       "exact multiple of maxInSize",
			pkNameList: []string{"id"},
			rowSize:    4,
			dbType:     "mysql",
			maxInSize:  2,
			expected:   "(`id`) IN ((?),(?)) OR (`id`) IN ((?),(?))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &baseExecutor{}
			result := executor.buildWhereConditionByPKs(tt.pkNameList, tt.rowSize, tt.dbType, tt.maxInSize)
			assert.Equal(t, tt.expected, result, "buildWhereConditionByPKs result should match")
		})
	}
}

// TestBaseExecutor_BuildPKParams tests the buildPKParams method
func TestBaseExecutor_BuildPKParams(t *testing.T) {
	tests := []struct {
		name         string
		rows         []types.RowImage
		pkNameList   []string
		expectedLen  int
		expectedVals []interface{}
	}{
		{
			name: "single PK, single row",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: int64(1)},
						{ColumnName: "name", Value: "Alice"},
					},
				},
			},
			pkNameList:   []string{"id"},
			expectedLen:  1,
			expectedVals: []interface{}{int64(1)},
		},
		{
			name: "single PK, multiple rows",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: int64(1)},
						{ColumnName: "name", Value: "Alice"},
					},
				},
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: int64(2)},
						{ColumnName: "name", Value: "Bob"},
					},
				},
			},
			pkNameList:   []string{"id"},
			expectedLen:  2,
			expectedVals: []interface{}{int64(1), int64(2)},
		},
		{
			name: "composite PK, single row",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "tenant_id", Value: int64(1)},
						{ColumnName: "user_id", Value: int64(100)},
						{ColumnName: "name", Value: "Alice"},
					},
				},
			},
			pkNameList:   []string{"tenant_id", "user_id"},
			expectedLen:  2,
			expectedVals: []interface{}{int64(1), int64(100)},
		},
		{
			name: "composite PK, multiple rows",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "tenant_id", Value: int64(1)},
						{ColumnName: "user_id", Value: int64(100)},
						{ColumnName: "name", Value: "Alice"},
					},
				},
				{
					Columns: []types.ColumnImage{
						{ColumnName: "tenant_id", Value: int64(1)},
						{ColumnName: "user_id", Value: int64(101)},
						{ColumnName: "name", Value: "Bob"},
					},
				},
			},
			pkNameList:   []string{"tenant_id", "user_id"},
			expectedLen:  4,
			expectedVals: []interface{}{int64(1), int64(100), int64(1), int64(101)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &baseExecutor{}
			params := executor.buildPKParams(tt.rows, tt.pkNameList)

			assert.Equal(t, tt.expectedLen, len(params), "params length should match")

			for i, expectedVal := range tt.expectedVals {
				assert.Equal(t, expectedVal, params[i].Value, "param value should match at index %d", i)
			}
		})
	}
}

// TestBaseExecutor_TraversalArgs tests the traversalArgs method
func TestBaseExecutor_TraversalArgs(t *testing.T) {
	tests := []struct {
		name         string
		node         ast.Node
		expectedLen  int
		expectedVals []int32
	}{
		{
			name:         "nil node",
			node:         nil,
			expectedLen:  0,
			expectedVals: []int32{},
		},
		{
			name:         "single ParamMarkerExpr",
			node:         &test_driver.ParamMarkerExpr{Order: 0},
			expectedLen:  1,
			expectedVals: []int32{0},
		},
		{
			name: "BinaryOperationExpr with params",
			node: &ast.BinaryOperationExpr{
				Op: opcode.EQ,
				L:  &test_driver.ParamMarkerExpr{Order: 0},
				R:  &test_driver.ParamMarkerExpr{Order: 1},
			},
			expectedLen:  2,
			expectedVals: []int32{0, 1},
		},
		{
			name: "BetweenExpr with params",
			node: &ast.BetweenExpr{
				Expr:  &ast.ColumnNameExpr{},
				Left:  &test_driver.ParamMarkerExpr{Order: 0},
				Right: &test_driver.ParamMarkerExpr{Order: 1},
			},
			expectedLen:  2,
			expectedVals: []int32{0, 1},
		},
		{
			name: "PatternInExpr with multiple params",
			node: &ast.PatternInExpr{
				List: []ast.ExprNode{
					&test_driver.ParamMarkerExpr{Order: 0},
					&test_driver.ParamMarkerExpr{Order: 1},
					&test_driver.ParamMarkerExpr{Order: 2},
				},
			},
			expectedLen:  3,
			expectedVals: []int32{0, 1, 2},
		},
		{
			name: "UnaryOperationExpr with param",
			node: &ast.UnaryOperationExpr{
				Op: opcode.Not,
				V:  &test_driver.ParamMarkerExpr{Order: 0},
			},
			expectedLen:  1,
			expectedVals: []int32{0},
		},
		{
			name: "IsNullExpr with param",
			node: &ast.IsNullExpr{
				Expr: &test_driver.ParamMarkerExpr{Order: 0},
			},
			expectedLen:  1,
			expectedVals: []int32{0},
		},
		{
			name: "PatternLikeExpr with params",
			node: &ast.PatternLikeExpr{
				Expr:    &test_driver.ParamMarkerExpr{Order: 0},
				Pattern: &test_driver.ParamMarkerExpr{Order: 1},
			},
			expectedLen:  2,
			expectedVals: []int32{0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &baseExecutor{}
			argsIndex := make([]int32, 0)
			executor.traversalArgs(tt.node, &argsIndex)

			assert.Equal(t, tt.expectedLen, len(argsIndex), "argsIndex length should match")
			assert.Equal(t, tt.expectedVals, argsIndex, "argsIndex values should match")
		})
	}
}
