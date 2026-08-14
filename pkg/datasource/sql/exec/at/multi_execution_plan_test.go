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
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestBuildMultiExecutionPlan(t *testing.T) {
	tests := []struct {
		name                string
		sourceQuery         string
		expectStatementSize int
		expectAggregatePath bool
	}{
		{
			name: "literal same table updates use aggregate path",
			sourceQuery: "UPDATE t_user SET name = 'user1' WHERE id = 1;" +
				"UPDATE t_user SET age = 18 WHERE id = 2",
			expectStatementSize: 2,
			expectAggregatePath: true,
		},
		{
			name: "parameterized same table updates use sequential path",
			sourceQuery: "UPDATE t_user SET name = ? WHERE id = ?;" +
				"UPDATE t_user SET age = ? WHERE id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "updates without where use sequential path",
			sourceQuery: "UPDATE t_user SET name = 'user1';" +
				"UPDATE t_user SET age = 18",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "literal same table deletes use aggregate path",
			sourceQuery: "DELETE FROM t_user WHERE id = 1;" +
				"DELETE FROM t_user WHERE id = 2",
			expectStatementSize: 2,
			expectAggregatePath: true,
		},
		{
			name: "parameterized same table deletes use sequential path",
			sourceQuery: "DELETE FROM t_user WHERE id = ?;" +
				"DELETE FROM t_user WHERE id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "deletes without where use sequential path",
			sourceQuery: "DELETE FROM t_user;" +
				"DELETE FROM t_user",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "multiple inserts use sequential path",
			sourceQuery: "INSERT INTO t_user(id, name) VALUES (?, ?);" +
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "literal update with limit uses sequential path",
			sourceQuery: "UPDATE t_user SET status = 1 " +
				"WHERE status = 0 LIMIT 10;" +
				"UPDATE t_user SET status = 2 " +
				"WHERE id = 1",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "updates on different tables use sequential path",
			sourceQuery: "UPDATE t_user SET name = ? WHERE id = ?;" +
				"UPDATE t_account SET balance = ? WHERE id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "deletes on different tables use sequential path",
			sourceQuery: "DELETE FROM t_user WHERE id = ?;" +
				"DELETE FROM t_user_log WHERE user_id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "update with limit uses sequential path",
			sourceQuery: "UPDATE t_user SET status = ? WHERE status = ? LIMIT ?;" +
				"UPDATE t_user SET status = ? WHERE id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "update with order by uses sequential path",
			sourceQuery: "UPDATE t_user SET status = ? WHERE status = ? ORDER BY id;" +
				"UPDATE t_user SET status = ? WHERE id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "delete with limit uses sequential path",
			sourceQuery: "DELETE FROM t_user WHERE status = ? LIMIT ?;" +
				"DELETE FROM t_user WHERE id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
		{
			name: "delete with order by uses sequential path",
			sourceQuery: "DELETE FROM t_user WHERE status = ? ORDER BY id;" +
				"DELETE FROM t_user WHERE id = ?",
			expectStatementSize: 2,
			expectAggregatePath: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseCtx, err := parser.DoParser(tt.sourceQuery)
			if !assert.NoError(t, err) {
				return
			}

			plan, err := buildMultiExecutionPlan(parseCtx, types.DBTypeMySQL)
			if !assert.NoError(t, err) || !assert.NotNil(t, plan) {
				return
			}

			assert.Len(t, plan.statements, tt.expectStatementSize)
			assert.Equal(t, tt.expectAggregatePath, plan.useAggregatePath)

			for index := range parseCtx.MultiStmt {
				assert.Same(t, parseCtx.MultiStmt[index], plan.statements[index])
			}
		})
	}
}

func TestBuildMultiExecutionPlanError(t *testing.T) {
	tests := []struct {
		name        string
		parseCtx    *types.ParseContext
		dbType      types.DBType
		expectError error
	}{
		{
			name:        "nil parse context",
			parseCtx:    nil,
			dbType:      types.DBTypeMySQL,
			expectError: errInvalidMultiSQL,
		},
		{
			name:        "single statement",
			parseCtx:    mustParseMultiExecutionPlanTestSQL(t, "UPDATE t_user SET name = ? WHERE id = ?"),
			dbType:      types.DBTypeMySQL,
			expectError: errInvalidMultiSQL,
		},
		{
			name: "unsupported database type",
			parseCtx: mustParseMultiExecutionPlanTestSQL(
				t,
				"UPDATE t_user SET name = ? WHERE id = ?;"+
					"UPDATE t_user SET age = ? WHERE id = ?",
			),
			dbType:      types.DBTypePostgreSQL,
			expectError: errUnsupportedMultiSQL,
		},
		{
			name: "unsupported select statement",
			parseCtx: mustParseMultiExecutionPlanTestSQL(
				t,
				"SELECT * FROM t_user WHERE id = ?;"+
					"UPDATE t_user SET name = ? WHERE id = ?",
			),
			dbType:      types.DBTypeMySQL,
			expectError: errUnsupportedMultiSQL,
		},
		{
			name: "nil child parse context",
			parseCtx: &types.ParseContext{
				SQLType:      types.SQLTypeMulti,
				ExecutorType: types.MultiExecutor,
				MultiStmt: []*types.ParseContext{
					mustParseMultiExecutionPlanTestSQL(t, "UPDATE t_user SET name = ? WHERE id = ?"),
					nil,
				},
			},
			dbType:      types.DBTypeMySQL,
			expectError: errInvalidMultiSQL,
		},
		{
			name: "update executor without update AST",
			parseCtx: &types.ParseContext{
				SQLType:      types.SQLTypeMulti,
				ExecutorType: types.MultiExecutor,
				MultiStmt: []*types.ParseContext{
					{
						SQLType:      types.SQLTypeUpdate,
						ExecutorType: types.UpdateExecutor,
					},
					mustParseMultiExecutionPlanTestSQL(t, "UPDATE t_user SET age = ? WHERE id = ?"),
				},
			},
			dbType:      types.DBTypeMySQL,
			expectError: errInvalidMultiSQL,
		},
		{
			name: "update join is unsupported",
			parseCtx: mustParseMultiExecutionPlanTestSQL(
				t,
				"UPDATE t_user u "+
					"JOIN t_account a ON a.user_id = u.id "+
					"SET u.status = 1 WHERE a.status = 0;"+
					"UPDATE t_user u "+
					"JOIN t_account a ON a.user_id = u.id "+
					"SET u.status = 2 WHERE a.status = 1",
			),
			dbType:      types.DBTypeMySQL,
			expectError: errUnsupportedMultiSQL,
		},
		{
			name: "delete join is unsupported",
			parseCtx: mustParseMultiExecutionPlanTestSQL(
				t,
				"DELETE u FROM t_user u "+
					"JOIN t_account a ON a.user_id = u.id "+
					"WHERE a.status = ?;"+
					"DELETE u FROM t_user u "+
					"JOIN t_account a ON a.user_id = u.id "+
					"WHERE a.status = ?",
			),
			dbType:      types.DBTypeMySQL,
			expectError: errUnsupportedMultiSQL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := buildMultiExecutionPlan(tt.parseCtx, tt.dbType)

			assert.Nil(t, plan)
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expectError)
		})
	}
}

func mustParseMultiExecutionPlanTestSQL(t *testing.T, sourceQuery string) *types.ParseContext {
	t.Helper()

	parseCtx, err := parser.DoParser(sourceQuery)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	return parseCtx
}
