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
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
)

func TestInsertOnUpdateBeforeImageSQL(t *testing.T) {
	var (
		ioe = insertOnUpdateExecutor{
			beforeImageSqlPrimaryKeys: make(map[string]bool),
		}
		tableMeta1 types.TableMeta
		// one index table
		tableMeta2  types.TableMeta
		columns     = make(map[string]types.ColumnMeta)
		index       = make(map[string]types.IndexMeta)
		index2      = make(map[string]types.IndexMeta)
		columnMeta1 []types.ColumnMeta
		columnMeta2 []types.ColumnMeta
		ColumnNames []string
	)
	columnId := types.ColumnMeta{
		ColumnDef:  nil,
		ColumnName: "id",
	}
	columnName := types.ColumnMeta{
		ColumnDef:  nil,
		ColumnName: "name",
	}
	columnAge := types.ColumnMeta{
		ColumnDef:  nil,
		ColumnName: "age",
	}
	columns["id"] = columnId
	columns["name"] = columnName
	columns["age"] = columnAge
	columnMeta1 = append(columnMeta1, columnId)
	columnMeta2 = append(columnMeta2, columnName, columnAge)
	index["id"] = types.IndexMeta{
		Name:    "PRIMARY",
		IType:   types.IndexTypePrimaryKey,
		Columns: columnMeta1,
	}
	index["id_name_age"] = types.IndexMeta{
		Name:    "name_age_idx",
		IType:   types.IndexUnique,
		Columns: columnMeta2,
	}

	ColumnNames = []string{"id", "name", "age"}
	tableMeta1 = types.TableMeta{
		TableName:   "t_user",
		Columns:     columns,
		Indexs:      index,
		ColumnNames: ColumnNames,
	}

	index2["id_name_age"] = types.IndexMeta{
		Name:    "name_age_idx",
		IType:   types.IndexUnique,
		Columns: columnMeta2,
	}

	tableMeta2 = types.TableMeta{
		TableName:   "t_user",
		Columns:     columns,
		Indexs:      index2,
		ColumnNames: ColumnNames,
	}

	tests := []struct {
		name             string
		execCtx          *types.ExecContext
		sourceQueryArgs  []driver.Value
		expectQuery1     string
		expectQueryArgs1 []driver.Value
		expectQuery2     string
		expectQueryArgs2 []driver.Value
	}{
		{
			execCtx: &types.ExecContext{
				Query:       "insert into t_user(id, name, age) values(?,?,?) on duplicate key update name = ?,age = ?",
				MetaDataMap: map[string]types.TableMeta{"t_user": tableMeta1},
			},
			sourceQueryArgs:  []driver.Value{1, "Jack1", 81, "Link", 18},
			expectQuery1:     "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs1: []driver.Value{1, "Jack1", 81},
			expectQuery2:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (id = ? ) ",
			expectQueryArgs2: []driver.Value{"Jack1", 81, 1},
		},
		{
			execCtx: &types.ExecContext{
				Query:       "insert into t_user(id, name, age) values(1,'Jack1',?) on duplicate key update name = 'Michael',age = ?",
				MetaDataMap: map[string]types.TableMeta{"t_user": tableMeta1},
			},
			sourceQueryArgs:  []driver.Value{81, "Link", 18},
			expectQuery1:     "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs1: []driver.Value{int64(1), "Jack1", 81},
			expectQuery2:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (id = ? ) ",
			expectQueryArgs2: []driver.Value{"Jack1", 81, int64(1)},
		},
		// multi insert one index
		{
			execCtx: &types.ExecContext{
				Query:       "insert into t_user(id, name, age) values(?,?,?),(?,?,?) on duplicate key update name = ?,age = ?",
				MetaDataMap: map[string]types.TableMeta{"t_user": tableMeta2},
			},
			sourceQueryArgs:  []driver.Value{1, "Jack1", 81, 2, "Michal", 35, "Link", 18},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack1", 81, "Michal", 35},
		},
		{
			execCtx: &types.ExecContext{
				Query:       "insert into t_user(id, name, age) values(?,'Jack1',?),(?,?,35) on duplicate key update name = 'Faker',age = ?",
				MetaDataMap: map[string]types.TableMeta{"t_user": tableMeta2},
			},
			sourceQueryArgs:  []driver.Value{1, 81, 2, "Michal", 26},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack1", 81, "Michal", int64(35)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.execCtx.Query)
			assert.Nil(t, err)
			tt.execCtx.ParseContext = c
			query, args, err := ioe.buildBeforeImageSQL(tt.execCtx.ParseContext.InsertStmt, tt.execCtx.MetaDataMap["t_user"], util.ValueToNamedValue(tt.sourceQueryArgs))
			assert.Nil(t, err)
			if query == tt.expectQuery1 {
				assert.Equal(t, tt.expectQuery1, query)
				assert.Equal(t, tt.expectQueryArgs1, util.NamedValueToValue(args))
			} else {
				assert.Equal(t, tt.expectQuery2, query)
				assert.Equal(t, tt.expectQueryArgs2, util.NamedValueToValue(args))
			}
		})
	}
}

func TestInsertOnUpdateAfterImageSQL(t *testing.T) {
	ioe := insertOnUpdateExecutor{}
	tests := []struct {
		name                      string
		beforeSelectSql           string
		BeforeImageSqlPrimaryKeys map[string]bool
		beforeSelectArgs          []driver.Value
		beforeImage               *types.RecordImage
		expectQuery               string
		expectQueryArgs           []driver.Value
	}{
		{
			beforeSelectSql:           "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? ) ",
			BeforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 81},
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{
								KeyType:    types.IndexTypePrimaryKey,
								ColumnName: "id",
								Value:      2,
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "name",
								Value:      "Jack",
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "age",
								Value:      18,
							},
						},
					},
				},
			},
			expectQuery:     "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs: []driver.Value{1, "Jack1", 81},
		},
		{
			beforeSelectSql:           "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? )  OR (id = ? )  OR (name = ?  and age = ? ) ",
			BeforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 30, 2, "Michael", 18},
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{
								KeyType:    types.IndexTypePrimaryKey,
								ColumnName: "id",
								Value:      1,
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "name",
								Value:      "Jack",
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "age",
								Value:      18,
							},
						},
					},
				},
			},
			expectQuery:     "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? )  OR (id = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs: []driver.Value{1, "Jack1", 30, 2, "Michael", 18},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ioe.beforeSelectSql = tt.beforeSelectSql
			ioe.beforeImageSqlPrimaryKeys = tt.BeforeImageSqlPrimaryKeys
			ioe.beforeSelectArgs = util.ValueToNamedValue(tt.beforeSelectArgs)
			query, args := ioe.buildAfterImageSQL(tt.beforeImage)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}
}
