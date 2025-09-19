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
	"strings"
	"testing"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestPostgreSQLMultiDeleteUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := GetPostgreSQLMultiDeleteUndoLogBuilder()
	executorType := builder.GetExecutorType()
	
	if executorType != types.MultiDeleteExecutor {
		t.Errorf("Expected MultiDeleteExecutor, got %v", executorType)
	}
}

func TestPostgreSQLMultiDeleteUndoLogBuilder_convertToPostgreSQLSyntax(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "convert backticks to double quotes",
			input:    "SELECT * FROM `users` u, `orders` o WHERE u.`id` = o.`user_id`",
			expected: `SELECT * FROM "users" u, "orders" o WHERE u."id" = o."user_id"`,
		},
		{
			name:     "no backticks",
			input:    "SELECT * FROM users u, orders o WHERE u.id = o.user_id",
			expected: "SELECT * FROM users u, orders o WHERE u.id = o.user_id",
		},
		{
			name:     "complex multi-table delete",
			input:    "SELECT * FROM `users` u JOIN `orders` o ON u.`id` = o.`user_id` WHERE o.`status` = 'cancelled'",
			expected: `SELECT * FROM "users" u JOIN "orders" o ON u."id" = o."user_id" WHERE o."status" = 'cancelled'`,
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

func TestPostgreSQLMultiDeleteUndoLogBuilder_convertToPostgreSQLParams(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single parameter",
			input:    "user_id = ?",
			expected: "user_id = $1",
		},
		{
			name:     "multiple parameters",
			input:    "user_id = ? AND status = ? AND created_at > ?",
			expected: "user_id = $1 AND status = $2 AND created_at > $3",
		},
		{
			name:     "no parameters",
			input:    "status = 'active'",
			expected: "status = 'active'",
		},
		{
			name:     "complex multi-table conditions",
			input:    "u.id = ? AND o.user_id = u.id AND o.status IN (?, ?, ?)",
			expected: "u.id = $1 AND o.user_id = u.id AND o.status IN ($2, $3, $4)",
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

func TestPostgreSQLMultiDeleteUndoLogBuilder_buildBeforeImageSQL_ValidQueries(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	tests := []struct {
		name        string
		query       string
		args        []driver.Value
		expectError bool
	}{
		{
			name:        "PostgreSQL simple delete",
			query:       "DELETE FROM users WHERE id = $1",
			args:        []driver.Value{123},
			expectError: false,
		},
		{
			name:        "PostgreSQL delete with conditions",
			query:       "DELETE FROM orders WHERE user_id = $1 AND status = $2",
			args:        []driver.Value{123, "cancelled"},
			expectError: false,
		},
		{
			name:        "invalid SQL statement",
			query:       "INVALID SQL",
			args:        []driver.Value{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, table, err := builder.buildSingleBeforeImageSQL(tt.query, tt.args)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if sql == "" {
				t.Error("Expected non-empty SQL")
			}
			
			if len(args) != len(tt.args) {
				t.Errorf("Expected %d args, got %d", len(tt.args), len(args))
			}
			
			if table == "" {
				t.Error("Expected table name")
			}
		})
	}
}

func TestPostgreSQLMultiDeleteUndoLogBuilder_AfterImage(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
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
		{
			TableName: "orders",
			Rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 100},
						{ColumnName: "user_id", Value: 1},
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

func TestPostgreSQLMultiDeleteUndoLogBuilder_buildBeforeImageSQL_InvalidStatement(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	invalidSQL := "SELECT * FROM users"
	args := []driver.Value{}
	
	_, _, _, err := builder.buildSingleBeforeImageSQL(invalidSQL, args)
	
	if err == nil {
		t.Error("Expected error for invalid delete statement, got nil")
	}
}

func TestPostgreSQLMultiDeleteUndoLogBuilder_splitStatements(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "single delete statement",
			query:    "DELETE FROM users WHERE id = 1",
			expected: []string{"DELETE FROM users WHERE id = 1"},
		},
		{
			name:     "multiple delete statements",
			query:    "DELETE FROM users WHERE id = 1; DELETE FROM orders WHERE user_id = 1;",
			expected: []string{"DELETE FROM users WHERE id = 1", "DELETE FROM orders WHERE user_id = 1"},
		},
		{
			name:     "mixed statements with non-delete",
			query:    "DELETE FROM users WHERE id = 1; SELECT * FROM orders; DELETE FROM logs WHERE user_id = 1;",
			expected: []string{"DELETE FROM users WHERE id = 1", "DELETE FROM logs WHERE user_id = 1"},
		},
		{
			name:     "empty query",
			query:    "",
			expected: []string{},
		},
		{
			name:     "query with extra whitespace",
			query:    "  DELETE FROM users WHERE id = 1  ;  DELETE FROM orders WHERE user_id = 1  ;  ",
			expected: []string{"DELETE FROM users WHERE id = 1", "DELETE FROM orders WHERE user_id = 1"},
		},
		{
			name:     "short statements",
			query:    "DEL; DELETE FROM users;",
			expected: []string{"DELETE FROM users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := builder.splitStatements(tt.query)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d statements, got %d", len(tt.expected), len(result))
				return
			}
			
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected statement[%d] = '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestPostgreSQLMultiDeleteUndoLogBuilder_extractArgsForStatement(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	tests := []struct {
		name           string
		stmt           string
		allArgs        []driver.Value
		offset         int
		expectedArgs   []driver.Value
		expectedOffset int
	}{
		{
			name:           "statement with ? parameters",
			stmt:           "DELETE FROM users WHERE id = ? AND name = ?",
			allArgs:        []driver.Value{1, "john", 2, "jane"},
			offset:         0,
			expectedArgs:   []driver.Value{1, "john"},
			expectedOffset: 2,
		},
		{
			name:           "statement with $ parameters",
			stmt:           "DELETE FROM users WHERE id = $1 AND name = $2",
			allArgs:        []driver.Value{1, "john", 2, "jane"},
			offset:         0,
			expectedArgs:   []driver.Value{1, "john"},
			expectedOffset: 2,
		},
		{
			name:           "statement with offset",
			stmt:           "DELETE FROM orders WHERE user_id = ?",
			allArgs:        []driver.Value{1, "john", 2, "jane"},
			offset:         2,
			expectedArgs:   []driver.Value{2},
			expectedOffset: 3,
		},
		{
			name:           "no parameters",
			stmt:           "DELETE FROM users WHERE active = true",
			allArgs:        []driver.Value{1, 2, 3},
			offset:         0,
			expectedArgs:   []driver.Value{},
			expectedOffset: 0,
		},
		{
			name:           "insufficient args",
			stmt:           "DELETE FROM users WHERE id = ? AND name = ? AND age = ?",
			allArgs:        []driver.Value{1, "john"},
			offset:         0,
			expectedArgs:   []driver.Value{1, "john"},
			expectedOffset: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, newOffset := builder.extractArgsForStatement(tt.stmt, tt.allArgs, tt.offset)
			
			if newOffset != tt.expectedOffset {
				t.Errorf("Expected offset %d, got %d", tt.expectedOffset, newOffset)
			}
			
			if len(args) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(args))
				return
			}
			
			for i, expected := range tt.expectedArgs {
				if args[i] != expected {
					t.Errorf("Expected arg[%d] = %v, got %v", i, expected, args[i])
				}
			}
		})
	}
}

