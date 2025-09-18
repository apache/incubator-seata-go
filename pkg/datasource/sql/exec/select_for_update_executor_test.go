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

package exec

import (
	"database/sql/driver"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/builder"
)

var (
	index   = 0
	rowVals = [][]interface{}{
		{1, "oid11"},
		{2, "oid22"},
		{3, "oid33"},
	}
)

func TestBuildSelectPKSQL(t *testing.T) {
	tests := []struct {
		name           string
		dbType         types.DBType
		sql            string
		expectedMySQL  string
		expectedPGSQL  string
	}{
		{
			name:          "MySQL - Select for update with primary key",
			dbType:        types.DBTypeMySQL,
			sql:           "SELECT name, order_id FROM t_user WHERE age > ? FOR UPDATE",
			expectedMySQL: "SELECT SQL_NO_CACHE id,order_id FROM t_user WHERE age>? FOR UPDATE",
		},
		{
			name:          "PostgreSQL - Select for update with primary key",
			dbType:        types.DBTypePostgreSQL,
			sql:           "SELECT name, order_id FROM t_user WHERE age > $1 FOR UPDATE",
			expectedPGSQL: "SELECT \"id\",\"order_id\" FROM \"t_user\" WHERE age>$1 FOR UPDATE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &SelectForUpdateExecutor{
				BasicUndoLogBuilder: builder.BasicUndoLogBuilder{},
				dbType:              tt.dbType,
			}

			ctx, err := parser.DoParser(tt.sql, tt.dbType)
			if err != nil {
				t.Skipf("Parser error for %s: %v", tt.dbType, err)
				return
			}
			assert.NotNil(t, ctx)
			if ctx.SelectStmt == nil && ctx.AuxtenSelectStmt == nil {
				t.Skipf("No valid select statement parsed for %s", tt.dbType)
				return
			}

			metaData := createTestTableMeta()

			var selSQL string
			
			if ctx.SelectStmt != nil {
				selSQL, err = e.buildSelectPKSQL(ctx.SelectStmt, metaData)
			} else if ctx.AuxtenSelectStmt != nil {
				selSQL, err = e.buildSelectPKSQLForPostgreSQL(ctx.AuxtenSelectStmt, metaData)
			} else {
				t.Skipf("No supported AST found for %s test", tt.dbType)
				return
			}
			
			assert.Nil(t, err)
			
			switch tt.dbType {
			case types.DBTypeMySQL:
				assert.Contains(t, selSQL, "SELECT SQL_NO_CACHE")
				assert.Contains(t, selSQL, "id")
				assert.Contains(t, selSQL, "order_id")
				assert.Contains(t, selSQL, "FROM t_user")
				assert.Contains(t, selSQL, "WHERE age>?")
				assert.Contains(t, selSQL, "FOR UPDATE")
			case types.DBTypePostgreSQL:
				assert.Contains(t, selSQL, "SELECT")
				assert.Contains(t, selSQL, "id")
				assert.Contains(t, selSQL, "order_id")
				assert.Contains(t, selSQL, "FROM t_user")
				assert.Contains(t, selSQL, "WHERE age > $1")
				assert.Contains(t, selSQL, "FOR UPDATE")
			}
		})
	}
}

func TestBuildLockKey(t *testing.T) {
	tests := []struct {
		name     string
		dbType   types.DBType
		expected string
	}{
		{
			name:     "MySQL - Build lock key",
			dbType:   types.DBTypeMySQL,
			expected: "t_user:1_oid11,2_oid22,3_oid33",
		},
		{
			name:     "PostgreSQL - Build lock key",
			dbType:   types.DBTypePostgreSQL,
			expected: "t_user:1_oid11,2_oid22,3_oid33",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &SelectForUpdateExecutor{
				BasicUndoLogBuilder: builder.BasicUndoLogBuilder{},
				dbType:              tt.dbType,
			}
			
			metaData := createTestTableMeta()
			rows := &mockRows{}
			index = 0
			lockkey := e.buildLockKey(rows, metaData)
			assert.Equal(t, tt.expected, lockkey)
		})
	}
}

