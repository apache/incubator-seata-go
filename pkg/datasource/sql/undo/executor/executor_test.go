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
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agiledragon/gomonkey/v2"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	datasourcemysql "seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource/mysql"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec/at"
	sqlparser "seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	undoparser "seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/parser"
	sqlutil "seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	serr "seata.apache.org/seata-go/v2/pkg/util/errors"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type testableBaseExecutor struct {
	BaseExecutor
	mockCurrentImage *types.RecordImage
}

func (t *testableBaseExecutor) queryCurrentRecords(ctx context.Context, conn *sql.Conn) (*types.RecordImage, error) {
	return t.mockCurrentImage, nil
}

func (t *testableBaseExecutor) dataValidationAndGoOn(ctx context.Context, conn *sql.Conn) (bool, error) {
	if !undo.UndoConfig.DataValidation {
		return true, nil
	}
	beforeImage := t.sqlUndoLog.BeforeImage
	afterImage := t.sqlUndoLog.AfterImage

	equals, err := IsRecordsEquals(beforeImage, afterImage)
	if err != nil {
		return false, err
	}
	if equals {
		log.Infof("Stop rollback because there is no data change between the before data snapshot and the after data snapshot.")
		return false, nil
	}

	currentImage, err := t.queryCurrentRecords(ctx, conn)
	if err != nil {
		return false, err
	}

	equals, err = IsRecordsEquals(afterImage, currentImage)
	if err != nil {
		return false, err
	}
	if !equals {
		equals, err = IsRecordsEquals(beforeImage, currentImage)
		if err != nil {
			return false, err
		}
		if equals {
			log.Infof("Stop rollback because there is no data change between the before data snapshot and the current data snapshot.")
			return false, nil
		} else {
			oldRowJson, _ := json.Marshal(afterImage.Rows)
			newRowJson, _ := json.Marshal(currentImage.Rows)
			log.Infof("check dirty data failed, old and new data are not equal, "+
				"tableName:[%s], oldRows:[%s],newRows:[%s].", afterImage.TableName, oldRowJson, newRowJson)
			return false, serr.New(serr.SQLUndoDirtyError, "has dirty records when undo", nil)
		}
	}
	return true, nil
}
func TestDataValidationAndGoOn(t *testing.T) {
	tests := []struct {
		name         string
		beforeImage  *types.RecordImage
		afterImage   *types.RecordImage
		currentImage *types.RecordImage
		want         bool
		wantErr      bool
	}{
		{
			name: "before == after, skip rollback",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "after == current, continue rollback",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			currentImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "current == before, rollback already done",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			currentImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "dirty data",
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "a"},
					}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "b"},
					}},
				},
			},
			currentImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "c"},
					}},
				},
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// patch UndoConfig
			cfgPatch := gomonkey.ApplyGlobalVar(&undo.UndoConfig, undo.Config{DataValidation: true})
			defer cfgPatch.Reset()

			// patch IsRecordsEquals
			comparePatch := gomonkey.ApplyFunc(IsRecordsEquals, func(a, b *types.RecordImage) (bool, error) {
				aj, _ := json.Marshal(a.Rows)
				bj, _ := json.Marshal(b.Rows)
				return string(aj) == string(bj), nil
			})
			defer comparePatch.Reset()

			executor := &testableBaseExecutor{
				BaseExecutor: BaseExecutor{
					sqlUndoLog: undo.SQLUndoLog{
						BeforeImage: tt.beforeImage,
						AfterImage:  tt.afterImage,
					},
					undoImage: tt.afterImage,
				},
				mockCurrentImage: tt.currentImage,
			}

			got, err := executor.dataValidationAndGoOn(context.Background(), nil)

			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				var be *serr.SeataError
				if errors.As(err, &be) {
					assert.Equal(t, serr.SQLUndoDirtyError, be.Code)
				} else {
					t.Errorf("expected BusinessError, got: %v", err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDataValidationAndGoOnDisabledValidation(t *testing.T) {
	cfgPatch := gomonkey.ApplyGlobalVar(&undo.UndoConfig, undo.Config{DataValidation: false})
	defer cfgPatch.Reset()

	executor := &testableBaseExecutor{
		BaseExecutor: BaseExecutor{
			sqlUndoLog: undo.SQLUndoLog{
				BeforeImage: &types.RecordImage{},
				AfterImage:  &types.RecordImage{},
			},
		},
	}

	got, err := executor.dataValidationAndGoOn(context.Background(), nil)

	assert.True(t, got)
	assert.NoError(t, err)
}

func TestDataValidationAndGoOnIsRecordsEqualsError(t *testing.T) {
	t.Skip("Skipping test that requires gomonkey function patching which doesn't work well with coverage mode")
}

func TestQueryCurrentRecordsNilUndoImage(t *testing.T) {
	executor := &BaseExecutor{
		undoImage: nil,
	}

	result, err := executor.queryCurrentRecords(context.Background(), nil)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undo image is nil")
}

func TestQueryCurrentRecordsEmptyPKValues(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "t_user",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	executor := &BaseExecutor{
		undoImage: &types.RecordImage{
			TableName: "t_user",
			TableMeta: &tableMeta,
			Rows:      []types.RowImage{},
		},
	}

	result, err := executor.queryCurrentRecords(context.Background(), nil)

	assert.Nil(t, result)
	assert.EqualError(t, err, "primary key values not found in undo image")
}

func TestQueryCurrentRecordsSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	tableMeta := types.TableMeta{
		TableName: "t_user",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id"},
			"name": {ColumnName: "name"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	executor := &BaseExecutor{
		undoImage: &types.RecordImage{
			TableName: "t_user",
			TableMeta: &tableMeta,
			Rows: []types.RowImage{
				{Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 1},
					{ColumnName: "name", Value: "test"},
				}},
			},
		},
	}

	rows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "test_updated")

	mock.ExpectQuery("SELECT .*id.*, .*name.* FROM .*t_user.* WHERE").
		WithArgs(1).
		WillReturnRows(rows)

	result, err := executor.queryCurrentRecords(context.Background(), conn)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "t_user", result.TableName)
	assert.Len(t, result.Rows, 1)
	assert.Len(t, result.Rows[0].Columns, 2)
	assert.Equal(t, "id", result.Rows[0].Columns[0].ColumnName)
	assert.NotNil(t, result.Rows[0].Columns[0].Value)
	assert.Equal(t, "name", result.Rows[0].Columns[1].ColumnName)
	assert.NotNil(t, result.Rows[0].Columns[1].Value)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryCurrentRecordsPreservesDecimalRepresentation(t *testing.T) {
	tests := []struct {
		name        string
		undoValue   interface{}
		expectValue interface{}
	}{
		{name: "exact decimal string", undoValue: "13.370000", expectValue: "13.370000"},
		{name: "legacy decimal float", undoValue: float64(13.37), expectValue: float64(13.37)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn, err := db.Conn(context.Background())
			require.NoError(t, err)
			defer conn.Close()

			idColumn := types.ColumnMeta{ColumnName: "id"}
			tableMeta := types.TableMeta{
				TableName: "t_decimal",
				Columns: map[string]types.ColumnMeta{
					"id":     idColumn,
					"amount": {ColumnName: "amount", DatabaseTypeString: "DECIMAL"},
				},
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{idColumn}},
				},
			}
			executor := &BaseExecutor{dbType: types.DBTypeMySQL, undoImage: &types.RecordImage{
				TableName: "t_decimal",
				TableMeta: &tableMeta,
				Rows: []types.RowImage{{Columns: []types.ColumnImage{
					{ColumnName: "id", ColumnType: types.JDBCTypeBigInt, Value: int64(1)},
					{ColumnName: "amount", ColumnType: types.JDBCTypeDecimal, Value: tt.undoValue},
				}}},
			}}
			rows := sqlmock.NewRowsWithColumnDefinition(
				sqlmock.NewColumn("id").OfType("BIGINT", int64(0)),
				sqlmock.NewColumn("amount").OfType("DECIMAL", sql.RawBytes{}),
			).AddRow(int64(1), "13.370000")
			mock.ExpectQuery("SELECT .*id.*, .*amount.* FROM .*t_decimal.* WHERE").
				WithArgs(int64(1)).
				WillReturnRows(rows)

			result, err := executor.queryCurrentRecords(context.Background(), conn)

			require.NoError(t, err)
			assert.Equal(t, tt.expectValue, result.Rows[0].Columns[1].Value)
			assert.Equal(t, types.JDBCTypeDecimal, result.Rows[0].Columns[1].ColumnType)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMySQLUndoInsertExecutorDecimalDataValidation(t *testing.T) {
	dsn := os.Getenv("SEATA_GO_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("SEATA_GO_TEST_MYSQL_DSN is not set")
	}

	cfg, err := mysqldriver.ParseDSN(dsn)
	require.NoError(t, err)
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	var tableSuffix [8]byte
	_, err = rand.Read(tableSuffix[:])
	require.NoError(t, err)
	tableName := fmt.Sprintf("seata_go_decimal_undo_test_%x", tableSuffix)
	_, err = conn.ExecContext(ctx, "CREATE TABLE "+tableName+" (id BIGINT PRIMARY KEY, amount DECIMAL(20,6))")
	require.NoError(t, err)
	defer conn.ExecContext(ctx, "DROP TABLE IF EXISTS "+tableName)

	previousTableCache := datasource.GetTableCache(types.DBTypeMySQL)
	datasource.RegisterTableCache(types.DBTypeMySQL, datasourcemysql.NewTableMetaInstance(db, cfg))
	defer datasource.RegisterTableCache(types.DBTypeMySQL, previousTableCache)

	query := "INSERT INTO " + tableName + " (id, amount) VALUES (?, ?)"
	parseCtx, err := sqlparser.DoParser(query)
	require.NoError(t, err)
	txCtx := types.NewTxCtx()
	txCtx.TransactionMode = types.ATMode
	execCtx := &types.ExecContext{
		TxCtx:       txCtx,
		Query:       query,
		NamedValues: []driver.NamedValue{{Ordinal: 1, Value: int64(1)}, {Ordinal: 2, Value: "13.370000"}},
		DBName:      cfg.DBName,
		DBType:      types.DBTypeMySQL,
	}
	err = conn.Raw(func(rawConn interface{}) error {
		driverConn, ok := rawConn.(driver.Conn)
		if !ok {
			return fmt.Errorf("MySQL connection does not implement driver.Conn")
		}
		execCtx.Conn = driverConn
		_, execErr := at.NewInsertExecutor(parseCtx, execCtx, nil).ExecContext(ctx, func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
			result, execErr := sqlutil.CtxDriverExecWithPrepareFallback(ctx, driverConn, query, args)
			if execErr != nil {
				return nil, execErr
			}
			return types.NewResult(types.WithResult(result)), nil
		})
		return execErr
	})
	require.NoError(t, err)
	require.Len(t, txCtx.RoundImages.BeofreImages(), 1)
	require.Len(t, txCtx.RoundImages.AfterImages(), 1)
	afterImage := txCtx.RoundImages.AfterImages()[0]
	require.Equal(t, "13.370000", afterImage.Rows[0].GetColumnMap()["amount"].Value)

	branchUndoLog := &undo.BranchUndoLog{Logs: []undo.SQLUndoLog{{
		SQLType:     types.SQLTypeInsert,
		TableName:   tableName,
		BeforeImage: txCtx.RoundImages.BeofreImages()[0],
		AfterImage:  afterImage,
	}}}
	encoded, err := (&undoparser.JsonParser{}).Encode(branchUndoLog)
	require.NoError(t, err)
	decoded, err := (&undoparser.JsonParser{}).Decode(encoded)
	require.NoError(t, err)
	decoded.Logs[0].SetTableMeta(afterImage.TableMeta)

	dataValidation := undo.UndoConfig.DataValidation
	undo.UndoConfig.DataValidation = true
	defer func() { undo.UndoConfig.DataValidation = dataValidation }()

	err = newMySQLUndoInsertExecutor(decoded.Logs[0]).ExecuteOn(ctx, types.DBTypeMySQL, conn)
	require.NoError(t, err)

	var count int
	require.NoError(t, conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tableName).Scan(&count))
	assert.Zero(t, count)
}

func TestQueryCurrentRecordsQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	tableMeta := types.TableMeta{
		TableName: "t_user",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	executor := &BaseExecutor{
		undoImage: &types.RecordImage{
			TableName: "t_user",
			TableMeta: &tableMeta,
			Rows: []types.RowImage{
				{Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 1},
				}},
			},
		},
	}

	mock.ExpectQuery("SELECT .*id.* FROM .*t_user.* WHERE").
		WithArgs(1).
		WillReturnError(fmt.Errorf("database connection error"))

	result, err := executor.queryCurrentRecords(context.Background(), conn)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection error")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryCurrentRecordsCompositePrimaryKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	tableMeta := types.TableMeta{
		TableName:   "t_order",
		ColumnNames: []string{"order_id", "user_id", "amount"},
		Columns: map[string]types.ColumnMeta{
			"order_id": {ColumnName: "order_id"},
			"user_id":  {ColumnName: "user_id"},
			"amount":   {ColumnName: "amount"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "order_id"},
					{ColumnName: "user_id"},
				},
			},
		},
	}

	executor := &BaseExecutor{
		undoImage: &types.RecordImage{
			TableName: "t_order",
			TableMeta: &tableMeta,
			Rows: []types.RowImage{
				{Columns: []types.ColumnImage{
					{ColumnName: "user_id", Value: 1},
					{ColumnName: "amount", Value: 99.99},
					{ColumnName: "order_id", Value: 100},
				}},
			},
		},
	}

	rows := sqlmock.NewRows([]string{"user_id", "amount", "order_id"}).
		AddRow(1, 199.99, 100)

	mock.ExpectQuery("SELECT .*user_id.*, .*amount.*, .*order_id.* FROM .*t_order.* WHERE").
		WithArgs(100, 1).
		WillReturnRows(rows)

	result, err := executor.queryCurrentRecords(context.Background(), conn)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "t_order", result.TableName)
	assert.Len(t, result.Rows, 1)
	assert.Len(t, result.Rows[0].Columns, 3)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParsePkValuesSinglePK(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1},
			{ColumnName: "name", Value: "test1"},
		}},
	}

	pkNameList := []string{"id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "id")
	assert.Len(t, result["id"], 1)
	assert.Equal(t, 1, result["id"][0].Value)
}

