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
			want:  `SELECT balance,id FROM "public"."Users" WHERE ID=1 FOR UPDATE`,
		},
		{
			name:  "unquoted names still fold",
			query: `UPDATE Users AS U SET balance=300 WHERE U.ID=1`,
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT balance,id FROM "public"."users" AS U WHERE U.ID=1 FOR UPDATE`,
		},
		{
			name:  "escaped schema and table with alias",
			query: `UPDATE "S""ales"."Us""ers" AS u SET balance=300 WHERE u.ID=1`,
			key:   types.TableMetaKey{Schema: `S"ales`, TableName: `Us"ers`},
			want:  `SELECT balance,id FROM "S""ales"."Us""ers" AS u WHERE u.ID=1 FOR UPDATE`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := parser.DoParserForDB(test.query, types.DBTypePostgreSQL)
			require.NoError(t, err)
			originalSource := parsed.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource)
			originalTable := originalSource.Source.(*ast.TableName)
			originalSchema, originalName := originalTable.Schema, originalTable.Name
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
