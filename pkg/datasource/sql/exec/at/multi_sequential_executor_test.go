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
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/test_driver"
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestRestoreStatementSQLFromMultiSQL(t *testing.T) {
	tests := []struct {
		name                string
		sourceQuery         string
		expectQueries       []string
		expectExecutorTypes []types.ExecutorType
	}{
		{
			name: "restore update and delete independently",
			sourceQuery: "UPDATE t_user SET name = ? WHERE id = ?;" +
				"DELETE FROM t_user_log WHERE user_id = ?",
			expectQueries: []string{
				"UPDATE t_user SET name = ? WHERE id = ?",
				"DELETE FROM t_user_log WHERE user_id = ?",
			},
			expectExecutorTypes: []types.ExecutorType{types.UpdateExecutor, types.DeleteExecutor},
		},
		{
			name: "restore multiple inserts independently",
			sourceQuery: "INSERT INTO t_user(id, name) VALUES (?, ?);" +
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
			expectQueries: []string{
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
			},
			expectExecutorTypes: []types.ExecutorType{types.InsertExecutor, types.InsertExecutor},
		},
		{
			name: "restore mixed statements in original order",
			sourceQuery: "INSERT INTO t_user(id, name) VALUES (?, ?);" +
				"UPDATE t_user SET name = ? WHERE id = ?;" +
				"DELETE FROM t_user_log WHERE user_id = ?",
			expectQueries: []string{
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
				"UPDATE t_user SET name = ? WHERE id = ?",
				"DELETE FROM t_user_log WHERE user_id = ?",
			},
			expectExecutorTypes: []types.ExecutorType{types.InsertExecutor, types.UpdateExecutor, types.DeleteExecutor},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseCtx, err := parser.DoParser(tt.sourceQuery)
			if !assert.NoError(t, err) {
				return
			}

			if !assert.Len(t, parseCtx.MultiStmt, len(tt.expectQueries)) {
				return
			}

			for index, statementCtx := range parseCtx.MultiStmt {
				stmtNode, err := getStatementNode(statementCtx)
				if !assert.NoError(t, err) {
					return
				}

				query, err := restoreStatementSQL(stmtNode)
				if !assert.NoError(t, err) {
					return
				}

				assert.Equal(t, tt.expectQueries[index], query)
				assert.NotContains(t, query, ";")

				restoredParseCtx, err := parser.DoParser(query)
				if !assert.NoError(t, err) {
					return
				}

				assert.Empty(t, restoredParseCtx.MultiStmt)
				assert.Equal(t, tt.expectExecutorTypes[index], restoredParseCtx.ExecutorType)
				assert.NotSame(t, statementCtx, restoredParseCtx)
			}
		})
	}
}

func TestCountStatementParameters(t *testing.T) {
	tests := []struct {
		name                 string
		sourceQuery          string
		expectParameterCount int
	}{
		{
			name:                 "insert with multiple rows",
			sourceQuery:          "INSERT INTO t_user(id, name) VALUES (?, ?), (?, ?)",
			expectParameterCount: 4,
		},
		{
			name:                 "update parameters in set where and limit",
			sourceQuery:          "UPDATE t_user SET name = ?, age = ? " + "WHERE id = ? LIMIT ?",
			expectParameterCount: 4,
		},
		{
			name:                 "question mark in string literal is not parameter",
			sourceQuery:          "UPDATE t_user SET description = 'ready?' " + "WHERE id = ?",
			expectParameterCount: 1,
		},
		{
			name:                 "delete without parameters",
			sourceQuery:          "DELETE FROM t_user WHERE id = 1",
			expectParameterCount: 0,
		},
		{
			name:                 "delete with multiple parameters",
			sourceQuery:          "DELETE FROM t_user " + "WHERE status = ? AND created_at < ?",
			expectParameterCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseCtx, err := parser.DoParser(tt.sourceQuery)
			if !assert.NoError(t, err) {
				return
			}

			stmtNode, err := getStatementNode(parseCtx)
			if !assert.NoError(t, err) {
				return
			}

			parameterCount, err := countStatementParameters(stmtNode)
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, tt.expectParameterCount, parameterCount)
		})
	}
}

