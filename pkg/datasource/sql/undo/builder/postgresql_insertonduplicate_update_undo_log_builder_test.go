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

package builder

import (
	"database/sql/driver"
	"testing"

	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := GetPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder()
	executorType := builder.GetExecutorType()
	
	if executorType != types.InsertOnDuplicateExecutor {
		t.Errorf("Expected InsertOnDuplicateExecutor, got %v", executorType)
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildSelectSQLByPKValues_SinglePK(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	tableName := "users"
	pkNameList := []string{"id"}
	pkValuesMap := map[string][]driver.Value{
		"id": {1, 2, 3},
	}

	sql, args, err := builder.buildSelectSQLByPKValues(tableName, pkNameList, pkValuesMap)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSQL := `SELECT * FROM "users" WHERE "id" IN ($1, $2, $3)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	expectedArgs := []driver.Value{1, 2, 3}
	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}

	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("Expected arg[%d] = %v, got %v", i, expectedArgs[i], arg)
		}
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildSelectSQLByPKValues_CompositePK(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	tableName := "user_roles"
	pkNameList := []string{"user_id", "role_id"}
	pkValuesMap := map[string][]driver.Value{
		"user_id": {1, 2},
		"role_id": {10, 20},
	}

	sql, args, err := builder.buildSelectSQLByPKValues(tableName, pkNameList, pkValuesMap)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSQL := `SELECT * FROM "user_roles" WHERE ("user_id" = $1 AND "role_id" = $2) OR ("user_id" = $3 AND "role_id" = $4)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	expectedArgs := []driver.Value{1, 10, 2, 20}
	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}

	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("Expected arg[%d] = %v, got %v", i, expectedArgs[i], arg)
		}
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildPostgreSQLPlaceholders(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	tests := []struct {
		name      string
		count     int
		startIndex int
		expected  string
	}{
		{
			name:      "single placeholder",
			count:     1,
			startIndex: 1,
			expected:  "$1",
		},
		{
			name:      "three placeholders",
			count:     3,
			startIndex: 1,
			expected:  "$1, $2, $3",
		},
		{
			name:      "placeholders with offset",
			count:     2,
			startIndex: 5,
			expected:  "$5, $6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := tt.startIndex
			result := builder.buildPostgreSQLPlaceholders(tt.count, &argIndex)
			
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}

			expectedFinalIndex := tt.startIndex + tt.count
			if argIndex != expectedFinalIndex {
				t.Errorf("Expected final argIndex %d, got %d", expectedFinalIndex, argIndex)
			}
		})
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildSelectSQLByPKValues_EmptyValues(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	tableName := "users"
	pkNameList := []string{"id"}
	pkValuesMap := map[string][]driver.Value{}

	_, _, err := builder.buildSelectSQLByPKValues(tableName, pkNameList, pkValuesMap)
	
	if err == nil {
		t.Error("Expected error for empty primary key values, got nil")
	}

	expectedErrorMsg := "no primary key values found"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_extractPrimaryKeyValues_EmptyStmt(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	execCtx := &types.ExecContext{
		ParseContext: &types.ParseContext{},
	}
	
	metaData := &types.TableMeta{
		TableName: "users",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	result, err := builder.extractPrimaryKeyValues(execCtx, []driver.Value{1}, metaData)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty statement, got %d items", len(result))
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildBeforeImageSQL_NoPK(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	execCtx := &types.ExecContext{
		Values: []driver.Value{1, "test"},
		ParseContext: &types.ParseContext{},
	}
	
	metaData := &types.TableMeta{
		TableName: "users",
		Indexs:    map[string]types.IndexMeta{},
	}

	_, _, err := builder.buildBeforeImageSQL(nil, execCtx, metaData)
	
	if err == nil {
		t.Error("Expected error for table with no primary key, got nil")
	}

	expectedErrorMsg := "no valid insert statement found"
	if err.Error() != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildImageParameters(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	t.Run("no valid insert statement", func(t *testing.T) {
		execCtx := &types.ExecContext{
			Values: []driver.Value{1, "test"},
			ParseContext: &types.ParseContext{},
		}
		metaData := &types.TableMeta{TableName: "users"}
		
		_, err := builder.buildImageParameters(execCtx, []driver.Value{1, "test"}, metaData)
		
		if err == nil {
			t.Error("Expected error for no valid insert statement, got nil")
		}
		
		expectedMsg := "no valid insert statement found"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_isIndexValueNotNull(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	tests := []struct {
		name        string
		index       types.IndexMeta
		paramMap    map[string][]driver.Value
		expected    bool
		description string
	}{
		{
			name: "all values present and not null",
			index: types.IndexMeta{
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
					{ColumnName: "email"},
				},
			},
			paramMap: map[string][]driver.Value{
				"id":    {123},
				"email": {"test@example.com"},
			},
			expected:    true,
			description: "should return true when all index columns have non-null values",
		},
		{
			name: "missing column with no default",
			index: types.IndexMeta{
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
					{ColumnName: "missing_col"},
				},
			},
			paramMap: map[string][]driver.Value{
				"id": {123},
			},
			expected:    false,
			description: "should return false when column is missing and has no default",
		},
		{
			name: "null value in column",
			index: types.IndexMeta{
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
					{ColumnName: "nullable_col"},
				},
			},
			paramMap: map[string][]driver.Value{
				"id":           {123},
				"nullable_col": {nil},
			},
			expected:    false,
			description: "should return false when column has null value",
		},
		{
			name: "empty value slice",
			index: types.IndexMeta{
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
					{ColumnName: "empty_col"},
				},
			},
			paramMap: map[string][]driver.Value{
				"id":        {123},
				"empty_col": {},
			},
			expected:    false,
			description: "should return false when column has empty value slice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.isIndexValueNotNull(tt.index, tt.paramMap)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v. %s", tt.expected, result, tt.description)
			}
		})
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_extractPKFromPostgreSQLInsert(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	metaData := &types.TableMeta{
		TableName: "users",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}
	
	t.Run("no columns in INSERT statement", func(t *testing.T) {
		stmt := &tree.Insert{}
		vals := []driver.Value{1, "test"}
		
		_, err := builder.extractPKFromPostgreSQLInsert(stmt, vals, metaData)
		
		if err == nil {
			t.Error("Expected error for INSERT with no columns, got nil")
		}
		
		expectedMsg := "INSERT statement has no column list"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
	
	t.Run("no primary key columns found", func(t *testing.T) {
		stmt := &tree.Insert{
			Columns: tree.NameList{tree.Name("name"), tree.Name("email")},
		}
		vals := []driver.Value{"John", "john@example.com"}
		
		_, err := builder.extractPKFromPostgreSQLInsert(stmt, vals, metaData)
		
		if err == nil {
			t.Error("Expected error for no PK columns, got nil")
		}
		
		expectedMsg := "no primary key columns found in INSERT statement"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
	
	t.Run("successful primary key extraction", func(t *testing.T) {
		stmt := &tree.Insert{
			Columns: tree.NameList{tree.Name("id"), tree.Name("name")},
		}
		vals := []driver.Value{123, "John"}
		
		result, err := builder.extractPKFromPostgreSQLInsert(stmt, vals, metaData)
		
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if len(result) != 1 {
			t.Errorf("Expected 1 result, got %d", len(result))
		}
		
		if result[0]["id"] != 123 {
			t.Errorf("Expected id=123, got %v", result[0]["id"])
		}
	})
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_getPkValues(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	meta := types.TableMeta{
		TableName: "users",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}
	
	t.Run("with insert result and lastInsertId", func(t *testing.T) {
		mockResult := &MockExecResult{
			lastInsertId: 100,
			rowsAffected: 2,
		}
		
		builder.InsertResult = &MockInsertResult{result: mockResult}
		builder.IncrementStep = 1
		
		execCtx := &types.ExecContext{
			Values: []driver.Value{},
		}
		parseCtx := &types.ParseContext{}
		
		result, err := builder.getPkValues(execCtx, parseCtx, meta)
		
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if len(result["id"]) != 2 {
			t.Errorf("Expected 2 values for id, got %d", len(result["id"]))
		}
		
		if result["id"][0] != int64(100) {
			t.Errorf("Expected first id=100, got %v", result["id"][0])
		}
		
		if result["id"][1] != int64(101) {
			t.Errorf("Expected second id=101, got %v", result["id"][1])
		}
	})
	
	t.Run("no insert result", func(t *testing.T) {
		builder.InsertResult = nil
		
		execCtx := &types.ExecContext{
			Values: []driver.Value{123},
			ParseContext: &types.ParseContext{},
		}
		parseCtx := &types.ParseContext{}
		
		_, err := builder.getPkValues(execCtx, parseCtx, meta)
		
		if err == nil {
			t.Error("Expected error when no insert result available, got nil")
		}
		
		expectedMsg := "no primary key values found for INSERT ON CONFLICT operation"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
}

type MockExecResult struct {
	lastInsertId int64
	rowsAffected int64
}

func (m *MockExecResult) LastInsertId() (int64, error) {
	return m.lastInsertId, nil
}

func (m *MockExecResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

type MockInsertResult struct {
	result *MockExecResult
}

func (m *MockInsertResult) GetRows() driver.Rows {
	return nil
}

func (m *MockInsertResult) GetResult() driver.Result {
	return m.result
}


func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildImageParametersFromPostgreSQL(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	metaData := &types.TableMeta{
		TableName: "users",
	}
	
	t.Run("no columns in statement", func(t *testing.T) {
		stmt := &tree.Insert{}
		vals := []driver.Value{123, "John"}
		
		_, err := builder.buildImageParametersFromPostgreSQL(stmt, vals, metaData)
		
		if err == nil {
			t.Error("Expected error for INSERT with no columns, got nil")
		}
		
		expectedMsg := "INSERT statement has no column list"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
	
	t.Run("successful parameter mapping", func(t *testing.T) {
		stmt := &tree.Insert{
			Columns: tree.NameList{tree.Name("id"), tree.Name("name")},
		}
		vals := []driver.Value{123, "John"}
		
		result, err := builder.buildImageParametersFromPostgreSQL(stmt, vals, metaData)
		
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if len(result) != 2 {
			t.Errorf("Expected 2 columns in result, got %d", len(result))
		}
		
		if result["id"][0] != 123 {
			t.Errorf("Expected id=123, got %v", result["id"][0])
		}
		
		if result["name"][0] != "John" {
			t.Errorf("Expected name=John, got %v", result["name"][0])
		}
	})
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_buildBeforeImageSQL_WithUniqueIndex(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	execCtx := &types.ExecContext{
		Values: []driver.Value{123, "test@example.com"},
		ParseContext: &types.ParseContext{
			AuxtenInsertStmt: &tree.Insert{
				Columns: tree.NameList{tree.Name("id"), tree.Name("email")},
			},
		},
	}
	
	metaData := &types.TableMeta{
		TableName: "users",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType:     types.IndexTypePrimaryKey,
				NonUnique: false,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
			"email_unique": {
				IType:     types.IndexTypePrimaryKey,
				NonUnique: false,
				Columns: []types.ColumnMeta{
					{ColumnName: "email"},
				},
			},
		},
	}
	
	sql, args, err := builder.buildBeforeImageSQL(nil, execCtx, metaData)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	expectedSQL := `SELECT * FROM "users" WHERE ("id" = $1) OR ("email" = $2)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}
	
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
	
	if args[0] != 123 {
		t.Errorf("Expected first arg=123, got %v", args[0])
	}
	
	if args[1] != "test@example.com" {
		t.Errorf("Expected second arg=test@example.com, got %v", args[1])
	}
}

func TestPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder_extractPrimaryKeyValues(t *testing.T) {
	builder := &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{}
	
	metaData := &types.TableMeta{
		TableName: "users",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}
	
	t.Run("with PostgreSQL AST", func(t *testing.T) {
		execCtx := &types.ExecContext{
			ParseContext: &types.ParseContext{
				AuxtenInsertStmt: &tree.Insert{
					Columns: tree.NameList{tree.Name("id"), tree.Name("name")},
				},
			},
		}
		vals := []driver.Value{123, "John"}
		
		result, err := builder.extractPrimaryKeyValues(execCtx, vals, metaData)
		
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if len(result) != 1 {
			t.Errorf("Expected 1 result, got %d", len(result))
		}
		
		if result[0]["id"] != 123 {
			t.Errorf("Expected id=123, got %v", result[0]["id"])
		}
	})
	
	
	t.Run("no valid statement", func(t *testing.T) {
		execCtx := &types.ExecContext{
			ParseContext: &types.ParseContext{},
		}
		vals := []driver.Value{123}
		
		result, err := builder.extractPrimaryKeyValues(execCtx, vals, metaData)
		
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		
		if len(result) != 0 {
			t.Errorf("Expected empty result, got %d items", len(result))
		}
	})
}