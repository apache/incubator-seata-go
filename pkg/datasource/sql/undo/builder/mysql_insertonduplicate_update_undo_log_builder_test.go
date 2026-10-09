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
	"strings"
	"testing"

	"database/sql/driver"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestInsertOnDuplicateBuildBeforeImageSQL(t *testing.T) {
	var (
		builder = MySQLInsertOnDuplicateUndoLogBuilder{
			BeforeImageSqlPrimaryKeys: make(map[string]bool),
		}
		tableMeta1 types.TableMeta
		//one index table
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

	ColumnNames = []string{"ID", "name", "age"}
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
	}{
		{
			name: "normal 1",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(?,?,?) on duplicate key update name = ?,age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{1, "Jack1", 81, "Link", 18},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (id = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack1", 81, 1},
		},
		{
			name: "normal 2",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(1,'Jack1',?) on duplicate key update name = 'Michael',age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{81, "Link", 18},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (id = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack1", 81, int64(1)},
		},
		{
			name: "multi insert one index",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(?,?,?),(?,?,?) on duplicate key update name = ?,age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta2}},
			},
			sourceQueryArgs:  []driver.Value{1, "Jack1", 81, 2, "Michal", 35, "Link", 18},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack1", 81, "Michal", 35},
		},
		{
			name: "multi insert one index",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(?,'Jack1',?),(?,?,35) on duplicate key update name = 'Faker',age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta2}},
			},
			sourceQueryArgs:  []driver.Value{1, 81, 2, "Michal", 26},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack1", 81, "Michal", int64(35)},
		},
		// Test case for null unique index
		{
			name: "null unique index",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(?, ?, ?) on duplicate key update age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{1, nil, 2, 5},
			expectQuery1:     "SELECT * FROM t_user  WHERE (id = ? ) ",
			expectQueryArgs1: []driver.Value{1},
		},
		// Test case for null primary key
		{
			name: "null primary key",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(?, ?, ?) on duplicate key update age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{nil, "Jack1", 5, 2},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack1", 5},
		},
		// Test case for null unique index with no primary key
		{
			name: "unique index with no primary key",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(name, age) values(?, ?) on duplicate key update age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta2}},
			},
			sourceQueryArgs:  []driver.Value{nil, 2, 5},
			expectQuery1:     "",
			expectQueryArgs1: nil,
		},
		// Test case for null unique index with no primary key
		{
			name: "no key",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(name) values(?) on duplicate key update age = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{"Jack", 5},
			expectQuery1:     "",
			expectQueryArgs1: nil,
		},
		// Test case for composite index with all columns
		{
			name: "composite_index_full",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(?,?,?) on duplicate key update other = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{1, "Jack", 25, "other"},
			expectQuery1:     "SELECT * FROM t_user  WHERE (name = ?  and age = ? )  OR (id = ? ) ",
			expectQueryArgs1: []driver.Value{"Jack", 25, 1},
		},
		// Test case for composite index with null value
		{
			name: "composite_index_with_null",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name, age) values(?,?,?) on duplicate key update other = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{1, "Jack", nil, "other"},
			expectQuery1:     "SELECT * FROM t_user  WHERE (id = ? ) ",
			expectQueryArgs1: []driver.Value{1},
		},
		// Test case for composite index with leftmost prefix only
		{
			name: "composite_index_leftmost_prefix",
			execCtx: &types.ExecContext{
				Query:           "insert into t_user(id, name) values(?,?) on duplicate key update other = ?",
				TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{"t_user": tableMeta1}},
			},
			sourceQueryArgs:  []driver.Value{1, "Jack", "other"},
			expectQuery1:     "SELECT * FROM t_user  WHERE (id = ? ) ",
			expectQueryArgs1: []driver.Value{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.execCtx.Query)
			assert.Nil(t, err)
			tt.execCtx.ParseContext = c
			meta, err := tt.execCtx.TableMetaReader.GetTableMeta(context.Background(), types.TableMetaKey{TableName: "t_user"})
			assert.NoError(t, err)
			for _, table := range []struct {
				name      string
				tableName string
			}{
				{name: "unqualified table", tableName: "`t_user`"},
				{name: "qualified table", tableName: "`tenant_a`.`t_user`"},
			} {
				t.Run(table.name, func(t *testing.T) {
					query, args, err := builder.buildBeforeImageSQL(tt.execCtx.ParseContext.InsertStmt, *meta, tt.sourceQueryArgs, table.tableName)
					require.NoError(t, err)
					wantQuery := strings.Replace(tt.expectQuery1, "FROM t_user", "FROM "+table.tableName, 1)
					assert.Equal(t, wantQuery, query)
					assert.Equal(t, tt.expectQueryArgs1, args)
				})
			}
		})
	}
}

