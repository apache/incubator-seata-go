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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/seata/seata-go/pkg/datasource/sql/mock"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/stretchr/testify/assert"

	_ "github.com/seata/seata-go/pkg/util/log"
)

func TestBuildSelectSQLByInsert(t *testing.T) {
	tests := []struct {
		name              string
		query             string
		queryArgs         []driver.Value
		NamedValues       []driver.NamedValue
		metaDataMap       map[string]types.TableMeta
		expectQuery       string
		expectQueryArgs   []driver.Value
		orExpectQuery     string
		orExpectQueryArgs []driver.Value
		mockInsertResult  mockInsertResult
		IncrementStep     int
	}{
		{
			name:  "test-1",
			query: "insert into user(id,name) values (19,'Tony'),(21,'tony')",
			metaDataMap: map[string]types.TableMeta{
				"user": {
					ColumnNames: []string{"id", "name"},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:   "id",
									DatabaseType: types.GetSqlDataType("BIGINT"),
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName: "id",
						},
						"name": {
							ColumnName: "name",
						},
					},
				},
			},
			expectQuery:     "SELECT * FROM user WHERE (`id`) IN ((?),(?)) ",
			expectQueryArgs: []driver.Value{int64(19), int64(21)},
		},
		{
			name:  "test-2",
			query: "insert into user(user_id,name) values (20,'Tony')",
			metaDataMap: map[string]types.TableMeta{
				"user": {
					ColumnNames: []string{"user_id", "name"},
					Indexs: map[string]types.IndexMeta{
						"user_id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "user_id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:   "user_id",
									DatabaseType: types.GetSqlDataType("BIGINT"),
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"user_id": {
							ColumnName: "user_id",
						},
						"name": {
							ColumnName: "name",
						},
					},
				},
			},
			expectQuery:     "SELECT * FROM user WHERE (`user_id`) IN ((?)) ",
			expectQueryArgs: []driver.Value{int64(20)},
		},
		{
			name:  "test-autoincrement-1",
			query: "insert into user(name) values ('Tony')",
			metaDataMap: map[string]types.TableMeta{
				"user": {
					ColumnNames: []string{"user_id", "name"},
					Indexs: map[string]types.IndexMeta{
						"user_id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "user_id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:    "user_id",
									DatabaseType:  types.GetSqlDataType("BIGINT"),
									Autoincrement: true,
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"user_id": {
							ColumnName:    "user_id",
							Autoincrement: true,
						},
						"name": {
							ColumnName: "name",
						},
					},
				},
			},
			mockInsertResult: NewMockInsertResult(100, 1),
			expectQuery:      "SELECT * FROM user WHERE (`user_id`) IN ((?)) ",
			expectQueryArgs:  []driver.Value{int64(100)},
		},
		{
			name:  "test-autoincrement-2",
			query: "insert into user(name) values ('Tony'),('Tom')",
			metaDataMap: map[string]types.TableMeta{
				"user": {
					ColumnNames: []string{"user_id", "name"},
					Indexs: map[string]types.IndexMeta{
						"user_id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "user_id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:    "user_id",
									DatabaseType:  types.GetSqlDataType("BIGINT"),
									Autoincrement: true,
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"user_id": {
							ColumnName:    "user_id",
							Autoincrement: true,
						},
						"name": {
							ColumnName: "name",
						},
					},
				},
			},
			mockInsertResult: NewMockInsertResult(100, 2),
			IncrementStep:    2,
			expectQuery:      "SELECT * FROM user WHERE (`user_id`) IN ((?),(?)) ",
			expectQueryArgs:  []driver.Value{int64(100), int64(102)},
		},
		{
			name:      "test-placeholder",
			query:     "insert into user(id,name) values (?,?)",
			queryArgs: []driver.Value{19, "Tom"},
			NamedValues: []driver.NamedValue{
				{
					Name:  "id",
					Value: 19,
				},
				{
					Name:  "name",
					Value: "Tom",
				},
			},
			metaDataMap: map[string]types.TableMeta{
				"user": {
					ColumnNames: []string{"id", "name"},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:   "id",
									DatabaseType: types.GetSqlDataType("BIGINT"),
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName: "id",
						},
						"name": {
							ColumnName: "name",
						},
					},
				},
			},
			expectQuery:     "SELECT * FROM user WHERE (`id`) IN ((?)) ",
			expectQueryArgs: []driver.Value{19},
		},
		{
			name:  "test-composite-id",
			query: "insert into user(id,name) values (19,'Tony')",
			metaDataMap: map[string]types.TableMeta{
				"user": {
					ColumnNames: []string{"id", "name"},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:   "id",
									DatabaseType: types.GetSqlDataType("BIGINT"),
								},
							},
						},
						"name": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "name",
							Columns: []types.ColumnMeta{
								{
									ColumnName:   "name",
									DatabaseType: types.GetSqlDataType("VARCHAR"),
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName: "id",
						},
						"name": {
							ColumnName: "name",
						},
					},
				},
			},
			expectQuery:       "SELECT * FROM user WHERE (`id`,`name`) IN ((?,?)) ",
			expectQueryArgs:   []driver.Value{int64(19), "Tony"},
			orExpectQuery:     "SELECT * FROM user WHERE (`name`,`id`) IN ((?,?)) ",
			orExpectQueryArgs: []driver.Value{"Tony", int64(19)},
		},
		{
			name:  "test-composite-id-2",
			query: "insert into user(id,name) values (19,'Tony'),(20,'Tom')",
			metaDataMap: map[string]types.TableMeta{
				"user": {
					ColumnNames: []string{"id", "name"},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:   "id",
									DatabaseType: types.GetSqlDataType("BIGINT"),
								},
							},
						},
						"name": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "name",
							Columns: []types.ColumnMeta{
								{
									ColumnName:   "name",
									DatabaseType: types.GetSqlDataType("VARCHAR"),
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName: "id",
						},
						"name": {
							ColumnName: "name",
						},
					},
				},
			},
			expectQuery:       "SELECT * FROM user WHERE (`id`,`name`) IN ((?,?),(?,?)) ",
			expectQueryArgs:   []driver.Value{int64(19), "Tony", int64(20), "Tom"},
			orExpectQuery:     "SELECT * FROM user WHERE (`name`,`id`) IN ((?,?),(?,?)) ",
			orExpectQueryArgs: []driver.Value{"Tony", int64(19), "Tom", int64(20)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := parser.DoParser(test.query)
			assert.Nil(t, err)
			exec := &types.ExecContext{}
			exec.ParseContext = c
			exec.MetaDataMap = test.metaDataMap
			exec.Values = test.queryArgs
			exec.NamedValues = test.NamedValues
			builder := MySQLInsertUndoLogBuilder{}
			builder.InsertResult = &test.mockInsertResult
			builder.IncrementStep = test.IncrementStep
			sql, values := builder.buildAfterImageSQL(context.Background(), exec)
			if test.orExpectQuery != "" && test.orExpectQueryArgs != nil {
				if test.orExpectQuery == sql {
					assert.Equal(t, test.orExpectQueryArgs, values)
					return
				}
			}
			assert.Equal(t, test.expectQuery, sql)
			assert.Equal(t, test.expectQueryArgs, values)
		})
	}
}

type mockInsertResult struct {
	lastInsertID int64
	rowsAffected int64
}

func NewMockInsertResult(lastInsertID int64, rowsAffected int64) mockInsertResult {
	return mockInsertResult{
		lastInsertID: lastInsertID,
		rowsAffected: rowsAffected,
	}
}

func (m *mockInsertResult) GetRows() driver.Rows {
	return &mock.MockTestDriverRows{}
}

func (m *mockInsertResult) GetResult() driver.Result {
	return sqlmock.NewResult(m.lastInsertID, m.rowsAffected)
}
