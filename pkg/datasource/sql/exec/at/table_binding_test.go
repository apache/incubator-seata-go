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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestPlainExecutionDoesNotBindTableReferences(t *testing.T) {
	Init()
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		queries := []string{"TABLE t", "TABLE t ORDER BY id LIMIT 3"}
		if dbType == types.DBTypeMySQL {
			queries = append(queries, "SELECT a.id FROM { OJ a LEFT JOIN b ON a.id=b.id } WHERE a.id=?")
		} else {
			queries = append(queries, `SELECT id FROM "Users" WHERE id=$1`)
		}
		for _, mode := range []types.TransactionMode{types.Local, types.ATMode, types.XAMode} {
			for _, query := range queries {
				t.Run(fmt.Sprintf("%v/%v/%s", dbType, mode, query), func(t *testing.T) {
					e, err := exec.BuildExecutor(dbType, mode, query)
					require.NoError(t, err)
					var args []driver.NamedValue
					if strings.Contains(query, "WHERE") {
						args = []driver.NamedValue{{Ordinal: 1, Value: int64(7)}}
					}
					execCtx := &types.ExecContext{
						DBType: dbType, Query: query, NamedValues: args,
						TxCtx: &types.TransactionContext{TransactionMode: mode},
					}
					calls := 0
					result, err := e.ExecWithNamedValue(context.Background(), execCtx, func(_ context.Context, got string, values []driver.NamedValue) (types.ExecResult, error) {
						calls++
						require.Equal(t, query, got)
						require.Equal(t, args, values)
						return &mockExecResult{rowsAffected: 1}, nil
					})
					require.NoError(t, err)
					require.Equal(t, &mockExecResult{rowsAffected: 1}, result)
					require.Equal(t, 1, calls)
					require.Nil(t, execCtx.TableMetaKey)
				})
			}
		}
	}
}

func TestLocalExecutionLeavesSyntaxValidationToDriver(t *testing.T) {
	Init()
	driverErr := errors.New("driver rejected SQL")
	for _, dbType := range []types.DBType{types.DBTypeMySQL, types.DBTypePostgreSQL} {
		t.Run(fmt.Sprint(dbType), func(t *testing.T) {
			const query = "SELECT FROM"
			e, err := exec.BuildExecutor(dbType, types.Local, query)
			require.NoError(t, err)
			calls := 0
			_, err = e.ExecWithNamedValue(context.Background(), &types.ExecContext{
				DBType: dbType, Query: query, TxCtx: types.NewTxCtx(),
			}, func(_ context.Context, got string, _ []driver.NamedValue) (types.ExecResult, error) {
				calls++
				require.Equal(t, query, got)
				return nil, driverErr
			})
			require.ErrorIs(t, err, driverErr)
			require.Equal(t, 1, calls)
		})
	}
}

