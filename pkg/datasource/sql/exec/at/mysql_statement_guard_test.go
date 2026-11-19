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

	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestMySQLATStatementReadClassification(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		reject string
	}{
		{name: "plain select", query: "SELECT id FROM account WHERE id = ?"},
		{name: "literal and comment", query: "SELECT 'INTO OUTFILE' /* FOR UPDATE */"},
		{name: "DATE literal", query: "SELECT DATE '2026-01-01'"},
		{name: "TIME literal", query: "SELECT TIME '12:34:56'"},
		{name: "TIMESTAMP literal", query: "SELECT TIMESTAMP '2026-01-01 12:34:56'"},
		{name: "ODBC temporal literals", query: "SELECT {d '2026-01-01'}, {t '12:34:56'}, {ts '2026-01-01 12:34:56'}"},
		{name: "native hex conversion", query: "SELECT HEX(255)"},
		{name: "native character conversion", query: "SELECT CHAR(65), CHAR(65 USING utf8mb4)"},
		{name: "native timestamp addition", query: "SELECT TIMESTAMPADD(DAY, 1, DATE '2026-01-01')"},
		{name: "native timestamp difference", query: "SELECT TIMESTAMPDIFF(DAY, '2026-01-01', '2026-01-02')"},
		{name: "extract date part", query: "SELECT EXTRACT(YEAR FROM DATE '2026-01-01')"},
		{name: "native and aggregate functions", query: "SELECT ABS(COALESCE(SUM(balance), 0)) FROM account"},
		{name: "connection reads", query: "SELECT @v, @@autocommit, LAST_INSERT_ID()"},
		{name: "cte", query: "WITH c AS (SELECT id FROM account) SELECT id FROM c"},
		{name: "union", query: "SELECT 1 UNION ALL SELECT 2"},
		{name: "table", query: "TABLE account"},
		{name: "values", query: "VALUES ROW(1)"},
		{name: "show", query: "SHOW VARIABLES WHERE Variable_name = 'autocommit'"},
		{name: "explain select", query: "EXPLAIN FORMAT=JSON SELECT id FROM account"},
		{name: "explain locking select", query: "EXPLAIN SELECT id FROM account FOR UPDATE"},
		{name: "explain update", query: "EXPLAIN UPDATE account SET balance = 100"},
		{name: "explain connection", query: "EXPLAIN FOR CONNECTION 1"},
		{name: "describe", query: "DESCRIBE account"},
		{name: "ddl", query: "CREATE TABLE abandoned (id INT)", reject: "CreateTableStmt"},
		{name: "truncate", query: "TRUNCATE TABLE account", reject: "TruncateTableStmt"},
		{name: "load data", query: "LOAD DATA LOCAL INFILE 'input.csv' INTO TABLE account", reject: "LoadDataStmt"},
		{name: "call", query: "CALL adjust_balance()", reject: "CallStmt"},
		{name: "unclassified set", query: "SET @v = 1", reject: "SetStmt"},
		{name: "outfile", query: "SELECT id FROM account INTO OUTFILE '/tmp/output'", reject: "SELECT INTO"},
		{name: "locking outfile", query: "SELECT id FROM account FOR UPDATE INTO OUTFILE '/tmp/output'", reject: "SELECT INTO"},
		{name: "share", query: "SELECT id FROM account FOR SHARE", reject: "locking clause"},
		{name: "lock in share mode", query: "SELECT id FROM account LOCK IN SHARE MODE", reject: "locking clause"},
		{name: "nowait", query: "SELECT id FROM account FOR UPDATE NOWAIT", reject: "locking clause"},
		{name: "skip locked", query: "SELECT id FROM account FOR UPDATE SKIP LOCKED", reject: "locking clause"},
		{name: "table list", query: "SELECT id FROM account FOR UPDATE OF account", reject: "locking clause"},
		{name: "no table", query: "SELECT 1 FOR UPDATE", reject: "locking clause"},
		{name: "nested locking read", query: "SELECT (SELECT id FROM account FOR UPDATE)", reject: "locking clause"},
		{name: "union locking read", query: "SELECT 1 UNION ALL (SELECT id FROM account FOR UPDATE)", reject: "locking clause"},
		{name: "variable assignment", query: "SELECT @v := 1", reject: "variable assignment"},
		{name: "variable assignment in native function", query: "SELECT HEX(@v := 1)", reject: "variable assignment"},
		{name: "variable assignment in extract", query: "SELECT EXTRACT(YEAR FROM (@v := '2026-01-01'))", reject: "variable assignment"},
		{name: "cte assignment", query: "WITH c AS (SELECT @v := 1) SELECT * FROM c", reject: "variable assignment"},
		{name: "union assignment", query: "SELECT 1 UNION ALL SELECT @v := 1", reject: "variable assignment"},
		{name: "show assignment", query: "SHOW TABLES WHERE @v := 1", reject: "variable assignment"},
		{name: "advisory lock", query: "SELECT GET_LOCK('key', 0)", reject: "GET_LOCK"},
		{name: "last insert id setter", query: "SELECT LAST_INSERT_ID(4)", reject: "LAST_INSERT_ID"},
		{name: "qualified function", query: "SELECT app.abs(1)", reject: "function"},
		{name: "qualified hex function", query: "SELECT app.hex(255)", reject: "function"},
		{name: "function named like a literal marker", query: "SELECT `'tidb``.(dateliteral`('2026-01-01')", reject: "function"},
		{name: "function named like char marker", query: "SELECT char_func(65, NULL)", reject: "function"},
		{name: "char marker with explicit charset string", query: "SELECT char_func(65, 'utf8mb4')", reject: "function"},
		{name: "unknown function", query: "SELECT adjust_balance()", reject: "function"},
		{name: "explain analyze select", query: "EXPLAIN ANALYZE SELECT id FROM account", reject: "EXPLAIN ANALYZE"},
		{name: "explain analyze update", query: "EXPLAIN ANALYZE UPDATE account SET balance = 0", reject: "EXPLAIN ANALYZE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []driver.NamedValue(nil)
			if tt.name == "plain select" {
				args = []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}
			}
			for _, mode := range []types.TransactionMode{types.ATMode, types.Local} {
				calls := 0
				wantResult := &mockExecResult{rowsAffected: 1}
				got, err := (&ATExecutor{}).ExecWithNamedValue(context.Background(), &types.ExecContext{
					Query:       tt.query,
					NamedValues: args,
					TxCtx:       &types.TransactionContext{TransactionMode: mode},
				}, func(_ context.Context, query string, values []driver.NamedValue) (types.ExecResult, error) {
					calls++
					require.Equal(t, tt.query, query)
					require.Equal(t, args, values)
					return wantResult, nil
				})
				if mode == types.ATMode && tt.reject != "" {
					require.ErrorIs(t, err, errMySQLATUnsupportedStatement)
					require.ErrorContains(t, err, tt.reject)
					require.Nil(t, got)
					require.Zero(t, calls)
				} else {
					require.NoError(t, err)
					require.Same(t, wantResult, got)
					require.Equal(t, 1, calls)
				}
			}
		})
	}
}

func TestMySQLATStatementPreservesParserErrors(t *testing.T) {
	for _, query := range []string{
		"SELECT id FROM account INTO DUMPFILE '/tmp/output'",
		"SELECT id INTO @v FROM account",
	} {
		t.Run(query, func(t *testing.T) {
			_, parserErr := parser.ParseSQLForDB(query, types.DBTypeMySQL)
			require.Error(t, parserErr)
			_, err := (&ATExecutor{}).ExecWithNamedValue(context.Background(), &types.ExecContext{
				Query: query,
				TxCtx: &types.TransactionContext{TransactionMode: types.ATMode},
			}, func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
				t.Fatal("unparsed statement reached the business callback")
				return nil, nil
			})
			require.EqualError(t, err, parserErr.Error())
		})
	}
}

func TestMySQLGlobalLockIntentDoesNotApplyATStatementGuard(t *testing.T) {
	calls := 0
	_, err := (&ATExecutor{}).ExecWithNamedValue(context.Background(), &types.ExecContext{
		Query:               "CALL adjust_balance()",
		TxCtx:               types.NewTxCtx(),
		IsRequireGlobalLock: true,
	}, func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
		calls++
		return &mockExecResult{rowsAffected: 1}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}
