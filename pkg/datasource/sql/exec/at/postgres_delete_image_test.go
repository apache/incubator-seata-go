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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestPostgresDeleteColumnIdentityProducesUndoAndLock(t *testing.T) {
	for _, test := range []struct {
		name      string
		predicate string
		column    string
		value     int64
		pk        string
		wantLock  string
	}{
		{"quoted column", `"ID"`, "ID", 1, "id", "PUBLIC.ACCOUNT:2"},
		{"unquoted column folds", `ID`, "id", 2, "id", "PUBLIC.ACCOUNT:2"},
		{"quoted primary key", `"ID"`, "ID", 1, "ID", "PUBLIC.ACCOUNT:1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			query := `DELETE FROM account WHERE ` + test.predicate + `=$1`
			parsed, err := parser.DoParserForDB(query, types.DBTypePostgreSQL)
			require.NoError(t, err)
			key := types.TableMetaKey{DBName: "app", Schema: "public", TableName: "account"}
			meta := &types.TableMeta{
				TableName: "public.account", ColumnNames: []string{"id", "ID"},
				Columns: map[string]types.ColumnMeta{
					"id": {ColumnName: "id", DatabaseTypeString: "BIGINT"},
					"ID": {ColumnName: "ID", DatabaseTypeString: "BIGINT"},
				},
				Indexs: map[string]types.IndexMeta{"PRIMARY": {
					IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: test.pk}},
				}},
			}
			args := []driver.NamedValue{{Ordinal: 1, Value: test.value}}
			conn := mock.NewMockTestDriverConn(gomock.NewController(t))
			queried := false
			conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), args).DoAndReturn(
				func(_ context.Context, sql string, _ []driver.NamedValue) (driver.Rows, error) {
					queried = true
					require.Equal(t, `SELECT * FROM "public"."account" WHERE "`+test.column+`"=$1 FOR UPDATE`, sql)
					return newDeleteRows(meta.ColumnNames, []driver.Value{int64(2), int64(1)}), nil
				},
			)
			txCtx := types.NewTxCtx()
			txCtx.TransactionMode = types.ATMode
			execCtx := &types.ExecContext{
				Query: query, NamedValues: args, DBType: types.DBTypePostgreSQL,
				Conn: conn, TxCtx: txCtx, TableMetaReader: &tableMetaReaderForTest{key: key, meta: meta},
			}
			result, err := (&postgresATExecutor{}).ExecWithNamedValue(context.Background(), execCtx,
				func(_ context.Context, sql string, gotArgs []driver.NamedValue) (types.ExecResult, error) {
					require.True(t, queried)
					require.Equal(t, query, sql)
					require.Equal(t, args, gotArgs)
					return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
				})
			require.NoError(t, err)
			affected, err := result.GetResult().RowsAffected()
			require.NoError(t, err)
			require.Equal(t, int64(1), affected)
			require.True(t, txCtx.HasUndoLog())
			before, after := txCtx.RoundImages.BeofreImages(), txCtx.RoundImages.AfterImages()
			require.Len(t, before, 1)
			require.Len(t, before[0].Rows, 1)
			require.Equal(t, int64(2), before[0].Rows[0].GetColumnMap()["id"].GetActualValue())
			require.Equal(t, int64(1), before[0].Rows[0].GetColumnMap()["ID"].GetActualValue())
			require.Len(t, after, 1)
			require.Empty(t, after[0].Rows)
			require.Equal(t, &key, before[0].TableMetaKey)
			require.Equal(t, &key, after[0].TableMetaKey)
			require.Equal(t, map[string]struct{}{test.wantLock: {}}, txCtx.LockKeys)
			require.Equal(t, query, parsed.DeleteStmt.Text())
		})
	}
}

func TestPostgresDeleteBeforeImagePreservesIdentifiersAndLiterals(t *testing.T) {
	for _, test := range []struct {
		query string
		want  string
	}{
		{`DELETE FROM account AS A WHERE A.ID=$1`, `FROM "public"."account" AS "a" WHERE "a"."id"=$1`},
		{`DELETE FROM account AS "A" WHERE "A"."ID"=$1`, `FROM "public"."account" AS "A" WHERE "A"."ID"=$1`},
		{`DELETE FROM account WHERE public.account."ID"=$1`, `FROM "public"."account" WHERE "public"."account"."ID"=$1`},
		{`DELETE FROM account WHERE "I""D.value"=$1`, `FROM "public"."account" WHERE "I""D.value"=$1`},
		{`DELETE FROM account WHERE "$1"=$1`, `FROM "public"."account" WHERE "$1"=$1`},
		{`DELETE FROM account WHERE Name='ALICE' AND "ID"=$1`, `FROM "public"."account" WHERE "name"='ALICE' AND "ID"=$1`},
		{`DELETE FROM account WHERE Name=$Tag$Dianne's "horse"; $9 ? _UTF8MB4ALICE$Tag$ AND "ID"=$1`, `FROM "public"."account" WHERE "name"=$Tag$Dianne's "horse"; $9 ? _UTF8MB4ALICE$Tag$ AND "ID"=$1`},
	} {
		t.Run(test.query, func(t *testing.T) {
			parsed, err := parser.DoParserForDB(test.query, types.DBTypePostgreSQL)
			require.NoError(t, err)
			originalWhere, originalFrom := parsed.DeleteStmt.Where, parsed.DeleteStmt.TableRefs
			executor := &deleteExecutor{parserCtx: parsed, execContext: &types.ExecContext{
				DBType: types.DBTypePostgreSQL, TableMetaKey: &types.TableMetaKey{Schema: "public", TableName: "account"},
			}}
			args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}
			got, gotArgs, err := executor.buildBeforeImageSQL(test.query, args)
			require.NoError(t, err)
			require.Equal(t, "SELECT * "+test.want+" FOR UPDATE", got)
			require.Equal(t, args, gotArgs)
			require.Same(t, originalWhere, parsed.DeleteStmt.Where)
			require.Same(t, originalFrom, parsed.DeleteStmt.TableRefs)
			require.Equal(t, test.query, parsed.DeleteStmt.Text())
		})
	}
}
