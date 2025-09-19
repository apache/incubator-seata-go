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
	"reflect"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/postgres"
	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"
	"github.com/arana-db/parser/test_driver"
	mysqlDriver "github.com/go-sql-driver/mysql"
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
		dbType            types.DBType
		query             string
		queryArgs         []driver.Value
		NamedValues       []driver.NamedValue
		metaData          types.TableMeta
		expectQuery       string
		expectQueryArgs   []driver.Value
		pgExpectQuery     string
		pgExpectQueryArgs []driver.Value
		mockInsertResult  mock.MockInsertResult
		IncrementStep     int
	}{
		{
			name:   "test-1-mysql",
			dbType: types.DBTypeMySQL,
			query:  "insert into user(id,name) values (19,'Tony'),(21,'tony')",
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
					"id":   {ColumnName: "id"},
					"name": {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT id, name FROM `user` WHERE (`id`) IN ((?),(?))",
			expectQueryArgs: []driver.Value{int64(19), int64(21)},
		},
		{
			name:   "test-2-mysql",
			dbType: types.DBTypeMySQL,
			query:  "insert into user(user_id,name) values (20,'Tony')",
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
					"user_id": {ColumnName: "user_id"},
					"name":    {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT user_id, name FROM `user` WHERE (`user_id`) IN ((?))",
			expectQueryArgs: []driver.Value{int64(20)},
		},
		{
			name:   "test-1-postgres",
			dbType: types.DBTypePostgreSQL,
			query:  "insert into \"user\"(id,name) values (19,'Tony'),(21,'tony')",
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
					"id":   {ColumnName: "id"},
					"name": {ColumnName: "name"},
				},
			},
			pgExpectQuery:     "SELECT \"id\", \"name\" FROM \"user\" WHERE (\"id\") IN (($1),($2)) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{int64(19), int64(21)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			switch test.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					mysqlCfg, ok := cfg.(*mysqlDriver.Config)
					if !ok {
						return mysql.NewTableMetaInstance(db, nil)
					}
					return mysql.NewTableMetaInstance(db, mysqlCfg)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					postgresCfg, ok := cfg.(string)
					if !ok {
						return postgres.NewTableMetaInstance(db, "")
					}
					return postgres.NewTableMetaInstance(db, postgresCfg)
				})
			}

			cache := datasource.GetTableCache(test.dbType)
			stub := gomonkey.ApplyMethod(reflect.TypeOf(cache), "GetTableMeta",
				func(_ datasource.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &test.metaData, nil
				})
			defer stub.Reset()

			c, err := parser.DoParser(test.query, test.dbType)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Values:      test.queryArgs,
				NamedValues: test.NamedValues,
				TxCtx: &types.TransactionContext{
					DBType: test.dbType,
				},
				Conn: mockDBConn(test.dbType),
			}

			executor := NewInsertExecutor(c, execCtx, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = &test.mockInsertResult
			executor.(*insertExecutor).incrementStep = test.IncrementStep

			sql, values, err := executor.(*insertExecutor).buildAfterImageSQL(context.Background())
			assert.Nil(t, err)

			if test.dbType == types.DBTypePostgreSQL && test.pgExpectQuery != "" {
				assert.Equal(t, test.pgExpectQuery, sql)
				assert.Equal(t, test.pgExpectQueryArgs, util.NamedValueToValue(values))
			} else {
				assert.Equal(t, test.expectQuery, sql)
				assert.Equal(t, test.expectQueryArgs, util.NamedValueToValue(values))
			}
		})
	}
}

func TestInsertExecutor_containsPK(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		meta     types.TableMeta
		parseCtx *types.ParseContext
		dbType   types.DBType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test-true-mysql",
			fields: fields{},
			args: args{
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
				parseCtx: &types.ParseContext{
					InsertStmt: &ast.InsertStmt{
						Columns: []*ast.ColumnName{{
							Name: model.CIStr{O: "id", L: "id"},
						}},
					},
				},
				dbType: types.DBTypeMySQL,
			},
			want: true,
		},
		{
			name:   "test-false-mysql",
			fields: fields{},
			args: args{
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
				parseCtx: &types.ParseContext{
					InsertStmt: &ast.InsertStmt{
						Columns: []*ast.ColumnName{{
							Name: model.CIStr{O: "name", L: "name"},
						}},
					},
				},
				dbType: types.DBTypeMySQL,
			},
			want: false,
		},
		{
			name:   "test-true-postgres",
			fields: fields{},
			args: args{
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
				parseCtx: &types.ParseContext{
					InsertStmt: &ast.InsertStmt{
						Columns: []*ast.ColumnName{{
							Name: model.CIStr{O: "id", L: "id"},
						}},
					},
				},
				dbType: types.DBTypePostgreSQL,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{
				TxCtx: &types.TransactionContext{
					DBType: tt.args.dbType,
				},
			}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			assert.Equalf(t, tt.want, executor.(*insertExecutor).containsPK(tt.args.meta, tt.args.parseCtx), "containsPK(%v, %v)", tt.args.meta, tt.args.parseCtx)
		})
	}
}

