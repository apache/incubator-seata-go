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
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestNewMySQLUndoInsertExecutor(t *testing.T) {
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLTypeInsert,
	}

	executor := newMySQLUndoInsertExecutor(sqlUndoLog)

	assert.NotNil(t, executor)
	assert.Equal(t, sqlUndoLog, executor.sqlUndoLog)
}

func TestMySQLUndoInsertExecutor_BuildUndoSQL(t *testing.T) {
	tests := []struct {
		name       string
		afterImage *types.RecordImage
		sqlUndoLog undo.SQLUndoLog
		wantSQL    string
		wantErr    bool
	}{
		{
			name: "build delete SQL with single primary key",
			afterImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Columns: map[string]types.ColumnMeta{
						"id":   {ColumnName: "id"},
						"name": {ColumnName: "name"},
					},
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
						},
					},
				},
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
						},
					},
				},
			},
			sqlUndoLog: undo.SQLUndoLog{
				TableName: "test_table",
			},
			wantSQL: "DELETE FROM test_table WHERE `id` = ? ",
			wantErr: false,
		},
		{
			name: "build delete SQL with composite primary key",
			afterImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Columns: map[string]types.ColumnMeta{
						"user_id":  {ColumnName: "user_id"},
						"order_id": {ColumnName: "order_id"},
						"amount":   {ColumnName: "amount"},
					},
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "user_id",
						},
					},
				},
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "user_id", KeyType: types.PrimaryKey.Number(), Value: 1},
							{ColumnName: "order_id", KeyType: types.PrimaryKey.Number(), Value: 100},
							{ColumnName: "amount", KeyType: types.IndexTypeNull, Value: 99.99},
						},
					},
				},
			},
			sqlUndoLog: undo.SQLUndoLog{
				TableName: "test_table",
			},
			wantSQL: "DELETE FROM test_table WHERE `user_id` = ? AND `order_id` = ? ",
			wantErr: false,
		},
		{
			name:       "build SQL with empty rows",
			afterImage: &types.RecordImage{Rows: []types.RowImage{}},
			sqlUndoLog: undo.SQLUndoLog{TableName: "test_table"},
			wantSQL:    "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock GetOrderedPkList function
			patches := gomonkey.ApplyFunc(GetOrderedPkList, func(image *types.RecordImage, row types.RowImage, dbType types.DBType) ([]types.ColumnImage, error) {
				var pkList []types.ColumnImage
				for _, col := range row.Columns {
					if col.KeyType == types.PrimaryKey.Number() {
						pkList = append(pkList, col)
					}
				}
				return pkList, nil
			})
			defer patches.Reset()

			// Mock BuildWhereConditionByPKs function
			patches.ApplyFunc(BuildWhereConditionByPKs, func(pkNameList []string, dbType types.DBType) string {
				if len(pkNameList) == 1 {
					return "`" + pkNameList[0] + "` = ?"
				} else if len(pkNameList) == 2 {
					return "`" + pkNameList[0] + "` = ? AND `" + pkNameList[1] + "` = ?"
				}
				return ""
			})

			executor := &mySQLUndoInsertExecutor{
				sqlUndoLog: tt.sqlUndoLog,
			}
			executor.sqlUndoLog.AfterImage = tt.afterImage

			got, err := executor.buildUndoSQL(types.DBTypeMySQL)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid undo log")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSQL, got)
			}
		})
	}
}

func TestMySQLUndoInsertExecutor_GenerateDeleteSql(t *testing.T) {
	tests := []struct {
		name       string
		image      *types.RecordImage
		rows       []types.RowImage
		sqlUndoLog undo.SQLUndoLog
		wantSQL    string
		wantErr    bool
	}{
		{
			name: "generate delete SQL success",
			image: &types.RecordImage{
				TableName: "test_table",
			},
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: 1},
						{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
					},
				},
			},
			sqlUndoLog: undo.SQLUndoLog{
				TableName: "test_table",
			},
			wantSQL: "DELETE FROM test_table WHERE `id` = ? ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock GetOrderedPkList function
			patches := gomonkey.ApplyFunc(GetOrderedPkList, func(image *types.RecordImage, row types.RowImage, dbType types.DBType) ([]types.ColumnImage, error) {
				var pkList []types.ColumnImage
				for _, col := range row.Columns {
					if col.KeyType == types.PrimaryKey.Number() {
						pkList = append(pkList, col)
					}
				}
				return pkList, nil
			})
			defer patches.Reset()

			// Mock BuildWhereConditionByPKs function
			patches.ApplyFunc(BuildWhereConditionByPKs, func(pkNameList []string, dbType types.DBType) string {
				return "`" + pkNameList[0] + "` = ?"
			})

			executor := &mySQLUndoInsertExecutor{
				sqlUndoLog: tt.sqlUndoLog,
			}

			got, err := executor.generateDeleteSql(tt.image, tt.rows, types.DBTypeMySQL, tt.sqlUndoLog)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSQL, got)
			}
		})
	}
}

func TestMySQLUndoInsertExecutor_ExecuteOn(t *testing.T) {
	tests := []struct {
		name        string
		afterImage  *types.RecordImage
		expectError bool
		setupMock   func(mock sqlmock.Sqlmock)
	}{
		{
			name: "execute on success",
			afterImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Columns: map[string]types.ColumnMeta{
						"id":   {ColumnName: "id"},
						"name": {ColumnName: "name"},
					},
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
						},
					},
				},
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
						},
					},
				},
			},
			expectError: false,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("DELETE FROM test_table").
					ExpectExec().
					WithArgs(sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "execute with prepare error",
			afterImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Columns: map[string]types.ColumnMeta{
						"id":   {ColumnName: "id"},
						"name": {ColumnName: "name"},
					},
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
						},
					},
				},
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
						},
					},
				},
			},
			expectError: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("DELETE FROM test_table").
					WillReturnError(assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup sqlmock
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			ctx := context.Background()
			conn, err := db.Conn(ctx)
			require.NoError(t, err)
			defer conn.Close()

			// Setup mock expectations
			tt.setupMock(mock)

			// Mock BaseExecutor.ExecuteOn to return nil
			patches := gomonkey.ApplyFunc((*BaseExecutor).ExecuteOn, func(be *BaseExecutor, ctx context.Context, dbType types.DBType, conn *sql.Conn) error {
				return nil
			})
			defer patches.Reset()

			executor := &mySQLUndoInsertExecutor{
				BaseExecutor: &BaseExecutor{
					sqlUndoLog: undo.SQLUndoLog{
						TableName:  tt.afterImage.TableName,
						AfterImage: tt.afterImage,
					},
				},
				sqlUndoLog: undo.SQLUndoLog{
					TableName:  tt.afterImage.TableName,
					AfterImage: tt.afterImage,
				},
			}

			err = executor.ExecuteOn(ctx, types.DBTypeMySQL, conn)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
