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

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestNewMySQLUndoUpdateExecutor(t *testing.T) {
	sqlUndoLog := undo.SQLUndoLog{
		TableName: "test_table",
		SQLType:   types.SQLTypeUpdate,
	}

	executor := newMySQLUndoUpdateExecutor(sqlUndoLog)

	assert.NotNil(t, executor)
	assert.Equal(t, sqlUndoLog, executor.sqlUndoLog)
	assert.NotNil(t, executor.baseExecutor)
	assert.Equal(t, sqlUndoLog, executor.baseExecutor.sqlUndoLog)
	assert.Equal(t, sqlUndoLog.AfterImage, executor.baseExecutor.undoImage)
}

func TestMySQLUndoUpdateExecutor_BuildUndoSQL(t *testing.T) {
	tests := []struct {
		name        string
		beforeImage *types.RecordImage
		wantSQL     string
		wantErr     bool
	}{
		{
			name: "build update SQL with single primary key",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Columns: map[string]types.ColumnMeta{
						"id":   {ColumnName: "id"},
						"name": {ColumnName: "name"},
						"age":  {ColumnName: "age"},
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
							{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "old_name"},
							{ColumnName: "age", KeyType: types.IndexTypeNull, Value: 25},
						},
					},
				},
			},
			wantSQL: "UPDATE test_table SET `name` = ? , `age` = ?  WHERE `id` = ? ",
			wantErr: false,
		},
		{
			name: "build update SQL with composite primary key",
			beforeImage: &types.RecordImage{
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
							{ColumnName: "user_id", KeyType: types.IndexTypePrimaryKey, Value: 1},
							{ColumnName: "order_id", KeyType: types.IndexTypePrimaryKey, Value: 100},
							{ColumnName: "amount", KeyType: types.IndexTypeNull, Value: 99.99},
						},
					},
				},
			},
			wantSQL: "UPDATE test_table SET `amount` = ?  WHERE `user_id` = ? AND `order_id` = ? ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock GetOrderedPkList function
			patches := gomonkey.ApplyFunc(GetOrderedPkList, func(image *types.RecordImage, row types.RowImage, dbType types.DBType) ([]types.ColumnImage, error) {
				var pkList []types.ColumnImage
				for _, col := range row.Columns {
					if col.KeyType == types.IndexTypePrimaryKey {
						pkList = append(pkList, col)
					}
				}
				return pkList, nil
			})
			defer patches.Reset()

			// Mock AddEscape function
			patches.ApplyFunc(AddEscape, func(columnName string, dbType types.DBType) string {
				return "`" + columnName + "`"
			})

			// Mock BuildWhereConditionByPKs function
			patches.ApplyFunc(BuildWhereConditionByPKs, func(pkNameList []string, dbType types.DBType) string {
				if len(pkNameList) == 1 {
					return "`" + pkNameList[0] + "` = ?"
				} else if len(pkNameList) == 2 {
					return "`" + pkNameList[0] + "` = ? AND `" + pkNameList[1] + "` = ?"
				}
				return ""
			})

			executor := &mySQLUndoUpdateExecutor{
				sqlUndoLog: undo.SQLUndoLog{
					TableName:   tt.beforeImage.TableName,
					BeforeImage: tt.beforeImage,
				},
			}

			got, err := executor.buildUndoSQL(types.DBTypeMySQL)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantSQL, got)
			}
		})
	}
}

func TestMySQLUndoUpdateExecutor_ExecuteOn(t *testing.T) {
	tests := []struct {
		name               string
		beforeImage        *types.RecordImage
		dataValidationPass bool
		expectError        bool
		setupMock          func(mock sqlmock.Sqlmock)
	}{
		{
			name: "execute on success",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "old_name"},
						},
					},
				},
			},
			dataValidationPass: true,
			expectError:        false,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE test_table").
					ExpectExec().
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "data validation failed, stop execution",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "old_name"},
						},
					},
				},
			},
			dataValidationPass: false,
			expectError:        false,
			setupMock:          func(mock sqlmock.Sqlmock) {},
		},
		{
			name: "execute with prepare error",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "old_name"},
						},
					},
				},
			},
			dataValidationPass: true,
			expectError:        true,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("UPDATE test_table").
					WillReturnError(assert.AnError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock GetOrderedPkList function
			patches := gomonkey.ApplyFunc(GetOrderedPkList, func(image *types.RecordImage, row types.RowImage, dbType types.DBType) ([]types.ColumnImage, error) {
				var pkList []types.ColumnImage
				for _, col := range row.Columns {
					if col.KeyType == types.IndexTypePrimaryKey {
						pkList = append(pkList, col)
					}
				}
				return pkList, nil
			})
			defer patches.Reset()

			// Mock AddEscape function
			patches.ApplyFunc(AddEscape, func(columnName string, dbType types.DBType) string {
				return "`" + columnName + "`"
			})

			// Mock BuildWhereConditionByPKs function
			patches.ApplyFunc(BuildWhereConditionByPKs, func(pkNameList []string, dbType types.DBType) string {
				if len(pkNameList) == 1 {
					return "`" + pkNameList[0] + "` = ?"
				}
				return ""
			})

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

			// Create mock baseExecutor
			mockBaseExecutor := &BaseExecutor{
				sqlUndoLog: undo.SQLUndoLog{
					TableName:   tt.beforeImage.TableName,
					BeforeImage: tt.beforeImage,
				},
			}

			// Mock BaseExecutor.dataValidationAndGoOn
			patches.ApplyFunc((*BaseExecutor).dataValidationAndGoOn, func(be *BaseExecutor, ctx context.Context, conn *sql.Conn) (bool, error) {
				return tt.dataValidationPass, nil
			})

			executor := &mySQLUndoUpdateExecutor{
				baseExecutor: mockBaseExecutor,
				sqlUndoLog: undo.SQLUndoLog{
					TableName:   tt.beforeImage.TableName,
					BeforeImage: tt.beforeImage,
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
