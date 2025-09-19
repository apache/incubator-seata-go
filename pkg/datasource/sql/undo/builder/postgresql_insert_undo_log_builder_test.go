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

func TestPostgreSQLInsertUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := GetPostgreSQLInsertUndoLogBuilder()
	executorType := builder.GetExecutorType()
	
	if executorType != types.InsertExecutor {
		t.Errorf("Expected InsertExecutor, got %v", executorType)
	}
}

func TestPostgreSQLInsertUndoLogBuilder_buildSelectSQLByPKValues_SinglePK(t *testing.T) {
	builder := &PostgreSQLInsertUndoLogBuilder{}
	
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

func TestPostgreSQLInsertUndoLogBuilder_buildSelectSQLByPKValues_CompositePK(t *testing.T) {
	builder := &PostgreSQLInsertUndoLogBuilder{}
	
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

func TestPostgreSQLInsertUndoLogBuilder_buildPostgreSQLPlaceholders(t *testing.T) {
	builder := &PostgreSQLInsertUndoLogBuilder{}
	
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

func TestPostgreSQLInsertUndoLogBuilder_buildSelectSQLByPKValues_EmptyValues(t *testing.T) {
	builder := &PostgreSQLInsertUndoLogBuilder{}
	
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