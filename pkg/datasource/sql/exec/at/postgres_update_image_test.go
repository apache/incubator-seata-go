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
	"strings"
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestPostgresUpdateBeforeImageUsesResolvedTableIdentity(t *testing.T) {
	previous := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previous })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	meta := postgresUpdateImageMeta()
	for _, test := range []struct {
		name  string
		query string
		key   types.TableMetaKey
		want  string
	}{
		{
			name:  "quoted table",
			query: `UPDATE "Users" SET balance=300 WHERE ID=1`,
			key:   types.TableMetaKey{Schema: "public", TableName: "Users"},
			want:  `SELECT "balance","id" FROM "public"."Users" WHERE "id"=1 FOR UPDATE`,
		},
		{
			name:  "unquoted names still fold",
			query: `UPDATE Users AS U SET balance=300 WHERE U.ID=1`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "balance","id" FROM "public"."users" AS "u" WHERE "u"."id"=1 FOR UPDATE`,
		},
		{
			name:  "escaped schema and table with alias",
			query: `UPDATE "S""ales"."Us""ers" AS u SET balance=300 WHERE u.ID=1`,
			key:   types.TableMetaKey{Schema: `S"ales`, TableName: `Us"ers`},
			want:  `SELECT "balance","id" FROM "S""ales"."Us""ers" AS "u" WHERE "u"."id"=1 FOR UPDATE`,
		},
		{
			name:  "quoted column and predicate",
			query: `UPDATE users SET "Balance"=300 WHERE "Balance"=100 AND ID=1`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "Balance","id" FROM "public"."users" WHERE "Balance"=100 AND "id"=1 FOR UPDATE`,
		},
		{
			name:  "qualified column and quoted alias",
			query: `UPDATE users AS "U" SET "Balance"=300 WHERE "U"."Balance"=100 AND "U".ID=1`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "Balance","id" FROM "public"."users" AS "U" WHERE "U"."Balance"=100 AND "U"."id"=1 FOR UPDATE`,
		},
		{
			name:  "schema qualified column",
			query: `UPDATE public.users SET "Balance"=300 WHERE public.users."Balance"=100`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "Balance","id" FROM "public"."users" WHERE "public"."users"."Balance"=100 FOR UPDATE`,
		},
		{
			name:  "escaped quote and dot in column name",
			query: `UPDATE users SET "Bal""ance.amount"=300 WHERE "Bal""ance.amount"=100`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "Bal""ance.amount","id" FROM "public"."users" WHERE "Bal""ance.amount"=100 FOR UPDATE`,
		},
		{
			name:  "column distinct from primary key by case",
			query: `UPDATE users SET "ID"=300 WHERE id=1`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "ID","id" FROM "public"."users" WHERE "id"=1 FOR UPDATE`,
		},
		{
			name:  "dollar literal remains a string without parameters",
			query: `UPDATE users SET balance=1 WHERE name=$$ALICE$$`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "balance","id" FROM "public"."users" WHERE "name"=$$ALICE$$ FOR UPDATE`,
		},
		{
			name:  "ordinary string alongside dollar literal",
			query: `UPDATE users SET balance=1 WHERE name='ALICE' AND name=$$ALICE$$`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "balance","id" FROM "public"."users" WHERE "name"='ALICE' AND "name"=$$ALICE$$ FOR UPDATE`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := parser.DoParserForDB(test.query, types.DBTypePostgreSQL)
			require.NoError(t, err)
			originalSource := parsed.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource)
			originalTable := originalSource.Source.(*ast.TableName)
			originalSchema, originalName := originalTable.Schema, originalTable.Name
			originalColumn := *parsed.UpdateStmt.List[0].Column
			executor := &updateExecutor{parserCtx: parsed, execContext: &types.ExecContext{
				DBType: types.DBTypePostgreSQL, TableMetaKey: &test.key,
				TableMetaReader: &stubTableMetaCache{meta: meta},
			}}

			query, args, err := executor.buildBeforeImageSQL(context.Background(), nil)

			require.NoError(t, err)
			require.Equal(t, test.want, query)
			require.Empty(t, args)
			require.Same(t, originalSource, parsed.UpdateStmt.TableRefs.TableRefs.Left)
			require.Same(t, originalTable, originalSource.Source)
			require.Equal(t, originalSchema, originalTable.Schema)
			require.Equal(t, originalName, originalTable.Name)
			require.Equal(t, originalColumn, *parsed.UpdateStmt.List[0].Column)
		})
	}
}

