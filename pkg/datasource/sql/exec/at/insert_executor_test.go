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
	"io"
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"
	"github.com/arana-db/parser/test_driver"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
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
		{
			name:  "single pk without explicit columns",
			query: "insert into user values (19,'Tony'),(21,'tony')",
			metaData: types.TableMeta{
				ColumnNames: []string{"id", "name"},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "id",
						Columns: []types.ColumnMeta{
							{
								ColumnName:         "id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"id":   {ColumnName: "id"},
					"name": {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT id, name FROM user WHERE (`id`) IN ((?),(?)) ",
			expectQueryArgs: []driver.Value{int64(19), int64(21)},
		},
		{
			name:  "composite pk explicit values",
			query: "insert into user(id,tenant_id,name) values (19,100,'Tony'),(21,101,'tony')",
			metaData: types.TableMeta{
				ColumnNames: []string{"id", "tenant_id", "name"},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{
								ColumnName:         "id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
							{
								ColumnName:         "tenant_id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"id":        {ColumnName: "id"},
					"tenant_id": {ColumnName: "tenant_id"},
					"name":      {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT id, tenant_id, name FROM user WHERE (`id`,`tenant_id`) IN ((?,?),(?,?)) ",
			expectQueryArgs: []driver.Value{int64(19), int64(100), int64(21), int64(101)},
		},
		{
			name:  "composite pk without explicit columns",
			query: "insert into user values (19,100,'Tony'),(21,101,'tony')",
			metaData: types.TableMeta{
				ColumnNames: []string{"id", "tenant_id", "name"},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{
								ColumnName:         "id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
							{
								ColumnName:         "tenant_id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"id":        {ColumnName: "id"},
					"tenant_id": {ColumnName: "tenant_id"},
					"name":      {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT id, tenant_id, name FROM user WHERE (`id`,`tenant_id`) IN ((?,?),(?,?)) ",
			expectQueryArgs: []driver.Value{int64(19), int64(100), int64(21), int64(101)},
		},
		{
			name:  "composite pk escaped explicit values",
			query: "insert into user(`id`,`tenant_id`,name) values (19,100,'Tony'),(21,101,'tony')",
			metaData: types.TableMeta{
				ColumnNames: []string{"id", "tenant_id", "name"},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{
								ColumnName:         "id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
							{
								ColumnName:         "tenant_id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"id":        {ColumnName: "id"},
					"tenant_id": {ColumnName: "tenant_id"},
					"name":      {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT id, tenant_id, name FROM user WHERE (`id`,`tenant_id`) IN ((?,?),(?,?)) ",
			expectQueryArgs: []driver.Value{int64(19), int64(100), int64(21), int64(101)},
		},
		{
			name:        "composite pk prepared explicit values",
			query:       "insert into user(id,tenant_id,name) values (?,?,?),(?,?,?)",
			NamedValues: []driver.NamedValue{{Ordinal: 1, Value: int64(19)}, {Ordinal: 2, Value: int64(100)}, {Ordinal: 3, Value: "Tony"}, {Ordinal: 4, Value: int64(21)}, {Ordinal: 5, Value: int64(101)}, {Ordinal: 6, Value: "tony"}},
			metaData: types.TableMeta{
				ColumnNames: []string{"id", "tenant_id", "name"},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{
								ColumnName:         "id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
							{
								ColumnName:         "tenant_id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"id":        {ColumnName: "id"},
					"tenant_id": {ColumnName: "tenant_id"},
					"name":      {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT id, tenant_id, name FROM user WHERE (`id`,`tenant_id`) IN ((?,?),(?,?)) ",
			expectQueryArgs: []driver.Value{int64(19), int64(100), int64(21), int64(101)},
		},
		{
			name:  "composite pk omitted auto increment column",
			query: "insert into user(tenant_id,name) values (100,'Tony'),(101,'tony')",
			metaData: types.TableMeta{
				ColumnNames: []string{"id", "tenant_id", "name"},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{
								ColumnName:         "id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
								Autoincrement:      true,
							},
							{
								ColumnName:         "tenant_id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"id":        {ColumnName: "id"},
					"tenant_id": {ColumnName: "tenant_id"},
					"name":      {ColumnName: "name"},
				},
			},
			mockInsertResult: mockInsertResult{lastInsertID: 19, rowsAffected: 2},
			IncrementStep:    2,
			expectQuery:      "SELECT id, tenant_id, name FROM user WHERE (`id`,`tenant_id`) IN ((?,?),(?,?)) ",
			expectQueryArgs:  []driver.Value{int64(19), int64(100), int64(21), int64(101)},
		},
		{
			name:  "composite pk null auto increment column",
			query: "insert into user(id,tenant_id,name) values (NULL,100,'Tony'),(NULL,101,'tony')",
			metaData: types.TableMeta{
				ColumnNames: []string{"id", "tenant_id", "name"},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{
								ColumnName:         "id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
								Autoincrement:      true,
							},
							{
								ColumnName:         "tenant_id",
								DatabaseType:       types.GetSqlDataType("BIGINT"),
								DatabaseTypeString: "BIGINT",
							},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"id":        {ColumnName: "id"},
					"tenant_id": {ColumnName: "tenant_id"},
					"name":      {ColumnName: "name"},
				},
			},
			mockInsertResult: mockInsertResult{lastInsertID: 19, rowsAffected: 2},
			IncrementStep:    2,
			expectQuery:      "SELECT id, tenant_id, name FROM user WHERE (`id`,`tenant_id`) IN ((?,?),(?,?)) ",
			expectQueryArgs:  []driver.Value{int64(19), int64(100), int64(21), int64(101)},
		},
		{
			name:  "test-composite-pk-allocation",
			query: "insert into user(tenant_id, id, name) values ('tenantA', 100, 'Tony'), ('tenantB', 101, 'Tom')",
			metaData: types.TableMeta{
				ColumnNames: []string{"tenant_id", "id", "name"},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "tenant_id", DatabaseType: types.GetSqlDataType("VARCHAR")},
							{ColumnName: "id", DatabaseType: types.GetSqlDataType("BIGINT")},
						},
					},
				},
				Columns: map[string]types.ColumnMeta{
					"tenant_id": {ColumnName: "tenant_id"},
					"id":        {ColumnName: "id"},
					"name":      {ColumnName: "name"},
				},
			},
			expectQuery:     "SELECT tenant_id, id, name FROM user WHERE (`tenant_id`,`id`) IN ((?,?),(?,?)) ",
			expectQueryArgs: []driver.Value{"tenantA", int64(100), "tenantB", int64(101)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: &test.metaData})

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
		})
	}
}

func TestInsertKeyPlanUsesParamMarkerOrder(t *testing.T) {
	meta := &types.TableMeta{
		TableName:   "user",
		ColumnNames: []string{"id", "name"},
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"}, "name": {ColumnName: "name"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{
				ColumnName: "id", DatabaseTypeString: "BIGINT",
			}},
		}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})

	for _, tt := range []struct {
		name    string
		query   string
		args    []driver.Value
		want    []driver.Value
		wantErr string
	}{
		{"function before pk", "insert into user(name,id) values (concat(?,?),?),(concat(?,?),?)", []driver.Value{"a", "b", 19, "c", "d", 21}, []driver.Value{19, 21}, ""},
		{"function after pk", "insert into user(id,name) values (?,concat(?,?)),(?,concat(?,?))", []driver.Value{19, "a", "b", 21, "c", "d"}, []driver.Value{19, 21}, ""},
		{"nested function", "insert into user(name,id) values (concat(upper(?),?),?),(lower(?),?)", []driver.Value{"a", "b", 19, "c", 21}, []driver.Value{19, 21}, ""},
		{"insufficient args", "insert into user(name,id) values (concat(?,?),?)", []driver.Value{"a", "b"}, nil, "parameter index 2 out of range"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parseCtx, err := parser.DoParser(tt.query)
			assert.NoError(t, err)
			executor := NewInsertExecutor(parseCtx, &types.ExecContext{NamedValues: util.ValueToNamedValue(tt.args)}, nil).(*insertExecutor)

			_, values, err := executor.buildAfterImageSQL(context.Background())
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, util.NamedValueToValue(values))
		})
	}

	autoMeta := *meta
	autoMeta.Indexs = map[string]types.IndexMeta{"PRIMARY": {
		IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id", DatabaseTypeString: "BIGINT", Autoincrement: true}},
	}}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: &autoMeta})
	parseCtx, err := parser.DoParser("insert into user(id,name) values (default,?)")
	assert.NoError(t, err)
	executor := NewInsertExecutor(parseCtx, &types.ExecContext{NamedValues: util.ValueToNamedValue([]driver.Value{"generated"})}, nil).(*insertExecutor)
	executor.businesSQLResult = &mockInsertResult{lastInsertID: 42, rowsAffected: 1}
	_, values, err := executor.buildAfterImageSQL(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []driver.Value{int64(42)}, util.NamedValueToValue(values))
}

func TestInsertKeyPlanZeroValueRespectsSQLMode(t *testing.T) {
	meta := &types.TableMeta{
		ColumnNames: []string{"id", "name"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{
				ColumnName: "id", Autoincrement: true,
			}},
		}},
	}

	for _, tt := range []struct {
		name         string
		query        string
		args         []driver.Value
		modePosition driver.Value
		wantPK       driver.Value
		checkSQLMode bool
	}{
		{name: "literal zero generates", query: "insert into user(id,name) values (0,'Tony')", modePosition: 0, wantPK: int64(42), checkSQLMode: true},
		{name: "prepared zero generates", query: "insert into user(id,name) values (?,?)", args: []driver.Value{int64(0), "Tony"}, modePosition: []byte("0"), wantPK: int64(42), checkSQLMode: true},
		{name: "literal zero remains explicit", query: "insert into user(id,name) values (0,'Tony')", modePosition: 1, wantPK: int64(0), checkSQLMode: true},
		{name: "prepared zero remains explicit", query: "insert into user(id,name) values (?,?)", args: []driver.Value{int64(0), "Tony"}, modePosition: []byte("1"), wantPK: int64(0), checkSQLMode: true},
		{name: "prepared false generates", query: "insert into user(id,name) values (?,?)", args: []driver.Value{false, "Tony"}, modePosition: 0, wantPK: int64(42), checkSQLMode: true},
		{name: "prepared false remains explicit", query: "insert into user(id,name) values (?,?)", args: []driver.Value{false, "Tony"}, modePosition: 1, wantPK: int64(0), checkSQLMode: true},
		{name: "prepared true remains explicit", query: "insert into user(id,name) values (?,?)", args: []driver.Value{true, "Tony"}, wantPK: int64(1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parseCtx, err := parser.DoParser(tt.query)
			if !assert.NoError(t, err) {
				return
			}
			conn := mock.NewMockTestDriverConn(gomock.NewController(t))
			if tt.checkSQLMode {
				conn.EXPECT().QueryContext(gomock.Any(), "SELECT FIND_IN_SET('NO_AUTO_VALUE_ON_ZERO', @@SESSION.sql_mode)", gomock.Any()).Return(&insertAfterImageRows{
					columns: []string{"FIND_IN_SET"},
					rows:    [][]driver.Value{{tt.modePosition}},
				}, nil)
			}
			execCtx := &types.ExecContext{
				Conn: conn, NamedValues: util.ValueToNamedValue(tt.args),
			}
			executor := NewInsertExecutor(parseCtx, execCtx, nil).(*insertExecutor)
			executor.businesSQLResult = &mockInsertResult{lastInsertID: 42, rowsAffected: 1}

			executor.keyPlan, err = executor.buildInsertKeyPlan(context.Background(), meta)
			if !assert.NoError(t, err) {
				return
			}
			values, err := executor.resolveInsertKeyPlan(execCtx)
			assert.NoError(t, err)
			assert.Equal(t, []interface{}{tt.wantPK}, values["id"])
		})
	}
}

func TestInsertKeyPlanEmptyValuesUsesLastInsertID(t *testing.T) {
	meta := &types.TableMeta{Indexs: map[string]types.IndexMeta{"PRIMARY": {
		IType: types.IndexTypePrimaryKey,
		Columns: []types.ColumnMeta{{
			ColumnName: "id", Autoincrement: true,
		}},
	}}}

	for _, query := range []string{
		"insert into user () values ()",
		"insert into user values ()",
	} {
		t.Run(query, func(t *testing.T) {
			parseCtx, err := parser.DoParser(query)
			if !assert.NoError(t, err) {
				return
			}
			execCtx := &types.ExecContext{}
			executor := NewInsertExecutor(parseCtx, execCtx, nil).(*insertExecutor)
			executor.businesSQLResult = &mockInsertResult{lastInsertID: 42, rowsAffected: 1}

			executor.keyPlan, err = executor.buildInsertKeyPlan(context.Background(), meta)
			if !assert.NoError(t, err) {
				return
			}
			values, err := executor.resolveInsertKeyPlan(execCtx)
			assert.NoError(t, err)
			assert.Equal(t, []interface{}{int64(42)}, values["id"])
		})
	}

	t.Run("explicit columns remain invalid", func(t *testing.T) {
		parseCtx, err := parser.DoParser("insert into user(id) values ()")
		if !assert.NoError(t, err) {
			return
		}
		executor := NewInsertExecutor(parseCtx, &types.ExecContext{}, nil).(*insertExecutor)

		_, err = executor.buildInsertKeyPlan(context.Background(), meta)

		assert.ErrorContains(t, err, "has 0 values, want 1")
	})
}

func TestBuildSelectSQLByInsertAddsOnlyMissingCompositePKs(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	meta := &types.TableMeta{
		TableName:   "user",
		ColumnNames: []string{"id", "tenant_id", "name"},
		Columns: map[string]types.ColumnMeta{
			"id":        {ColumnName: "id"},
			"tenant_id": {ColumnName: "tenant_id"},
			"name":      {ColumnName: "name"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{
				{ColumnName: "id", DatabaseTypeString: "BIGINT", Autoincrement: true},
				{ColumnName: "tenant_id", DatabaseTypeString: "BIGINT"},
			},
		}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})
	parseCtx, err := parser.DoParser("insert into user(tenant_id,name) values (100,'Tony')")
	assert.NoError(t, err)
	executor := NewInsertExecutor(parseCtx, &types.ExecContext{}, nil).(*insertExecutor)
	executor.businesSQLResult = &mockInsertResult{lastInsertID: 19, rowsAffected: 1}

	sql, values, err := executor.buildAfterImageSQL(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "SELECT tenant_id, name, id FROM user WHERE (`id`,`tenant_id`) IN ((?,?)) ", sql)
	assert.Equal(t, []driver.Value{int64(19), int64(100)}, util.NamedValueToValue(values))
}

func TestInsertExecutorRejectsUnsafeSourcesBeforeBusinessSQL(t *testing.T) {
	meta := &types.TableMeta{
		TableName: "user", ColumnNames: []string{"id", "name"},
		Columns: map[string]types.ColumnMeta{"id": {ColumnName: "id"}, "name": {ColumnName: "name"}},
		Indexs:  map[string]types.IndexMeta{"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})

	for _, query := range []string{
		"insert ignore into user(id,name) values (1,'a'),(2,'b')",
		"insert into user(id,name) select id,name from other_user",
		"insert into user(id,name) values (uuid(),'generated')",
	} {
		parseCtx, err := parser.DoParser(query)
		assert.NoError(t, err)
		called := false
		executor := NewInsertExecutor(parseCtx, &types.ExecContext{Query: query, TxCtx: types.NewTxCtx()}, nil)
		_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
			called = true
			return mockInsertResult{rowsAffected: 1}, nil
		})
		assert.Error(t, err)
		assert.False(t, called)
	}
}

func TestInsertExecutorRejectsIncompleteAfterImage(t *testing.T) {
	for _, tt := range []struct {
		name  string
		query string
		rows  [][]driver.Value
		want  string
	}{
		{
			name:  "empty after image",
			query: "insert into user(id,name) values (1,'a')",
			want:  "has 0 rows, expected 1",
		},
		{
			name:  "missing expected row",
			query: "insert into user(id,name) values (1,'a'),(2,'b')",
			rows:  [][]driver.Value{{int64(1), "a"}},
			want:  "has 1 rows, expected 2",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			txCtx, err := executeInsertWithAfterRows(t, tt.query, tt.rows)

			assert.Error(t, err)
			assert.ErrorContains(t, err, "insert after image for table user")
			assert.ErrorContains(t, err, tt.want)
			assert.Empty(t, txCtx.RoundImages.BeofreImages())
			assert.Empty(t, txCtx.RoundImages.AfterImages())
			assert.Empty(t, txCtx.LockKeys)
		})
	}
}

