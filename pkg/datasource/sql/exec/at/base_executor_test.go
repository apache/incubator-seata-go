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

	"github.com/arana-db/parser/mysql"
	parserTypes "github.com/arana-db/parser/types"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

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

func TestBaseExecutor_GetScanSlice(t *testing.T) {
	exec := &baseExecutor{}

	tests := []struct {
		name        string
		columnNames []string
		tableMeta   *types.TableMeta
		validate    func(t *testing.T, result []interface{})
	}{
		{
			name:        "nullable int column",
			columnNames: []string{"id"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeLong,
							Flag: 0,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*sql.NullInt64)
				assert.True(t, ok, "expected *sql.NullInt64")
			},
		},
		{
			name:        "not null int column",
			columnNames: []string{"id"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeLonglong,
							Flag: mysql.NotNullFlag,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*int64)
				assert.True(t, ok, "expected *int64")
			},
		},
		{
			name:        "nullable float column",
			columnNames: []string{"price"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"price": {
						ColumnName: "price",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeDouble,
							Flag: 0,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*sql.NullFloat64)
				assert.True(t, ok, "expected *sql.NullFloat64")
			},
		},
		{
			name:        "not null decimal column",
			columnNames: []string{"amount"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"amount": {
						ColumnName: "amount",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeNewDecimal,
							Flag: mysql.NotNullFlag,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*float64)
				assert.True(t, ok, "expected *float64")
			},
		},
		{
			name:        "datetime column",
			columnNames: []string{"created_at"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"created_at": {
						ColumnName: "created_at",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeDatetime,
							Flag: 0,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*sql.NullTime)
				assert.True(t, ok, "expected *sql.NullTime")
			},
		},
		{
			name:        "varchar column",
			columnNames: []string{"name"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"name": {
						ColumnName: "name",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeVarchar,
							Flag: 0,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*sql.NullString)
				assert.True(t, ok, "expected *sql.NullString")
			},
		},
		{
			name:        "json column",
			columnNames: []string{"config"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"config": {
						ColumnName: "config",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeJSON,
							Flag: 0,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*sql.NullString)
				assert.True(t, ok, "expected *sql.NullString")
			},
		},
		{
			name:        "nil FieldType fallback",
			columnNames: []string{"unknown"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"unknown": {
						ColumnName: "unknown",
						FieldType:  nil,
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 1)
				_, ok := result[0].(*sql.RawBytes)
				assert.True(t, ok, "expected *sql.RawBytes for nil FieldType")
			},
		},
		{
			name:        "multiple columns mixed types",
			columnNames: []string{"id", "name", "price", "created_at"},
			tableMeta: &types.TableMeta{
				Columns: map[string]types.ColumnMeta{
					"id": {
						ColumnName: "id",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeLonglong,
							Flag: mysql.NotNullFlag | mysql.PriKeyFlag,
						},
					},
					"name": {
						ColumnName: "name",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeVarchar,
							Flag: 0,
						},
					},
					"price": {
						ColumnName: "price",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeDouble,
							Flag: 0,
						},
					},
					"created_at": {
						ColumnName: "created_at",
						FieldType: &parserTypes.FieldType{
							Tp:   mysql.TypeTimestamp,
							Flag: mysql.NotNullFlag,
						},
					},
				},
			},
			validate: func(t *testing.T, result []interface{}) {
				assert.Len(t, result, 4)
				_, ok := result[0].(*int64)
				assert.True(t, ok, "id should be *int64")
				_, ok = result[1].(*sql.NullString)
				assert.True(t, ok, "name should be *sql.NullString")
				_, ok = result[2].(*sql.NullFloat64)
				assert.True(t, ok, "price should be *sql.NullFloat64")
				_, ok = result[3].(*sql.NullTime)
				assert.True(t, ok, "created_at should be *sql.NullTime")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := exec.GetScanSlice(tt.columnNames, tt.tableMeta)
			tt.validate(t, result)
		})
	}
}
