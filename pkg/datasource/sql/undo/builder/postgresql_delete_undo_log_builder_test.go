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

func TestPostgreSQLDeleteUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := GetPostgreSQLDeleteUndoLogBuilder()
	executorType := builder.GetExecutorType()
	
	if executorType != types.DeleteExecutor {
		t.Errorf("Expected DeleteExecutor, got %v", executorType)
	}
}

func TestPostgreSQLDeleteUndoLogBuilder_convertToPostgreSQLSyntax(t *testing.T) {
	builder := &PostgreSQLDeleteUndoLogBuilder{}
	
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
			name:     "complex SQL with backticks",
			input:    "SELECT * FROM `users` u WHERE u.`id` IN (SELECT `user_id` FROM `orders` WHERE `status` = 'active')",
			expected: `SELECT * FROM "users" u WHERE u."id" IN (SELECT "user_id" FROM "orders" WHERE "status" = 'active')`,
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

func TestPostgreSQLDeleteUndoLogBuilder_convertToPostgreSQLParams(t *testing.T) {
	builder := &PostgreSQLDeleteUndoLogBuilder{}
	
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
			name:     "complex WHERE clause",
			input:    "id BETWEEN ? AND ? OR (name = ? AND age IN (?, ?, ?))",
			expected: "id BETWEEN $1 AND $2 OR (name = $3 AND age IN ($4, $5, $6))",
		},
		{
			name:     "nested conditions",
			input:    "(status = ? OR priority = ?) AND created_at > ?",
			expected: "(status = $1 OR priority = $2) AND created_at > $3",
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

func TestPostgreSQLDeleteUndoLogBuilder_buildBeforeImageSQL_SimpleDelete(t *testing.T) {
	builder := &PostgreSQLDeleteUndoLogBuilder{}
	
	query := "DELETE FROM users WHERE id = $1"
	args := []driver.Value{123}

	sql, resultArgs, err := builder.buildBeforeImageSQL(query, args)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSQL := `SELECT * FROM "users" WHERE id = $1`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(resultArgs) != len(args) {
		t.Errorf("Expected %d args, got %d", len(args), len(resultArgs))
	}

	for i, arg := range resultArgs {
		if arg != args[i] {
			t.Errorf("Expected arg[%d] = %v, got %v", i, args[i], arg)
		}
	}
}

func TestPostgreSQLDeleteUndoLogBuilder_buildBeforeImageSQL_ComplexDelete(t *testing.T) {
	builder := &PostgreSQLDeleteUndoLogBuilder{}
	
	query := "DELETE FROM orders WHERE user_id = $1 AND status = $2 AND created_at < $3"
	args := []driver.Value{456, "pending", "2023-01-01"}

	sql, resultArgs, err := builder.buildBeforeImageSQL(query, args)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSQL := `SELECT * FROM "orders" WHERE ((user_id = $1) AND (status = $2)) AND (created_at < $3)`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(resultArgs) != len(args) {
		t.Errorf("Expected %d args, got %d", len(args), len(resultArgs))
	}

	for i, arg := range resultArgs {
		if arg != args[i] {
			t.Errorf("Expected arg[%d] = %v, got %v", i, args[i], arg)
		}
	}
}

func TestPostgreSQLDeleteUndoLogBuilder_buildBeforeImageSQL_WithOrderBy(t *testing.T) {
	builder := &PostgreSQLDeleteUndoLogBuilder{}
	
	query := "DELETE FROM products WHERE category = $1 ORDER BY created_at ASC LIMIT 10"
	args := []driver.Value{"electronics"}

	sql, resultArgs, err := builder.buildBeforeImageSQL(query, args)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSQL := `SELECT * FROM "products" WHERE category = $1 ORDER BY created_at ASC LIMIT 10`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(resultArgs) != len(args) {
		t.Errorf("Expected %d args, got %d", len(args), len(resultArgs))
	}
}

func TestPostgreSQLDeleteUndoLogBuilder_AfterImage(t *testing.T) {
	builder := &PostgreSQLDeleteUndoLogBuilder{}
	
	beforeImages := []*types.RecordImage{
		{
			TableName: "users",
			Rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "John"},
					},
				},
			},
		},
	}

	afterImages, err := builder.AfterImage(nil, nil, beforeImages)
	
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(afterImages) != 0 {
		t.Errorf("Expected empty after images for DELETE operation, got %d images", len(afterImages))
	}
}