func TestInsertExecutorRejectsUnexpectedAfterImagePrimaryKey(t *testing.T) {
	for _, tt := range []struct {
		name  string
		query string
		rows  [][]driver.Value
		want  string
	}{
		{
			name:  "unexpected primary key",
			query: "insert into user(id,name) values (1,'a')",
			rows:  [][]driver.Value{{int64(2), "a"}},
			want:  "contains unexpected primary key [2]",
		},
		{
			name:  "duplicate primary key",
			query: "insert into user(id,name) values (1,'a'),(2,'b')",
			rows:  [][]driver.Value{{int64(1), "a"}, {int64(1), "duplicate"}},
			want:  "contains duplicate primary key [1]",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			txCtx, err := executeInsertWithAfterRows(t, tt.query, tt.rows)

			assert.Error(t, err)
			assert.ErrorContains(t, err, "insert after image for table user")
			assert.ErrorContains(t, err, tt.want)
			assert.Empty(t, txCtx.RoundImages.BeofreImages())
			assert.Empty(t, txCtx.RoundImages.AfterImages())
			assert.Empty(t, txCtx.LockKeys)
		})
	}
}

func executeInsertWithAfterRows(t *testing.T, query string, rows [][]driver.Value) (*types.TransactionContext, error) {
	t.Helper()
	meta := &types.TableMeta{
		TableName:   "user",
		ColumnNames: []string{"id", "name"},
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id", DatabaseTypeString: "BIGINT"},
			"name": {ColumnName: "name", DatabaseTypeString: "VARCHAR"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{
				ColumnName: "id", DatabaseTypeString: "BIGINT",
			}},
		}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})

	parseCtx, err := parser.DoParser(query)
	assert.NoError(t, err)
	conn := mock.NewMockTestDriverConn(gomock.NewController(t))
	conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(&insertAfterImageRows{
		columns: []string{"id", "name"},
		rows:    rows,
	}, nil)
	txCtx := types.NewTxCtx()
	executor := NewInsertExecutor(parseCtx, &types.ExecContext{
		Query: query,
		TxCtx: txCtx,
		Conn:  conn,
	}, nil)

	_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
		return mockInsertResult{rowsAffected: int64(len(parseCtx.InsertStmt.Lists))}, nil
	})
	return txCtx, err
}

