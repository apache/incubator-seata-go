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
	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
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

func TestMySQLAdapterPreservesAggregateDelete(t *testing.T) {
	const query = "DELETE FROM orders WHERE id=1; DELETE FROM orders WHERE id=2"
	id := types.ColumnMeta{ColumnName: "id", DatabaseTypeString: "BIGINT"}
	meta := &types.TableMeta{
		TableName: "orders", ColumnNames: []string{"id"},
		Columns: map[string]types.ColumnMeta{"id": id},
		Indexs:  map[string]types.IndexMeta{"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{id}}},
	}
	reader := &tableMetaReaderForTest{key: types.TableMetaKey{DBName: "app", TableName: meta.TableName}, meta: meta}
	conn := mock.NewMockTestDriverConn(gomock.NewController(t))
	conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Len(0)).
		Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1)}, []driver.Value{int64(2)}), nil)
	txCtx := types.NewTxCtx()
	txCtx.TransactionMode = types.ATMode
	execCtx := &types.ExecContext{Query: query, DBName: "app", Conn: conn, TxCtx: txCtx, TableMetaReader: reader}
	hook := &mockSQLHook{sqlType: types.SQLTypeMulti}
	root := &ATExecutor{hooks: []exec.SQLHook{hook}}

	businessResult := types.NewResult(types.WithResult(driver.RowsAffected(1)))
	callbacks := 0
	result, err := root.ExecWithNamedValue(context.Background(), execCtx, func(_ context.Context, sql string, args []driver.NamedValue) (types.ExecResult, error) {
		callbacks++
		assert.Equal(t, query, sql)
		assert.Empty(t, args)
		return businessResult, nil
	})

	assert.NoError(t, err)
	assert.Same(t, businessResult, result)
	assert.Equal(t, 1, callbacks)
	assert.Equal(t, 1, hook.beforeCallCount)
	assert.Equal(t, 1, hook.afterCallCount)
	assert.Len(t, txCtx.RoundImages.BeofreImages(), 1)
	assert.Len(t, txCtx.RoundImages.AfterImages(), 1)
	assert.Equal(t, map[string]struct{}{"ORDERS:1,2": {}}, txCtx.LockKeys)
}

func TestMultiExecutorWriteDatabaseBoundary(t *testing.T) {
	for _, tc := range []struct {
		name       string
		query      string
		currentDB  string
		wantReject bool
	}{
		{"later write in batch", "UPDATE db_a.account SET balance=1 WHERE id=1; DELETE FROM db_b.account WHERE id=2", "", true},
		{"aggregate writes", "UPDATE db_b.account SET balance=1 WHERE id=1; UPDATE db_b.account SET balance=2 WHERE id=2", "", true},
		{"same database batch", "INSERT INTO db_a.account (id) VALUES (1); UPDATE db_a.account SET balance=2", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkWriteDatabaseBoundary(t, tc.query, tc.currentDB, tc.wantReject)
		})
	}
}