func TestCloneNamedValues(t *testing.T) {
	tests := []struct {
		name              string
		sourceNamedValues []driver.NamedValue
		expectNamedValues []driver.NamedValue
	}{
		{
			name:              "rebase single named value",
			sourceNamedValues: []driver.NamedValue{{Name: "user_id", Ordinal: 3, Value: int64(10)}},
			expectNamedValues: []driver.NamedValue{{Name: "user_id", Ordinal: 1, Value: int64(10)}},
		},
		{
			name:              "rebase multiple named values",
			sourceNamedValues: []driver.NamedValue{{Ordinal: 4, Value: "user"}, {Ordinal: 5, Value: int64(18)}, {Ordinal: 6, Value: int64(10)}},
			expectNamedValues: []driver.NamedValue{{Ordinal: 1, Value: "user"}, {Ordinal: 2, Value: int64(18)}, {Ordinal: 3, Value: int64(10)}},
		},
		{
			name:              "clone empty named values",
			sourceNamedValues: []driver.NamedValue{},
			expectNamedValues: []driver.NamedValue{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalNamedValues := make([]driver.NamedValue, len(tt.sourceNamedValues))
			copy(originalNamedValues, tt.sourceNamedValues)

			cloned := cloneNamedValues(tt.sourceNamedValues)
			assert.Equal(t, tt.expectNamedValues, cloned)
			assert.Equal(t, originalNamedValues, tt.sourceNamedValues)

			if len(cloned) == 0 {
				return
			}

			cloned[0].Ordinal = 100
			cloned[0].Value = "changed"

			assert.Equal(t, originalNamedValues, tt.sourceNamedValues)
		})
	}
}

func TestReparseStatementResetsParameterOrders(t *testing.T) {
	sourceQuery := "UPDATE t_user SET name = ? WHERE id = ?;" + "DELETE FROM t_user_log WHERE user_id = ?"

	parseCtx, err := parser.DoParser(sourceQuery)
	if !assert.NoError(t, err) {
		return
	}

	if !assert.Len(t, parseCtx.MultiStmt, 2) {
		return
	}

	originalUpdateNode, err := getStatementNode(parseCtx.MultiStmt[0])
	if !assert.NoError(t, err) {
		return
	}

	originalDeleteNode, err := getStatementNode(parseCtx.MultiStmt[1])
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []int{0, 1}, collectParameterOrdersForTest(t, originalUpdateNode))
	assert.Equal(t, []int{2}, collectParameterOrdersForTest(t, originalDeleteNode))

	deleteQuery, err := restoreStatementSQL(originalDeleteNode)
	if !assert.NoError(t, err) {
		return
	}

	reparsedDeleteCtx, err := parser.DoParser(deleteQuery)
	if !assert.NoError(t, err) {
		return
	}

	reparsedDeleteNode, err := getStatementNode(reparsedDeleteCtx)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []int{0}, collectParameterOrdersForTest(t, reparsedDeleteNode))

	globalNamedValues := []driver.NamedValue{{Ordinal: 1, Value: "user"}, {Ordinal: 2, Value: int64(10)}, {Ordinal: 3, Value: int64(10)}}
	deleteNamedValues := cloneNamedValues(globalNamedValues[2:])

	assert.Equal(t, []driver.NamedValue{{Ordinal: 1, Value: int64(10)}}, deleteNamedValues)
	assert.Equal(t, []int{2}, collectParameterOrdersForTest(t, originalDeleteNode))
}

type parameterOrderCollectorForTest struct {
	orders []int
}

func (c *parameterOrderCollectorForTest) Enter(node ast.Node) (ast.Node, bool) {
	if marker, ok := node.(*test_driver.ParamMarkerExpr); ok {
		c.orders = append(c.orders, marker.Order)
	}

	return node, false
}

func (c *parameterOrderCollectorForTest) Leave(node ast.Node) (ast.Node, bool) {
	return node, true
}

func collectParameterOrdersForTest(t *testing.T, stmt ast.StmtNode) []int {
	t.Helper()

	if !assert.NotNil(t, stmt) {
		t.FailNow()
	}

	collector := new(parameterOrderCollectorForTest)

	if _, ok := stmt.Accept(collector); !assert.True(t, ok) {
		t.FailNow()
	}

	return collector.orders
}

