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
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestGetScanSlicePreservesDecimal(t *testing.T) {
	executor := baseExecutor{}
	meta := &types.TableMeta{Columns: map[string]types.ColumnMeta{
		"amount": {ColumnName: "amount", DatabaseTypeString: "DECIMAL"},
	}}

	scanSlice := executor.GetScanSlice([]string{"amount"}, meta)

	assert.IsType(t, &sql.NullString{}, scanSlice[0])
}

func TestBaseExecBuildLockKey(t *testing.T) {
	var exec baseExecutor

	columnID := types.ColumnMeta{
		ColumnName: "id",
	}
	columnUserId := types.ColumnMeta{
		ColumnName: "userId",
	}
	columnName := types.ColumnMeta{
		ColumnName: "name",
	}
	columnAge := types.ColumnMeta{
		ColumnName: "age",
	}
	columnNonExistent := types.ColumnMeta{
		ColumnName: "non_existent",
	}

	columnsTwoPk := []types.ColumnMeta{columnID, columnUserId}
	columnsThreePk := []types.ColumnMeta{columnID, columnUserId, columnAge}
	columnsMixPk := []types.ColumnMeta{columnName, columnAge}

	getColumnImage := func(columnName string, value interface{}) types.ColumnImage {
		return types.ColumnImage{KeyType: types.IndexTypePrimaryKey, ColumnName: columnName, Value: value}
	}

	tests := []struct {
		name     string
		metaData types.TableMeta
		records  types.RecordImage
		expected string
	}{
		{
			"Two Primary Keys",
			types.TableMeta{
				TableName: "test_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsTwoPk},
				},
			},
			types.RecordImage{
				TableName: "test_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "user1")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "user2")}},
				},
			},
			"TEST_NAME:1_user1,2_user2",
		},
		{
			"Three Primary Keys",
			types.TableMeta{
				TableName: "test2_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsThreePk},
				},
			},
			types.RecordImage{
				TableName: "test2_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "one"), getColumnImage("age", "11")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "two"), getColumnImage("age", "22")}},
					{Columns: []types.ColumnImage{getColumnImage("id", 3), getColumnImage("userId", "three"), getColumnImage("age", "33")}},
				},
			},
			"TEST2_NAME:1_one_11,2_two_22,3_three_33",
		},
		{
			name: "Single Primary Key",
			metaData: types.TableMeta{
				TableName: "single_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "single_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 100)}},
				},
			},
			expected: "SINGLE_KEY:100",
		},
		{
			name: "Mixed Type Keys",
			metaData: types.TableMeta{
				TableName: "mixed_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: columnsMixPk},
				},
			},
			records: types.RecordImage{
				TableName: "mixed_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("name", "mike"), getColumnImage("age", 25)}},
				},
			},
			expected: "MIXED_KEY:mike_25",
		},
		{
			name: "Empty Records",
			metaData: types.TableMeta{
				TableName: "empty",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records:  types.RecordImage{TableName: "empty"},
			expected: "EMPTY:",
		},
		{
			name: "Special Characters",
			metaData: types.TableMeta{
				TableName: "special",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "special",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", "A,b_c")}},
				},
			},
			expected: "SPECIAL:A,b_c",
		},
		{
			name: "Non-existent Key Name",
			metaData: types.TableMeta{
				TableName: "error_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnNonExistent}},
				},
			},
			records: types.RecordImage{
				TableName: "error_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", 1)}},
				},
			},
			expected: "ERROR_KEY:",
		},
		{
			name: "Multiple Rows With Nil PK Value",
			metaData: types.TableMeta{
				TableName: "nil_pk",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnID}},
				},
			},
			records: types.RecordImage{
				TableName: "nil_pk",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("id", nil)}},
					{Columns: []types.ColumnImage{getColumnImage("id", 123)}},
					{Columns: []types.ColumnImage{getColumnImage("id", nil)}},
				},
			},
			expected: "NIL_PK:,123,",
		},
		{
			name: "PK As Bool And Float",
			metaData: types.TableMeta{
				TableName: "type_pk",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{columnName, columnAge}},
				},
			},
			records: types.RecordImage{
				TableName: "type_pk",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("name", true), getColumnImage("age", 3.14)}},
					{Columns: []types.ColumnImage{getColumnImage("name", false), getColumnImage("age", 0.0)}},
				},
			},
			expected: "TYPE_PK:true_3.14,false_0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := exec.buildLockKey(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}

