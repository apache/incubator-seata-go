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
	"database/sql/driver"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"
	"github.com/arana-db/parser/test_driver"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/mysql"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
)

func TestBuildSelectSQLByInsert(t *testing.T) {
	tests := []struct {
		name              string
		query             string
		queryArgs         []driver.Value
		NamedValues       []driver.NamedValue
		metaData          types.TableMeta
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
			metaData: types.TableMeta{
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

			expectQuery:     "SELECT id, name FROM user WHERE (`id`) IN ((?),(?)) ",
			expectQueryArgs: []driver.Value{int64(19), int64(21)},
		},
		{
			name:  "test-2",
			query: "insert into user(user_id,name) values (20,'Tony')",
			metaData: types.TableMeta{
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
			expectQuery:     "SELECT user_id, name FROM user WHERE (`user_id`) IN ((?)) ",
			expectQueryArgs: []driver.Value{int64(20)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, mysql.NewTableMetaInstance(nil, nil))
			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypeMySQL)), "GetTableMeta",
				func(_ *mysql.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &test.metaData, nil
				})

			c, err := parser.DoParser(test.query)
			assert.Nil(t, err)

			executor := NewInsertExecutor(c, &types.ExecContext{
				Values:      test.queryArgs,
				NamedValues: test.NamedValues,
			}, []exec.SQLHook{})

			executor.(*insertExecutor).businesSQLResult = &test.mockInsertResult
			executor.(*insertExecutor).incrementStep = test.IncrementStep

			sql, values, err := executor.(*insertExecutor).buildAfterImageSQL(context.Background())
			assert.Nil(t, err)
			if test.orExpectQuery != "" && test.orExpectQueryArgs != nil {
				if test.orExpectQuery == sql {
					assert.Equal(t, test.orExpectQueryArgs, values)
					return
				}
			}
			assert.Equal(t, test.expectQuery, sql)
			assert.Equal(t, test.expectQueryArgs, util.NamedValueToValue(values))
			stub.Reset()
		})
	}
}

func TestMySQLInsertUndoLogBuilder_containsPK(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		meta     types.TableMeta
		parseCtx *types.ParseContext
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "test-true", fields: fields{}, args: args{meta: types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{{
						ColumnName: "id",
					}},
				},
			},
		}, parseCtx: &types.ParseContext{
			InsertStmt: &ast.InsertStmt{
				Columns: []*ast.ColumnName{{
					Name: model.CIStr{O: "id", L: "id"},
				}},
			},
		}}, want: true},
		{name: "test-false", fields: fields{}, args: args{meta: types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{{
						ColumnName: "id",
					}},
				},
			},
		}, parseCtx: &types.ParseContext{
			InsertStmt: &ast.InsertStmt{
				Columns: []*ast.ColumnName{{
					Name: model.CIStr{O: "name", L: "name"},
				}},
			},
		}}, want: false},
		{name: "test-false", fields: fields{}, args: args{meta: types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{{
						ColumnName: "id",
					}},
				},
			},
		}, parseCtx: &types.ParseContext{}}, want: false},
		{name: "test-false", fields: fields{}, args: args{meta: types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{{
						ColumnName: "id",
					}},
				},
			},
		}, parseCtx: &types.ParseContext{
			InsertStmt: &ast.InsertStmt{
				Columns: []*ast.ColumnName{{}},
			},
		}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			assert.Equalf(t, tt.want, executor.(*insertExecutor).containsPK(tt.args.meta, tt.args.parseCtx), "containsPK(%v, %v)", tt.args.meta, tt.args.parseCtx)
		})
	}
}