func TestExecSequentialPreservesOrderAndArguments(t *testing.T) {
	tests := []struct {
		name                string
		sourceQuery         string
		namedValues         []driver.NamedValue
		expectQueries       []string
		expectedNamedValues [][]driver.NamedValue
		expectExecutorTypes []types.ExecutorType
	}{
		{
			name: "multiple inserts execute in original order",
			sourceQuery: "INSERT INTO t_user(id, name) VALUES (?, ?);" +
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
			namedValues: sequentialNamedValuesForTest(int64(1), "user1", int64(2), "user2"),
			expectQueries: []string{
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
			},
			expectedNamedValues: [][]driver.NamedValue{
				sequentialNamedValuesForTest(int64(1), "user1"),
				sequentialNamedValuesForTest(int64(2), "user2"),
			},
			expectExecutorTypes: []types.ExecutorType{types.InsertExecutor, types.InsertExecutor},
		},
		{
			name: "mixed DML executes in original order",
			sourceQuery: "INSERT INTO t_user(id, name) VALUES (?, ?);" +
				"UPDATE t_user SET name = ? WHERE id = ?;" +
				"DELETE FROM t_user WHERE id = ?",
			namedValues: sequentialNamedValuesForTest(int64(1), "user1", "Updated user1", int64(1), int64(1)),
			expectQueries: []string{
				"INSERT INTO t_user(id, name) VALUES (?, ?)",
				"UPDATE t_user SET name = ? WHERE id = ?",
				"DELETE FROM t_user WHERE id = ?",
			},
			expectedNamedValues: [][]driver.NamedValue{
				sequentialNamedValuesForTest(int64(1), "user1"),
				sequentialNamedValuesForTest("Updated user1", int64(1)),
				sequentialNamedValuesForTest(int64(1)),
			},
			expectExecutorTypes: []types.ExecutorType{types.InsertExecutor, types.UpdateExecutor, types.DeleteExecutor},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factoryRecorder := installSequentialFactoriesForTest(t)
			var events []string
			hook := &sequentialHookForTest{events: &events}

			multiExec := newMultiExecutorForSequentialTest(t, tt.sourceQuery, tt.namedValues, []exec.SQLHook{hook})
			executions := make([]sequentialExecutionForTest, 0, len(tt.expectQueries))

			result, err := multiExec.ExecContext(context.Background(), func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				executions = append(executions, sequentialExecutionForTest{
					query:       query,
					namedValues: cloneNamedValuesForSequentialTest(args),
				})

				events = append(events, "execute:"+query)
				return types.NewResult(types.WithResult(driver.RowsAffected(len(executions)))), nil
			})

			if !assert.NoError(t, err) || !assert.NotNil(t, result) {
				return
			}

			assertSequentialFactoryCallsForTest(t, factoryRecorder.calls, tt.expectExecutorTypes, tt.expectQueries, tt.expectedNamedValues)

			expectedExecutions := make([]sequentialExecutionForTest, len(tt.expectQueries))
			for index := range tt.expectQueries {
				expectedExecutions[index] = sequentialExecutionForTest{
					query:       tt.expectQueries[index],
					namedValues: tt.expectedNamedValues[index],
				}
			}

			assert.Equal(t, expectedExecutions, executions)

			// Hooks belong to the complete multi-SQL operation, rather than to each child statement.
			assert.Equal(t, 1, hook.beforeCount)
			assert.Equal(t, 1, hook.afterCount)
			assert.Same(t, multiExec.execContext, hook.beforeExecCtx)
			assert.Same(t, multiExec.execContext, hook.afterExecCtx)

			expectedEvents := []string{"before"}
			for _, query := range tt.expectQueries {
				expectedEvents = append(expectedEvents, "execute:"+query)
			}

			expectedEvents = append(expectedEvents, "after")
			assert.Equal(t, expectedEvents, events)

			rowsAffected, err := result.GetResult().RowsAffected()
			if assert.NoError(t, err) {
				assert.Equal(t, int64(len(tt.expectQueries)), rowsAffected)
			}

		})
	}
}