func TestBaseExecutorPrepareUndoPair(t *testing.T) {
	meta := &types.TableMeta{
		TableName: "test_table",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType:   types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
			},
		},
	}
	rows := func(id int) []types.RowImage {
		return []types.RowImage{{Columns: []types.ColumnImage{{
			ColumnName: "id",
			KeyType:    types.IndexTypePrimaryKey,
			Value:      id,
		}}}}
	}

	tests := []struct {
		name       string
		sqlType    types.SQLType
		beforeRows []types.RowImage
		afterRows  []types.RowImage
		wantLock   string
		wantPairs  int
	}{
		{name: "empty update", sqlType: types.SQLTypeUpdate, wantPairs: 0},
		{name: "unchanged update", sqlType: types.SQLTypeUpdate, beforeRows: rows(1), afterRows: rows(1), wantPairs: 0},
		{name: "reordered unchanged update", sqlType: types.SQLTypeUpdate, beforeRows: append(rows(1), rows(2)...), afterRows: append(rows(2), rows(1)...), wantPairs: 0},
		{name: "update locks after image", sqlType: types.SQLTypeUpdate, beforeRows: rows(1), afterRows: rows(2), wantLock: "TEST_TABLE:2", wantPairs: 1},
		{name: "delete locks before image", sqlType: types.SQLTypeDelete, beforeRows: rows(1), wantLock: "TEST_TABLE:1", wantPairs: 1},
		{name: "insert locks after image", sqlType: types.SQLTypeInsert, afterRows: rows(2), wantLock: "TEST_TABLE:2", wantPairs: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txCtx := types.NewTxCtx()
			execCtx := &types.ExecContext{TxCtx: txCtx}
			beforeImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: tt.sqlType, Rows: tt.beforeRows}
			afterImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: tt.sqlType, Rows: tt.afterRows}

			err := (&baseExecutor{}).prepareUndoPair(execCtx, beforeImage, afterImage)

			assert.NoError(t, err)
			assert.Len(t, txCtx.RoundImages.BeofreImages(), tt.wantPairs)
			assert.Len(t, txCtx.RoundImages.AfterImages(), tt.wantPairs)
			assert.Len(t, txCtx.LockKeys, tt.wantPairs)
			if tt.wantLock != "" {
				assert.Contains(t, txCtx.LockKeys, tt.wantLock)
			}
		})
	}
}

func TestBaseExecutorPrepareUndoPairKeepsCompositeLockKeyFormat(t *testing.T) {
	meta := &types.TableMeta{
		TableName:   "test_table",
		ColumnNames: []string{"id", "tenant_id", "value"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}, {ColumnName: "tenant_id"}},
		}},
	}
	beforeRows := []types.RowImage{{Columns: []types.ColumnImage{
		{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
		{ColumnName: "tenant_id", KeyType: types.IndexTypePrimaryKey, Value: 2},
		{ColumnName: "value", Value: "before"},
	}}}
	afterRows := []types.RowImage{{Columns: []types.ColumnImage{
		{ColumnName: "id", KeyType: types.IndexTypePrimaryKey, Value: 1},
		{ColumnName: "tenant_id", KeyType: types.IndexTypePrimaryKey, Value: 2},
		{ColumnName: "value", Value: "after"},
	}}}
	txCtx := types.NewTxCtx()
	execCtx := &types.ExecContext{TxCtx: txCtx, DBType: types.DBTypeMySQL}
	beforeImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: types.SQLTypeUpdate, Rows: beforeRows}
	afterImage := &types.RecordImage{TableName: meta.TableName, TableMeta: meta, SQLType: types.SQLTypeUpdate, Rows: afterRows}

	err := (&baseExecutor{}).prepareUndoPair(execCtx, beforeImage, afterImage)

	assert.NoError(t, err)
	assert.Contains(t, txCtx.LockKeys, "TEST_TABLE:1_2")
}

func TestBaseExecutorPrepareUndoPairRejectsInvalidLockImage(t *testing.T) {
	meta := &types.TableMeta{
		TableName: "test_table",
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	primaryKey := func(name string, value interface{}) types.ColumnImage {
		return types.ColumnImage{ColumnName: name, KeyType: types.IndexTypePrimaryKey, Value: value}
	}
	validRows := []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", 1)}}}
	tests := []struct {
		name      string
		meta      *types.TableMeta
		afterRows []types.RowImage
		wantErr   string
	}{
		{name: "chosen lock image is empty", meta: meta, wantErr: "lock image rows are empty"},
		{name: "primary key metadata is empty", meta: &types.TableMeta{TableName: "test_table"}, afterRows: validRows, wantErr: "primary key metadata is empty"},
		{name: "primary key is missing", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{{ColumnName: "name", Value: "test"}}}}, wantErr: "not found in row image"},
		{name: "primary key is duplicated", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", 1), primaryKey("ID", 1)}}}, wantErr: "found more than once"},
		{name: "extra primary key is present", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", 1), primaryKey("tenant_id", 2)}}}, wantErr: "not defined in table metadata"},
		{name: "primary key value is nil", meta: meta, afterRows: []types.RowImage{{Columns: []types.ColumnImage{primaryKey("id", nil)}}}, wantErr: "is nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txCtx := types.NewTxCtx()
			execCtx := &types.ExecContext{TxCtx: txCtx, DBType: types.DBTypeMySQL}
			beforeImage := &types.RecordImage{TableName: "test_table", TableMeta: tt.meta, SQLType: types.SQLTypeUpdate, Rows: validRows}
			afterImage := &types.RecordImage{TableName: "test_table", TableMeta: tt.meta, SQLType: types.SQLTypeUpdate, Rows: tt.afterRows}

			err := (&baseExecutor{}).prepareUndoPair(execCtx, beforeImage, afterImage)

			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, txCtx.LockKeys)
			assert.Empty(t, txCtx.RoundImages.BeofreImages())
			assert.Empty(t, txCtx.RoundImages.AfterImages())
		})
	}
}

