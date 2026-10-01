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
	"fmt"
	"regexp"
	"testing"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	undoexecutor "seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/executor"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	_ "seata.apache.org/seata-go/v2/pkg/util/log"
)

func TestBuildSelectSQLByUpdateJoin(t *testing.T) {
	tableMetas := map[string]*types.TableMeta{
		"table1": {
			TableName: "table1",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"name": {
					ColumnDef:  nil,
					ColumnName: "name",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "name", "age"},
		},
		"table2": {
			TableName: "table2",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"name": {
					ColumnDef:  nil,
					ColumnName: "name",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
				"kk": {
					ColumnDef:  nil,
					ColumnName: "kk",
				},
				"addr": {
					ColumnDef:  nil,
					ColumnName: "addr",
				},
			},
			ColumnNames: []string{"id", "name", "age", "kk", "addr"},
		},
		"table3": {
			TableName: "table3",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "age"},
		},
		"table4": {
			TableName: "table4",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "age"},
		},
	}

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     map[string]string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id and t1.age=? set t1.name = 'WILL',t2.name = ?",
			sourceQueryArgs: []driver.Value{18, "Jack"},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id AND t1.age=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id AND t1.age=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{18},
		},
		{
			sourceQuery:     "update table1 AS t1 inner join table2 AS t2 on t1.id = t2.id set t1.name = 'WILL',t2.name = 'WILL' where t1.id=?",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			sourceQuery:     "update table1 AS t1 right join table2 AS t2 on t1.id = t2.id set t1.name = 'WILL',t2.name = 'WILL' where t1.id=?",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 RIGHT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 RIGHT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			sourceQuery:     "update table1 t1 inner join table2 t2 on t1.id = t2.id set t1.name = ?, t1.age = ? where t1.id = ? and t1.name = ? and t2.age between ? and ?",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, "Jack", 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.age,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? AND t1.name=? AND t2.age BETWEEN ? AND ? GROUP BY t1.name,t1.age,t1.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, "Jack", 18, 28},
		},
		{
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id set t1.name = ?, t1.age = ? where t1.id=? and t2.id is null and t1.age IN (?,?)",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.age,t1.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? AND t2.id IS NULL AND t1.age IN (?,?) GROUP BY t1.name,t1.age,t1.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 18, 28},
		},
		{
			sourceQuery:     "update table1 t1 inner join table2 t2 on t1.id = t2.id set t1.name = ?, t2.age = ? where t2.kk between ? and ? and t2.addr in(?,?) and t2.age > ? order by t1.name desc limit ?",
			sourceQueryArgs: []driver.Value{"Jack", 18, 10, 20, "Beijing", "Guangzhou", 18, 2},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t2.kk BETWEEN ? AND ? AND t2.addr IN (?,?) AND t2.age>? GROUP BY t1.name,t1.id ORDER BY t1.name DESC LIMIT ? FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.age,t2.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t2.kk BETWEEN ? AND ? AND t2.addr IN (?,?) AND t2.age>? GROUP BY t2.age,t2.id ORDER BY t1.name DESC LIMIT ? FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{10, 20, "Beijing", "Guangzhou", 18, 2},
		},
		{
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id inner join table3 t3 on t3.id = t2.id right join table4 t4 on t4.id = t2.id set t1.name = ?,t2.name = ? where t1.id=? and t3.age=? and t4.age>30",
			sourceQueryArgs: []driver.Value{"Jack", "WILL", 1, 10},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM ((table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id) JOIN table3 AS t3 ON t3.id=t2.id) RIGHT JOIN table4 AS t4 ON t4.id=t2.id WHERE t1.id=? AND t3.age=? AND t4.age>30 GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM ((table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id) JOIN table3 AS t3 ON t3.id=t2.id) RIGHT JOIN table4 AS t4 ON t4.id=t2.id WHERE t1.id=? AND t3.age=? AND t4.age>30 GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery)
			assert.Nil(t, err)
			executor := NewUpdateJoinExecutor(c, &types.ExecContext{Values: tt.sourceQueryArgs, NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs)}, []exec.SQLHook{})
			tableNames := executor.(*updateJoinExecutor).parseTableName(c.UpdateStmt.TableRefs.TableRefs)
			for _, table := range tableNames {
				tbName := table.ref.TableName
				query, args, err := executor.(*updateJoinExecutor).buildBeforeImageSQL(context.Background(), tableMetas[tbName], table, util.ValueToNamedValue(tt.sourceQueryArgs))
				assert.Nil(t, err)
				if query == "" {
					continue
				}
				assert.Equal(t, tt.expectQuery[tbName], query)
				assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
			}
		})
	}
}

