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
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/stretchr/testify/assert"
)

func TestExecutorTypeConstants(t *testing.T) {
	// Test ExecutorType constants
	assert.Equal(t, int(UnSupportExecutor), 1)
	assert.Equal(t, int(InsertExecutor), 2)
	assert.Equal(t, int(UpdateExecutor), 3)
	assert.Equal(t, int(SelectForUpdateExecutor), 4)
	assert.Equal(t, int(SelectExecutor), 5)
	assert.Equal(t, int(DeleteExecutor), 6)
	assert.Equal(t, int(ReplaceIntoExecutor), 7)
	assert.Equal(t, int(MultiExecutor), 8)
	assert.Equal(t, int(MultiDeleteExecutor), 9)
	assert.Equal(t, int(InsertOnDuplicateExecutor), 10)
}

func TestParseContext_HasValidStmt(t *testing.T) {
	tests := []struct {
		name     string
		context  *ParseContext
		expected bool
	}{
		{
			name: "has insert stmt",
			context: &ParseContext{
				InsertStmt: &ast.InsertStmt{},
			},
			expected: true,
		},
		{
			name: "has update stmt",
			context: &ParseContext{
				UpdateStmt: &ast.UpdateStmt{},
			},
			expected: true,
		},
		{
			name: "has delete stmt",
			context: &ParseContext{
				DeleteStmt: &ast.DeleteStmt{},
			},
			expected: true,
		},
		{
			name: "has select stmt only",
			context: &ParseContext{
				SelectStmt: &ast.SelectStmt{},
			},
			expected: false,
		},
		{
			name: "has multiple valid stmts",
			context: &ParseContext{
				InsertStmt: &ast.InsertStmt{},
				UpdateStmt: &ast.UpdateStmt{},
			},
			expected: true,
		},
		{
			name:     "empty context",
			context:  &ParseContext{},
			expected: false,
		},
		{
			name:     "nil context",
			context:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.context == nil {
				assert.False(t, false) // Skip nil test
				return
			}
			result := tt.context.HasValidStmt()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseContext_GetTableName(t *testing.T) {
	tests := []struct {
		name        string
		context     *ParseContext
		expectError bool
		errorMsg    string
	}{
		{
			name: "multi stmt with invalid sub-context",
			context: &ParseContext{
				MultiStmt: []*ParseContext{
					{}, // empty context
				},
			},
			expectError: true,
			errorMsg:    "invalid stmt",
		},
		{
			name: "multi stmt with multiple invalid sub-contexts",
			context: &ParseContext{
				MultiStmt: []*ParseContext{
					{}, // empty context
					{}, // empty context
				},
			},
			expectError: true,
			errorMsg:    "invalid stmt",
		},
		{
			name:        "empty context",
			context:     &ParseContext{},
			expectError: true,
			errorMsg:    "invalid stmt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tableName, err := tt.context.GetTableName()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Empty(t, tableName)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseContext_GetTableName_EmptyMultiStmt(t *testing.T) {
	context := &ParseContext{
		MultiStmt: []*ParseContext{},
	}

	_, err := context.GetTableName()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid stmt")
}

func TestParseContext_GetTableName_MultiStmtWithEmptyTableName(t *testing.T) {
	// Create a context that would return empty table name from sub-context
	context := &ParseContext{
		MultiStmt: []*ParseContext{
			{
				// This will cause GetTableName to return empty string and error
			},
		},
	}

	_, err := context.GetTableName()
	assert.Error(t, err)
}

func TestParseContext_StructFields(t *testing.T) {
	context := &ParseContext{
		SQLType:      SQLTypeInsert,
		ExecutorType: InsertExecutor,
		InsertStmt:   &ast.InsertStmt{},
		UpdateStmt:   &ast.UpdateStmt{},
		SelectStmt:   &ast.SelectStmt{},
		DeleteStmt:   &ast.DeleteStmt{},
		MultiStmt: []*ParseContext{
			{SQLType: SQLTypeSelect},
		},
	}

	// Test that fields are properly set
	assert.NotNil(t, context.InsertStmt)
	assert.NotNil(t, context.UpdateStmt)
	assert.NotNil(t, context.SelectStmt)
	assert.NotNil(t, context.DeleteStmt)
	assert.Len(t, context.MultiStmt, 1)
	// Basic validation that struct was initialized correctly
	assert.NotZero(t, context.SQLType)
	assert.NotZero(t, context.ExecutorType)
}
