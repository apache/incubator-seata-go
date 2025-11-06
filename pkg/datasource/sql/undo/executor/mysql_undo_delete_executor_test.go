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

package executor

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestNewMySQLUndoDeleteExecutor(t *testing.T) {
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLType_DELETE,
	}

	executor := newMySQLUndoDeleteExecutor(sqlUndoLog)
	assert.NotNil(t, executor)
	assert.NotNil(t, executor.baseExecutor)
	assert.Equal(t, "test_table", executor.sqlUndoLog.TableName)
}

func TestMySQLUndoDeleteExecutor_BuildUndoSQL(t *testing.T) {
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLType_DELETE,
		BeforeImage: &types.RecordImage{
			TableName: "test_table",
			Rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
						{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
					},
				},
			},
		},
	}

	executor := newMySQLUndoDeleteExecutor(sqlUndoLog)
	sql, err := executor.buildUndoSQL(types.DBTypeMySQL)
	assert.NoError(t, err)
	assert.Contains(t, sql, "INSERT INTO")
	assert.Contains(t, sql, "test_table")
	assert.Contains(t, sql, "VALUES")
}

func TestMySQLUndoDeleteExecutor_BuildUndoSQL_EmptyRows(t *testing.T) {
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLType_DELETE,
		BeforeImage: &types.RecordImage{
			TableName: "test_table",
			Rows:      []types.RowImage{},
		},
	}

	executor := newMySQLUndoDeleteExecutor(sqlUndoLog)
	sql, err := executor.buildUndoSQL(types.DBTypeMySQL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid undo log")
	assert.Empty(t, sql)
}

func TestMySQLUndoDeleteExecutor_ExecuteOn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLType_DELETE,
		BeforeImage: &types.RecordImage{
			TableName: "test_table",
			TableMeta: &types.TableMeta{
				TableName: "test_table",
			},
			Rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
						{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
					},
				},
			},
		},
	}

	executor := newMySQLUndoDeleteExecutor(sqlUndoLog)

	// Mock the prepare and exec
	mock.ExpectPrepare("INSERT INTO (.+)").
		ExpectExec().
		WithArgs("test", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	err = executor.ExecuteOn(ctx, types.DBTypeMySQL, conn)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