func TestUpdateJoinKeepsSameNamedTablesSeparate(t *testing.T) {
	parsed, err := parser.DoParser("UPDATE one.users a JOIN two.users b ON a.id=b.id SET a.id=1")
	assert.NoError(t, err)
	executor := NewUpdateJoinExecutor(parsed, &types.ExecContext{}, nil).(*updateJoinExecutor)
	tables := executor.parseTableName(parsed.UpdateStmt.TableRefs.TableRefs)
	assert.Equal(t, []joinTableRef{
		{ref: types.TableRef{Qualifier: "one", TableName: "users"}, alias: "a"},
		{ref: types.TableRef{Qualifier: "two", TableName: "users"}, alias: "b"},
	}, tables)
}

func TestUpdateJoinSameNamedTablesWithoutAliasesUsesQualifiedColumns(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	parsed, err := parser.DoParser("UPDATE one.users JOIN two.users ON one.users.id=two.users.id SET one.users.name='A',two.users.name='B'")
	if !assert.NoError(t, err) {
		return
	}
	executor := NewUpdateJoinExecutor(parsed, &types.ExecContext{}, nil).(*updateJoinExecutor)
	tables := executor.parseTableName(parsed.UpdateStmt.TableRefs.TableRefs)
	assert.Len(t, tables, 2)
	meta := &types.TableMeta{TableName: "users", Indexs: map[string]types.IndexMeta{
		"id": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
	}}
	for _, table := range tables {
		query, _, err := executor.buildBeforeImageSQL(context.Background(), meta, table, nil)
		assert.NoError(t, err)
		qualified := table.ref.Qualifier + ".users"
		assert.Contains(t, query, "SELECT SQL_NO_CACHE "+qualified+".name,"+qualified+".id FROM")
		assert.Contains(t, query, " GROUP BY "+qualified+".name,"+qualified+".id FOR UPDATE")
		assert.NotContains(t, query, "SELECT SQL_NO_CACHE one.users.name,two.users.name")

		before := types.RecordImage{Rows: []types.RowImage{{Columns: []types.ColumnImage{{ColumnName: "id", Value: int64(1)}}}}}
		afterQuery, _, err := executor.buildAfterImageSQL(context.Background(), before, meta, table)
		assert.NoError(t, err)
		assert.Contains(t, afterQuery, qualified+".name,"+qualified+".id FROM")
		assert.Contains(t, afterQuery, " GROUP BY "+qualified+".name,"+qualified+".id")
	}

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})
	for _, table := range tables {
		query, _, err := executor.buildBeforeImageSQL(context.Background(), meta, table, nil)
		assert.NoError(t, err)
		assert.Contains(t, query, "SELECT SQL_NO_CACHE "+table.ref.Qualifier+".users.* FROM")
	}
}