func TestMySQLInsertUndoLogBuilder_containPK(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		columnName string
		meta       types.TableMeta
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test-true",
			fields: fields{},
			args: args{
				columnName: "id",
				meta: types.TableMeta{
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "id",
							}},
						},
					},
				},
			},
			want: true,
		},
		{
			name:   "test-false",
			fields: fields{},
			args: args{
				columnName: "id",
				meta: types.TableMeta{
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "bizId",
							}},
						},
					},
				},
			},
			want: false,
		},
		{
			name:   "test-false",
			fields: fields{},
			args: args{
				columnName: "id",
				meta: types.TableMeta{
					Indexs: map[string]types.IndexMeta{},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			assert.Equalf(t, tt.want, executor.(*insertExecutor).containPK(tt.args.columnName, tt.args.meta), "isPKColumn(%v, %v)", tt.args.columnName, tt.args.meta)
		})
	}
}

func TestMySQLInsertUndoLogBuilder_getPkIndex(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		InsertStmt *ast.InsertStmt
		meta       types.TableMeta
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]int
	}{
		{name: "test-0", fields: fields{}, args: args{
			InsertStmt: &ast.InsertStmt{
				Columns: []*ast.ColumnName{
					{
						Name: model.CIStr{O: "id", L: "id"},
					},
					{
						Name: model.CIStr{O: "name", L: "name"},
					},
				},
			},
			meta: types.TableMeta{
				ColumnNames: []string{"id"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{{
							ColumnName: "id",
						}},
					},
				},
			},
		}, want: map[string]int{
			"id": 0,
		}},
		{name: "test-1", fields: fields{}, args: args{
			InsertStmt: &ast.InsertStmt{
				Columns: []*ast.ColumnName{
					{
						Name: model.CIStr{O: "name", L: "name"},
					},
					{
						Name: model.CIStr{O: "id", L: "id"},
					},
				},
			},
			meta: types.TableMeta{
				ColumnNames: []string{"id"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{{
							ColumnName: "id",
						}},
					},
				},
			},
		}, want: map[string]int{
			"id": 1,
		}},
		{name: "test-null", fields: fields{}, args: args{
			InsertStmt: &ast.InsertStmt{},
			meta: types.TableMeta{
				ColumnNames: []string{"id"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{{
							ColumnName: "id",
						}},
					},
				},
			},
		}, want: map[string]int{}},
		{name: "test-1", fields: fields{}, args: args{
			InsertStmt: &ast.InsertStmt{
				Columns: []*ast.ColumnName{
					{
						Name: model.CIStr{O: "name", L: "name"},
					},
					{
						Name: model.CIStr{O: "id", L: "id"},
					},
				},
			},
			meta: types.TableMeta{},
		}, want: map[string]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep
			assert.Equalf(t, tt.want, executor.(*insertExecutor).getPkIndex(tt.args.InsertStmt, tt.args.meta), "getPkIndexArray(%v, %v)", tt.args.InsertStmt, tt.args.meta)
		})
	}
}

func genIntDatum(id int64) test_driver.Datum {
	tmp := test_driver.Datum{}
	tmp.SetInt64(id)
	return tmp
}

func genStrDatum(str string) test_driver.Datum {
	tmp := test_driver.Datum{}
	tmp.SetBytesAsString([]byte(str))
	return tmp
}

func TestMySQLInsertUndoLogBuilder_parsePkValuesFromStatement(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		insertStmt *ast.InsertStmt
		meta       types.TableMeta
		nameValues []driver.NamedValue
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]interface{}
	}{
		{
			name:   "test-1",
			fields: fields{},
			args: args{
				insertStmt: &ast.InsertStmt{
					Columns: []*ast.ColumnName{
						{
							Name: model.CIStr{O: "id", L: "id"},
						},
					},
					Lists: [][]ast.ExprNode{
						{
							&test_driver.ValueExpr{
								Datum: genIntDatum(1),
							},
						},
					},
				},
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName: "id",
						},
					},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "id",
							}},
						},
					},
				},
				nameValues: []driver.NamedValue{
					{
						Name:  "name",
						Value: "Tom",
					},
					{
						Name:  "id",
						Value: 1,
					},
				},
			},
			want: map[string][]interface{}{
				"id": {int64(1)},
			},
		},
		{
			name:   "test-placeholder",
			fields: fields{},
			args: args{
				insertStmt: &ast.InsertStmt{
					Columns: []*ast.ColumnName{
						{
							Name: model.CIStr{O: "id", L: "id"},
						},
					},
					Lists: [][]ast.ExprNode{
						{
							&test_driver.ValueExpr{
								Datum: genStrDatum("?"),
							},
						},
					},
				},
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName: "id",
						},
					},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "id",
							}},
						},
					},
				},
				nameValues: []driver.NamedValue{
					{
						Name:  "id",
						Value: int64(1),
					},
				},
			},
			want: map[string][]interface{}{
				"id": {int64(1)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			got, err := executor.(*insertExecutor).parsePkValuesFromStatement(tt.args.insertStmt, tt.args.meta, tt.args.nameValues)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "parsePkValuesFromStatement(%v, %v, %v)", tt.args.insertStmt, tt.args.meta, tt.args.nameValues)
		})
	}
}