func TestExecSequentialReturnsFinalStatementResult(t *testing.T) {
	sourceQuery := "INSERT INTO t_user(id, name) VALUES (?, ?);" +
		"INSERT INTO t_user(id, name) VALUES (?, ?)"
	namedValues := sequentialNamedValuesForTest(int64(1), "user1", int64(2), "user2")
	results := []*mockExecResult{
		{lastInsertID: 101, rowsAffected: 2},
		{lastInsertID: 202, rowsAffected: 5},
	}

	installSequentialFactoriesForTest(t)
	multiExec := newMultiExecutorForSequentialTest(t, sourceQuery, namedValues, nil)
	executionCount := 0

	result, err := multiExec.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
		result := results[executionCount]
		executionCount++
		return result, nil
	})

	if !assert.NoError(t, err) || !assert.Same(t, results[1], result) {
		return
	}
	assert.Equal(t, len(results), executionCount)

	rowsAffected, err := result.GetResult().RowsAffected()
	if assert.NoError(t, err) {
		assert.Equal(t, int64(5), rowsAffected)
	}

	lastInsertID, err := result.GetResult().LastInsertId()
	if assert.NoError(t, err) {
		assert.Equal(t, int64(202), lastInsertID)
	}
}

func TestExecSequentialStopsOnMiddleFailure(t *testing.T) {
	sourceQuery := "INSERT INTO t_user(id, name) VALUES (?,?);" +
		"INSERT INTO t_user(id, name) VALUES (?,?);" +
		"INSERT INTO t_user(id, name) VALUES (?,?)"

	nameValues := sequentialNamedValuesForTest(int64(1), "user1", int64(2), "user2", int64(3), "user3")
	factoryRecorder := installSequentialFactoriesForTest(t)

	var events []string
	hook := &sequentialHookForTest{
		events: &events,
	}

	multiExec := newMultiExecutorForSequentialTest(t, sourceQuery, nameValues, []exec.SQLHook{hook})
	expectedError := errors.New("second statement failed")
	executions := make([]sequentialExecutionForTest, 0, 2)

	result, err := multiExec.ExecContext(context.Background(), func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		executions = append(executions, sequentialExecutionForTest{
			query:       query,
			namedValues: cloneNamedValuesForSequentialTest(args),
		})

		events = append(events, "execute:"+query)
		if len(executions) == 2 {
			return nil, expectedError
		}

		return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
	},
	)

	assert.Nil(t, result)
	if !assert.ErrorIs(t, err, expectedError) {
		return
	}

	assert.Contains(t, err.Error(), "execute statement 1")

	assert.Len(t, factoryRecorder.calls, 2)
	assert.Len(t, executions, 2)

	expectedQuery := "INSERT INTO t_user(id, name) VALUES (?,?)"

	assert.Equal(t, expectedQuery, executions[0].query)
	assert.Equal(t, expectedQuery, executions[1].query)

	assert.Equal(t, sequentialNamedValuesForTest(int64(1), "user1"), executions[0].namedValues)
	assert.Equal(t, sequentialNamedValuesForTest(int64(2), "user2"), executions[1].namedValues)

	assert.Equal(t, 1, hook.beforeCount)
	assert.Equal(t, 1, hook.afterCount)

	assert.Equal(t, []string{"before", "execute:" + expectedQuery, "execute:" + expectedQuery, "after"}, events)
}

func TestExecSequentialRejectsArgumentCountMismatchBeforeSideEffects(t *testing.T) {
	sourceQuery := "INSERT INTO t_user(id,name) VALUES (?,?);" + "INSERT INTO t_user(id,name) VALUES (?,?)"

	tests := []struct {
		name        string
		namedValues []driver.NamedValue
	}{
		{
			name:        "too few arguments",
			namedValues: sequentialNamedValuesForTest(int64(1), "user1", int64(2)),
		},
		{
			name:        "too many arguments",
			namedValues: sequentialNamedValuesForTest(int64(1), "user1", int64(2), "user2", "unexpected"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factoryRecorder := installSequentialFactoriesForTest(t)
			hook := new(sequentialHookForTest)

			multiExec := newMultiExecutorForSequentialTest(t, sourceQuery, tt.namedValues, []exec.SQLHook{hook})
			callbackCount := 0
			result, err := multiExec.ExecContext(context.Background(), func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				callbackCount++
				return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
			})

			assert.Nil(t, result)
			if !assert.ErrorIs(t, err, errInvalidMultiSQL) {
				return
			}
			assert.Contains(t, err.Error(), "statements require 4 arguments")
			assert.Contains(t, err.Error(), fmt.Sprintf("but %d were provided", len(tt.namedValues)))

			assert.Empty(t, factoryRecorder.calls)
			assert.Zero(t, callbackCount)
			assert.Zero(t, hook.beforeCount)
			assert.Zero(t, hook.afterCount)
		})
	}
}