func TestUpdateJoinCaseSensitiveIdentityKeepsImagesAndUndoSeparate(t *testing.T) {
	previousUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })

	cases := []struct {
		name      string
		tables    string
		selectors [2]string
		refs      [2]types.TableRef
	}{
		{
			name: "aliases", tables: "t1 a JOIN t2 A ON a.id=A.id",
			selectors: [2]string{"a", "A"},
			refs:      [2]types.TableRef{{TableName: "t1"}, {TableName: "t2"}},
		},
		{
			name: "tables", tables: "users JOIN Users ON users.id=Users.id",
			selectors: [2]string{"users", "Users"},
			refs:      [2]types.TableRef{{TableName: "users"}, {TableName: "Users"}},
		},
		{
			name: "databases", tables: "one.users JOIN ONE.users ON one.users.id=ONE.users.id",
			selectors: [2]string{"one.users", "ONE.users"},
			refs:      [2]types.TableRef{{Qualifier: "one", TableName: "users"}, {Qualifier: "ONE", TableName: "users"}},
		},
	}
	for _, tc := range cases {
		for _, validate := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/dataValidation=%t", tc.name, validate), func(t *testing.T) {
				undo.UndoConfig = undo.Config{OnlyCareUpdateColumns: true, DataValidation: validate}
				ctx := context.Background()
				db, sqlMock, err := sqlmock.New()
				require.NoError(t, err)
				defer db.Close()
				conn, err := db.Conn(ctx)
				require.NoError(t, err)
				defer conn.Close()

				reader := mock.NewMockTableMetaCache(gomock.NewController(t))
				var keys [2]types.TableMetaKey
				var metas [2]*types.TableMeta
				oldBalances, newBalances := [2]int64{10, 20}, [2]int64{100, 200}
				for i, ref := range tc.refs {
					dbName := ref.Qualifier
					if dbName == "" {
						dbName = "app"
					}
					keys[i] = types.TableMetaKey{DBName: dbName, TableName: ref.TableName}
					metas[i] = &types.TableMeta{
						TableName: ref.TableName, ColumnNames: []string{"id", "balance"},
						Columns: map[string]types.ColumnMeta{
							"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
							"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
						},
						Indexs: map[string]types.IndexMeta{"PRIMARY": {
							IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
						}},
					}
					reader.EXPECT().ResolveTableMetaKey(gomock.Any(), gomock.Any(), ref).Return(keys[i], nil)
					reader.EXPECT().GetTableMeta(gomock.Any(), keys[i]).Return(metas[i], nil).Times(2)
					prefix := "SELECT SQL_NO_CACHE " + tc.selectors[i] + ".balance," + tc.selectors[i] + ".id FROM "
					sqlMock.ExpectQuery("^" + regexp.QuoteMeta(prefix) + ".* FOR UPDATE$").WithArgs(int64(1)).
						WillReturnRows(sqlmock.NewRows([]string{"balance", "id"}).AddRow(oldBalances[i], int64(1)))
				}
				query := fmt.Sprintf("UPDATE %s SET %s.balance=100,%s.balance=200 WHERE %s.id=?",
					tc.tables, tc.selectors[0], tc.selectors[1], tc.selectors[0])
				sqlMock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(int64(1)).WillReturnResult(sqlmock.NewResult(0, 2))
				for i, selector := range tc.selectors {
					prefix := "SELECT SQL_NO_CACHE " + selector + ".balance," + selector + ".id FROM "
					sqlMock.ExpectQuery("^" + regexp.QuoteMeta(prefix) + ".* GROUP BY " + regexp.QuoteMeta(selector+".id") + "$").
						WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"balance", "id"}).AddRow(newBalances[i], int64(1)))
				}

				parsed, err := parser.DoParser(query)
				require.NoError(t, err)
				txCtx := types.NewTxCtx()
				err = conn.Raw(func(raw interface{}) error {
					driverConn := raw.(driver.Conn)
					execCtx := &types.ExecContext{
						Query: query, NamedValues: []driver.NamedValue{{Ordinal: 1, Value: int64(1)}},
						DBType: types.DBTypeMySQL, DbVersion: "8.0.29", Conn: driverConn,
						TxCtx: txCtx, TableMetaReader: reader,
					}
					_, err := NewUpdateJoinExecutor(parsed, execCtx, nil).ExecContext(ctx,
						func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
							result, err := driverConn.(driver.ExecerContext).ExecContext(ctx, query, args)
							return types.NewResult(types.WithResult(result)), err
						})
					return err
				})
				require.NoError(t, err)
				before, after := txCtx.RoundImages.BeofreImages(), txCtx.RoundImages.AfterImages()
				require.Len(t, before, 2)
				require.Len(t, after, 2)
				for i := len(before) - 1; i >= 0; i-- {
					require.Equal(t, &keys[i], before[i].TableMetaKey)
					require.Equal(t, &keys[i], after[i].TableMetaKey)
					require.Len(t, before[i].Rows, 1)
					require.Len(t, after[i].Rows, 1)
					require.Len(t, before[i].Rows[0].Columns, 2)
					require.Len(t, after[i].Rows[0].Columns, 2)
					require.Equal(t, oldBalances[i], before[i].Rows[0].Columns[0].GetActualValue())
					require.Equal(t, newBalances[i], after[i].Rows[0].Columns[0].GetActualValue())

					log := undo.SQLUndoLog{SQLType: types.SQLTypeUpdate, TableName: before[i].TableName,
						TableMetaKey: before[i].TableMetaKey, BeforeImage: before[i], AfterImage: after[i]}
					log.SetTableMeta(metas[i])
					qualified := "`" + keys[i].DBName + "`.`" + keys[i].TableName + "`"
					if validate {
						sqlMock.ExpectQuery("^" + regexp.QuoteMeta("SELECT balance, id FROM "+qualified+" WHERE ") + ".* FOR UPDATE$").
							WithArgs(int64(1)).WillReturnRows(sqlmock.NewRowsWithColumnDefinition(
							sqlmock.NewColumn("balance").OfType("BIGINT", int64(0)),
							sqlmock.NewColumn("id").OfType("BIGINT", int64(0)),
						).AddRow(newBalances[i], int64(1)))
					}
					sqlMock.ExpectPrepare("^"+regexp.QuoteMeta("UPDATE "+qualified+" SET balance = ? WHERE id = ?")+"$").
						WillBeClosed().ExpectExec().WithArgs(oldBalances[i], int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
					rollback := undoexecutor.NewMySQLUndoExecutorHolder().GetUpdateExecutor(log)
					require.NoError(t, rollback.ExecuteOn(ctx, types.DBTypeMySQL, conn))
				}
				require.NoError(t, sqlMock.ExpectationsWereMet())
			})
		}
	}
}
