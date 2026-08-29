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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

func TestNewMySQLUndoDeleteExecutor(t *testing.T) {
	sqlUndoLog := undo.SQLUndoLog{
		TableName:   "test_table",
		SQLType:     types.SQLTypeDelete,
		BeforeImage: &types.RecordImage{TableName: "before"},
		AfterImage:  &types.RecordImage{TableName: "after"},
	}

	executor := newMySQLUndoDeleteExecutor(sqlUndoLog)

	assert.NotNil(t, executor)
	assert.Equal(t, sqlUndoLog, executor.sqlUndoLog)
	assert.NotNil(t, executor.baseExecutor)
	assert.Equal(t, sqlUndoLog, executor.baseExecutor.sqlUndoLog)
	assert.Equal(t, sqlUndoLog.BeforeImage, executor.baseExecutor.undoImage)
}

func TestMySQLUndoDeleteExecutor_BuildUndoSQL(t *testing.T) {
	tests := []struct {
		name        string
		beforeImage *types.RecordImage
		wantSQL     string
		wantErr     bool
	}{
		{
			name: "build undo SQL with single row",
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
							IType:   types.IndexTypePrimaryKey,
							Columns: []types.ColumnMeta{{ColumnName: "id"}},
						},
					},
					ColumnNames: []string{"id", "name", "age"},
				},
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: 1},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
							{ColumnName: "age", KeyType: types.IndexTypeNull, Value: 25},
						},
					},
				},
			},
			wantSQL: "INSERT INTO test_table (name, age, id) VALUES (?, ?, ?)",
			wantErr: false,
		},
		{
			name: "build undo SQL with composite primary key",
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
							{ColumnName: "order_id", KeyType: types.PrimaryKey.Number(), Value: 100},
							{ColumnName: "amount", KeyType: types.IndexTypeNull, Value: 99.99},
							{ColumnName: "user_id", KeyType: types.PrimaryKey.Number(), Value: 1},
						},
					},
				},
			},
			wantSQL: "INSERT INTO test_table (amount, user_id, order_id) VALUES (?, ?, ?)",
			wantErr: false,
		},
		{
			name:        "build undo SQL with empty rows",
			beforeImage: &types.RecordImage{Rows: []types.RowImage{}},
			wantSQL:     "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &mySQLUndoDeleteExecutor{
				sqlUndoLog: undo.SQLUndoLog{
					TableName:   tt.beforeImage.TableName,
					BeforeImage: tt.beforeImage,
				},
			}

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

func TestMySQLUndoDeleteExecutor_ExecuteOn(t *testing.T) {
	dataValidation := undo.UndoConfig.DataValidation
	undo.UndoConfig.DataValidation = true
	defer func() { undo.UndoConfig.DataValidation = dataValidation }()

	tableMeta := &types.TableMeta{
		TableName: "test_table",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
		},
	}

	tests := []struct {
		name        string
		beforeImage *types.RecordImage
		currentRows *sqlmock.Rows
		expectError bool
		setupMock   func(mock sqlmock.Sqlmock)
	}{
		{
			name: "restore when current image is empty",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: int64(1)},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
							{ColumnName: "age", KeyType: types.IndexTypeNull, Value: 25},
						},
					},
				},
			},
			currentRows: sqlmock.NewRows([]string{"id", "name", "age"}),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO test_table").
					ExpectExec().
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "skip when current image already equals before image",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{{Columns: []types.ColumnImage{
					{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: int64(1)},
				}}},
			},
			currentRows: sqlmock.NewRowsWithColumnDefinition(
				sqlmock.NewColumn("id").OfType("BIGINT", int64(0)),
			).AddRow(1),
			setupMock: func(mock sqlmock.Sqlmock) {},
		},
		{
			name: "reject dirty current image",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{{Columns: []types.ColumnImage{
					{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: int64(1)},
				}}},
			},
			currentRows: sqlmock.NewRowsWithColumnDefinition(
				sqlmock.NewColumn("id").OfType("BIGINT", int64(0)),
			).AddRow(2),
			expectError: true,
			setupMock:   func(mock sqlmock.Sqlmock) {},
		},
		{
			name: "execute with prepare error",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{ColumnName: "id", KeyType: types.PrimaryKey.Number(), Value: int64(1)},
							{ColumnName: "name", KeyType: types.IndexTypeNull, Value: "test"},
						},
					},
				},
			},
			currentRows: sqlmock.NewRows([]string{"id", "name"}),
			expectError: true,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectPrepare("INSERT INTO test_table").
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
			tt.beforeImage.TableMeta = tableMeta
			mock.ExpectQuery("SELECT .* FROM test_table .* FOR UPDATE").
				WithArgs(1).
				WillReturnRows(tt.currentRows)
			tt.setupMock(mock)

			executor := newMySQLUndoDeleteExecutor(undo.SQLUndoLog{
				TableName:   tt.beforeImage.TableName,
				BeforeImage: tt.beforeImage,
				AfterImage:  &types.RecordImage{TableName: tt.beforeImage.TableName, TableMeta: tableMeta},
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

type mockConn struct {
	mockStmt *mockStmt
}

func (m *mockConn) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	if m.mockStmt.prepareErr {
		return nil, assert.AnError
	}
	return &sql.Stmt{}, nil
}

type mockStmt struct {
	execSuccess bool
	prepareErr  bool
}

func (m *mockStmt) ExecContext(ctx context.Context, args ...interface{}) (sql.Result, error) {
	if m.execSuccess {
		return &mockResult{}, nil
	}
	return nil, assert.AnError
}

func (m *mockStmt) Exec(args ...interface{}) (sql.Result, error) {
	if m.execSuccess {
		return &mockResult{}, nil
	}
	return nil, assert.AnError
}

func (m *mockStmt) Close() error {
	return nil
}

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}