func TestBuildImageSelectColumns(t *testing.T) {
	compositeMeta := func() *types.TableMeta {
		return &types.TableMeta{
			ColumnNames: []string{"id", "tenant_id", "name"},
			Indexs: map[string]types.IndexMeta{"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", Autoincrement: true},
					{ColumnName: "tenant_id"},
				},
			}},
		}
	}

	tests := []struct {
		name              string
		meta              *types.TableMeta
		requested         []string
		dbType            types.DBType
		onlyCareRequested bool
		want              []string
		wantErr           string
	}{
		{
			name:              "adds only missing composite primary key",
			meta:              compositeMeta(),
			requested:         []string{"tenant_id", "name"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			want:              []string{"tenant_id", "name", "id"},
		},
		{
			name:              "preserves complete composite primary key",
			meta:              compositeMeta(),
			requested:         []string{"tenant_id", "name", "id"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			want:              []string{"tenant_id", "name", "id"},
		},
		{
			name:              "matches escaped primary key case insensitively",
			meta:              compositeMeta(),
			requested:         []string{"`TENANT_ID`", "order"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			want:              []string{"`TENANT_ID`", "`order`", "id"},
		},
		{
			name:              "uses all columns without explicit columns",
			meta:              compositeMeta(),
			dbType:            types.DBTypePostgreSQL,
			onlyCareRequested: true,
			want:              []string{`"id"`, `"tenant_id"`, `"name"`},
		},
		{
			name:              "uses all columns when only care is disabled",
			meta:              compositeMeta(),
			requested:         []string{"name"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: false,
			want:              []string{"id", "tenant_id", "name"},
		},
		{
			name:    "rejects nil metadata",
			dbType:  types.DBTypeMySQL,
			wantErr: "table meta is nil",
		},
		{
			name: "rejects missing primary key metadata",
			meta: &types.TableMeta{
				ColumnNames: []string{"name"},
			},
			dbType:  types.DBTypeMySQL,
			wantErr: "primary key metadata is empty",
		},
		{
			name:              "rejects duplicate requested columns",
			meta:              compositeMeta(),
			requested:         []string{"id", "`ID`"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			wantErr:           "found more than once",
		},
		{
			name: "rejects duplicate primary key metadata",
			meta: &types.TableMeta{
				ColumnNames: []string{"id", "name"},
				Indexs: map[string]types.IndexMeta{"PRIMARY": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
						{ColumnName: "`ID`"},
					},
				}},
			},
			requested:         []string{"name"},
			dbType:            types.DBTypeMySQL,
			onlyCareRequested: true,
			wantErr:           "exists more than once in metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildImageSelectColumns(tt.meta, tt.requested, tt.dbType, tt.onlyCareRequested)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildImageSelectColumnsDoesNotMutateInput(t *testing.T) {
	meta := &types.TableMeta{
		ColumnNames: []string{"id", "name", "age"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	requested := make([]string, 2, 3)
	copy(requested, []string{"name", "age"})

	_, err := buildImageSelectColumns(meta, requested, types.DBTypePostgreSQL, true)

	assert.NoError(t, err)
	assert.Equal(t, []string{"name", "age"}, requested)
}

func TestBaseExecBuildLockKey_EscapedColumnNames(t *testing.T) {
	var exec baseExecutor

	getColumnImage := func(columnName string, value interface{}) types.ColumnImage {
		return types.ColumnImage{KeyType: types.IndexTypePrimaryKey, ColumnName: columnName, Value: value}
	}

	tests := []struct {
		name     string
		metaData types.TableMeta
		records  types.RecordImage
		expected string
	}{
		{
			name: "Backtick-escaped single PK",
			metaData: types.TableMeta{
				TableName: "test_table",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
				},
			},
			records: types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("`id`", 1), {ColumnName: "`name`", Value: "test"}}},
				},
			},
			expected: "TEST_TABLE:1",
		},
		{
			name: "Backtick-escaped composite PK",
			metaData: types.TableMeta{
				TableName: "orders",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{
						{ColumnName: "order_id"},
						{ColumnName: "user_id"},
					}},
				},
			},
			records: types.RecordImage{
				TableName: "orders",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{getColumnImage("`order_id`", 100), getColumnImage("`user_id`", 1)}},
					{Columns: []types.ColumnImage{getColumnImage("`order_id`", 200), getColumnImage("`user_id`", 2)}},
				},
			},
			expected: "ORDERS:100_1,200_2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := exec.buildLockKey(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}