type sequentialExecutionForTest struct {
	query       string
	namedValues []driver.NamedValue
}

type sequentialFactoryCallForTest struct {
	factoryExecutorType types.ExecutorType
	parseExecutorType   types.ExecutorType

	query       string
	namedValues []driver.NamedValue

	childHookCount    int
	parseContextMatch bool
	isSingleStatement bool
}

type sequentialFactoryRecorderForTest struct {
	calls []sequentialFactoryCallForTest
}

func (r *sequentialFactoryRecorderForTest) build(factoryExecutorType types.ExecutorType, parseCtx *types.ParseContext,
	execCtx *types.ExecContext, hooks []exec.SQLHook,
) executor {
	r.calls = append(r.calls,
		sequentialFactoryCallForTest{
			factoryExecutorType: factoryExecutorType,
			parseExecutorType:   parseCtx.ExecutorType,
			query:               execCtx.Query,
			namedValues:         cloneNamedValuesForSequentialTest(execCtx.NamedValues),
			childHookCount:      len(hooks),
			parseContextMatch:   parseCtx == execCtx.ParseContext,
			isSingleStatement:   len(parseCtx.MultiStmt) == 0,
		},
	)

	return &sequentialCallbackExecutorForTest{baseExecutor: baseExecutor{hooks: append([]exec.SQLHook(nil), hooks...)}, execCtx: execCtx}
}

type sequentialCallbackExecutorForTest struct {
	baseExecutor
	execCtx *types.ExecContext
}

func (e *sequentialCallbackExecutorForTest) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if err := e.beforeHooks(ctx, e.execCtx); err != nil {
		return nil, err
	}
	defer func() {
		e.afterHooks(ctx, e.execCtx)
	}()
	return f(ctx, e.execCtx.Query, e.execCtx.NamedValues)
}

type sequentialHookForTest struct {
	beforeCount int
	afterCount  int

	beforeExecCtx *types.ExecContext
	afterExecCtx  *types.ExecContext

	events *[]string
}

func (h *sequentialHookForTest) Type() types.SQLType {
	return types.SQLTypeMulti
}

func (h *sequentialHookForTest) Before(ctx context.Context, execCtx *types.ExecContext) error {
	h.beforeCount++
	h.beforeExecCtx = execCtx
	if h.events != nil {
		*h.events = append(*h.events, "before")
	}
	return nil
}

func (h *sequentialHookForTest) After(ctx context.Context, execCtx *types.ExecContext) error {
	h.afterCount++
	h.afterExecCtx = execCtx
	if h.events != nil {
		*h.events = append(*h.events, "after")
	}
	return nil
}

func installSequentialFactoriesForTest(t *testing.T) *sequentialFactoryRecorderForTest {
	t.Helper()

	originalInsertExecutor := newInsertExecutor
	originalUpdateExecutor := newUpdateExecutor
	originalDeleteExecutor := newDeleteExecutor

	t.Cleanup(func() {
		newInsertExecutor = originalInsertExecutor
		newUpdateExecutor = originalUpdateExecutor
		newDeleteExecutor = originalDeleteExecutor
	})

	recorder := new(sequentialFactoryRecorderForTest)

	newInsertExecutor = func(parseCtx *types.ParseContext, execCtx *types.ExecContext, hooks []exec.SQLHook) executor {
		return recorder.build(types.InsertExecutor, parseCtx, execCtx, hooks)
	}

	newUpdateExecutor = func(parseCtx *types.ParseContext, execCtx *types.ExecContext, hooks []exec.SQLHook) executor {
		return recorder.build(types.UpdateExecutor, parseCtx, execCtx, hooks)
	}

	newDeleteExecutor = func(parseCtx *types.ParseContext, execCtx *types.ExecContext, hooks []exec.SQLHook) executor {
		return recorder.build(types.DeleteExecutor, parseCtx, execCtx, hooks)
	}

	return recorder
}