func TestInsertExecutor_containPK(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		columnName string
		meta       types.TableMeta
		dbType     types.DBType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test-true-mysql",
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
				dbType: types.DBTypeMySQL,
			},
			want: true,
		},
		{
			name:   "test-false-mysql",
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
				dbType: types.DBTypeMySQL,
			},
			want: false,
		},

		{
			name:   "test-true-postgres-quoted",
			fields: fields{},
			args: args{
				columnName: "\"id\"",
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
				dbType: types.DBTypePostgreSQL,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{
				TxCtx: &types.TransactionContext{
					DBType: tt.args.dbType,
				},
			}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			assert.Equalf(t, tt.want, executor.(*insertExecutor).containPK(tt.args.columnName, tt.args.meta), "containPK(%v, %v)", tt.args.columnName, tt.args.meta)
		})
	}
}

func TestInsertExecutor_getPkIndex(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		InsertStmt *ast.InsertStmt
		meta       types.TableMeta
		dbType     types.DBType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]int
	}{
		{
			name:   "test-0-mysql",
			fields: fields{},
			args: args{
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
						"id": {ColumnName: "id"},
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
				dbType: types.DBTypeMySQL,
			},
			want: map[string]int{"id": 0},
		},
		{
			name:   "test-1-mysql",
			fields: fields{},
			args: args{
				InsertStmt: &ast.InsertStmt{
					Columns: []*ast.ColumnName{
						{Name: model.CIStr{O: "name", L: "name"}},
						{Name: model.CIStr{O: "id", L: "id"}},
					},
				},
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns:     map[string]types.ColumnMeta{"id": {ColumnName: "id"}},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "id",
							}},
						},
					},
				},
				dbType: types.DBTypeMySQL,
			},
			want: map[string]int{"id": 1},
		},
		{
			name:   "test-postgres-quoted-column",
			fields: fields{},
			args: args{
				InsertStmt: &ast.InsertStmt{
					Columns: []*ast.ColumnName{
						{Name: model.CIStr{O: "\"name\"", L: "name"}},
						{Name: model.CIStr{O: "\"id\"", L: "id"}},
					},
				},
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns:     map[string]types.ColumnMeta{"id": {ColumnName: "id"}},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "id",
							}},
						},
					},
				},
				dbType: types.DBTypePostgreSQL,
			},
			want: map[string]int{"id": 1},
		},
		{
			name:   "test-null",
			fields: fields{},
			args: args{
				InsertStmt: &ast.InsertStmt{},
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns:     map[string]types.ColumnMeta{"id": {ColumnName: "id"}},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName: "id",
							}},
						},
					},
				},
				dbType: types.DBTypeMySQL,
			},
			want: map[string]int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{
				TxCtx: &types.TransactionContext{
					DBType: tt.args.dbType,
				},
			}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			assert.Equalf(t, tt.want, executor.(*insertExecutor).getPkIndex(tt.args.InsertStmt, tt.args.meta, tt.args.dbType),
				"getPkIndex(%v, %v, %v)", tt.args.InsertStmt, tt.args.meta, tt.args.dbType)
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