func TestMySQLInsertUndoLogBuilder_getPkValuesByColumn(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		execCtx *types.ExecContext
		meta    types.TableMeta
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]interface{}
	}{
		{
			name:   "test-1",
			fields: fields{},
			args: args{
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName: "id",
						},
					},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "id",
							}},
						},
					},
				},
				execCtx: &types.ExecContext{
					ParseContext: &types.ParseContext{
						InsertStmt: &ast.InsertStmt{
							Table: &ast.TableRefsClause{
								TableRefs: &ast.Join{
									Left: &ast.TableSource{
										Source: &ast.TableName{
											Name: model.CIStr{
												O: "test",
											},
										},
									},
								},
							},
							Columns: []*ast.ColumnName{
								{
									Name: model.CIStr{O: "id", L: "id"},
								},
							},
							Lists: [][]ast.ExprNode{
								{
									&test_driver.ValueExpr{
										Datum: genIntDatum(1),
									},
								},
							},
						},
					},
				},
			},
			want: map[string][]interface{}{
				"id": {int64(1)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, mysql.NewTableMetaInstance(nil, nil))
			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypeMySQL)), "GetTableMeta",
				func(_ *mysql.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.args.meta, nil
				})

			executor := NewInsertExecutor(tt.args.execCtx.ParseContext, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			got, err := executor.(*insertExecutor).getPkValuesByColumn(context.Background(), tt.args.execCtx)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "getPkValuesByColumn(%v)", tt.args.execCtx)
			stub.Reset()
		})
	}
}

func TestMySQLInsertUndoLogBuilder_getPkValuesByAuto(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		execCtx *types.ExecContext
		meta    types.TableMeta
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test-2",
			fields: fields{
				InsertResult:  &mockInsertResult{lastInsertID: 100, rowsAffected: 1},
				IncrementStep: 1,
			},
			args: args{
				meta: types.TableMeta{
					ColumnNames: []string{"id", "name"},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
							Columns: []types.ColumnMeta{
								{
									ColumnName:    "id",
									DatabaseType:  types.GetSqlDataType("BIGINT"),
									Autoincrement: true,
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
				execCtx: &types.ExecContext{
					ParseContext: &types.ParseContext{
						InsertStmt: &ast.InsertStmt{
							Table: &ast.TableRefsClause{
								TableRefs: &ast.Join{
									Left: &ast.TableSource{
										Source: &ast.TableName{
											Name: model.CIStr{
												O: "test",
											},
										},
									},
								},
							},
							Columns: []*ast.ColumnName{
								{
									Name: model.CIStr{O: "name", L: "name"},
								},
							},
							Lists: [][]ast.ExprNode{
								{
									&test_driver.ValueExpr{
										Datum: genStrDatum("Tom"),
									},
								},
							},
						},
					},
				},
			},
			want: map[string][]interface{}{
				"id": {int64(100)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, mysql.NewTableMetaInstance(nil, nil))
			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypeMySQL)), "GetTableMeta",
				func(_ *mysql.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.args.meta, nil
				})
			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep
			executor.(*insertExecutor).parserCtx = tt.args.execCtx.ParseContext

			got, err := executor.(*insertExecutor).getPkValuesByAuto(context.Background(), tt.args.execCtx)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "getPkValuesByAuto(%v)", tt.args.execCtx)
			stub.Reset()
		})
	}
}