func TestInsertIgnoreNoOpSkipsAfterImage(t *testing.T) {
	meta := &types.TableMeta{
		TableName: "user", ColumnNames: []string{"id", "name"},
		Columns: map[string]types.ColumnMeta{"id": {ColumnName: "id"}, "name": {ColumnName: "name"}},
		Indexs:  map[string]types.IndexMeta{"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})
	query := "insert ignore into user(id,name) values (?,?)"
	parseCtx, err := parser.DoParser(query)
	assert.NoError(t, err)
	txCtx := types.NewTxCtx()
	executor := NewInsertExecutor(parseCtx, &types.ExecContext{
		Query: query, NamedValues: util.ValueToNamedValue([]driver.Value{1, "existing"}), TxCtx: txCtx,
	}, nil)

	_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
		return mockInsertResult{rowsAffected: 0}, nil
	})
	assert.NoError(t, err)
	assert.Empty(t, txCtx.RoundImages.BeofreImages())
	assert.Empty(t, txCtx.RoundImages.AfterImages())
	assert.Empty(t, txCtx.LockKeys)
}

func TestBuildPostgreSQLReturningInsertSQL(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})

	parseCtx, err := parser.DoParser("INSERT INTO t_user(name, age) VALUES ($1, $2);")
	assert.NoError(t, err)

	executor := NewInsertExecutor(parseCtx, &types.ExecContext{
		Query:  "INSERT INTO t_user(name, age) VALUES ($1, $2);",
		DBType: types.DBTypePostgreSQL,
	}, []exec.SQLHook{})

	meta := &types.TableMeta{
		TableName:   "t_user",
		ColumnNames: []string{"id", "name", "age"},
		Columns: map[string]types.ColumnMeta{
			"id": {
				ColumnName:         "id",
				DatabaseTypeString: "INTEGER",
			},
			"name": {
				ColumnName:         "name",
				DatabaseTypeString: "VARCHAR",
			},
			"age": {
				ColumnName:         "age",
				DatabaseTypeString: "INTEGER",
			},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}

	sql, err := executor.(*insertExecutor).buildPostgreSQLReturningInsertSQL(meta)
	assert.NoError(t, err)
	assert.Equal(t, `INSERT INTO t_user(name, age) VALUES ($1, $2) RETURNING "id", "name", "age"`, sql)
}

