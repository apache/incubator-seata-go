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

package builder

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

func TestGetMySQLMultiDeleteUndoLogBuilder(t *testing.T) {
	builder := GetMySQLMultiDeleteUndoLogBuilder()
	assert.NotNil(t, builder)
	assert.IsType(t, &MySQLMultiDeleteUndoLogBuilder{}, builder)
}

func TestMySQLMultiDeleteUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := &MySQLMultiDeleteUndoLogBuilder{}
	executorType := builder.GetExecutorType()
	assert.Equal(t, types.MultiDeleteExecutor, executorType)
}

func TestMySQLMultiDeleteUndoLogBuilder_AfterImage(t *testing.T) {
	builder := &MySQLMultiDeleteUndoLogBuilder{}

	execCtx := &types.ExecContext{}
	beforeImages := []*types.RecordImage{}

	images, err := builder.AfterImage(context.Background(), execCtx, beforeImages)
	// AfterImage for multi DELETE should return nil
	assert.NoError(t, err)
	assert.Nil(t, images)
}

func TestMySQLMultiDeleteUndoLogBuilder_BeforeImage_SingleDelete(t *testing.T) {
	log.Init()
	for _, query := range []string{"DELETE FROM t_user WHERE id IN (?, ?)", "DELETE FROM t_user WHERE id IN (?, ?);"} {
		t.Run(query, func(t *testing.T) {
			conn := &legacyBuilderTestConn{}
			execCtx := legacyBuilderTestContext(t, query, conn)
			execCtx.Values = []driver.Value{int64(1), int64(2)}

			images, err := (&MySQLMultiDeleteUndoLogBuilder{}).BeforeImage(context.Background(), execCtx)
			if !assert.NoError(t, err) || !assert.Len(t, images, 1) {
				return
			}
			assert.True(t, conn.prepared)
			assert.Equal(t, "SELECT SQL_NO_CACHE * FROM t_user WHERE id IN (?,?) FOR UPDATE", conn.query)
			assert.Equal(t, execCtx.Values, conn.args)
			assert.Equal(t, "t_user", images[0].TableName)
			if assert.Len(t, images[0].Rows, 2) {
				for i, row := range images[0].Rows {
					if assert.Len(t, row.Columns, 1) {
						assert.Equal(t, "id", row.Columns[0].ColumnName)
						assert.Equal(t, int64(i+1), row.Columns[0].Value)
						assert.Equal(t, types.IndexTypePrimaryKey, row.Columns[0].KeyType)
					}
				}
			}
			assert.Equal(t, map[string]struct{}{"T_USER:1,2": {}}, execCtx.TxCtx.LockKeys)
		})
	}
}

func TestGetMySQLMultiUpdateUndoLogBuilder(t *testing.T) {
	builder := GetMySQLMultiUpdateUndoLogBuilder()
	assert.NotNil(t, builder)
	assert.IsType(t, &MySQLMultiUpdateUndoLogBuilder{}, builder)
}

func TestMySQLMultiUpdateUndoLogBuilder_GetExecutorType(t *testing.T) {
	builder := &MySQLMultiUpdateUndoLogBuilder{}
	executorType := builder.GetExecutorType()
	assert.Equal(t, types.UpdateExecutor, executorType)
}

func TestUpdateVisitor_Enter(t *testing.T) {
	visitor := &updateVisitor{}

	node := &ast.SelectStmt{}
	result, skipChildren := visitor.Enter(node)

	assert.Equal(t, node, result)
	assert.True(t, skipChildren)
}

func TestUpdateVisitor_Leave(t *testing.T) {
	visitor := &updateVisitor{}

	node := &ast.SelectStmt{}
	result, ok := visitor.Leave(node)

	assert.Equal(t, node, result)
	assert.True(t, ok)
}

func TestMySQLMultiUpdateUndoLogBuilder_buildBeforeImageSQL_EmptyStmts(t *testing.T) {
	builder := &MySQLMultiUpdateUndoLogBuilder{}

	stmts := []*ast.UpdateStmt{}
	args := []driver.Value{}

	_, _, err := builder.buildBeforeImageSQL(stmts, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid muliti update stmt")
}
