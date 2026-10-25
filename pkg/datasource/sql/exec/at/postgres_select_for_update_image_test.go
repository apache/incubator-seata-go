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
	"strings"
	"testing"

	"github.com/arana-db/parser/format"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestPostgresLockQueryPreservesColumnIdentity(t *testing.T) {
	for _, test := range []struct {
		name  string
		query string
		pk    string
		key   types.TableMetaKey
		want  string
	}{
		{
			name:  "quoted predicate distinct from lowercase primary key",
			query: `SELECT balance FROM account WHERE "ID"=$1 FOR UPDATE`,
			pk:    "id",
			key:   types.TableMetaKey{Schema: "public", TableName: "account"},
			want:  `SELECT "id" FROM "public"."account" WHERE "ID"=$1 FOR UPDATE`,
		},
		{
			name:  "unquoted identifiers still fold",
			query: `SELECT BALANCE FROM ACCOUNT WHERE ID=$1 FOR UPDATE`,
			pk:    "id",
			key:   types.TableMetaKey{Schema: "public", TableName: "account"},
			want:  `SELECT "id" FROM "public"."account" WHERE "id"=$1 FOR UPDATE`,
		},
		{
			name:  "quoted metadata primary key",
			query: `SELECT balance FROM account WHERE id=$1 FOR UPDATE`,
			pk:    "ID",
			key:   types.TableMetaKey{Schema: "public", TableName: "account"},
			want:  `SELECT "ID" FROM "public"."account" WHERE "id"=$1 FOR UPDATE`,
		},
		{
			name:  "unquoted alias still folds",
			query: `SELECT U.balance FROM account AS U WHERE U.ID=$1 FOR UPDATE`,
			pk:    "id",
			key:   types.TableMetaKey{Schema: "public", TableName: "account"},
			want:  `SELECT "id" FROM "public"."account" AS "u" WHERE "u"."id"=$1 FOR UPDATE`,
		},
		{
			name:  "quoted alias and predicate",
			query: `SELECT "U".balance FROM account AS "U" WHERE "U"."ID"=$1 FOR UPDATE`,
			pk:    "id",
			key:   types.TableMetaKey{Schema: "public", TableName: "account"},
			want:  `SELECT "id" FROM "public"."account" AS "U" WHERE "U"."ID"=$1 FOR UPDATE`,
		},
		{
			name:  "escaped schema table alias and column names",
			query: `SELECT balance FROM "S""ales"."Ac""count" AS "A""cc" WHERE "A""cc"."I""D"=$1 FOR UPDATE`,
			pk:    `I"D`,
			key:   types.TableMetaKey{Schema: `S"ales`, TableName: `Ac"count`},
			want:  `SELECT "I""D" FROM "S""ales"."Ac""count" AS "A""cc" WHERE "A""cc"."I""D"=$1 FOR UPDATE`,
		},
		{
			name:  "parameter token distinct from quoted identifier",
			query: `SELECT balance FROM account WHERE "$1"=$1 AND ID=$2 FOR UPDATE`,
			pk:    "id",
			key:   types.TableMetaKey{Schema: "public", TableName: "account"},
			want:  `SELECT "id" FROM "public"."account" WHERE "$1"=$1 AND "id"=$2 FOR UPDATE`,
		},
		{
			name:  "untagged dollar string bypasses charset rewriting",
			query: `SELECT id FROM users WHERE name=$$prefix _UTF8MB4foo suffix$$ FOR UPDATE`,
			pk:    "id",
			key:   types.TableMetaKey{Schema: "public", TableName: "users"},
			want:  `SELECT "id" FROM "public"."users" WHERE "name"=$$prefix _UTF8MB4foo suffix$$ FOR UPDATE`,
		},
		{
			name:  "ordinary and dollar strings preserve their contents",
			query: `SELECT balance FROM account WHERE NAME='Dianne''s "horse"' AND NAME=$Tag$Dianne's "horse" $1 ? _UTF8MB4ALICE$Tag$ AND "ID"=$1 FOR UPDATE`,
			pk:    "id",
			key:   types.TableMetaKey{Schema: "public", TableName: "account"},
			want:  `SELECT "id" FROM "public"."account" WHERE "name"='Dianne''s "horse"' AND "name"=$Tag$Dianne's "horse" $1 ? _UTF8MB4ALICE$Tag$ AND "ID"=$1 FOR UPDATE`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := parser.DoParserForDB(test.query, types.DBTypePostgreSQL)
			require.NoError(t, err)
			originalFrom, originalWhere := parsed.SelectStmt.From, parsed.SelectStmt.Where
			var originalSQL strings.Builder
			restoreFlags := format.RestoreKeyWordUppercase | format.RestoreNameDoubleQuotes | format.RestoreStringSingleQuotes
			require.NoError(t, parsed.SelectStmt.Restore(format.NewRestoreCtx(restoreFlags, &originalSQL)))
			meta := &types.TableMeta{
				TableName: test.key.TableName,
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: test.pk,
						Columns:    []types.ColumnMeta{{ColumnName: test.pk}},
					},
				},
			}
			executor := &selectForUpdateExecutor{execContext: &types.ExecContext{
				DBType: types.DBTypePostgreSQL, TableMetaKey: &test.key,
			}}

			got, err := executor.buildSelectPKSQL(parsed.SelectStmt, meta)

			require.NoError(t, err)
			require.Equal(t, test.want, got)
			var unchangedSQL strings.Builder
			require.NoError(t, parsed.SelectStmt.Restore(format.NewRestoreCtx(restoreFlags, &unchangedSQL)))
			require.Equal(t, originalSQL.String(), unchangedSQL.String())
			require.Same(t, originalFrom, parsed.SelectStmt.From)
			require.Same(t, originalWhere, parsed.SelectStmt.Where)
			require.Equal(t, test.query, parsed.SelectStmt.Text())
		})
	}
}
