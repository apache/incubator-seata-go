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
	_ "github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/stretchr/testify/assert"
)

func TestBuildSelectSQLByUpdate(t *testing.T) {
	var (
		builder = MySQLUpdateUndoLogBuilder{}
	)

	tests := []struct {
		name            string
		sourceSQL       string
		sourceArgs      []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceSQL:       "update t_user set name = ?, age = ? where id = ?",
			sourceArgs:      []driver.Value{"Jack", 1, 100},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=?",
			expectQueryArgs: []driver.Value{100},
		},
		{
			sourceSQL:       "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age between ? and ?",
			sourceArgs:      []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age BETWEEN ? AND ?",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceSQL:       "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age in (?,?)",
			sourceArgs:      []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age IN (?,?)",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceSQL:       "update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc limit ?",
			sourceArgs:      []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, 2},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE kk BETWEEN ? AND ? AND id=? AND addr IN (?,?) AND age>? ORDER BY name DESC LIMIT ?",
			expectQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args, err := builder.buildUndoLogSelectSQL(tt.sourceSQL, tt.sourceArgs)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, args)
		})
	}
}

func TestBuildRecordImages(t *testing.T) {
	query := "update t_user set name = 'Jack', age = 12, addr = ? where id = ?"

	p, err := parser.DoParser(query)
	assert.Nil(t, err)

	fields := []*ast.SelectField{}
	fieldNames := make([]string, 0)

	for _, column := range p.UpdateStmt.List {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: column.Column,
			},
		})
		fieldNames = append(fieldNames, column.Column.Name.O)
	}

	assert.NotNil(t, fields)
	assert.NotNil(t, fieldNames)
}