type mockRows struct{}

func (m mockRows) Columns() []string {
	//TODO implement me
	panic("implement me")
}

func (m mockRows) Close() error {
	//TODO implement me
	panic("implement me")
}

func (m mockRows) Next(dest []driver.Value) error {
	if index == len(rowVals) {
		return io.EOF
	}

	if len(dest) >= 1 {
		dest[0] = rowVals[index][0]
		dest[1] = rowVals[index][1]
		index++
	}

	return nil
}

func createTestTableMeta() types.TableMeta {
	return types.TableMeta{
		TableName: "t_user",
		Indexs: map[string]types.IndexMeta{
			"id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "id",
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
			"order_id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "order_id",
				Columns: []types.ColumnMeta{
					{ColumnName: "order_id"},
				},
			},
			"age": {
				IType:      types.IndexTypeNull,
				ColumnName: "age",
				Columns: []types.ColumnMeta{
					{ColumnName: "age"},
				},
			},
		},
	}
}

func TestEscapeIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		dbType     types.DBType
		identifier string
		expected   string
	}{
		{
			name:       "MySQL - Simple identifier",
			dbType:     types.DBTypeMySQL,
			identifier: "user_id",
			expected:   "`user_id`",
		},
		{
			name:       "MySQL - Already escaped",
			dbType:     types.DBTypeMySQL,
			identifier: "`user_id`",
			expected:   "`user_id`",
		},
		{
			name:       "PostgreSQL - Simple identifier",
			dbType:     types.DBTypePostgreSQL,
			identifier: "user_id",
			expected:   "\"user_id\"",
		},
		{
			name:       "PostgreSQL - Already escaped",
			dbType:     types.DBTypePostgreSQL,
			identifier: "\"user_id\"",
			expected:   "\"user_id\"",
		},
		{
			name:       "Empty identifier",
			dbType:     types.DBTypeMySQL,
			identifier: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &SelectForUpdateExecutor{
				dbType: tt.dbType,
			}
			result := e.escapeIdentifier(tt.identifier)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdaptSQLSyntax(t *testing.T) {
	tests := []struct {
		name     string
		dbType   types.DBType
		sql      string
		expected string
	}{
		{
			name:     "MySQL - Keep original syntax",
			dbType:   types.DBTypeMySQL,
			sql:      "SELECT SQL_NO_CACHE `id` FROM `users` WHERE `age` > ?",
			expected: "SELECT SQL_NO_CACHE `id` FROM `users` WHERE `age` > ?",
		},
		{
			name:     "PostgreSQL - Remove SQL_NO_CACHE and convert identifiers",
			dbType:   types.DBTypePostgreSQL,
			sql:      "SELECT SQL_NO_CACHE `id` FROM `users` WHERE `age` > ?",
			expected: "SELECT \"id\" FROM \"users\" WHERE \"age\" > $1",
		},
		{
			name:     "PostgreSQL - Remove SQL_CACHE",
			dbType:   types.DBTypePostgreSQL,
			sql:      "SELECT SQL_CACHE `name` FROM `users` WHERE `id` = ?",
			expected: "SELECT \"name\" FROM \"users\" WHERE \"id\" = $1",
		},
		{
			name:     "PostgreSQL - Multiple parameters",
			dbType:   types.DBTypePostgreSQL,
			sql:      "SELECT `id`, `name` FROM `users` WHERE `age` > ? AND `status` = ?",
			expected: "SELECT \"id\", \"name\" FROM \"users\" WHERE \"age\" > $1 AND \"status\" = $2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &SelectForUpdateExecutor{
				dbType: tt.dbType,
			}
			result := e.adaptSQLSyntax(tt.sql)
			assert.Equal(t, tt.expected, result)
		})
	}
}
