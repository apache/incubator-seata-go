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
	"github.com/arana-db/parser/format"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestPostgresDeleteAndLockQueriesUseResolvedTableIdentity(t *testing.T) {
	for _, test := range []struct {
		name string
		from string
		key  types.TableMetaKey
		want string
	}{
		{"quoted table", `"Users"`, types.TableMetaKey{Schema: "public", TableName: "Users"}, `"public"."Users"`},
		{"unquoted table still folds", "Users AS U", types.TableMetaKey{Schema: "public", TableName: "users"}, `"public"."users" AS U`},
		{"resolved search path", "users", types.TableMetaKey{Schema: "tenant", TableName: "users"}, `"tenant"."users"`},
		{"escaped names and alias", `"S""ales"."Us""ers" AS u`, types.TableMetaKey{Schema: `S"ales`, TableName: `Us"ers`}, `"S""ales"."Us""ers" AS u`},
	} {
		for _, statement := range []string{"delete", "select for update"} {
			t.Run(test.name+"/"+statement, func(t *testing.T) {
				query := "DELETE FROM " + test.from + " WHERE id=$1"
				if statement == "select for update" {
					query = "SELECT balance FROM " + test.from + " WHERE id=$1 FOR UPDATE"
				}
				parsed, err := parser.DoParserForDB(query, types.DBTypePostgreSQL)
				require.NoError(t, err)
				execCtx := &types.ExecContext{DBType: types.DBTypePostgreSQL, TableMetaKey: &test.key}
				var from *ast.TableRefsClause
				if statement == "delete" {
					from = parsed.DeleteStmt.TableRefs
				} else {
					from = parsed.SelectStmt.From
				}
				originalSource := from.TableRefs.Left.(*ast.TableSource)
				originalTable := originalSource.Source.(*ast.TableName)
				originalSchema, originalName := originalTable.Schema, originalTable.Name
				var got, fields string
				if statement == "delete" {
					args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}
					var gotArgs []driver.NamedValue
					got, gotArgs, err = (&deleteExecutor{parserCtx: parsed, execContext: execCtx}).buildBeforeImageSQL(query, args)
					require.Equal(t, args, gotArgs)
					fields = "*"
				} else {
					got, err = (&selectForUpdateExecutor{execContext: execCtx}).buildSelectPKSQL(parsed.SelectStmt, postgresUpdateImageMeta())
					fields = "id"
				}
				require.NoError(t, err)
				require.Equal(t, "SELECT "+fields+" FROM "+test.want+" WHERE id=$1 FOR UPDATE", got)
				require.Same(t, originalSource, from.TableRefs.Left)
				require.Same(t, originalTable, originalSource.Source)
				require.Equal(t, originalSchema, originalTable.Schema)
				require.Equal(t, originalName, originalTable.Name)
			})
		}
	}
}

func TestPostgresDeleteDoesNotCaptureBeforeImageFromLowercaseTable(t *testing.T) {
	const query = `DELETE FROM "Users" WHERE id=$1`
	parsed, err := parser.DoParserForDB(query, types.DBTypePostgreSQL)
	require.NoError(t, err)
	key := types.TableMetaKey{DBName: "app", Schema: "public", TableName: "Users"}
	reader := &tableMetaReaderForTest{key: key, meta: postgresUpdateImageMeta()}
	conn := mock.NewMockTestDriverConn(gomock.NewController(t))
	args := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}
	conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), args).DoAndReturn(
		func(_ context.Context, sql string, _ []driver.NamedValue) (driver.Rows, error) {
			balance := int64(100) // The lowercase users table has a different row with the same PK.
			if sql == `SELECT * FROM "public"."Users" WHERE id=$1 FOR UPDATE` {
				balance = 200
			}
			return newDeleteRows([]string{"id", "balance"}, []driver.Value{int64(1), balance}), nil
		},
	)
	txCtx := types.NewTxCtx()
	execCtx := &types.ExecContext{
		Query: query, NamedValues: args, DBType: types.DBTypePostgreSQL,
		TableMetaReader: reader, Conn: conn, TxCtx: txCtx,
	}
	called := false
	_, err = NewDeleteExecutor(parsed, execCtx, nil).ExecContext(context.Background(), func(_ context.Context, got string, gotArgs []driver.NamedValue) (types.ExecResult, error) {
		called = true
		require.Equal(t, query, got)
		require.Equal(t, args, gotArgs)
		return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
	})

	require.NoError(t, err)
	require.True(t, called)
	before, after := txCtx.RoundImages.BeofreImages(), txCtx.RoundImages.AfterImages()
	require.Len(t, before, 1)
	require.Len(t, after, 1)
	require.Equal(t, int64(200), before[0].Rows[0].Columns[1].GetActualValue())
	require.Empty(t, after[0].Rows)
	require.Equal(t, &key, before[0].TableMetaKey)
	require.Equal(t, &key, after[0].TableMetaKey)
}

func TestPostgresLockQueryPreservesNestedJoins(t *testing.T) {
	const query = `SELECT u.id FROM "Users" AS u JOIN b ON u.id=b.bid JOIN c ON b.bid=c.cid WHERE u.id=$1 FOR UPDATE`
	parsed, err := parser.DoParserForDB(query, types.DBTypePostgreSQL)
	require.NoError(t, err)
	originalFrom := parsed.SelectStmt.From
	var originalSQL strings.Builder
	require.NoError(t, parsed.SelectStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &originalSQL)))
	executor := &selectForUpdateExecutor{execContext: &types.ExecContext{
		DBType: types.DBTypePostgreSQL, TableMetaKey: &types.TableMetaKey{Schema: "public", TableName: "Users"},
	}}

	got, err := executor.buildSelectPKSQL(parsed.SelectStmt, postgresUpdateImageMeta())

	require.NoError(t, err)
	require.Equal(t, `SELECT id FROM ("public"."Users" AS u JOIN b ON u.id=b.bid) JOIN c ON b.bid=c.cid WHERE u.id=$1 FOR UPDATE`, got)
	var unchangedSQL strings.Builder
	require.NoError(t, parsed.SelectStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, &unchangedSQL)))
	require.Equal(t, originalSQL.String(), unchangedSQL.String())
	require.Same(t, originalFrom, parsed.SelectStmt.From)
}
