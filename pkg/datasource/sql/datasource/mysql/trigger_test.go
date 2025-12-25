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

package mysql

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/testdata"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func initMockIndexMeta() []types.IndexMeta {
	return []types.IndexMeta{
		{
			IType:      types.IndexTypePrimaryKey,
			ColumnName: "id",
			Columns: []types.ColumnMeta{
				{
					ColumnName:   "id",
					DatabaseType: types.GetSqlDataType("BIGINT"),
				},
			},
		},
	}
}

func initMockColumnMeta() []types.ColumnMeta {
	return []types.ColumnMeta{
		{
			ColumnName: "id",
		},
		{
			ColumnName: "name",
		},
	}
}

func initGetIndexesStub(m *mysqlTrigger, indexMeta []types.IndexMeta) *gomonkey.Patches {
	getIndexesStub := gomonkey.ApplyPrivateMethod(m, "getIndexes",
		func(_ *mysqlTrigger, ctx context.Context, dbName string, tableName string, conn *sql.Conn) ([]types.IndexMeta, error) {
			return indexMeta, nil
		})
	return getIndexesStub
}

func initGetColumnMetasStub(m *mysqlTrigger, columnMeta []types.ColumnMeta) *gomonkey.Patches {
	getColumnMetasStub := gomonkey.ApplyPrivateMethod(m, "getColumnMetas",
		func(_ *mysqlTrigger, ctx context.Context, dbName string, table string, conn *sql.Conn) ([]types.ColumnMeta, error) {
			return columnMeta, nil
		})
	return getColumnMetasStub
}

func Test_mysqlTrigger_LoadOne(t *testing.T) {
	wantTableMeta := testdata.MockWantTypesMeta("test")
	type args struct {
		ctx       context.Context
		dbName    string
		tableName string
		conn      *sql.Conn
	}
	tests := []struct {
		name          string
		args          args
		columnMeta    []types.ColumnMeta
		indexMeta     []types.IndexMeta
		wantTableMeta *types.TableMeta
	}{
		{
			name:          "1",
			args:          args{ctx: context.Background(), dbName: "dbName", tableName: "test", conn: nil},
			indexMeta:     initMockIndexMeta(),
			columnMeta:    initMockColumnMeta(),
			wantTableMeta: &wantTableMeta,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mysqlTrigger{}

			getColumnMetasStub := initGetColumnMetasStub(m, tt.columnMeta)
			defer getColumnMetasStub.Reset()

			getIndexesStub := initGetIndexesStub(m, tt.indexMeta)
			defer getIndexesStub.Reset()

			got, err := m.LoadOne(tt.args.ctx, tt.args.dbName, tt.args.tableName, tt.args.conn)
			if err != nil {
				t.Errorf("LoadOne() error = %v", err)
				return
			}

			assert.Equal(t, tt.wantTableMeta, got)
		})
	}
}

func initMockResourceManager(branchType branch.BranchType, ctrl *gomock.Controller) *mock.MockDataSourceManager {
	mockResourceMgr := mock.NewMockDataSourceManager(ctrl)
	mockResourceMgr.SetBranchType(branchType)
	mockResourceMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).AnyTimes().Return(int64(0), nil)
	rm.GetRmCacheInstance().RegisterResourceManager(mockResourceMgr)
	mockResourceMgr.EXPECT().RegisterResource(gomock.Any()).AnyTimes().Return(nil)
	mockResourceMgr.EXPECT().CreateTableMetaCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)
	return mockResourceMgr
}

