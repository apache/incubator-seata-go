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
	"context"
	"database/sql/driver"
	"testing"

	"github.com/arana-db/parser/ast"
	_ "github.com/arana-db/parser/test_driver"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	_ "seata.apache.org/seata-go/v2/pkg/util/log"
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

func TestMySQLMultiUpdateUndoLogBuilder_BeforeImageParameters(t *testing.T) {
	testBuilderBeforeImageParameters(t, &MySQLMultiUpdateUndoLogBuilder{},
		"UPDATE t_user SET id=? WHERE id=?; UPDATE t_user SET id=? WHERE id=?",
		[]driver.Value{int64(11), int64(1), int64(22), int64(2)})
}

func TestMySQLMultiUpdateUndoLogBuilder_AfterImageSQL(t *testing.T) {
	b := &MySQLMultiUpdateUndoLogBuilder{}
	cases := []struct {
		name  string
		ids   []driver.Value
		query string
	}{
		{"single_row", []driver.Value{int64(100)}, "SELECT * FROM t_user WHERE  (`id`) IN ((?)) "},
		{"multiple_rows", []driver.Value{int64(1), int64(2)}, "SELECT * FROM t_user WHERE  (`id`) IN ((?),(?)) "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := &types.RecordImage{TableName: "t_user"}
			for _, id := range tc.ids {
				before.Rows = append(before.Rows, types.RowImage{Columns: []types.ColumnImage{
					{ColumnName: "id", Value: id, KeyType: types.IndexTypePrimaryKey},
				}})
			}
			query, args := b.buildAfterImageSQL(before, legacyBuilderTestMeta())
			assert.Equal(t, tc.query, query)
			assert.Equal(t, tc.ids, args)
			_, err := parser.DoParser(query)
			assert.NoError(t, err)

			for _, source := range []string{
				"UPDATE t_user SET id=11 WHERE id=1",
				"UPDATE t_user SET id=11 WHERE id=1; UPDATE t_user SET id=22 WHERE id=2",
			} {
				t.Run(source, func(t *testing.T) {
					conn := &legacyBuilderTestConn{}
					execCtx := legacyBuilderTestContext(t, source, conn)
					images, err := b.AfterImage(context.Background(), execCtx, []*types.RecordImage{before})
					if !assert.NoError(t, err) {
						return
					}
					assert.True(t, conn.prepared)
					assert.Equal(t, tc.query, conn.query)
					assert.Equal(t, tc.ids, conn.args)
					if assert.Len(t, images, 1) {
						assert.Equal(t, "t_user", images[0].TableName)
						assert.Equal(t, execCtx.ParseContext.SQLType, images[0].SQLType)
						assert.Len(t, images[0].Rows, 2)
					}
				})
			}
		})
	}
}

func TestMySQLMultiUpdateUndoLogBuilder_AfterImageEmpty(t *testing.T) {
	cases := []struct {
		name   string
		before []*types.RecordImage
	}{
		{"nil_images", nil},
		{"empty_images", []*types.RecordImage{}},
		{"no_rows", []*types.RecordImage{{
			TableName: "t_user", SQLType: types.SQLTypeUpdate, Rows: []types.RowImage{},
		}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn := &legacyBuilderTestConn{}
			execCtx := legacyBuilderTestContext(t, "UPDATE t_user SET id=11 WHERE id=1", conn)
			images, err := (&MySQLMultiUpdateUndoLogBuilder{}).AfterImage(context.Background(), execCtx, tc.before)
			assert.NoError(t, err)
			assert.False(t, conn.prepared, "empty before image must not query the database")
			if len(tc.before) == 0 {
				assert.Equal(t, tc.before, images)
			} else if assert.Len(t, images, 1) {
				assert.NotSame(t, tc.before[0], images[0])
				assert.Empty(t, images[0].Rows)
				assert.Equal(t, "t_user", images[0].TableName)
				assert.Equal(t, execCtx.ParseContext.SQLType, images[0].SQLType)
				metaData := execCtx.MetaDataMap["t_user"]
				assert.Equal(t, &metaData, images[0].TableMeta)
				assert.NotNil(t, images[0].PrimaryKeyMap)
				assert.Empty(t, images[0].PrimaryKeyMap)
			}
		})
	}
}