func TestATTableBindingRoutesODBCStatements(t *testing.T) {
	Init()
	for _, tc := range []struct {
		name         string
		query        string
		mode         types.TransactionMode
		globalLock   bool
		executorType types.ExecutorType
		want         [][]types.TableRef
	}{
		{"ordinary write control", "UPDATE a LEFT JOIN b ON a.id=b.id SET a.balance=1", types.ATMode, false, types.UpdateExecutor, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"ordinary locking read control", "SELECT a.id FROM a LEFT JOIN b ON a.id=b.id FOR UPDATE", types.ATMode, false, types.SelectForUpdateExecutor, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"ordinary global lock control", "SELECT a.id FROM a LEFT JOIN b ON a.id=b.id FOR UPDATE", types.Local, true, types.SelectForUpdateExecutor, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"ordinary later statement control", "UPDATE a SET balance=1; UPDATE a LEFT JOIN b ON a.id=b.id SET a.balance=2", types.ATMode, false, types.MultiExecutor, [][]types.TableRef{{{TableName: "a"}}, {{TableName: "a"}, {TableName: "b"}}}},
		{"write", "UPDATE { OJ a LEFT JOIN b ON a.id=b.id } SET a.balance=1", types.ATMode, false, types.UpdateExecutor, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"locking read", "SELECT a.id FROM { OJ a LEFT JOIN b ON a.id=b.id } FOR UPDATE", types.ATMode, false, types.SelectForUpdateExecutor, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"global lock", "SELECT a.id FROM { OJ a LEFT JOIN b ON a.id=b.id } FOR UPDATE", types.Local, true, types.SelectForUpdateExecutor, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"later statement", "UPDATE a SET balance=1; UPDATE { OJ a LEFT JOIN b ON a.id=b.id } SET a.balance=2", types.ATMode, false, types.MultiExecutor, [][]types.TableRef{{{TableName: "a"}}, {{TableName: "a"}, {TableName: "b"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			originalUpdate, originalSelect, originalMulti := newUpdateExecutor, newSelectForUpdateExecutor, newMultiExecutor
			t.Cleanup(func() {
				newUpdateExecutor, newSelectForUpdateExecutor, newMultiExecutor = originalUpdate, originalSelect, originalMulti
			})
			factoryCalls := 0
			// Only replace execution after the real parser and binder have run.
			// This checks binding and routing, not multi-SQL JOIN execution support.
			factory := func(kind types.ExecutorType) func(*types.ParseContext, *types.ExecContext, []exec.SQLHook) executor {
				return func(parsed *types.ParseContext, execCtx *types.ExecContext, _ []exec.SQLHook) executor {
					factoryCalls++
					require.Equal(t, tc.executorType, kind)
					statements := parsed.MultiStmt
					if len(statements) == 0 {
						statements = []*types.ParseContext{parsed}
					}
					require.Len(t, statements, len(tc.want))
					for i, statement := range statements {
						var clause *ast.TableRefsClause
						if statement.UpdateStmt != nil {
							clause = statement.UpdateStmt.TableRefs
						} else {
							clause = statement.SelectStmt.From
						}
						left := clause.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName)
						want := map[*ast.TableName]types.TableRef{left: tc.want[i][0]}
						if len(tc.want[i]) == 2 {
							right := clause.TableRefs.Right.(*ast.TableSource).Source.(*ast.TableName)
							want[right] = tc.want[i][1]
						}
						require.Equal(t, want, statement.TableRefs)
					}
					return &mockExecutor{execContextFunc: func(ctx context.Context, callback exec.CallbackWithNamedValue) (types.ExecResult, error) {
						return callback(ctx, execCtx.Query, execCtx.NamedValues)
					}}
				}
			}
			newUpdateExecutor = factory(types.UpdateExecutor)
			newSelectForUpdateExecutor = factory(types.SelectForUpdateExecutor)
			newMultiExecutor = factory(types.MultiExecutor)
			e, err := exec.BuildExecutor(types.DBTypeMySQL, tc.mode, tc.query)
			require.NoError(t, err, "classification must not bind tables")
			calls := 0
			result, err := e.ExecWithNamedValue(context.Background(), &types.ExecContext{
				DBType: types.DBTypeMySQL, Query: tc.query,
				TxCtx: &types.TransactionContext{TransactionMode: tc.mode}, IsRequireGlobalLock: tc.globalLock,
			}, func(_ context.Context, got string, _ []driver.NamedValue) (types.ExecResult, error) {
				calls++
				require.Equal(t, tc.query, got)
				return &mockExecResult{rowsAffected: 1}, nil
			})
			require.NoError(t, err)
			require.Equal(t, &mockExecResult{rowsAffected: 1}, result)
			require.Equal(t, 1, factoryCalls)
			require.Equal(t, 1, calls)
		})
	}
}

func TestATExecutionRejectsMismatchedTableBindingsBeforeBusinessSQL(t *testing.T) {
	Init()
	for _, tc := range []struct {
		name       string
		query      string
		mode       types.TransactionMode
		globalLock bool
		mismatchAt int
	}{
		{"write", "UPDATE a SET balance=1", types.ATMode, false, 0},
		{"locking read", "SELECT id FROM a FOR UPDATE", types.ATMode, false, 0},
		{"global lock", "SELECT id FROM a FOR UPDATE", types.Local, true, 0},
		{"later statement", "UPDATE a SET balance=1; UPDATE b SET balance=2; UPDATE c SET balance=3", types.ATMode, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := parser.ParseSQLForDB(tc.query, types.DBTypeMySQL)
			require.NoError(t, err)
			statements := parsed.MultiStmt
			if len(statements) == 0 {
				statements = []*types.ParseContext{parsed}
			}
			mismatched := statements[tc.mismatchAt]
			if mismatched.UpdateStmt != nil {
				mismatched.UpdateStmt.SetText(nil, "UPDATE other_table SET balance=1")
			} else {
				mismatched.SelectStmt.SetText(nil, "SELECT id FROM other_table FOR UPDATE")
			}
			originalParse := parseSQLQuery
			originalPlain, originalUpdate, originalSelect, originalMulti := newPlainExecutor, newUpdateExecutor, newSelectForUpdateExecutor, newMultiExecutor
			t.Cleanup(func() {
				parseSQLQuery = originalParse
				newPlainExecutor, newUpdateExecutor, newSelectForUpdateExecutor, newMultiExecutor = originalPlain, originalUpdate, originalSelect, originalMulti
			})
			parseSQLQuery = func(query string) (*types.ParseContext, error) {
				require.Equal(t, tc.query, query)
				return parsed, nil
			}
			factoryCalls := 0
			factory := func(_ *types.ParseContext, execCtx *types.ExecContext, _ []exec.SQLHook) executor {
				factoryCalls++
				return &mockExecutor{execContextFunc: func(ctx context.Context, callback exec.CallbackWithNamedValue) (types.ExecResult, error) {
					return callback(ctx, execCtx.Query, execCtx.NamedValues)
				}}
			}
			newUpdateExecutor, newSelectForUpdateExecutor, newMultiExecutor = factory, factory, factory
			newPlainExecutor = func(parsed *types.ParseContext, execCtx *types.ExecContext) executor {
				return factory(parsed, execCtx, nil)
			}
			e, err := exec.BuildExecutor(types.DBTypeMySQL, tc.mode, tc.query)
			require.NoError(t, err)
			calls := 0
			result, err := e.ExecWithNamedValue(context.Background(), &types.ExecContext{
				DBType: types.DBTypeMySQL, Query: tc.query,
				TxCtx: &types.TransactionContext{TransactionMode: tc.mode}, IsRequireGlobalLock: tc.globalLock,
			}, func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
				calls++
				return &mockExecResult{rowsAffected: 1}, nil
			})
			require.ErrorContains(t, err, "does not match AST table")
			require.Nil(t, result)
			require.Zero(t, factoryCalls)
			require.Zero(t, calls)
			for _, statement := range statements[tc.mismatchAt:] {
				require.Nil(t, statement.TableRefs, "failed and unvisited statements must not expose unverified bindings")
			}
		})
	}
}