func newMultiExecutorForSequentialTest(t *testing.T, sourceQuery string, namedValues []driver.NamedValue, hooks []exec.SQLHook) *multiExecutor {
	t.Helper()

	parseCtx, err := parser.DoParser(sourceQuery)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	execCtx := &types.ExecContext{
		TxCtx:        types.NewTxCtx(),
		Query:        sourceQuery,
		ParseContext: parseCtx,
		NamedValues:  namedValues,
		DBType:       types.DBTypeMySQL,
	}

	builtExecutor := NewMultiExecutor(parseCtx, execCtx, hooks)

	multiExec, ok := builtExecutor.(*multiExecutor)
	if !assert.True(t, ok) {
		t.FailNow()
	}

	return multiExec
}

func sequentialNamedValuesForTest(values ...driver.Value) []driver.NamedValue {
	namedValues := make([]driver.NamedValue, len(values))

	for index, value := range values {
		namedValues[index] = driver.NamedValue{
			Ordinal: index + 1,
			Value:   value,
		}
	}

	return namedValues
}

func cloneNamedValuesForSequentialTest(values []driver.NamedValue) []driver.NamedValue {
	cloned := make([]driver.NamedValue, len(values))
	copy(cloned, values)
	return cloned
}

func assertSequentialFactoryCallsForTest(t *testing.T, calls []sequentialFactoryCallForTest,
	expectExecutorTypes []types.ExecutorType, expectQueries []string, expectNamedValues [][]driver.NamedValue) {
	t.Helper()

	if !assert.Len(t, calls, len(expectQueries)) {
		return
	}

	for index, call := range calls {
		assert.Equal(t, expectExecutorTypes[index], call.factoryExecutorType)
		assert.Equal(t, expectExecutorTypes[index], call.parseExecutorType)
		assert.Equal(t, expectQueries[index], call.query)
		assert.Equal(t, expectNamedValues[index], call.namedValues)

		assert.True(t, call.parseContextMatch)
		assert.True(t, call.isSingleStatement)

		assert.Zero(t, call.childHookCount)
	}
}

func TestExecSequentialRunsInsertSpecificHook(t *testing.T) {
	expectedErr := errors.New("insert blocked")
	insertHook := &statementSpecificHookForTest{sqlType: types.SQLTypeInsert, beforeErr: expectedErr}

	originalHooksForSQLType := hooksForSQLType
	t.Cleanup(func() {
		hooksForSQLType = originalHooksForSQLType
	})

	hooksForSQLType = func(sqlType types.SQLType) []exec.SQLHook {
		if sqlType == types.SQLTypeInsert {
			return []exec.SQLHook{insertHook}
		}
		return nil
	}

	sourceQuery := "INSERT INTO forbidden_table(id) VALUES (?);" + "INSERT INTO forbidden_table(id) VALUES (?)"

	multiExec := newMultiExecutorForSequentialTest(t, sourceQuery, sequentialNamedValuesForTest(int64(1), int64(2)), nil)
	callbackCount := 0
	result, err := multiExec.ExecContext(context.Background(),
		func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
			callbackCount++
			return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
		},
	)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
	assert.Zero(t, callbackCount)
	assert.Equal(t, 1, insertHook.beforeCount)
	assert.Zero(t, insertHook.afterCount)
}

type statementSpecificHookForTest struct {
	sqlType       types.SQLType
	beforeErr     error
	beforeCount   int
	afterCount    int
	beforeExecCtx *types.ExecContext
	afterExecCtx  *types.ExecContext
}

func (h *statementSpecificHookForTest) Type() types.SQLType {
	return h.sqlType
}

func (h *statementSpecificHookForTest) Before(ctx context.Context, execCtx *types.ExecContext) error {
	h.beforeCount++
	h.beforeExecCtx = execCtx
	return h.beforeErr
}

func (h *statementSpecificHookForTest) After(ctx context.Context, execCtx *types.ExecContext) error {
	h.afterCount++
	h.afterExecCtx = execCtx
	return nil
}