func TestMySQLInsertUndoLogBuilder_autoGeneratePks(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		execCtx        *types.ExecContext
		autoColumnName string
		lastInsetId    int64
		updateCount    int64
		meta           types.TableMeta
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]interface{}
	}{
		{name: "test", fields: fields{
			IncrementStep: 1,
		}, args: args{
			meta: types.TableMeta{
				ColumnNames: []string{"id"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{{
							ColumnName: "id",
						}},
					},
				},
			},
			execCtx: &types.ExecContext{
				ParseContext: &types.ParseContext{
					InsertStmt: &ast.InsertStmt{
						Table: &ast.TableRefsClause{
							TableRefs: &ast.Join{
								Left: &ast.TableSource{
									Source: &ast.TableName{
										Name: model.CIStr{
											O: "test",
										},
									},
								},
							},
						},
						Columns: []*ast.ColumnName{
							{
								Name: model.CIStr{O: "id", L: "id"},
							},
						},
						Lists: [][]ast.ExprNode{
							{
								&test_driver.ValueExpr{
									Datum: genIntDatum(1),
								},
							},
						},
					},
				},
			},
			autoColumnName: "id",
			lastInsetId:    100,
			updateCount:    1,
		}, want: map[string][]interface{}{
			"id": {int64(100)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, mysql.NewTableMetaInstance(nil, nil))
			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypeMySQL)), "GetTableMeta",
				func(_ *mysql.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.args.meta, nil
				})

			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			got, err := executor.(*insertExecutor).autoGeneratePks(tt.args.execCtx, tt.args.autoColumnName, tt.args.lastInsetId, tt.args.updateCount)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "autoGeneratePks(%v, %v, %v, %v)", tt.args.execCtx, tt.args.autoColumnName, tt.args.lastInsetId, tt.args.updateCount)
			stub.Reset()
		})
	}
}

// TestInsertExecutor_BeforeImage tests the beforeImage method
func TestInsertExecutor_BeforeImage(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		metaData   types.TableMeta
		wantErr    bool
		expectRows int
	}{
		{
			name:  "simple insert - before image should be empty",
			query: "INSERT INTO users (id, name) VALUES (1, 'Alice')",
			metaData: types.TableMeta{
				TableName:   "users",
				ColumnNames: []string{"id", "name"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
					},
					"name": {
						ColumnName: "name",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "id"},
						},
					},
				},
			},
			wantErr:    false,
			expectRows: 0,
		},
		{
			name:  "insert with auto-increment - before image should be empty",
			query: "INSERT INTO users (name) VALUES ('Bob')",
			metaData: types.TableMeta{
				TableName:   "users",
				ColumnNames: []string{"id", "name"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName:    "id",
						Autoincrement: true,
					},
					"name": {
						ColumnName: "name",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "id", Autoincrement: true},
						},
					},
				},
			},
			wantErr:    false,
			expectRows: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, mysql.NewTableMetaInstance(nil, nil))
			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypeMySQL)), "GetTableMeta",
				func(_ *mysql.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.metaData, nil
				})
			defer stub.Reset()

			parseCtx, err := parser.DoParser(tt.query)
			assert.Nil(t, err)

			executor := NewInsertExecutor(parseCtx, &types.ExecContext{DBName: "test"}, []exec.SQLHook{})

			image, err := executor.(*insertExecutor).beforeImage(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, image)
				assert.Equal(t, tt.expectRows, len(image.Rows), "before image should have expected rows")
				assert.Equal(t, image.SQLType, types.SQLTypeInsert)
				assert.Equal(t, tt.metaData.TableName, image.TableName)
			}
		})
	}
}