func TestBuildPostgreSQLReturningInsertSQLAddsOnlyMissingCompositePKs(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	parseCtx, err := parser.DoParser("INSERT INTO t_user(tenant_id, name) VALUES ($1, $2)")
	assert.NoError(t, err)
	executor := NewInsertExecutor(parseCtx, &types.ExecContext{
		Query:  "INSERT INTO t_user(tenant_id, name) VALUES ($1, $2)",
		DBType: types.DBTypePostgreSQL,
	}, nil)
	meta := &types.TableMeta{
		TableName:   "t_user",
		ColumnNames: []string{"id", "tenant_id", "name"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{
				{ColumnName: "id", Autoincrement: true},
				{ColumnName: "tenant_id"},
			},
		}},
	}

	sql, err := executor.(*insertExecutor).buildPostgreSQLReturningInsertSQL(meta)

	assert.NoError(t, err)
	assert.Equal(t, `INSERT INTO t_user(tenant_id, name) VALUES ($1, $2) RETURNING "tenant_id", "name", "id"`, sql)
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
		{name: "test-escaped-backtick-true", fields: fields{}, args: args{meta: types.TableMeta{
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
					Name: model.CIStr{O: "`id`", L: "`id`"},
				}, {
					Name: model.CIStr{O: "`name`", L: "`name`"},
				}},
			},
		}}, want: true},
		// Issue #702: mixed escaped and unescaped columns
		{name: "test-mixed-escape-true", fields: fields{}, args: args{meta: types.TableMeta{
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
					Name: model.CIStr{O: "`id`", L: "`id`"},
				}},
			},
		}}, want: true},
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
							&test_driver.ParamMarkerExpr{},
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
			datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: &tt.args.meta})

			executor := NewInsertExecutor(tt.args.execCtx.ParseContext, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			got, err := executor.(*insertExecutor).getPkValuesByColumn(context.Background(), tt.args.execCtx)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "getPkValuesByColumn(%v)", tt.args.execCtx)
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
			datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: &tt.args.meta})
			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep
			executor.(*insertExecutor).parserCtx = tt.args.execCtx.ParseContext

			got, err := executor.(*insertExecutor).getPkValuesByAuto(context.Background(), tt.args.execCtx)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "getPkValuesByAuto(%v)", tt.args.execCtx)
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
		{name: "query auto increment step", fields: fields{
			IncrementStep: 0,
		}, args: args{
			execCtx: &types.ExecContext{
				Conn: &autoIncrementStepConn{value: []byte("2")},
			},
			autoColumnName: "id",
			lastInsetId:    100,
			updateCount:    3,
		}, want: map[string][]interface{}{
			"id": {int64(100), int64(102), int64(104)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: &tt.args.meta})

			executor := NewInsertExecutor(nil, &types.ExecContext{}, []exec.SQLHook{})
			executor.(*insertExecutor).businesSQLResult = tt.fields.InsertResult
			executor.(*insertExecutor).incrementStep = tt.fields.IncrementStep

			got, err := executor.(*insertExecutor).autoGeneratePks(tt.args.execCtx, tt.args.autoColumnName, tt.args.lastInsetId, tt.args.updateCount)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "autoGeneratePks(%v, %v, %v, %v)", tt.args.execCtx, tt.args.autoColumnName, tt.args.lastInsetId, tt.args.updateCount)
		})
	}
}