func TestPostgreSQLMultiDeleteUndoLogBuilder_parseTableName(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	tests := []struct {
		name           string
		tableName      string
		expectedSchema string
		expectedTable  string
	}{
		{
			name:           "table with schema",
			tableName:      "public.users",
			expectedSchema: "public",
			expectedTable:  "users",
		},
		{
			name:           "table without schema",
			tableName:      "users",
			expectedSchema: "",
			expectedTable:  "users",
		},
		{
			name:           "complex schema name",
			tableName:      "my_schema.user_profiles",
			expectedSchema: "my_schema",
			expectedTable:  "user_profiles",
		},
		{
			name:           "quoted identifiers",
			tableName:      `"MySchema"."UserTable"`,
			expectedSchema: `"MySchema"`,
			expectedTable:  `"UserTable"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table := builder.parseTableName(tt.tableName)
			
			if schema != tt.expectedSchema {
				t.Errorf("Expected schema '%s', got '%s'", tt.expectedSchema, schema)
			}
			
			if table != tt.expectedTable {
				t.Errorf("Expected table '%s', got '%s'", tt.expectedTable, table)
			}
		})
	}
}

func TestPostgreSQLMultiDeleteUndoLogBuilder_MultiStatement_Integration(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	tests := []struct {
		name               string
		query              string
		args               []driver.Value
		expectedStatements int
		expectedTables     []string
	}{
		{
			name:               "multiple delete statements",
			query:              "DELETE FROM users WHERE id = $1; DELETE FROM orders WHERE user_id = $2;",
			args:               []driver.Value{123, 456},
			expectedStatements: 2,
			expectedTables:     []string{"users", "orders"},
		},
		{
			name:               "schema qualified tables",
			query:              "DELETE FROM public.users WHERE id = $1; DELETE FROM sales.orders WHERE id = $2;",
			args:               []driver.Value{123, 456},
			expectedStatements: 2,
			expectedTables:     []string{"public.users", "sales.orders"},
		},
		{
			name:               "mixed statements with non-delete",
			query:              "DELETE FROM users WHERE id = $1; SELECT * FROM orders; DELETE FROM logs WHERE user_id = $2;",
			args:               []driver.Value{123, 456},
			expectedStatements: 2,
			expectedTables:     []string{"users", "logs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statements := builder.splitStatements(tt.query)
			
			if len(statements) != tt.expectedStatements {
				t.Errorf("Expected %d statements, got %d", tt.expectedStatements, len(statements))
				return
			}
			
			argOffset := 0
			for i, stmt := range statements {
				stmtArgs, newOffset := builder.extractArgsForStatement(stmt, tt.args, argOffset)
				argOffset = newOffset
				
				if len(stmtArgs) == 0 && strings.Contains(stmt, "$") {
					t.Errorf("Statement %d should have args but got none: %s", i, stmt)
				}
				
				expectedTable := tt.expectedTables[i]
				if strings.Contains(stmt, expectedTable) {
					if strings.Contains(expectedTable, ".") {
						schema, table := builder.parseTableName(expectedTable)
						if schema == "" || table == "" {
							t.Errorf("Failed to parse schema.table: %s", expectedTable)
						}
					}
				}
			}
		})
	}
}

func TestPostgreSQLMultiDeleteUndoLogBuilder_ParameterDistribution(t *testing.T) {
	builder := &PostgreSQLMultiDeleteUndoLogBuilder{}
	
	query := "DELETE FROM users WHERE id = $1 AND status = $2; DELETE FROM orders WHERE user_id = $3 AND amount > $4; DELETE FROM logs WHERE user_id = $5;"
	args := []driver.Value{123, "active", 456, 100.50, 789}
	
	statements := builder.splitStatements(query)
	if len(statements) != 3 {
		t.Fatalf("Expected 3 statements, got %d", len(statements))
	}
	
	argOffset := 0
	expectedArgCounts := []int{2, 2, 1}
	
	for i, stmt := range statements {
		stmtArgs, newOffset := builder.extractArgsForStatement(stmt, args, argOffset)
		
		if len(stmtArgs) != expectedArgCounts[i] {
			t.Errorf("Statement %d: expected %d args, got %d", i, expectedArgCounts[i], len(stmtArgs))
		}
		
		if newOffset != argOffset+expectedArgCounts[i] {
			t.Errorf("Statement %d: expected offset %d, got %d", i, argOffset+expectedArgCounts[i], newOffset)
		}
		
		argOffset = newOffset
	}
	
	if argOffset != len(args) {
		t.Errorf("Expected total offset %d, got %d", len(args), argOffset)
	}
}