func TestInsertExecutor_parsePkValuesFromStatement(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		insertStmt *ast.InsertStmt
		meta       types.TableMeta
		nameValues []driver.NamedValue
		dbType     types.DBType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]interface{}
	}{
		{
			name:   "test-1-mysql",
			fields: fields{},
			args: args{
				insertStmt: &ast.InsertStmt{
					Columns: []*ast.ColumnName{
						{Name: model.CIStr{O: "id", L: "id"}},
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
					Columns:     map[string]types.ColumnMeta{"id": {ColumnName: "id"}},
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
					{Name: "name", Value: "Tom"},
					{Name: "id", Value: 1},
				},
				dbType: types.DBTypeMySQL,
			},
			want: map[string][]interface{}{"id": {int64(1)}},
		},
		{
			name:   "test-placeholder-mysql",
			fields: fields{},
			args: args{
				insertStmt: &ast.InsertStmt{
					Columns: []*ast.ColumnName{{Name: model.CIStr{O: "id", L: "id"}}},
					Lists: [][]ast.ExprNode{
						{&test_driver.ValueExpr{Datum: genStrDatum("?")}},
					},
				},
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns:     map[string]types.ColumnMeta{"id": {ColumnName: "id"}},
					Indexs: map[string]types.IndexMeta{
						"id": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
					},
				},
				nameValues: []driver.NamedValue{{Name: "id", Value: int64(1)}},
				dbType:     types.DBTypeMySQL,
			},
			want: map[string][]interface{}{"id": {int64(1)}},
		},

		{
			name:   "test-placeholder-postgres",
			fields: fields{},
			args: args{
				insertStmt: &ast.InsertStmt{
					Columns: []*ast.ColumnName{{Name: model.CIStr{O: "id", L: "id"}}},
					Lists: [][]ast.ExprNode{
						{&test_driver.ValueExpr{Datum: genStrDatum("$1")}},
					},
				},
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns:     map[string]types.ColumnMeta{"id": {ColumnName: "id"}},
					Indexs: map[string]types.IndexMeta{
						"id": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
					},
				},
				nameValues: []driver.NamedValue{{Name: "id", Value: int64(1)}},
				dbType:     types.DBTypePostgreSQL,
			},
			want: map[string][]interface{}{"id": {int64(1)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewInsertExecutor(nil, &types.ExecContext{
				TxCtx: &types.TransactionContext{
					DBType: tt.args.dbType,
				},
			}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			got, err := executor.(*insertExecutor).parsePkValuesFromStatement(
				tt.args.insertStmt, tt.args.meta, tt.args.nameValues, tt.args.dbType)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got,
				"parsePkValuesFromStatement(%v, %v, %v, %v)", tt.args.insertStmt, tt.args.meta, tt.args.nameValues, tt.args.dbType)
		})
	}
}

func TestInsertExecutor_getPkValuesByColumn(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		execCtx *types.ExecContext
		meta    types.TableMeta
		dbType  types.DBType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]interface{}
	}{
		{
			name:   "test-1-mysql",
			fields: fields{},
			args: args{
				dbType: types.DBTypeMySQL,
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {ColumnName: "id"},
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
										Source: &ast.TableName{Name: model.CIStr{O: "test"}},
									},
								},
							},
							Columns: []*ast.ColumnName{{Name: model.CIStr{O: "id", L: "id"}}},
							Lists: [][]ast.ExprNode{
								{&test_driver.ValueExpr{Datum: genIntDatum(1)}},
							},
						},
					},
				},
			},
			want: map[string][]interface{}{"id": {int64(1)}},
		},

		{
			name:   "test-1-postgres",
			fields: fields{},
			args: args{
				dbType: types.DBTypePostgreSQL,
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName:    "id",
							Autoincrement: true,
							Extra:         "nextval('test_id_seq'::regclass)",
						},
					},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName:    "id",
								Autoincrement: true,
								Extra:         "nextval('test_id_seq'::regclass)",
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
										Source: &ast.TableName{Name: model.CIStr{O: "test"}},
									},
								},
							},
							Columns: []*ast.ColumnName{{Name: model.CIStr{O: "id", L: "id"}}},
							Lists: [][]ast.ExprNode{
								{&test_driver.ValueExpr{Datum: genIntDatum(2)}},
							},
						},
					},
				},
			},
			want: map[string][]interface{}{"id": {int64(2)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.args.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					mysqlCfg, ok := cfg.(*mysqlDriver.Config)
					if !ok {
						return mysql.NewTableMetaInstance(db, nil)
					}
					return mysql.NewTableMetaInstance(db, mysqlCfg)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					dsn, ok := cfg.(string)
					if !ok {
						return postgres.NewTableMetaInstance(db, "")
					}
					return postgres.NewTableMetaInstance(db, dsn)
				})
			}

			cache := datasource.GetTableCache(tt.args.dbType)
			stub := gomonkey.ApplyMethod(reflect.TypeOf(cache), "GetTableMeta",
				func(_ datasource.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.args.meta, nil
				})
			defer stub.Reset()

			execCtx := &types.ExecContext{
				ParseContext: tt.args.execCtx.ParseContext,
				TxCtx: &types.TransactionContext{
					DBType: tt.args.dbType,
				},
				Conn: mockDBConn(tt.args.dbType),
			}

			executor := NewInsertExecutor(execCtx.ParseContext, execCtx, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			got, err := executor.(*insertExecutor).getPkValuesByColumn(context.Background(), execCtx)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "getPkValuesByColumn(%v)", execCtx)
		})
	}
}

