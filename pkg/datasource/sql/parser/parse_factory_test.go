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

package parser

import (
	"context"
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	aparser "github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	"seata.apache.org/seata-go/v2/pkg/util/bytes"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource/postgres"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"

	_ "github.com/arana-db/parser/test_driver"
)

func TestDoParser(t *testing.T) {
	type tt struct {
		sql     string
		sqlType types.SQLType
		types   types.ExecutorType
	}

	for _, t2 := range [...]tt{
		// replace
		{sql: "REPLACE INTO foo VALUES (1 || 2)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES (1 | 2)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES (false || true)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES (bar(5678))", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES ()", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo (a,b) VALUES (42,314)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo () VALUES ()", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUE ()", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO ta TABLE tb", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO t.a TABLE t.b", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		// insert
		{sql: "INSERT INTO foo VALUES (1234)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (1234, 5678)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO t1 (SELECT * FROM t2)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (1 || 2)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (1 | 2)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (false || true)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (bar(5678))", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo (a) VALUES (42)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		// update
		{sql: "UPDATE LOW_PRIORITY IGNORE t SET id = id + 1 ORDER BY id DESC;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t SET id = id + 1 ORDER BY id DESC;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t SET id = id + 1 ORDER BY id DESC limit 3 ;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t SET id = id + 1, name = 'jojo';", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE items,month SET items.price=month.price WHERE items.id=month.id;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE user T0 LEFT OUTER JOIN user_profile T1 ON T1.id = T0.profile_id SET T0.profile_id = 1 WHERE T0.profile_id IN (1);", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t1, t2 set t1.profile_id = 1, t2.profile_id = 1 where ta.a=t.ba", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		// delete
		{sql: "DELETE from t1 where a=1 limit 1", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE FROM t1 WHERE t1.a > 0 ORDER BY t1.a LIMIT 1", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE FROM x.y z WHERE z.a > 0", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE FROM t1 AS w WHERE a > 0", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE from t1 partition (p0,p1)", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "delete low_priority t1, t2 from t1, t2", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "delete quick t1, t2 from t1, t2", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "delete ignore t1, t2 from t1, t2", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
	} {
		parser, err := DoParser(t2.sql)
		assert.NoError(t, err)
		assert.Equal(t, parser.ExecutorType, t2.types)
		assert.Equal(t, parser.SQLType, t2.sqlType)
	}
}

func TestParseSQLForDBSeparatesClassificationFromTableBinding(t *testing.T) {
	for _, tc := range []struct {
		name    string
		query   string
		sqlType types.SQLType
		want    [][]types.TableRef
	}{
		{"table statement", "TABLE t", types.SQLTypeSelect, [][]types.TableRef{{{TableName: "t"}}}},
		{"qualified table with order and limit", "TABLE `Db`.`Us``ers` ORDER BY id, name LIMIT 2, 3", types.SQLTypeSelect, [][]types.TableRef{{{Qualifier: "Db", TableName: "Us`ers", QualifierQuoted: true, TableNameQuoted: true}}}},
		{"parenthesized table", "(TABLE Db.Users)", types.SQLTypeSelect, [][]types.TableRef{{{Qualifier: "Db", TableName: "Users"}}}},
		{"odbc join", "SELECT a.id FROM { OJ a LEFT JOIN b ON a.id=b.id }", types.SQLTypeSelect, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"quoted odbc join with comments and strings", "SELECT '{ OJ ignored }' FROM { oj /* } FROM ignored */ `Db`.`Us``ers` a LEFT JOIN `Other`.`Users` b ON a.name='} JOIN ignored' } ORDER BY a.id, b.id LIMIT 3", types.SQLTypeSelect, [][]types.TableRef{{{Qualifier: "Db", TableName: "Us`ers", QualifierQuoted: true, TableNameQuoted: true}, {Qualifier: "Other", TableName: "Users", QualifierQuoted: true, TableNameQuoted: true}}}},
		{"nested odbc join", "SELECT a.id FROM { OJ a LEFT JOIN ({ OJ b LEFT JOIN c ON b.id=c.id }) ON a.id=b.id }", types.SQLTypeSelect, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}, {TableName: "c"}}}},
		{"odbc join in table list", "SELECT a.id FROM first_table, { OJ a LEFT JOIN b ON a.id=b.id }, last_table", types.SQLTypeSelect, [][]types.TableRef{{{TableName: "first_table"}, {TableName: "a"}, {TableName: "b"}, {TableName: "last_table"}}}},
		{"odbc join with derived select", "SELECT a.id FROM { OJ a LEFT JOIN (SELECT id FROM hidden) b ON a.id=b.id }", types.SQLTypeSelect, [][]types.TableRef{{{TableName: "a"}}}},
		{"odbc join with derived table statement", "SELECT a.id FROM { OJ a LEFT JOIN (TABLE hidden) b ON a.id=b.id }", types.SQLTypeSelect, [][]types.TableRef{{{TableName: "a"}}}},
		{"odbc date in join predicate", "SELECT a.id FROM { OJ a LEFT JOIN b ON a.created_at={d '2026-10-01'} }", types.SQLTypeSelect, [][]types.TableRef{{{TableName: "a"}, {TableName: "b"}}}},
		{"multi statement", "SELECT id FROM users; TABLE t; SELECT a.id FROM { OJ a LEFT JOIN b ON a.id=b.id }", types.SQLTypeMulti, [][]types.TableRef{{{TableName: "users"}}, {{TableName: "t"}}, {{TableName: "a"}, {TableName: "b"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseSQLForDB(tc.query, types.DBTypeMySQL)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, tc.sqlType, parsed.SQLType)
			statements := parsed.MultiStmt
			if len(statements) == 0 {
				statements = []*types.ParseContext{parsed}
			}
			if !assert.Len(t, statements, len(tc.want)) {
				return
			}
			for _, statement := range statements {
				assert.Equal(t, types.SelectExecutor, statement.ExecutorType)
				assert.Nil(t, statement.TableRefs)
			}

			checkBindings := func(parsed *types.ParseContext) {
				statements := parsed.MultiStmt
				if len(statements) == 0 {
					statements = []*types.ParseContext{parsed}
				}
				if !assert.Len(t, statements, len(tc.want)) {
					return
				}
				for i, statement := range statements {
					tables := tableNames(statement)
					if !assert.Len(t, tables, len(tc.want[i])) {
						continue
					}
					assert.Len(t, statement.TableRefs, len(tc.want[i]))
					for j, table := range tables {
						ref, ok := statement.TableRefs[table]
						assert.True(t, ok, "every source AST table must have a bound reference")
						assert.Equal(t, tc.want[i][j], ref)
					}
				}
			}
			if assert.NoError(t, BindTableRefs(parsed)) {
				checkBindings(parsed)
			}
			strict, err := DoParserForDB(tc.query, types.DBTypeMySQL)
			if assert.NoError(t, err) && assert.NotNil(t, strict) {
				checkBindings(strict)
			}
		})
	}
}

func TestBindTableRefsRejectsMismatchedStatementSource(t *testing.T) {
	for _, tc := range []struct {
		name       string
		query      string
		mismatchAt int
	}{
		{"single statement", "SELECT id FROM users", 0},
		{"later statement", "SELECT id FROM first_table; SELECT id FROM users; SELECT id FROM last_table", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseSQLForDB(tc.query, types.DBTypeMySQL)
			if !assert.NoError(t, err) {
				return
			}
			statements := parsed.MultiStmt
			if len(statements) == 0 {
				statements = []*types.ParseContext{parsed}
			}
			statements[tc.mismatchAt].SelectStmt.SetText(nil, "SELECT id FROM other_table")
			assert.ErrorContains(t, BindTableRefs(parsed), "does not match AST table")
			for _, statement := range statements[tc.mismatchAt:] {
				assert.Nil(t, statement.TableRefs, "failed and unvisited statements must not expose unverified bindings")
			}
		})
	}
}

func TestParseSQLForDBStillRejectsInvalidSQL(t *testing.T) {
	parsed, err := ParseSQLForDB("SELECT FROM", types.DBTypeMySQL)
	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestBindTableRefsUsesEachStatementSource(t *testing.T) {
	parsed, err := ParseSQLForDB(`UPDATE Space.Users SET id=1; SELECT id, users FROM "Users" FOR UPDATE`, types.DBTypePostgreSQL)
	if !assert.NoError(t, err) || !assert.Len(t, parsed.MultiStmt, 2) {
		return
	}
	for _, statement := range parsed.MultiStmt {
		assert.Nil(t, statement.TableRefs)
	}
	if !assert.NoError(t, BindTableRefs(parsed)) {
		return
	}
	first, err := parsed.MultiStmt[0].GetTableRef()
	assert.NoError(t, err)
	second, err := parsed.MultiStmt[1].GetTableRef()
	assert.NoError(t, err)
	assert.Equal(t, types.TableRef{Qualifier: "Space", TableName: "Users"}, first)
	assert.Equal(t, types.TableRef{TableName: "Users", TableNameQuoted: true}, second)
}

func TestBindTableRefsRejectsIncompleteExistingBindings(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*types.ParseContext, []*ast.TableName)
	}{
		{"missing table", func(parsed *types.ParseContext, tables []*ast.TableName) {
			delete(parsed.TableRefs, tables[0])
		}},
		{"different node", func(parsed *types.ParseContext, tables []*ast.TableName) {
			parsed.TableRefs[&ast.TableName{}] = parsed.TableRefs[tables[0]]
			delete(parsed.TableRefs, tables[0])
		}},
		{"different identity", func(parsed *types.ParseContext, tables []*ast.TableName) {
			parsed.TableRefs[tables[0]] = types.TableRef{TableName: "Other"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := DoParserForDB(`UPDATE "One" JOIN "Two" ON 1=1 SET id=1`, types.DBTypePostgreSQL)
			if !assert.NoError(t, err) {
				return
			}
			tc.mutate(parsed, tableNames(parsed))
			assert.ErrorContains(t, BindTableRefs(parsed), "cannot bind")
		})
	}
	assert.ErrorContains(t, BindTableRefs(nil), "nil parse context")
}

func TestDoParserForDBPreservesTableIdentifierQuotes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dbType types.DBType
		query  string
		want   types.TableRef
	}{
		{"postgres quoted", types.DBTypePostgreSQL, `UPDATE "Space"."Users" AS u SET id = 1`, types.TableRef{Qualifier: "Space", TableName: "Users", QualifierQuoted: true, TableNameQuoted: true}},
		{"postgres quoted after tab comment", types.DBTypePostgreSQL, "UPDATE --\tUsers\n \"Users\" SET id = 1", types.TableRef{TableName: "Users", TableNameQuoted: true}},
		{"postgres unquoted", types.DBTypePostgreSQL, `UPDATE Space.Users AS u SET id = 1`, types.TableRef{Qualifier: "Space", TableName: "Users"}},
		{"mysql quoted", types.DBTypeMySQL, "INSERT INTO `shop`.`orders` (id) VALUES (1)", types.TableRef{Qualifier: "shop", TableName: "orders", QualifierQuoted: true, TableNameQuoted: true}},
		{"delete alias", types.DBTypeMySQL, "DELETE FROM shop.orders AS o WHERE o.id=1", types.TableRef{Qualifier: "shop", TableName: "orders"}},
		{"delete target alias before table", types.DBTypeMySQL, "DELETE t FROM `t` AS t WHERE t.id=1", types.TableRef{TableName: "t", TableNameQuoted: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := DoParserForDB(tc.query, tc.dbType)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			got, err := parsed.GetTableRef()
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDoParserForDBUsesEachStatementTableRef(t *testing.T) {
	parsed, err := DoParserForDB(`UPDATE "One" SET id=1; UPDATE "Two" SET id=2`, types.DBTypePostgreSQL)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Len(t, parsed.MultiStmt, 2)
	first, err := parsed.MultiStmt[0].GetTableRef()
	assert.NoError(t, err)
	second, err := parsed.MultiStmt[1].GetTableRef()
	assert.NoError(t, err)
	assert.Equal(t, types.TableRef{TableName: "One", TableNameQuoted: true}, first)
	assert.Equal(t, types.TableRef{TableName: "Two", TableNameQuoted: true}, second)
}

func TestDoParserForDBBindsMySQLExecutableComments(t *testing.T) {
	qualified := types.TableRef{Qualifier: "shop", TableName: "Orders", QualifierQuoted: true, TableNameQuoted: true}
	for _, tc := range []struct {
		name  string
		query string
		want  types.TableRef
	}{
		{"complete versioned statement", "/*!40101 UPDATE t SET balance=2 WHERE id=1 */", types.TableRef{TableName: "t"}},
		{"complete unversioned statement", "/*! UPDATE `shop`.`Orders` SET balance=2 WHERE id=1 */", qualified},
		{"versioned table reference", "UPDATE /*!40101 `shop`.`Orders` */ SET balance=2 WHERE id=1", qualified},
		{"unversioned table reference", "UPDATE /*! `shop`.`Orders` */ SET balance=2 WHERE id=1", qualified},
		{"version is not filtered by parser", "UPDATE /*!99999 `shop`.`Orders` */ SET balance=2 WHERE id=1", qualified},
		{"short digits belong to identifier", "UPDATE /*!1234t*/ SET balance=2 WHERE id=1", types.TableRef{TableName: "1234t"}},
		{"five digit prefix before identifier", "UPDATE /*!401011234t*/ SET balance=2 WHERE id=1", types.TableRef{TableName: "1234t"}},
		{"split qualified reference", "UPDATE /*!40101 `shop`. */ `Orders` SET balance=2 WHERE id=1", qualified},
		{"ordinary comment remains ignored", "UPDATE /* other.table */ `shop`.`Orders` SET balance=2 WHERE id=1", qualified},
		{"comment terminator in string", "/*! UPDATE `shop`.`Orders` SET name='*/' WHERE id=1 */", qualified},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := DoParserForDB(tc.query, types.DBTypeMySQL)
			if !assert.NoError(t, err) {
				return
			}
			ref, err := parsed.GetTableRef()
			assert.NoError(t, err)
			assert.Equal(t, tc.want, ref)
		})
	}
}

func TestBindTableRefsDefaultsToMySQLCommentSyntax(t *testing.T) {
	for _, dbType := range []types.DBType{0, types.DBTypeUnknown} {
		parsed, err := DoParserForDB("/*! UPDATE t SET balance=1 */", dbType)
		if !assert.NoError(t, err) {
			continue
		}
		ref, err := parsed.GetTableRef()
		assert.NoError(t, err)
		assert.Equal(t, types.TableRef{TableName: "t"}, ref)
	}
}

func TestDoParserForDBKeepsExecutableCommentsWithinEachStatement(t *testing.T) {
	parsed, err := DoParserForDB("/*!40101 UPDATE `one`.`Users` SET balance=1 */; UPDATE /*! `two`.`Users` */ SET balance=2", types.DBTypeMySQL)
	if !assert.NoError(t, err) || !assert.Len(t, parsed.MultiStmt, 2) {
		return
	}
	for i, qualifier := range []string{"one", "two"} {
		ref, err := parsed.MultiStmt[i].GetTableRef()
		assert.NoError(t, err)
		assert.Equal(t, types.TableRef{Qualifier: qualifier, TableName: "Users", QualifierQuoted: true, TableNameQuoted: true}, ref)
	}
}

func TestBindTableRefsDoesNotExpandPostgresExecutableComments(t *testing.T) {
	for _, query := range []string{
		`/*!40101 UPDATE "Users" SET id=1 */`,
		`UPDATE /*! "public". */ "Users" SET id=1`,
		`UPDATE "Users" SET id=1; UPDATE /*! "public". */ "Users" SET id=2`,
	} {
		t.Run(query, func(t *testing.T) {
			// The shared parser expands MySQL comments even in ANSI_QUOTES mode.
			// PostgreSQL treats them as ordinary comments, so binding must still fail.
			parsed, err := ParseSQLForDB(query, types.DBTypePostgreSQL)
			if !assert.NoError(t, err) {
				return
			}
			assert.Error(t, BindTableRefs(parsed))
			strict, err := DoParserForDB(query, types.DBTypePostgreSQL)
			assert.Error(t, err)
			assert.Nil(t, strict, "strict parsing must not expose a context after binding fails")
		})
	}
}

func TestDoParserForDBBindsOnlyStatementTableReferences(t *testing.T) {
	quotedUsers := types.TableRef{TableName: "Users", TableNameQuoted: true}
	plainUsers := types.TableRef{TableName: "users"}
	for _, tc := range []struct {
		name  string
		query string
		want  []types.TableRef
	}{
		{"projection comma", `SELECT id, users FROM "Users" FOR UPDATE`, []types.TableRef{quotedUsers}},
		{"function arguments and strings", `SELECT COALESCE(id, users), 'FROM users, users' FROM "Users" FOR UPDATE`, []types.TableRef{quotedUsers}},
		{"projection subquery", `SELECT (SELECT users FROM users), users FROM "Users" FOR UPDATE`, []types.TableRef{quotedUsers}},
		{"predicate subquery", `SELECT id FROM "Users" WHERE id IN (SELECT id FROM users) FOR UPDATE`, []types.TableRef{quotedUsers}},
		{"derived query", `SELECT u.id FROM (SELECT id FROM users) x JOIN "Users" u ON x.id=u.id FOR UPDATE`, []types.TableRef{quotedUsers}},
		{"cte query", `WITH x AS (SELECT id FROM users) SELECT users FROM "Users" FOR UPDATE`, []types.TableRef{quotedUsers}},
		{"table list", `SELECT id, users FROM users u, "Users" v WHERE u.id=v.id FOR UPDATE`, []types.TableRef{plainUsers, quotedUsers}},
		{"grouped join", `SELECT u.id FROM (users u JOIN "Users" v ON u.id=COALESCE(v.id, u.id)) FOR UPDATE`, []types.TableRef{plainUsers, quotedUsers}},
		{"join subquery", `SELECT u.id FROM users u JOIN "Users" v ON u.id IN (SELECT id FROM users) FOR UPDATE`, []types.TableRef{plainUsers, quotedUsers}},
		{"join index hint", `SELECT u.id FROM users u USE INDEX FOR JOIN (idx) JOIN "Users" v ON u.id=v.id FOR UPDATE`, []types.TableRef{plainUsers, quotedUsers}},
		{"order index hint", `SELECT u.id FROM users u FORCE INDEX FOR ORDER BY (idx) JOIN "Users" v ON u.id=v.id FOR UPDATE`, []types.TableRef{plainUsers, quotedUsers}},
		{"group index hint", `SELECT u.id FROM users u IGNORE KEY FOR GROUP BY (idx), "Users" v WHERE u.id=v.id FOR UPDATE`, []types.TableRef{plainUsers, quotedUsers}},
		{"update modifiers", `UPDATE LOW_PRIORITY IGNORE "Users" SET id=COALESCE(id, users)`, []types.TableRef{quotedUsers}},
		{"update joined table list", `UPDATE users u, "Users" v SET u.id=v.id`, []types.TableRef{plainUsers, quotedUsers}},
		{"insert without into", `INSERT "Users" (id, users) VALUES (1, 2)`, []types.TableRef{quotedUsers}},
		{"insert select", `INSERT INTO "Users" (id) SELECT id FROM users`, []types.TableRef{quotedUsers}},
		{"delete target aliases", `DELETE users, u FROM "Users" AS users JOIN users AS u ON users.id=u.id`, []types.TableRef{quotedUsers, plainUsers}},
		{"delete using source", `DELETE FROM users USING "Users" AS users WHERE users.id=1`, []types.TableRef{quotedUsers}},
		{"quoted punctuation", `SELECT id, users FROM "Space.with.dot"."Semi;colon" FOR UPDATE`, []types.TableRef{{Qualifier: "Space.with.dot", TableName: "Semi;colon", QualifierQuoted: true, TableNameQuoted: true}}},
		{"escaped identifier quote", `SELECT id FROM "Us""ers" FOR UPDATE`, []types.TableRef{{TableName: `Us"ers`, TableNameQuoted: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := DoParserForDB(tc.query, types.DBTypePostgreSQL)
			if !assert.NoError(t, err) {
				return
			}
			var got []types.TableRef
			for _, table := range tableNames(parsed) {
				ref, ok := parsed.TableRefs[table]
				assert.True(t, ok, "every source AST table must have a bound reference")
				got = append(got, ref)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCaptureTableRefsRejectsUnboundAndReorderedSources(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query string
		sql   string
	}{
		{"projection is not a source", `SELECT id FROM "Users" FOR UPDATE`, `SELECT id, Users FROM others FOR UPDATE`},
		{"missing source", `SELECT id FROM "Users" FOR UPDATE`, `SELECT Users`},
		{"extra source", `SELECT id FROM "Users" FOR UPDATE`, `SELECT id FROM "Users", others FOR UPDATE`},
		{"different source order", `UPDATE "One" JOIN "Two" ON 1=1 SET id=1`, `UPDATE "Two" JOIN "One" ON 1=1 SET id=1`},
		{"table statement identity mismatch", `TABLE "Users"`, `TABLE "Other"`},
		{"odbc source order mismatch", `SELECT * FROM "One" JOIN "Two" ON 1=1`, `SELECT * FROM { OJ "Two" JOIN "One" ON 1=1 }`},
		{"odbc missing closing brace", `SELECT * FROM "One" JOIN "Two" ON 1=1`, `SELECT * FROM { OJ "One" JOIN "Two" ON 1=1`},
		{"odbc extra closing brace", `SELECT * FROM "One" JOIN "Two" ON 1=1`, `SELECT * FROM { OJ "One" JOIN "Two" ON 1=1 } }`},
		{"odbc unsupported escape", `SELECT * FROM "One" JOIN "Two" ON 1=1`, `SELECT * FROM { wrong "One" JOIN "Two" ON 1=1 }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := DoParserForDB(tc.query, types.DBTypePostgreSQL)
			if !assert.NoError(t, err) {
				return
			}
			parsed.TableRefs = nil
			assert.ErrorContains(t, captureTableRefs(parsed, tc.sql), "cannot")
			assert.Nil(t, parsed.TableRefs, "failed binding must not expose default unquoted references")
		})
	}
}

func TestDoParserForDBKeepsQueryScopesWithinEachStatement(t *testing.T) {
	parsed, err := DoParserForDB(`SELECT id, users FROM "Users" FOR UPDATE; SELECT (SELECT id FROM "Users"), Users FROM users FOR UPDATE`, types.DBTypePostgreSQL)
	if !assert.NoError(t, err) || !assert.Len(t, parsed.MultiStmt, 2) {
		return
	}
	first, err := parsed.MultiStmt[0].GetTableRef()
	assert.NoError(t, err)
	second, err := parsed.MultiStmt[1].GetTableRef()
	assert.NoError(t, err)
	assert.Equal(t, types.TableRef{TableName: "Users", TableNameQuoted: true}, first)
	assert.Equal(t, types.TableRef{TableName: "users"}, second)
}

func TestQuotedSelectTableResolvesWithPostgresQuotedName(t *testing.T) {
	parsed, err := DoParserForDB(`SELECT id, users FROM "Users" FOR UPDATE`, types.DBTypePostgreSQL)
	if !assert.NoError(t, err) {
		return
	}
	ref, err := parsed.GetTableRef()
	if !assert.NoError(t, err) {
		return
	}
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return
	}
	defer db.Close()
	conn, err := db.Conn(context.Background())
	if !assert.NoError(t, err) {
		return
	}
	defer conn.Close()
	mock.ExpectQuery("to_regclass").WithArgs(`"Users"`).WillReturnRows(
		sqlmock.NewRows([]string{"database", "schema", "table"}).AddRow("app", "public", "Users"),
	)
	cache := &postgres.TableMetaCache{}
	err = conn.Raw(func(raw any) error {
		key, err := cache.ResolveTableMetaKey(context.Background(), raw.(driver.Conn), ref)
		assert.Equal(t, types.TableMetaKey{DBName: "app", Schema: "public", TableName: "Users"}, key)
		return err
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDoParserForDBKeepsJoinTableReferencesSeparate(t *testing.T) {
	parsed, err := DoParserForDB(`UPDATE "one"."Users" u JOIN "two"."Users" v ON u.id=v.id SET u.id=1`, types.DBTypePostgreSQL)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	left := parsed.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName)
	right := parsed.UpdateStmt.TableRefs.TableRefs.Right.(*ast.TableSource).Source.(*ast.TableName)
	assert.Equal(t, types.TableRef{Qualifier: "one", TableName: "Users", QualifierQuoted: true, TableNameQuoted: true}, parsed.TableRefs[left])
	assert.Equal(t, types.TableRef{Qualifier: "two", TableName: "Users", QualifierQuoted: true, TableNameQuoted: true}, parsed.TableRefs[right])
	primary, err := parsed.GetTableRef()
	assert.NoError(t, err)
	assert.Equal(t, parsed.TableRefs[left], primary)
}

func TestCopyTableRefsKeepsOriginalQuotingAfterReparse(t *testing.T) {
	original, err := DoParserForDB(`UPDATE Space.Users SET id=1`, types.DBTypePostgreSQL)
	assert.NoError(t, err)
	reparsed, err := DoParserForDB("UPDATE `Space`.`Users` SET id=1", types.DBTypePostgreSQL)
	assert.NoError(t, err)
	CopyTableRefs(reparsed, original)
	assert.NoError(t, BindTableRefs(reparsed))
	ref, err := reparsed.GetTableRef()
	assert.NoError(t, err)
	assert.Equal(t, types.TableRef{Qualifier: "Space", TableName: "Users"}, ref)
}

func TestK(t *testing.T) {
	sql := "update aa set name = ?, age = ? where id = 123"
	p := aparser.New()
	stmt, _, _ := p.Parse(sql, "", "")

	var bytes = bytes.NewByteBuffer([]byte{})
	var cc = format.NewRestoreCtx(format.RestoreKeyWordUppercase, bytes)
	stmt[0].Restore(cc)

	fmt.Println(stmt)
}