func TestPostgresUpdateBeforeImagePreservesDollarStrings(t *testing.T) {
	previous := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previous })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	for _, literal := range []string{
		`$$$$`,
		`$$ALICE$$`,
		`$CaseTag$ALICE$CaseTag$`,
		`$Tag$ALICE $tag$ $Tag$`,
		`$$Dianne's "horse"; /* $9 */ ? _UTF8MB4ALICE$$`,
		`$Tag$literal $1 and $2 with $$ and ' quotes$Tag$`,
		"$$中文\nALICE$$",
	} {
		t.Run(literal, func(t *testing.T) {
			query := `UPDATE users AS "U" SET "Balance"=$1 WHERE "U".Name=` + literal + ` AND "U".ID=$2`
			parsed, err := parser.DoParserForDB(query, types.DBTypePostgreSQL)
			require.NoError(t, err)
			key := types.TableMetaKey{Schema: "public", TableName: "users"}
			executor := &updateExecutor{parserCtx: parsed, execContext: &types.ExecContext{
				DBType: types.DBTypePostgreSQL, TableMetaKey: &key,
				TableMetaReader: &stubTableMetaCache{meta: postgresUpdateImageMeta()},
			}}
			args := []driver.NamedValue{{Ordinal: 1, Value: int64(300)}, {Ordinal: 2, Value: int64(7)}}

			imageSQL, imageArgs, err := executor.buildBeforeImageSQL(context.Background(), args)

			require.NoError(t, err)
			require.Equal(t, `SELECT "Balance","id" FROM "public"."users" AS "U" WHERE "U"."name"=`+literal+` AND "U"."id"=$1 FOR UPDATE`, imageSQL)
			require.Equal(t, []driver.NamedValue{{Ordinal: 1, Value: int64(7)}}, imageArgs)
			require.Equal(t, query, parsed.UpdateStmt.Text())
		})
	}
}

func TestPostgresUpdateDoesNotCaptureBeforeImageFromLowercaseTable(t *testing.T) {
	previous := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previous })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	const query = `UPDATE "Users" SET balance=300 WHERE id=1`
	parsed, err := parser.DoParserForDB(query, types.DBTypePostgreSQL)
	require.NoError(t, err)
	key := types.TableMetaKey{DBName: "app", Schema: "public", TableName: "Users"}
	reader := &tableMetaReaderForTest{key: key, meta: postgresUpdateImageMeta()}
	conn := mock.NewMockTestDriverConn(gomock.NewController(t))
	balance := int64(200)
	conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, sql string, _ []driver.NamedValue) (driver.Rows, error) {
			value := int64(100) // A separate lowercase users table has the same PK.
			if strings.Contains(sql, `FROM "public"."Users"`) {
				value = balance
			}
			return newDeleteRows([]string{"balance", "id"}, []driver.Value{value, int64(1)}), nil
		},
	).Times(2)
	txCtx := types.NewTxCtx()
	execCtx := &types.ExecContext{Query: query, DBType: types.DBTypePostgreSQL, TableMetaReader: reader, Conn: conn, TxCtx: txCtx}
	executor := NewUpdateExecutor(parsed, execCtx, nil)

	_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
		balance = 300
		return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
	})

	require.NoError(t, err)
	before := txCtx.RoundImages.BeofreImages()
	after := txCtx.RoundImages.AfterImages()
	require.Len(t, before, 1)
	require.Len(t, after, 1)
	require.Equal(t, int64(200), before[0].Rows[0].Columns[0].GetActualValue())
	require.Equal(t, int64(300), after[0].Rows[0].Columns[0].GetActualValue())
	require.Equal(t, &key, before[0].TableMetaKey)
	require.Equal(t, &key, after[0].TableMetaKey)
}

