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

package at

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	_ "seata.apache.org/seata-go/v2/pkg/util/log"
)

type stubTableMetaCache struct {
	meta *types.TableMeta
}

func (s *stubTableMetaCache) Init(ctx context.Context, conn *sql.DB) error {
	return nil
}

func (s *stubTableMetaCache) GetTableMeta(ctx context.Context, dbName, table string) (*types.TableMeta, error) {
	return s.meta, nil
}

func (s *stubTableMetaCache) Destroy() error {
	return nil
}

func TestBuildSelectSQLByUpdate(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() {
		undo.UndoConfig = originalUndoConfig
	})

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{
		meta: &types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
		},
	})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age between ? and ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age in (?,?)",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc limit ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, 2},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE kk BETWEEN ? AND ? AND id=? AND addr IN (?,?) AND age>? ORDER BY name DESC LIMIT ? FOR UPDATE",
			expectQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery)
			assert.Nil(t, err)
			executor := NewUpdateExecutor(c, &types.ExecContext{Values: tt.sourceQueryArgs, NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs)}, []exec.SQLHook{})
			query, args, err := executor.(*updateExecutor).buildBeforeImageSQL(context.Background(), util.ValueToNamedValue(tt.sourceQueryArgs))
			assert.Nil(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}
}

func TestBuildSelectSQLByUpdate_PostgreSQL(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() {
		undo.UndoConfig = originalUndoConfig
	})

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	datasource.RegisterTableCache(types.DBTypePostgreSQL, &stubTableMetaCache{
		meta: &types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
		},
	})

	sourceQueryArgs := []driver.Value{"Jack", 1, 100, 18, 28}
	c, err := parser.DoParser("update t_user set name = $1, age = $2 where id = $3 and name = 'Jack' and age between $4 and $5")
	assert.Nil(t, err)

	executor := NewUpdateExecutor(c, &types.ExecContext{
		DBType:      types.DBTypePostgreSQL,
		DBName:      "public",
		Values:      sourceQueryArgs,
		NamedValues: util.ValueToNamedValue(sourceQueryArgs),
	}, []exec.SQLHook{})

	query, args, err := executor.(*updateExecutor).buildBeforeImageSQL(context.Background(), util.ValueToNamedValue(sourceQueryArgs))
	assert.Nil(t, err)
	assert.Equal(t, "SELECT name,age,id FROM t_user WHERE id=$1 AND name='Jack' AND age BETWEEN $2 AND $3 FOR UPDATE", query)
	assert.Equal(t, []driver.Value{100, 18, 28}, util.NamedValueToValue(args))

	meta := &types.TableMeta{
		TableName: "t_user",
		Indexs: map[string]types.IndexMeta{
			"id": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}
	afterSQL, afterArgs := executor.(*updateExecutor).buildAfterImageSQL(types.RecordImage{
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{
				{ColumnName: "name", Value: "Jack"},
				{ColumnName: "age", Value: 1},
				{ColumnName: "id", Value: 100},
			}},
		},
	}, meta)
	assert.NotContains(t, afterSQL, "SQL_NO_CACHE")
	assert.NotContains(t, afterSQL, "`")
	assert.Contains(t, afterSQL, `("id") IN (($1))`)
	assert.Equal(t, []driver.Value{100}, util.NamedValueToValue(afterArgs))
}
