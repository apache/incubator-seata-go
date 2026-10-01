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

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

func Test_multiDeleteExecutor_buildBeforeImageSQL(t *testing.T) {
	log.Init()
	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "delete from table_update_executor_test where id = ?; delete from table_update_executor_test",
			sourceQueryArgs: []driver.Value{3},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test FOR UPDATE",
			expectQueryArgs: []driver.Value{},
		},
		{
			sourceQuery:     "delete from table_update_executor_test2 where id = ?; delete from table_update_executor_test2 where id = ?",
			sourceQueryArgs: []driver.Value{3, 2},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test2 WHERE (id=?) OR (id=?) FOR UPDATE",
			expectQueryArgs: []driver.Value{3, 2},
		},
		{
			sourceQuery:     "delete from table_update_executor_test2 where id = ?; delete from table_update_executor_test2 where name = ? and age = ?",
			sourceQueryArgs: []driver.Value{3, "seata-go", 4},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test2 WHERE (id=?) OR (name=? AND age=?) FOR UPDATE",
			expectQueryArgs: []driver.Value{3, "seata-go", 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParser, err := parser.DoParser(tt.sourceQuery)
			assert.Nil(t, err)
			executor := NewMultiDeleteExecutor(queryParser, &types.ExecContext{Query: tt.sourceQuery, Values: tt.sourceQueryArgs, NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs)}, []exec.SQLHook{})
			query, args, err := executor.buildBeforeImageSQL()
			assert.Nil(t, err)
			assert.Equal(t, query, tt.expectQuery)
			assert.Equal(t, util.ValueToNamedValue(tt.expectQueryArgs), args)
		})
	}
}

func TestMultiDeleteBeforeImagePreservesTableSource(t *testing.T) {
	for _, tt := range []struct {
		name       string
		source     string
		wantSource string
		column     string
		indexHint  bool
	}{
		{name: "partition", source: "orders PARTITION(p0)", wantSource: "`app`.`orders` PARTITION(`p0`)", column: "id"},
		{name: "alias", source: "orders AS o", wantSource: "`app`.`orders` AS `o`", column: "o.id"},
		{name: "partition and alias", source: "orders PARTITION(p0) AS o", wantSource: "`app`.`orders` PARTITION(`p0`) AS `o`", column: "o.id"},
		{name: "index hint", source: "orders AS o", wantSource: "`app`.`orders` AS `o` USE INDEX (`PRIMARY`)", column: "o.id", indexHint: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			query := "DELETE FROM " + tt.source + " WHERE " + tt.column + "=1; DELETE FROM " + tt.source + " WHERE " + tt.column + "=11"
			parseCtx, err := parser.DoParser(query)
			require.NoError(t, err)
			originalSources := make([]string, len(parseCtx.MultiStmt))
			for i, statement := range parseCtx.MultiStmt {
				if tt.indexHint {
					table := statement.DeleteStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName)
					table.IndexHints = []*ast.IndexHint{{HintType: ast.HintUse, HintScope: ast.HintForScan, IndexNames: []model.CIStr{model.NewCIStr("PRIMARY")}}}
				}
				originalSources[i], err = statement.GetTableName()
				require.NoError(t, err)
			}
			plan, err := buildMultiExecutionPlan(parseCtx, types.DBTypeMySQL)
			require.NoError(t, err)
			require.True(t, plan.useAggregatePath)
			executor := NewMultiDeleteExecutor(parseCtx, &types.ExecContext{
				TableMetaKey: &types.TableMetaKey{DBName: "app", TableName: "orders"},
			}, nil)

			selectSQL, args, err := executor.buildBeforeImageSQL()
			require.NoError(t, err)
			assert.Equal(t, "SELECT SQL_NO_CACHE * FROM "+tt.wantSource+" WHERE ("+tt.column+"=1) OR ("+tt.column+"=11) FOR UPDATE", selectSQL)
			assert.Empty(t, args)
			for i, statement := range parseCtx.MultiStmt {
				source, err := statement.GetTableName()
				require.NoError(t, err)
				assert.Equal(t, originalSources[i], source, "before-image generation must not change the business SQL AST")
			}
		})
	}
}

func TestMultiDeleteExecutesBeforeImageWithPartitionAndAlias(t *testing.T) {
	query := "DELETE FROM app.orders PARTITION(p0) AS o WHERE o.id=1; DELETE FROM app.orders PARTITION(p0) AS o WHERE o.id=11"
	parseCtx, err := parser.DoParser(query)
	require.NoError(t, err)
	meta := &types.TableMeta{
		TableName:   "orders",
		ColumnNames: []string{"id"},
		Columns:     map[string]types.ColumnMeta{"id": {ColumnName: "id", DatabaseTypeString: "BIGINT"}},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	conn := mock.NewMockTestDriverConn(gomock.NewController(t))
	conn.EXPECT().QueryContext(gomock.Any(),
		"SELECT SQL_NO_CACHE * FROM `app`.`orders` PARTITION(`p0`) AS `o` WHERE (o.id=1) OR (o.id=11) FOR UPDATE",
		gomock.Any()).Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1)}), nil)
	execCtx := &types.ExecContext{
		Query: query, Conn: conn, TxCtx: types.NewTxCtx(), DBType: types.DBTypeMySQL,
		TableMetaReader: &stubTableMetaCache{meta: meta},
	}
	callbacks := 0
	_, err = NewMultiExecutor(parseCtx, execCtx, nil).ExecContext(context.Background(),
		func(_ context.Context, actualQuery string, _ []driver.NamedValue) (types.ExecResult, error) {
			callbacks++
			assert.Equal(t, query, actualQuery)
			return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
		})
	require.NoError(t, err)
	assert.Equal(t, 1, callbacks)
	assert.Equal(t, &types.TableMetaKey{DBName: "app", TableName: "orders"}, execCtx.TableMetaKey)
	images := execCtx.TxCtx.RoundImages.BeofreImages()
	require.Len(t, images, 1)
	require.Len(t, images[0].Rows, 1)
	assert.EqualValues(t, 1, images[0].Rows[0].GetColumnMap()["id"].Value)
	assert.Equal(t, execCtx.TableMetaKey, images[0].TableMetaKey)
}
