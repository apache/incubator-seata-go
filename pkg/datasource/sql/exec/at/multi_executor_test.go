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

	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestMultiExecutorFallsBackToSequentialWhenStatementSpecificHookExists(t *testing.T) {
	sourceQuery := "UPDATE t_user SET name = 'user1' WHERE id = 1;" + "UPDATE t_user SET age = 18 WHERE id = 2"

	parseCtx, err := parser.DoParser(sourceQuery)
	if !assert.NoError(t, err) {
		return
	}

	plan, err := buildMultiExecutionPlan(parseCtx, types.DBTypeMySQL)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, plan.useAggregatePath)

	updateHook := &statementSpecificHookForTest{sqlType: types.SQLTypeUpdate}

	originalHooksForSQLType := hooksForSQLType
	t.Cleanup(func() { hooksForSQLType = originalHooksForSQLType })

	hooksForSQLType = func(sqlType types.SQLType) []exec.SQLHook {
		if sqlType == types.SQLTypeUpdate {
			return []exec.SQLHook{updateHook}
		}
		return nil
	}

	factoryRecorder := installSequentialFactoriesForTest(t)

	execCtx := &types.ExecContext{
		TxCtx:        types.NewTxCtx(),
		Query:        sourceQuery,
		ParseContext: parseCtx,
		DBType:       types.DBTypeMySQL,
	}

	multiExec := NewMultiExecutor(parseCtx, execCtx, nil)

	callbackQueries := make([]string, 0, 2)

	result, err := multiExec.ExecContext(context.Background(), func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		callbackQueries = append(callbackQueries, query)
		return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
	},
	)

	if !assert.NoError(t, err) || !assert.NotNil(t, result) {
		return
	}

	assert.Equal(t, []string{"UPDATE t_user SET name = 'user1' WHERE id = 1", "UPDATE t_user SET age = 18 WHERE id = 2"}, callbackQueries)
	assert.Len(t, factoryRecorder.calls, 2)
	assert.Equal(t, 2, updateHook.beforeCount)
	assert.Equal(t, 2, updateHook.afterCount)
}