func TestMetadataResolutionBindsBeforeUsingTableRef(t *testing.T) {
	ctx, err := parser.ParseSQLForDB(`UPDATE "Users" SET id=1`, types.DBTypePostgreSQL)
	require.NoError(t, err)
	require.Nil(t, ctx.TableRefs)
	reader := &tableMetaReaderForTest{key: types.TableMetaKey{DBName: "app", Schema: "public", TableName: "Users"}}
	err = (&baseExecutor{}).resolveTableMetaKey(context.Background(), &types.ExecContext{TableMetaReader: reader}, ctx)
	require.NoError(t, err)
	require.Equal(t, types.TableRef{TableName: "Users", TableNameQuoted: true}, reader.resolved)

	ctx.TableRefs = nil
	ctx.UpdateStmt.SetText(nil, `UPDATE other SET id=1`)
	reader.resolved = types.TableRef{}
	err = (&baseExecutor{}).resolveTableMetaKey(context.Background(), &types.ExecContext{TableMetaReader: reader}, ctx)
	require.ErrorContains(t, err, "cannot bind")
	require.Empty(t, reader.resolved, "invalid identity must not reach metadata resolution")
}

func TestATExecutionAcceptsMySQLExecutableComments(t *testing.T) {
	previous := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previous })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})
	for _, query := range []string{
		"/*!40101 UPDATE account SET balance=200 WHERE id=1 */",
		"UPDATE /*!40101 account */ SET balance=200 WHERE id=1",
	} {
		t.Run(query, func(t *testing.T) {
			key := types.TableMetaKey{DBName: "app", TableName: "account"}
			meta := &types.TableMeta{
				TableName: "account", ColumnNames: []string{"id", "balance"},
				Columns: map[string]types.ColumnMeta{
					"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
					"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
				},
				Indexs: map[string]types.IndexMeta{"PRIMARY": {
					IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
				}},
			}
			reader := &tableMetaReaderForTest{key: key, meta: meta}
			conn := mock.NewMockTestDriverConn(gomock.NewController(t))
			balance := int64(100)
			conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, sql string, _ []driver.NamedValue) (driver.Rows, error) {
					require.Contains(t, sql, "account")
					return newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), balance}), nil
				},
			).Times(2)
			txCtx := types.NewTxCtx()
			txCtx.TransactionMode = types.ATMode
			execCtx := &types.ExecContext{Query: query, DBType: types.DBTypeMySQL, TableMetaReader: reader, Conn: conn, TxCtx: txCtx}
			called := false
			_, err := (&ATExecutor{}).ExecWithNamedValue(context.Background(), execCtx, func(_ context.Context, got string, _ []driver.NamedValue) (types.ExecResult, error) {
				called = true
				require.Equal(t, query, got, "the business SQL retains its executable comment")
				balance = 200
				return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
			})
			require.NoError(t, err)
			require.True(t, called)
			require.Equal(t, types.TableRef{TableName: "account"}, reader.resolved)
			before, after := txCtx.RoundImages.BeofreImages(), txCtx.RoundImages.AfterImages()
			require.Len(t, before, 1)
			require.Len(t, after, 1)
			require.Equal(t, int64(100), before[0].Rows[0].Columns[1].GetActualValue())
			require.Equal(t, int64(200), after[0].Rows[0].Columns[1].GetActualValue())
			require.Equal(t, &key, before[0].TableMetaKey)
			require.Equal(t, &key, after[0].TableMetaKey)
		})
	}
}