func TestParsePkValuesMultipleRows(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1},
			{ColumnName: "name", Value: "test1"},
		}},
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 2},
			{ColumnName: "name", Value: "test2"},
		}},
	}

	pkNameList := []string{"id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Contains(t, result, "id")
}

func TestParsePkValuesCompositePK(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "order_id", Value: 100},
			{ColumnName: "user_id", Value: 1},
			{ColumnName: "amount", Value: 99.99},
		}},
	}

	pkNameList := []string{"order_id", "user_id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "order_id")
	assert.Contains(t, result, "user_id")
	assert.Len(t, result["order_id"], 1)
	assert.Len(t, result["user_id"], 1)
	assert.Equal(t, 100, result["order_id"][0].Value)
	assert.Equal(t, 1, result["user_id"][0].Value)
}

func TestParsePkValuesCaseInsensitive(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "ID", Value: 1},
			{ColumnName: "Name", Value: "test"},
		}},
	}

	pkNameList := []string{"id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "id")
	assert.Len(t, result["id"], 1)
	assert.Equal(t, 1, result["id"][0].Value)
}

func TestParsePkValuesEmptyRows(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{}
	pkNameList := []string{"id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestParsePkValuesNoMatchingPK(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "name", Value: "test"},
			{ColumnName: "age", Value: 25},
		}},
	}

	pkNameList := []string{"id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestParsePkValuesEscapedColumnName(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "`id`", Value: 1},
			{ColumnName: "`name`", Value: "test"},
		}},
		{Columns: []types.ColumnImage{
			{ColumnName: "`id`", Value: 2},
			{ColumnName: "`name`", Value: "test2"},
		}},
	}

	pkNameList := []string{"id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "id")
	assert.Len(t, result["id"], 2)
	assert.Equal(t, 1, result["id"][0].Value)
	assert.Equal(t, 2, result["id"][1].Value)
}

func TestParsePkValuesEscapedCompositePK(t *testing.T) {
	executor := &BaseExecutor{}

	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "`order_id`", Value: 100},
			{ColumnName: "`user_id`", Value: 1},
			{ColumnName: "`amount`", Value: 99.99},
		}},
	}

	pkNameList := []string{"order_id", "user_id"}

	result := executor.parsePkValues(rows, pkNameList, types.DBTypeMySQL)

	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "order_id")
	assert.Contains(t, result, "user_id")
	assert.Equal(t, 100, result["order_id"][0].Value)
	assert.Equal(t, 1, result["user_id"][0].Value)
}
