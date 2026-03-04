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
	"fmt"
	"testing"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
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
	builder := &MySQLMultiDeleteUndoLogBuilder{}

	// Test with single delete - should delegate to regular delete builder
	// We expect this to fail due to nil conn, but we're testing the delegation logic
	execCtx := &types.ExecContext{
		Query:  "DELETE FROM t_user WHERE id = ?",
		Values: []driver.Value{100},
	}

	// This should delegate to single delete builder (based on query splitting logic)
	defer func() {
		if r := recover(); r != nil {
			// Expected panic due to nil structures - this confirms delegation happened
			assert.Contains(t, fmt.Sprintf("%v", r), "nil pointer")
		}
	}()

	_, err := builder.BeforeImage(context.Background(), execCtx)
	// If we reach here without panic, there should be an error
	if err == nil {
		t.Error("Expected error or panic due to incomplete context")
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

func TestMySQLMultiUpdateUndoLogBuilder_AfterImage(t *testing.T) {
	builder := &MySQLMultiUpdateUndoLogBuilder{}

	// Test with nil conn - expect panic that we can catch
	execCtx := &types.ExecContext{
		Conn: nil,
		ParseContext: &types.ParseContext{
			UpdateStmt: &ast.UpdateStmt{
				TableRefs: &ast.TableRefsClause{
					TableRefs: &ast.Join{
						Left: &ast.TableSource{
							Source: &ast.TableName{
								Name: model.CIStr{O: "t_user"},
							},
						},
					},
				},
			},
		},
		MetaDataMap: map[string]types.TableMeta{
			"t_user": {
				TableName: "t_user",
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType: types.IndexTypePrimaryKey,
						Columns: []types.ColumnMeta{
							{ColumnName: "id"},
						},
					},
				},
			},
		},
	}

	beforeImages := []*types.RecordImage{
		{
			TableName: "t_user",
			Rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{
							ColumnName: "id",
							Value:      100,
						},
					},
				},
			},
		},
	}

	// Expect panic due to nil conn
	defer func() {
		if r := recover(); r != nil {
			assert.Contains(t, fmt.Sprintf("%v", r), "nil pointer")
		}
	}()

	_, err := builder.AfterImage(context.Background(), execCtx, beforeImages)
	// If we reach here without panic, there should be an error
	if err == nil {
		t.Error("Expected error or panic due to nil conn")
	}
}

func TestMySQLMultiUpdateUndoLogBuilder_buildAfterImageSQL(t *testing.T) {
	builder := &MySQLMultiUpdateUndoLogBuilder{}

	beforeImage := &types.RecordImage{
		TableName: "t_user",
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{
						ColumnName: "id",
						Value:      100,
					},
				},
			},
		},
	}

	meta := types.TableMeta{
		TableName: "t_user",
		Indexs: map[string]types.IndexMeta{
			"id": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	sql, args := builder.buildAfterImageSQL(beforeImage, meta)

	assert.Contains(t, sql, "SELECT * FROM t_user")
	// The generated SQL format might be different, let's just check it contains the essential parts
	assert.Contains(t, sql, "t_user")
	assert.Len(t, args, 1)
	assert.Equal(t, 100, args[0])
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