func Test_mysqlTrigger_LoadAll(t *testing.T) {
	sql.Register("seata-at-mysql", &mock.MockTestDriver{})

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	conn := sql.Conn{}
	type args struct {
		ctx    context.Context
		dbName string
		conn   *sql.Conn
		tables []string
	}
	tests := []struct {
		name       string
		args       args
		columnMeta []types.ColumnMeta
		indexMeta  []types.IndexMeta
		want       []types.TableMeta
	}{
		{
			name: "test-01",
			args: args{
				ctx:    nil,
				dbName: "dbName",
				conn:   &conn,
				tables: []string{
					"test_01",
					"test_02",
				},
			},
			indexMeta:  initMockIndexMeta(),
			columnMeta: initMockColumnMeta(),
			want:       []types.TableMeta{testdata.MockWantTypesMeta("test_01"), testdata.MockWantTypesMeta("test_02")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mysqlTrigger{}

			getColumnMetasStub := initGetColumnMetasStub(m, tt.columnMeta)
			defer getColumnMetasStub.Reset()

			getIndexesStub := initGetIndexesStub(m, tt.indexMeta)
			defer getIndexesStub.Reset()

			got, err := m.LoadAll(tt.args.ctx, tt.args.dbName, tt.args.conn, tt.args.tables...)
			if err != nil {
				t.Errorf("LoadAll() error = %v", err)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewMysqlTrigger(t *testing.T) {
	trigger := NewMysqlTrigger()
	assert.NotNil(t, trigger)
}

func Test_mysqlTrigger_LoadOne_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		columnMetaErr error
		indexMetaErr  error
		columnMeta    []types.ColumnMeta
		indexMeta     []types.IndexMeta
		expectError   bool
		errorContains string
	}{
		{
			name:          "error_getting_columns",
			columnMetaErr: errors.New("column query error"),
			expectError:   true,
			errorContains: "Could not found any columnMeta",
		},
		{
			name:          "error_getting_indexes",
			columnMeta:    initMockColumnMeta(),
			indexMetaErr:  errors.New("index query error"),
			expectError:   true,
			errorContains: "Could not found any index",
		},
		{
			name:          "no_indexes_found",
			columnMeta:    initMockColumnMeta(),
			indexMeta:     []types.IndexMeta{},
			expectError:   true,
			errorContains: "could not found any index",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mysqlTrigger{}

			if tt.columnMetaErr != nil {
				getColumnMetasStub := gomonkey.ApplyPrivateMethod(m, "getColumnMetas",
					func(_ *mysqlTrigger, ctx context.Context, dbName string, table string, conn *sql.Conn) ([]types.ColumnMeta, error) {
						return nil, tt.columnMetaErr
					})
				defer getColumnMetasStub.Reset()
			} else {
				getColumnMetasStub := initGetColumnMetasStub(m, tt.columnMeta)
				defer getColumnMetasStub.Reset()
			}

			if tt.indexMetaErr != nil {
				getIndexesStub := gomonkey.ApplyPrivateMethod(m, "getIndexes",
					func(_ *mysqlTrigger, ctx context.Context, dbName string, tableName string, conn *sql.Conn) ([]types.IndexMeta, error) {
						return nil, tt.indexMetaErr
					})
				defer getIndexesStub.Reset()
			} else {
				getIndexesStub := initGetIndexesStub(m, tt.indexMeta)
				defer getIndexesStub.Reset()
			}

			_, err := m.LoadOne(context.Background(), "testdb", "testtable", nil)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_mysqlTrigger_LoadOne_ComplexIndexes(t *testing.T) {
	m := &mysqlTrigger{}

	// Test composite indexes
	columnMeta := []types.ColumnMeta{
		{ColumnName: "id"},
		{ColumnName: "user_id"},
		{ColumnName: "email"},
	}

	indexMeta := []types.IndexMeta{
		{
			IType:      types.IndexTypePrimaryKey,
			Name:       "PRIMARY",
			ColumnName: "id",
			Columns:    []types.ColumnMeta{},
		},
		{
			IType:      types.IndexUnique,
			Name:       "idx_unique_email",
			ColumnName: "email",
			NonUnique:  false,
			Columns:    []types.ColumnMeta{},
		},
		{
			IType:      types.IndexNormal,
			Name:       "idx_user",
			ColumnName: "user_id",
			NonUnique:  true,
			Columns:    []types.ColumnMeta{},
		},
	}

	getColumnMetasStub := initGetColumnMetasStub(m, columnMeta)
	defer getColumnMetasStub.Reset()

	getIndexesStub := initGetIndexesStub(m, indexMeta)
	defer getIndexesStub.Reset()

	tableMeta, err := m.LoadOne(context.Background(), "testdb", "testtable", nil)

	assert.NoError(t, err)
	assert.NotNil(t, tableMeta)
	assert.Equal(t, 3, len(tableMeta.Columns))
	assert.Equal(t, 3, len(tableMeta.Indexs))
}

func Test_mysqlTrigger_getColumnMetas(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	defer conn.Close()

	tests := []struct {
		name          string
		setupMock     func()
		expectError   bool
		errorContains string
		expectedCount int
	}{
		{
			name: "success_with_multiple_columns",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE",
					"COLUMN_TYPE", "COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
				}).
					AddRow("users", "testdb", "id", "BIGINT", "BIGINT(20)", "PRI", "NO", nil, "auto_increment").
					AddRow("users", "testdb", "name", "VARCHAR", "VARCHAR(100)", "", "YES", []byte("default"), "").
					AddRow("users", "testdb", "age", "INT", "INT(11)", "", "NO", []byte("0"), "")

				mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 3,
		},
		{
			name: "prepare_error",
			setupMock: func() {
				mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
					WillReturnError(errors.New("prepare failed"))
			},
			expectError:   true,
			errorContains: "prepare failed",
		},
		{
			name: "query_error",
			setupMock: func() {
				mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnError(errors.New("query failed"))
			},
			expectError:   true,
			errorContains: "query failed",
		},
		{
			name: "no_columns_found",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE",
					"COLUMN_TYPE", "COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
				})

				mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   true,
			errorContains: "can't find column",
		},
		{
			name: "nullable_column",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE",
					"COLUMN_TYPE", "COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
				}).
					AddRow("users", "testdb", "optional_field", "VARCHAR", "VARCHAR(50)", "", "YES", nil, "")

				mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			m := &mysqlTrigger{}
			columns, err := m.getColumnMetas(context.Background(), "testdb", "users", conn)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(columns))

				if tt.name == "nullable_column" && len(columns) > 0 {
					assert.Equal(t, int8(1), columns[0].IsNullable)
				}
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_mysqlTrigger_getIndexes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	defer conn.Close()

	tests := []struct {
		name          string
		setupMock     func()
		expectError   bool
		errorContains string
		expectedCount int
		validateIndex func(*testing.T, []types.IndexMeta)
	}{
		{
			name: "success_primary_key",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
					AddRow("PRIMARY", "id", 0)

				mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 1,
			validateIndex: func(t *testing.T, indexes []types.IndexMeta) {
				assert.Equal(t, types.IndexTypePrimaryKey, indexes[0].IType)
				assert.Equal(t, "PRIMARY", indexes[0].Name)
				assert.False(t, indexes[0].NonUnique)
			},
		},
		{
			name: "success_unique_index",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
					AddRow("idx_email", "email", 0)

				mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 1,
			validateIndex: func(t *testing.T, indexes []types.IndexMeta) {
				assert.Equal(t, types.IndexUnique, indexes[0].IType)
				assert.False(t, indexes[0].NonUnique)
			},
		},
		{
			name: "success_normal_index",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
					AddRow("idx_name", "name", 1)

				mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 1,
			validateIndex: func(t *testing.T, indexes []types.IndexMeta) {
				assert.Equal(t, types.IndexNormal, indexes[0].IType)
				assert.True(t, indexes[0].NonUnique)
			},
		},
		{
			name: "prepare_error",
			setupMock: func() {
				mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
					WillReturnError(errors.New("prepare failed"))
			},
			expectError:   true,
			errorContains: "prepare failed",
		},
		{
			name: "query_error",
			setupMock: func() {
				mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnError(errors.New("query failed"))
			},
			expectError:   true,
			errorContains: "query failed",
		},
		{
			name: "composite_index",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
					AddRow("idx_composite", "col1", 1).
					AddRow("idx_composite", "col2", 1)

				mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "mixed_case_primary",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{"INDEX_NAME", "COLUMN_NAME", "NON_UNIQUE"}).
					AddRow("Primary", "id", 0)

				mock.ExpectPrepare("SELECT (.+) FROM `INFORMATION_SCHEMA`.`STATISTICS`").
					ExpectQuery().
					WithArgs("testdb", "users").
					WillReturnRows(rows)
			},
			expectError:   false,
			expectedCount: 1,
			validateIndex: func(t *testing.T, indexes []types.IndexMeta) {
				assert.Equal(t, types.IndexTypePrimaryKey, indexes[0].IType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			m := &mysqlTrigger{}
			indexes, err := m.getIndexes(context.Background(), "testdb", "users", conn)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(indexes))

				if tt.validateIndex != nil {
					tt.validateIndex(t, indexes)
				}
			}

			// Ensure all expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func Test_mysqlTrigger_LoadAll_ErrorHandling(t *testing.T) {
	m := &mysqlTrigger{}

	// Test with one successful and one failed table
	columnMeta := initMockColumnMeta()
	indexMeta := initMockIndexMeta()

	callCount := 0
	getColumnMetasStub := gomonkey.ApplyPrivateMethod(m, "getColumnMetas",
		func(_ *mysqlTrigger, ctx context.Context, dbName string, table string, conn *sql.Conn) ([]types.ColumnMeta, error) {
			callCount++
			if callCount == 2 {
				return nil, errors.New("column error")
			}
			return columnMeta, nil
		})
	defer getColumnMetasStub.Reset()

	getIndexesStub := initGetIndexesStub(m, indexMeta)
	defer getIndexesStub.Reset()

	// LoadAll should continue even if one table fails
	result, err := m.LoadAll(context.Background(), "testdb", nil, "table1", "table2", "table3")

	assert.NoError(t, err)
	// Should have 2 tables (table1 and table3), table2 failed
	assert.Equal(t, 2, len(result))
}

func Test_mysqlTrigger_LoadOne_MultipleIndexesOnSameColumn(t *testing.T) {
	m := &mysqlTrigger{}

	columnMeta := []types.ColumnMeta{
		{ColumnName: "id"},
		{ColumnName: "email"},
	}

	// Same index appears multiple times (composite index with multiple columns)
	indexMeta := []types.IndexMeta{
		{
			IType:      types.IndexTypePrimaryKey,
			Name:       "PRIMARY",
			ColumnName: "id",
			Columns:    []types.ColumnMeta{},
		},
		{
			IType:      types.IndexNormal,
			Name:       "idx_composite",
			ColumnName: "id",
			NonUnique:  true,
			Columns:    []types.ColumnMeta{},
		},
		{
			IType:      types.IndexNormal,
			Name:       "idx_composite",
			ColumnName: "email",
			NonUnique:  true,
			Columns:    []types.ColumnMeta{},
		},
	}

	getColumnMetasStub := initGetColumnMetasStub(m, columnMeta)
	defer getColumnMetasStub.Reset()

	getIndexesStub := initGetIndexesStub(m, indexMeta)
	defer getIndexesStub.Reset()

	tableMeta, err := m.LoadOne(context.Background(), "testdb", "testtable", nil)

	assert.NoError(t, err)
	assert.NotNil(t, tableMeta)

	// Should have 2 unique index names (PRIMARY and idx_composite)
	assert.Equal(t, 2, len(tableMeta.Indexs))

	// The composite index should have 2 columns
	compositeIdx := tableMeta.Indexs["idx_composite"]
	assert.Equal(t, 2, len(compositeIdx.Columns))
}

func Test_mysqlTrigger_getColumnMetas_DataTypes(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}
	defer conn.Close()

	rows := sqlmock.NewRows([]string{
		"TABLE_NAME", "TABLE_SCHEMA", "COLUMN_NAME", "DATA_TYPE",
		"COLUMN_TYPE", "COLUMN_KEY", "IS_NULLABLE", "COLUMN_DEFAULT", "EXTRA",
	}).
		AddRow("test", "testdb", "id", "BIGINT", "BIGINT(20)", "PRI", "NO", nil, "auto_increment").
		AddRow("test", "testdb", "name", "VARCHAR", "VARCHAR(255)", "", "YES", []byte("''"), "").
		AddRow("test", "testdb", "created_at", "DATETIME", "DATETIME", "", "NO", []byte("CURRENT_TIMESTAMP"), "on update CURRENT_TIMESTAMP").
		AddRow("test", "testdb", "price", "DECIMAL", "DECIMAL(10,2)", "", "YES", nil, "")

	mock.ExpectPrepare("SELECT (.+) FROM INFORMATION_SCHEMA.COLUMNS").
		ExpectQuery().
		WithArgs("testdb", "test").
		WillReturnRows(rows)

	m := &mysqlTrigger{}
	columns, err := m.getColumnMetas(context.Background(), "testdb", "test", conn)

	assert.NoError(t, err)
	assert.Equal(t, 4, len(columns))

	// Verify auto_increment
	assert.True(t, columns[0].Autoincrement)
	assert.False(t, columns[1].Autoincrement)

	// Verify nullable
	assert.Equal(t, int8(0), columns[0].IsNullable)
	assert.Equal(t, int8(1), columns[1].IsNullable)

	// Verify column defaults
	assert.Nil(t, columns[0].ColumnDef)
	assert.Equal(t, []byte("''"), columns[1].ColumnDef)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
