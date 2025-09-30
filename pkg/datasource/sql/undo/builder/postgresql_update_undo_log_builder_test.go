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

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestPostgreSQLUpdateUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := GetPostgreSQLUpdateUndoLogBuilder()
	executorType := builder.GetExecutorType()

	if executorType != types.UpdateExecutor {
		t.Errorf("Expected UpdateExecutor, got %v", executorType)
	}
}

func TestPostgreSQLUpdateUndoLogBuilder_convertToPostgreSQLSyntax(t *testing.T) {
	builder := &PostgreSQLUpdateUndoLogBuilder{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "convert backticks to double quotes",
			input:    "SELECT * FROM `users` WHERE `id` = ? AND `name` = ?",
			expected: `SELECT * FROM "users" WHERE "id" = ? AND "name" = ?`,
		},
		{
			name:     "no backticks",
			input:    "SELECT * FROM users WHERE id = ?",
			expected: "SELECT * FROM users WHERE id = ?",
		},
		{
			name:     "mixed backticks",
			input:    "SELECT `id`, name FROM `users` WHERE `status` = 'active'",
			expected: `SELECT "id", name FROM "users" WHERE "status" = 'active'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.convertToPostgreSQLSyntax(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestPostgreSQLUpdateUndoLogBuilder_convertToPostgreSQLParams(t *testing.T) {
	builder := &PostgreSQLUpdateUndoLogBuilder{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single parameter",
			input:    "id = ?",
			expected: "id = $1",
		},
		{
			name:     "multiple parameters",
			input:    "id = ? AND name = ? AND age > ?",
			expected: "id = $1 AND name = $2 AND age > $3",
		},
		{
			name:     "no parameters",
			input:    "status = 'active'",
			expected: "status = 'active'",
		},
		{
			name:     "complex condition",
			input:    "id BETWEEN ? AND ? OR name IN (?, ?, ?)",
			expected: "id BETWEEN $1 AND $2 OR name IN ($3, $4, $5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.convertToPostgreSQLParams(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestPostgreSQLUpdateUndoLogBuilder_buildAfterImageSQL(t *testing.T) {
	builder := &PostgreSQLUpdateUndoLogBuilder{}

	beforeImage := &types.RecordImage{
		TableName: "users",
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 1},
					{ColumnName: "name", Value: "John"},
				},
			},
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 2},
					{ColumnName: "name", Value: "Jane"},
				},
			},
		},
	}

	metaData := &types.TableMeta{
		TableName: "users",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id"},
			"name": {ColumnName: "name"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	sql, args, err := builder.buildAfterImageSQL(beforeImage, metaData)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSQL := `SELECT * FROM "users" WHERE ("id" = $1) OR ("id" = $2)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	expectedArgs := []driver.Value{1, 2}
	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}

	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("Expected arg[%d] = %v, got %v", i, expectedArgs[i], arg)
		}
	}
}

func TestPostgreSQLUpdateUndoLogBuilder_buildAfterImageSQL_CompositePK(t *testing.T) {
	builder := &PostgreSQLUpdateUndoLogBuilder{}

	beforeImage := &types.RecordImage{
		TableName: "user_roles",
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "user_id", Value: 1},
					{ColumnName: "role_id", Value: 10},
				},
			},
		},
	}

	metaData := &types.TableMeta{
		TableName: "user_roles",
		Columns: map[string]types.ColumnMeta{
			"user_id": {ColumnName: "user_id"},
			"role_id": {ColumnName: "role_id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "user_id"},
					{ColumnName: "role_id"},
				},
			},
		},
	}

	sql, args, err := builder.buildAfterImageSQL(beforeImage, metaData)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSQL := `SELECT * FROM "user_roles" WHERE ("user_id" = $1 AND "role_id" = $2)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	expectedArgs := []driver.Value{1, 10}
	if len(args) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(args))
	}

	for i, arg := range args {
		if arg != expectedArgs[i] {
			t.Errorf("Expected arg[%d] = %v, got %v", i, expectedArgs[i], arg)
		}
	}
}

func TestPostgreSQLUpdateUndoLogBuilder_buildAfterImageSQL_EmptyRows(t *testing.T) {
	builder := &PostgreSQLUpdateUndoLogBuilder{}

	beforeImage := &types.RecordImage{
		TableName: "users",
		Rows:      []types.RowImage{},
	}

	metaData := &types.TableMeta{
		TableName: "users",
	}

	sql, args, err := builder.buildAfterImageSQL(beforeImage, metaData)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sql != "" || len(args) != 0 {
		t.Errorf("Expected empty SQL and args for empty rows, got sql=%s, args=%v", sql, args)
	}
}