func TestInsertOnDuplicateBeforeImageUsesQualifiedTable(t *testing.T) {
	for _, tt := range []struct {
		name      string
		query     string
		key       types.TableMetaKey
		wantQuery string
	}{
		{
			name:      "qualified table",
			query:     "INSERT INTO tenant_a.t_user (id) VALUES (?) ON DUPLICATE KEY UPDATE value = 2",
			key:       types.TableMetaKey{DBName: "tenant_a", TableName: "t_user"},
			wantQuery: "SELECT * FROM `tenant_a`.`t_user`  WHERE (id = ? ) ",
		},
		{
			name:      "escaped identifiers",
			query:     "INSERT INTO `tenant``a`.`t``user` (id) VALUES (?) ON DUPLICATE KEY UPDATE value = 2",
			key:       types.TableMetaKey{DBName: "tenant`a", TableName: "t`user"},
			wantQuery: "SELECT * FROM `tenant``a`.`t``user`  WHERE (id = ? ) ",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err)
			defer db.Close()
			mock.ExpectPrepare(tt.wantQuery).ExpectQuery().WithArgs(int64(7)).WillReturnRows(
				sqlmock.NewRows([]string{"id"}).AddRow(int64(7)),
			)
			parseCtx, err := parser.DoParserForDB(tt.query, types.DBTypeMySQL)
			require.NoError(t, err)
			id := types.ColumnMeta{ColumnName: "id", DatabaseTypeString: "BIGINT"}
			meta := types.TableMeta{
				TableName: tt.key.TableName,
				Columns:   map[string]types.ColumnMeta{"id": id},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {Name: "PRIMARY", IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{id}},
				},
			}
			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			defer conn.Close()
			var images []*types.RecordImage
			err = conn.Raw(func(raw any) error {
				execCtx := &types.ExecContext{
					Query:           tt.query,
					ParseContext:    parseCtx,
					DBType:          types.DBTypeMySQL,
					Conn:            raw.(driver.Conn),
					NamedValues:     []driver.NamedValue{{Ordinal: 1, Value: int64(7)}},
					TableMetaReader: testTableMetaReader{metas: map[string]types.TableMeta{tt.key.TableName: meta}},
				}
				var imageErr error
				images, imageErr = GetMySQLInsertOnDuplicateUndoLogBuilder().BeforeImage(context.Background(), execCtx)
				return imageErr
			})
			require.NoError(t, err)
			require.Len(t, images, 1)
			assert.Equal(t, &tt.key, images[0].TableMetaKey)
			require.Len(t, images[0].Rows, 1)
			require.Len(t, images[0].Rows[0].Columns, 1)
			assert.Equal(t, "id", images[0].Rows[0].Columns[0].ColumnName)
			assert.Equal(t, int64(7), images[0].Rows[0].Columns[0].Value)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInsertOnDuplicateBuildAfterImageSQL(t *testing.T) {
	var (
		builder = MySQLInsertOnDuplicateUndoLogBuilder{}
	)
	tests := []struct {
		name                      string
		beforeSelectSql           string
		BeforeImageSqlPrimaryKeys map[string]bool
		beforeSelectArgs          []driver.Value
		beforeImages              []*types.RecordImage
		expectQuery               string
		expectQueryArgs           []driver.Value
	}{
		{
			beforeSelectSql:           "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? ) ",
			BeforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 81},
			beforeImages: []*types.RecordImage{
				{
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
			},
			expectQuery:     "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? ) ",
			expectQueryArgs: []driver.Value{1, "Jack1", 81},
		},
		{
			beforeSelectSql:           "SELECT * FROM t_user  WHERE (id = ? )  OR (name = ?  and age = ? )  OR (id = ? )  OR (name = ?  and age = ? ) ",
			BeforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 30, 2, "Michael", 18},
			beforeImages: []*types.RecordImage{
				{
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
									Value:      "Michael",
								},
								{
									KeyType:    types.IndexUnique,
									ColumnName: "age",
									Value:      30,
								},
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
			builder.BeforeSelectSql = tt.beforeSelectSql
			builder.BeforeImageSqlPrimaryKeys = tt.BeforeImageSqlPrimaryKeys
			builder.Args = tt.beforeSelectArgs
			query, args := builder.buildAfterImageSQL(context.TODO(), tt.beforeImages)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, args)
		})
	}
}