func TestInsertExecutor_getPkValuesByAuto(t *testing.T) {
	type fields struct {
		InsertResult  types.ExecResult
		IncrementStep int
	}
	type args struct {
		execCtx *types.ExecContext
		meta    types.TableMeta
		dbType  types.DBType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test-2-mysql",
			fields: fields{
				InsertResult:  &mock.MockInsertResult{LastInsertID: 100, RowsAffected: 1},
				IncrementStep: 1,
			},
			args: args{
				dbType: types.DBTypeMySQL,
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
						"id":   {ColumnName: "id"},
						"name": {ColumnName: "name"},
					},
				},
				execCtx: &types.ExecContext{
					ParseContext: &types.ParseContext{
						InsertStmt: &ast.InsertStmt{
							Table: &ast.TableRefsClause{
								TableRefs: &ast.Join{
									Left: &ast.TableSource{
										Source: &ast.TableName{Name: model.CIStr{O: "test"}},
									},
								},
							},
							Columns: []*ast.ColumnName{{Name: model.CIStr{O: "name", L: "name"}}},
							Lists: [][]ast.ExprNode{
								{&test_driver.ValueExpr{Datum: genStrDatum("Tom")}},
							},
						},
					},
				},
			},
			want:    map[string][]interface{}{"id": {int64(100)}},
			wantErr: assert.NoError,
		},

		{
			name: "test-2-postgres",
			fields: fields{
				InsertResult: &mock.MockInsertResult{
					Rows: &mock.GenericMockRows{
						ColumnNames: []string{"id"},
						Data:        [][]driver.Value{{int64(200)}},
					},
					RowsAffected: 1,
				},
				IncrementStep: 1,
			},
			args: args{
				dbType: types.DBTypePostgreSQL,
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
									Extra:         "nextval('test_id_seq'::regclass)",
								},
							},
						},
					},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName:    "id",
							Autoincrement: true,
							Extra:         "nextval('test_id_seq'::regclass)",
						},
						"name": {ColumnName: "name"},
					},
				},
				execCtx: &types.ExecContext{
					ParseContext: &types.ParseContext{
						InsertStmt: &ast.InsertStmt{
							Table: &ast.TableRefsClause{
								TableRefs: &ast.Join{
									Left: &ast.TableSource{
										Source: &ast.TableName{Name: model.CIStr{O: "test"}},
									},
								},
							},
							Columns: []*ast.ColumnName{{Name: model.CIStr{O: "name", L: "name"}}},
							Lists: [][]ast.ExprNode{
								{&test_driver.ValueExpr{Datum: genStrDatum("Jerry")}},
							},
						},
					},
				},
			},
			want:    map[string][]interface{}{"id": {int64(200)}},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.args.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					mysqlCfg, ok := cfg.(*mysqlDriver.Config)
					if !ok {
						return mysql.NewTableMetaInstance(db, nil)
					}
					return mysql.NewTableMetaInstance(db, mysqlCfg)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					dsn, ok := cfg.(string)
					if !ok {
						return postgres.NewTableMetaInstance(db, "")
					}
					return postgres.NewTableMetaInstance(db, dsn)
				})
			}

			cache := datasource.GetTableCache(tt.args.dbType)
			stub := gomonkey.ApplyMethod(reflect.TypeOf(cache), "GetTableMeta",
				func(_ datasource.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.args.meta, nil
				})
			defer stub.Reset()

			execCtx := &types.ExecContext{
				ParseContext: tt.args.execCtx.ParseContext,
				TxCtx: &types.TransactionContext{
					DBType: tt.args.dbType,
				},
				Conn: mockDBConn(tt.args.dbType),
			}

			executor := NewInsertExecutor(execCtx.ParseContext, execCtx, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep
			executor.(*insertExecutor).parserCtx = tt.args.execCtx.ParseContext

			got, err := executor.(*insertExecutor).getPkValuesByAuto(context.Background(), execCtx)
			tt.wantErr(t, err)
			assert.Equalf(t, tt.want, got, "getPkValuesByAuto(%v)", execCtx)
		})
	}
}