func TestPostgresUpdateColumnIdentityProducesUndoAndLock(t *testing.T) {
	previous := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previous })
	for _, test := range []struct {
		name       string
		assignment string
		column     string
		onlyCare   bool
	}{
		{"quoted column", `"Balance"`, "Balance", true},
		{"unquoted column still folds", "BALANCE", "balance", true},
		{"all columns", `"Balance"`, "Balance", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: test.onlyCare})
			query := "UPDATE users SET " + test.assignment + "=$1 WHERE id=$2"
			key := types.TableMetaKey{DBName: "app", Schema: "public", TableName: "users"}
			meta := postgresUpdateImageMeta()
			meta.TableName = "public.users"
			meta.ColumnNames = append(meta.ColumnNames, "Balance")
			meta.Columns["Balance"] = types.ColumnMeta{ColumnName: "Balance", DatabaseTypeString: "BIGINT"}
			values := map[string]int64{"id": 1, "balance": 100, "Balance": 10}
			beforeValue := values[test.column]
			conn := mock.NewMockTestDriverConn(gomock.NewController(t))
			queries := 0
			conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, sql string, args []driver.NamedValue) (driver.Rows, error) {
					queries++
					require.Equal(t, []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}, args)
					require.NotContains(t, sql, `"$1"`)
					if queries == 1 {
						require.Contains(t, sql, "=$1 FOR UPDATE")
					}
					columns := meta.ColumnNames
					if test.onlyCare {
						// An unquoted Balance reads the separate lowercase column.
						column := "balance"
						if strings.HasPrefix(sql, `SELECT "Balance",`) {
							column = "Balance"
						}
						columns = []string{column, "id"}
					} else {
						require.True(t, strings.HasPrefix(sql, "SELECT * FROM "), sql)
					}
					row := make([]driver.Value, 0, len(columns))
					for _, column := range columns {
						row = append(row, values[column])
					}
					return newDeleteRows(columns, row), nil
				},
			).Times(2)
			txCtx := types.NewTxCtx()
			txCtx.TransactionMode = types.ATMode
			execCtx := &types.ExecContext{
				Query: query, DBType: types.DBTypePostgreSQL, Conn: conn, TxCtx: txCtx,
				NamedValues:     []driver.NamedValue{{Ordinal: 1, Value: int64(20)}, {Ordinal: 2, Value: int64(1)}},
				TableMetaReader: &tableMetaReaderForTest{key: key, meta: meta},
			}
			_, err := (&postgresATExecutor{}).ExecWithNamedValue(context.Background(), execCtx,
				func(_ context.Context, sql string, args []driver.NamedValue) (types.ExecResult, error) {
					require.Equal(t, 1, queries)
					require.Equal(t, query, sql)
					require.Equal(t, execCtx.NamedValues, args)
					values[test.column] = 20
					return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
				})
			require.NoError(t, err)
			require.True(t, txCtx.HasUndoLog())
			require.Len(t, txCtx.RoundImages.BeofreImages(), 1)
			require.Len(t, txCtx.RoundImages.AfterImages(), 1)
			before := txCtx.RoundImages.BeofreImages()[0]
			after := txCtx.RoundImages.AfterImages()[0]
			require.Equal(t, beforeValue, before.Rows[0].GetColumnMap()[test.column].GetActualValue())
			require.Equal(t, int64(20), after.Rows[0].GetColumnMap()[test.column].GetActualValue())
			require.Equal(t, &key, before.TableMetaKey)
			require.Equal(t, &key, after.TableMetaKey)
			require.Equal(t, map[string]struct{}{"PUBLIC.USERS:1": {}}, txCtx.LockKeys)
		})
	}
}

func postgresUpdateImageMeta() *types.TableMeta {
	return &types.TableMeta{
		TableName:   "public.Users",
		ColumnNames: []string{"id", "balance"},
		Columns: map[string]types.ColumnMeta{
			"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
			"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
}