// TestInsertExecutor_ExecContext tests the ExecContext method
func TestInsertExecutor_ExecContext(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		metaData        types.TableMeta
		mockResult      types.ExecResult
		wantErr         bool
		expectImageRows int
	}{
		{
			name:  "successful insert execution",
			query: "INSERT INTO users (id, name) VALUES (1, 'Alice')",
			metaData: types.TableMeta{
				TableName:   "users",
				ColumnNames: []string{"id", "name"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName:         "id",
						DatabaseTypeString: "BIGINT",
					},
					"name": {
						ColumnName:         "name",
						DatabaseTypeString: "VARCHAR",
					},
				},
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
			},
			mockResult:      &mockInsertResult{lastInsertID: 1, rowsAffected: 1},
			wantErr:         false,
			expectImageRows: 1,
		},
		{
			name:  "insert with auto-increment",
			query: "INSERT INTO users (name) VALUES ('Bob')",
			metaData: types.TableMeta{
				TableName:   "users",
				ColumnNames: []string{"id", "name"},
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName:         "id",
						DatabaseTypeString: "BIGINT",
						Autoincrement:      true,
					},
					"name": {
						ColumnName:         "name",
						DatabaseTypeString: "VARCHAR",
					},
				},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "id",
						Columns: []types.ColumnMeta{
							{
								ColumnName:    "id",
								DatabaseType:  types.GetSqlDataType("BIGINT"),
								Autoincrement: true,
							},
						},
					},
				},
			},
			mockResult:      &mockInsertResult{lastInsertID: 100, rowsAffected: 1},
			wantErr:         false,
			expectImageRows: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, mysql.NewTableMetaInstance(nil, nil))
			stubMeta := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypeMySQL)), "GetTableMeta",
				func(_ *mysql.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.metaData, nil
				})
			defer stubMeta.Reset()

			parseCtx, err := parser.DoParser(tt.query)
			assert.Nil(t, err)

			mockConn := &mockQueryerConn{
				queryResult: &mockDriverRows{
					columns: tt.metaData.ColumnNames,
					rows: [][]driver.Value{
						{int64(1), "Alice"},
					},
				},
			}

			execCtx := &types.ExecContext{
				Conn:         mockConn,
				DBName:       "test",
				Query:        tt.query,
				NamedValues:  []driver.NamedValue{},
				ParseContext: parseCtx,
				TxCtx: &types.TransactionContext{
					LockKeys:    make(map[string]struct{}),
					RoundImages: &types.RoundRecordImage{},
				},
			}

			executor := NewInsertExecutor(parseCtx, execCtx, []exec.SQLHook{})

			callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				return tt.mockResult, nil
			}

			result, err := executor.ExecContext(context.Background(), callback)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockResult, result)

				// Verify before image is empty
				beforeImages := execCtx.TxCtx.RoundImages.BeofreImages()
				assert.Equal(t, 1, len(beforeImages))
				assert.Equal(t, 0, len(beforeImages[0].Rows))

				// Verify after image is populated
				afterImages := execCtx.TxCtx.RoundImages.AfterImages()
				assert.Equal(t, 1, len(afterImages))
				assert.Equal(t, tt.expectImageRows, len(afterImages[0].Rows))
			}
		})
	}
}

// Mock types for testing
type mockQueryerConn struct {
	queryResult driver.Rows
}

func (m *mockQueryerConn) Prepare(query string) (driver.Stmt, error) {
	return nil, nil
}

func (m *mockQueryerConn) Close() error {
	return nil
}

func (m *mockQueryerConn) Begin() (driver.Tx, error) {
	return nil, nil
}

func (m *mockQueryerConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return m.queryResult, nil
}

func (m *mockQueryerConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	return m.queryResult, nil
}

type mockDriverRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (m *mockDriverRows) Columns() []string {
	return m.columns
}

func (m *mockDriverRows) Close() error {
	return nil
}

func (m *mockDriverRows) Next(dest []driver.Value) error {
	if m.index >= len(m.rows) {
		return driver.ErrSkip
	}
	copy(dest, m.rows[m.index])
	m.index++
	return nil
}
