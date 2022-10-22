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

	"github.com/arana-db/parser/ast"
	_ "github.com/arana-db/parser/test_driver"
	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	_ "github.com/seata/seata-go/pkg/util/log"
)

func TestBuildSelectSQLByMultiUpdate(t *testing.T) {
	var builder MySQLMultiUpdateUndoLogBuilder
	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ?;update t_user set name = ?, age = ? where id = ?;",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, "TOM", 2, 200},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? OR id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 200},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age between ? and ?;update t_user set name = ?, age = ? where id = ? and name = 'Jack2' and age between ? and ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 28, 38},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age BETWEEN ? AND ? OR id=? AND name=_UTF8MB4Jack2 AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28, 200, 28, 38},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age in (?,?);update t_user set name = ?, age = ? where id = ? and name = 'Jack2' and age in (?,?)",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 48, 58},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age IN (?,?) OR id=? AND name=_UTF8MB4Jack2 AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28, 200, 48, 58},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery)
			assert.Nil(t, err)
			var updateStmts []*ast.UpdateStmt
			for _, v := range c.MultiStmt {
				updateStmts = append(updateStmts, v.UpdateStmt)
			}

			query, args, err := builder.buildBeforeImageSQL(updateStmts, tt.sourceQueryArgs)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, args)
		})
	}

	sourceQuery := "update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc;update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name"
	sourceQueryArgs := []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, 2, "Jack2", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, 2}
	c, err := parser.DoParser(sourceQuery)
	assert.NoError(t, err)
	var updateStmts []*ast.UpdateStmt
	for _, v := range c.MultiStmt {
		updateStmts = append(updateStmts, v.UpdateStmt)
	}
	_, _, err = builder.buildBeforeImageSQL(updateStmts, sourceQueryArgs)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "multi update SQL with orderBy condition is not support yet")
}
