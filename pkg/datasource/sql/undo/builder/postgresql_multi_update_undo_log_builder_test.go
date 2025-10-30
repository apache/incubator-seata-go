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

func TestPostgreSQLMultiUpdateUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := GetPostgreSQLMultiUpdateUndoLogBuilder()
	executorType := builder.GetExecutorType()
	
	if executorType != types.UpdateExecutor {
		t.Errorf("Expected UpdateExecutor, got %v", executorType)
	}
}

func TestPostgreSQLMultiUpdateUndoLogBuilder_convertToPostgreSQLSyntax(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
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
			name:     "complex multi-table update",
			input:    "SELECT * FROM `users` u JOIN `orders` o ON u.`id` = o.`user_id` WHERE o.`status` = 'processing'",
			expected: `SELECT * FROM "users" u JOIN "orders" o ON u."id" = o."user_id" WHERE o."status" = 'processing'`,
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_convertToPostgreSQLParams(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
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
			input:    "user_id = ? AND status = ? AND updated_at > ?",
			expected: "user_id = $1 AND status = $2 AND updated_at > $3",
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_buildBeforeImageSQL_ValidQueries(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
	tests := []struct {
		name        string
		query       string
		args        []driver.Value
		expectError bool
	}{
		{
			name:        "PostgreSQL simple update",
			query:       "UPDATE users SET name = $1 WHERE id = $2",
			args:        []driver.Value{"John", 123},
			expectError: false,
		},
		{
			name:        "PostgreSQL update with conditions",
			query:       "UPDATE orders SET status = $1 WHERE user_id = $2 AND amount > $3",
			args:        []driver.Value{"processed", 123, 100.50},
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_splitStatements(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
	tests := []struct {
		name     string
		query    string
		expected []string
	}{
		{
			name:     "single update statement",
			query:    "UPDATE users SET name = 'John' WHERE id = 1",
			expected: []string{"UPDATE users SET name = 'John' WHERE id = 1"},
		},
		{
			name:     "multiple update statements",
			query:    "UPDATE users SET name = 'John' WHERE id = 1; UPDATE orders SET status = 'processed' WHERE user_id = 1;",
			expected: []string{"UPDATE users SET name = 'John' WHERE id = 1", "UPDATE orders SET status = 'processed' WHERE user_id = 1"},
		},
		{
			name:     "mixed statements with non-update",
			query:    "UPDATE users SET name = 'John' WHERE id = 1; SELECT * FROM orders; UPDATE logs SET processed = true WHERE user_id = 1;",
			expected: []string{"UPDATE users SET name = 'John' WHERE id = 1", "UPDATE logs SET processed = true WHERE user_id = 1"},
		},
		{
			name:     "empty query",
			query:    "",
			expected: []string{},
		},
		{
			name:     "query with extra whitespace",
			query:    "  UPDATE users SET name = 'John' WHERE id = 1  ;  UPDATE orders SET status = 'processed' WHERE user_id = 1  ;  ",
			expected: []string{"UPDATE users SET name = 'John' WHERE id = 1", "UPDATE orders SET status = 'processed' WHERE user_id = 1"},
		},
		{
			name:     "short statements",
			query:    "UPD; UPDATE users SET name = 'John';",
			expected: []string{"UPDATE users SET name = 'John'"},
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_splitStatementsAdvanced(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
	tests := []struct {
		name        string
		query       string
		expected    []string
		expectError bool
	}{
		{
			name:     "statements with quoted semicolons",
			query:    "UPDATE users SET col = 'a;b' WHERE id = 1; UPDATE orders SET note = 'test;data' WHERE id = 2;",
			expected: []string{"UPDATE users SET col = 'a;b' WHERE id = 1", "UPDATE orders SET note = 'test;data' WHERE id = 2"},
		},
		{
			name:     "statements with comments containing semicolons",
			query:    "UPDATE users SET name = 'John' WHERE id = 1 -- comment; with semicolon\n; UPDATE orders SET status = 'done';",
			expected: []string{"UPDATE users SET name = 'John' WHERE id = 1 -- comment; with semicolon", "UPDATE orders SET status = 'done'"},
		},
		{
			name:     "statements with block comments",
			query:    "UPDATE users SET name = 'John' /* comment; with semicolon */ WHERE id = 1; UPDATE orders SET status = 'done';",
			expected: []string{"UPDATE users SET name = 'John' /* comment; with semicolon */ WHERE id = 1", "UPDATE orders SET status = 'done'"},
		},
		{
			name:     "statements with escaped quotes",
			query:    "UPDATE users SET col = 'Don''t; stop' WHERE id = 1; UPDATE orders SET note = 'test' WHERE id = 2;",
			expected: []string{"UPDATE users SET col = 'Don''t; stop' WHERE id = 1", "UPDATE orders SET note = 'test' WHERE id = 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.splitStatementsAdvanced(tt.query)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_extractArgsForStatement(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
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
			stmt:           "UPDATE users SET name = ? WHERE id = ?",
			allArgs:        []driver.Value{"john", 1, "jane", 2},
			offset:         0,
			expectedArgs:   []driver.Value{"john", 1},
			expectedOffset: 2,
		},
		{
			name:           "statement with $ parameters",
			stmt:           "UPDATE users SET name = $1 WHERE id = $2",
			allArgs:        []driver.Value{"john", 1, "jane", 2},
			offset:         0,
			expectedArgs:   []driver.Value{"john", 1},
			expectedOffset: 2,
		},
		{
			name:           "statement with offset",
			stmt:           "UPDATE orders SET status = ?",
			allArgs:        []driver.Value{"john", 1, "processed", 3},
			offset:         2,
			expectedArgs:   []driver.Value{"processed"},
			expectedOffset: 3,
		},
		{
			name:           "no parameters",
			stmt:           "UPDATE users SET active = true",
			allArgs:        []driver.Value{1, 2, 3},
			offset:         0,
			expectedArgs:   []driver.Value{},
			expectedOffset: 0,
		},
		{
			name:           "insufficient args",
			stmt:           "UPDATE users SET name = ? WHERE id = ? AND age = ?",
			allArgs:        []driver.Value{"john", 1},
			offset:         0,
			expectedArgs:   []driver.Value{"john", 1},
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_extractArgsForStatementAdvanced(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
	tests := []struct {
		name           string
		stmt           string
		allArgs        []driver.Value
		offset         int
		expectedArgs   []driver.Value
		expectedOffset int
		expectError    bool
	}{
		{
			name:           "statement with quoted parameters should ignore them",
			stmt:           "UPDATE users SET col = '$1' WHERE id = ?",
			allArgs:        []driver.Value{"value", 123},
			offset:         0,
			expectedArgs:   []driver.Value{"value"},
			expectedOffset: 1,
		},
		{
			name:           "statement with comment containing parameters",
			stmt:           "UPDATE users SET name = ? WHERE id = ? -- comment with ?",
			allArgs:        []driver.Value{"john", 1, "extra"},
			offset:         0,
			expectedArgs:   []driver.Value{"john", 1},
			expectedOffset: 2,
		},
		{
			name:           "mixed $ parameters",
			stmt:           "UPDATE users SET name = $1, age = $2 WHERE id = $3",
			allArgs:        []driver.Value{"john", 25, 1},
			offset:         0,
			expectedArgs:   []driver.Value{"john", 25, 1},
			expectedOffset: 3,
		},
		{
			name:           "escaped quotes with parameters",
			stmt:           "UPDATE users SET col = 'Don''t use $1' WHERE id = ?",
			allArgs:        []driver.Value{123},
			offset:         0,
			expectedArgs:   []driver.Value{123},
			expectedOffset: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, newOffset, err := builder.extractArgsForStatementAdvanced(tt.stmt, tt.allArgs, tt.offset)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_parseTableName(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_MultiStatement_Integration(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
	tests := []struct {
		name               string
		query              string
		args               []driver.Value
		expectedStatements int
		expectedTables     []string
	}{
		{
			name:               "multiple update statements",
			query:              "UPDATE users SET name = $1 WHERE id = $2; UPDATE orders SET status = $3 WHERE user_id = $4;",
			args:               []driver.Value{"John", 123, "processed", 456},
			expectedStatements: 2,
			expectedTables:     []string{"users", "orders"},
		},
		{
			name:               "schema qualified tables",
			query:              "UPDATE public.users SET name = $1 WHERE id = $2; UPDATE sales.orders SET status = $3 WHERE id = $4;",
			args:               []driver.Value{"John", 123, "shipped", 456},
			expectedStatements: 2,
			expectedTables:     []string{"public.users", "sales.orders"},
		},
		{
			name:               "mixed statements with non-update",
			query:              "UPDATE users SET name = $1 WHERE id = $2; SELECT * FROM orders; UPDATE logs SET processed = $3 WHERE user_id = $4;",
			args:               []driver.Value{"John", 123, true, 456},
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

func TestPostgreSQLMultiUpdateUndoLogBuilder_ParameterDistribution(t *testing.T) {
	builder := &PostgreSQLMultiUpdateUndoLogBuilder{}
	
	query := "UPDATE users SET name = $1, status = $2 WHERE id = $3; UPDATE orders SET amount = $4 WHERE user_id = $5; UPDATE logs SET processed = $6;"
	args := []driver.Value{"John", "active", 123, 100.50, 456, true}
	
	statements := builder.splitStatements(query)
	if len(statements) != 3 {
		t.Fatalf("Expected 3 statements, got %d", len(statements))
	}
	
	argOffset := 0
	expectedArgCounts := []int{3, 2, 1}
	
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