func TestCanAutoGeneratePKs_CompositePK(t *testing.T) {
	tests := []struct {
		name      string
		pkMetaMap map[string]types.ColumnMeta
		want      bool
	}{
		{
			name: "composite primary key with one autoincrement column",
			pkMetaMap: map[string]types.ColumnMeta{
				"tenant_id": {ColumnName: "tenant_id", Autoincrement: false},
				"id":        {ColumnName: "id", Autoincrement: true},
			},
			want: true,
		},
		{
			name: "composite primary key without any autoincrement column",
			pkMetaMap: map[string]types.ColumnMeta{
				"group_id": {ColumnName: "group_id", Autoincrement: false},
				"user_id":  {ColumnName: "user_id", Autoincrement: false},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canAutoGeneratePKs(tt.pkMetaMap)
			assert.Equal(t, tt.want, got)
		})
	}
}

type autoIncrementStepConn struct {
	value driver.Value
}

func (c *autoIncrementStepConn) Prepare(query string) (driver.Stmt, error) {
	return &autoIncrementStepStmt{value: c.value}, nil
}

func (c *autoIncrementStepConn) Close() error {
	return nil
}

func (c *autoIncrementStepConn) Begin() (driver.Tx, error) {
	return nil, nil
}

type autoIncrementStepStmt struct {
	value driver.Value
}

func (s *autoIncrementStepStmt) Close() error {
	return nil
}

func (s *autoIncrementStepStmt) NumInput() int {
	return 0
}

func (s *autoIncrementStepStmt) Exec(args []driver.Value) (driver.Result, error) {
	return nil, nil
}

func (s *autoIncrementStepStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &autoIncrementStepRows{value: s.value}, nil
}

type autoIncrementStepRows struct {
	value driver.Value
}

type insertAfterImageRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func (r *insertAfterImageRows) Columns() []string {
	return r.columns
}

func (r *insertAfterImageRows) Close() error {
	return nil
}

func (r *insertAfterImageRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}

func (r *autoIncrementStepRows) Columns() []string {
	return []string{"Variable_name", "Value"}
}

func (r *autoIncrementStepRows) Close() error {
	return nil
}

func (r *autoIncrementStepRows) Next(dest []driver.Value) error {
	dest[0] = "auto_increment_increment"
	dest[1] = r.value
	return nil
}