func TestInsertExecutor_autoGeneratePks(t *testing.T) {
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
		dbType         types.DBType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]interface{}
	}{

		{
			name: "test-mysql-single",
			fields: fields{
				IncrementStep: 1,
			},
			args: args{
				dbType:         types.DBTypeMySQL,
				autoColumnName: "id",
				lastInsetId:    100,
				updateCount:    1,
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName:    "id",
							Autoincrement: true,
						},
					},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName:    "id",
								Autoincrement: true,
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
										Source: &ast.TableName{Name: model.CIStr{O: "test"}},
									},
								},
							},
							Columns: []*ast.ColumnName{{Name: model.CIStr{O: "id", L: "id"}}},
							Lists:   [][]ast.ExprNode{{&test_driver.ValueExpr{Datum: genIntDatum(1)}}},
						},
					},
				},
			},
			want: map[string][]interface{}{"id": {int64(100)}},
		},

		{
			name: "test-mysql-batch",
			fields: fields{
				IncrementStep: 2,
			},
			args: args{
				dbType:         types.DBTypeMySQL,
				autoColumnName: "id",
				lastInsetId:    200,
				updateCount:    3,
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {ColumnName: "id", Autoincrement: true},
					},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName:    "id",
								Autoincrement: true,
							}},
						},
					},
				},
				execCtx: &types.ExecContext{
					ParseContext: &types.ParseContext{InsertStmt: &ast.InsertStmt{Table: &ast.TableRefsClause{}}},
				},
			},
			want: map[string][]interface{}{"id": {int64(200), int64(202), int64(204)}},
		},

		{
			name: "test-postgres-sequence",
			fields: fields{
				IncrementStep: 0,
			},
			args: args{
				dbType:         types.DBTypePostgreSQL,
				autoColumnName: "id",
				lastInsetId:    300,
				updateCount:    2,
				meta: types.TableMeta{
					ColumnNames: []string{"id"},
					Columns: map[string]types.ColumnMeta{
						"id": {
							ColumnName:    "id",
							Autoincrement: true,
							Extra:         "nextval('test_id_seq'::regclass)",
						},
					},
					Indexs: map[string]types.IndexMeta{
						"id": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{
								ColumnName:    "id",
								Autoincrement: true,
								Extra:         "nextval('test_id_seq'::regclass)",
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
										Source: &ast.TableName{Name: model.CIStr{O: "test"}},
									},
								},
							},
							Columns: []*ast.ColumnName{{Name: model.CIStr{O: "name", L: "name"}}},
							Lists:   [][]ast.ExprNode{{&test_driver.ValueExpr{Datum: genStrDatum("Alice")}}},
						},
					},
				},
			},
			want: map[string][]interface{}{"id": {int64(300), int64(301)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.args.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					mysqlCfg, ok := cfg.(*mysqlDriver.Config)
					if !ok {
						return mysql.NewTableMetaInstance(db, nil)
					}
					return mysql.NewTableMetaInstance(db, mysqlCfg)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					dsn, ok := cfg.(string)
					if !ok {
						dsn = ""
					}
					return postgres.NewTableMetaInstance(db, dsn)
				})
			}

			cache := datasource.GetTableCache(tt.args.dbType)
			stub := gomonkey.ApplyMethod(reflect.TypeOf(cache), "GetTableMeta",
				func(_ datasource.TableMetaCache, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &tt.args.meta, nil
				})
			defer stub.Reset()

			execCtx := &types.ExecContext{
				ParseContext: tt.args.execCtx.ParseContext,
				TxCtx: &types.TransactionContext{
					DBType: tt.args.dbType,
				},
				Conn: mockDBConn(tt.args.dbType),
			}

			executor := NewInsertExecutor(execCtx.ParseContext, execCtx, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep
			executor.(*insertExecutor).parserCtx = tt.args.execCtx.ParseContext

			got, err := executor.(*insertExecutor).autoGeneratePks(
				execCtx,
				tt.args.autoColumnName,
				tt.args.lastInsetId,
				tt.args.updateCount,
			)
			assert.Nil(t, err, "autoGeneratePks failed: %v", err)
			assert.Equalf(t, tt.want, got,
				"autoGeneratePks(execCtx: %v, autoCol: %s, lastId: %d, count: %d)",
				execCtx, tt.args.autoColumnName, tt.args.lastInsetId, tt.args.updateCount)
		})
	}
}

func mockDBConn(dbType types.DBType) driver.Conn {
	return mock.NewGenericMockConn(dbType)
}
