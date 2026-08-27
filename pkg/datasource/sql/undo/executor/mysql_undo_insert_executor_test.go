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
							IType:   types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{ColumnName: "id"}},
						},
					},
					ColumnNames: []string{"id", "name"},
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
			wantSQL: "DELETE FROM test_table WHERE id = ?  ",
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
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{
								{ColumnName: "user_id"},
								{ColumnName: "order_id"},
							},
						},
					},
					ColumnNames: []string{"user_id", "order_id", "amount"},
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
			wantSQL: "DELETE FROM test_table WHERE user_id = ?  and order_id = ?  ",
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
				TableMeta: &types.TableMeta{
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType:   types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{ColumnName: "id"}},
						},
					},
					ColumnNames: []string{"id"},
				},
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
			wantSQL: "DELETE FROM test_table WHERE id = ?  ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	unchangedImage := &types.RecordImage{TableName: "test_table"}
	tests := []struct {
		name           string
		beforeImage    *types.RecordImage
		afterImage     *types.RecordImage
		dataValidation bool
		expectError    bool
		setupMock      func(mock sqlmock.Sqlmock)
	}{
		{
			name: "execute composite primary key in metadata order",
			afterImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Columns: map[string]types.ColumnMeta{
						"pk1":  {ColumnName: "pk1"},
						"pk2":  {ColumnName: "pk2"},
						"name": {ColumnName: "name"},
					},
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType: types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{
								{ColumnName: "pk1"},
								{ColumnName: "pk2"},
							},
						},
					},
					ColumnNames: []string{"pk1", "name", "pk2"},
				},
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "pk2", KeyType: types.PrimaryKey.Number(), Value: 2},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
							{ColumnName: "pk1", KeyType: types.PrimaryKey.Number(), Value: 1},
						},
					},
				},
			},
			expectError: false,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("DELETE FROM test_table").
					ExpectExec().
					WithArgs(1, 2).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:           "skip when before and after images are equal",
			beforeImage:    unchangedImage,
			afterImage:     unchangedImage,
			dataValidation: true,
			setupMock:      func(mock sqlmock.Sqlmock) {},
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
							IType:   types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{ColumnName: "id"}},
						},
					},
					ColumnNames: []string{"id", "name"},
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
		{
			name: "execute with ordered pk list error",
			afterImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{TableName: "test_table"},
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: 1},
						},
					},
				},
			},
			expectError: true,
			setupMock:   func(mock sqlmock.Sqlmock) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataValidation := undo.UndoConfig.DataValidation
			undo.UndoConfig.DataValidation = tt.dataValidation
			defer func() { undo.UndoConfig.DataValidation = dataValidation }()

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

			executor := newMySQLUndoInsertExecutor(undo.SQLUndoLog{
				TableName:   tt.afterImage.TableName,
				BeforeImage: tt.beforeImage,
				AfterImage:  tt.afterImage,
			})

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
func TestMySQLUndoInsertExecutor_BuildUndoSQL_CompositePK(t *testing.T) {
	afterImage := &types.RecordImage{
		TableName: "test_table",
		TableMeta: &types.TableMeta{
			TableName: "test_table",
			Columns: map[string]types.ColumnMeta{
				"tenant_id": {ColumnName: "tenant_id"},
				"id":        {ColumnName: "id"},
			},
			Indexs: map[string]types.IndexMeta{
				"PRIMARY": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "tenant_id"},
						{ColumnName: "id"},
					},
				},
			},
			ColumnNames: []string{"tenant_id", "id"},
		},
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "tenant_id", KeyType: types.IndexTypePrimaryKey, Value: "tenant_1"},
					{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 100},
				},
			},
		},
	}

	sqlUndoLog := undo.SQLUndoLog{
		TableName:  "test_table",
		AfterImage: afterImage,
	}

	executor := &mySQLUndoInsertExecutor{
		sqlUndoLog: sqlUndoLog,
	}

	gotSQL, err := executor.buildUndoSQL(types.DBTypeMySQL)

	assert.NoError(t, err)
	assert.Equal(t, "DELETE FROM test_table WHERE tenant_id = ?  and id = ?  ", gotSQL)
}

func TestMySQLUndoExecutorsPropagateBuildUndoSQLError(t *testing.T) {
	dataValidation := undo.UndoConfig.DataValidation
	undo.UndoConfig.DataValidation = false
	defer func() { undo.UndoConfig.DataValidation = dataValidation }()

	image := &types.RecordImage{
		TableMeta: &types.TableMeta{Indexs: map[string]types.IndexMeta{
			"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
		}},
		Rows: []types.RowImage{{Columns: []types.ColumnImage{
			{ColumnName: "order_id", KeyType: types.PrimaryKey.Number(), Value: 1},
		}}},
	}
	sqlUndoLog := undo.SQLUndoLog{BeforeImage: image, AfterImage: image}
	executors := map[string]undo.UndoExecutor{
		"insert": newMySQLUndoInsertExecutor(sqlUndoLog),
		"delete": newMySQLUndoDeleteExecutor(sqlUndoLog),
		"update": newMySQLUndoUpdateExecutor(sqlUndoLog),
	}

	for name, executor := range executors {
		t.Run(name, func(t *testing.T) {
			err := executor.ExecuteOn(context.Background(), types.DBTypeMySQL, nil)
			assert.ErrorContains(t, err, "primary key")
		})
	}
}
