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

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableMeta_GetPrimaryKeyTypeStrMap(t *testing.T) {
	type fields struct {
		TableName   string
		Columns     map[string]ColumnMeta
		Indexs      map[string]IndexMeta
		ColumnNames []string
	}

	tests := []struct {
		name   string
		fields fields
		want   map[string]string
	}{
		{name: "test-1", fields: fields{TableName: "test", Indexs: map[string]IndexMeta{
			"id": {
				Name:       "id",
				ColumnName: "id",
				IType:      IndexTypePrimaryKey,
				Columns: []ColumnMeta{
					{
						ColumnName:         "id",
						DatabaseTypeString: "BIGINT",
					},
				},
			},
		}}, want: map[string]string{
			"id": "BIGINT",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := TableMeta{
				TableName:   tt.fields.TableName,
				Columns:     tt.fields.Columns,
				Indexs:      tt.fields.Indexs,
				ColumnNames: tt.fields.ColumnNames,
			}
			got, err := m.GetPrimaryKeyTypeStrMap()
			assert.Nil(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTableMeta_GetPrimaryKeyType(t *testing.T) {
	type fields struct {
		TableName   string
		Columns     map[string]ColumnMeta
		Indexs      map[string]IndexMeta
		ColumnNames []string
	}
	tests := []struct {
		name   string
		fields fields
		want   int32
	}{
		{name: "test-1", fields: fields{TableName: "test", Indexs: map[string]IndexMeta{
			"id": {
				Name:       "id",
				ColumnName: "id",
				IType:      IndexTypePrimaryKey,
				Columns: []ColumnMeta{
					{
						ColumnName:         "id",
						DatabaseTypeString: "BIGINT",
						DatabaseType:       GetSqlDataType("BIGINT"),
					},
				},
			},
		}}, want: GetSqlDataType("BIGINT")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := TableMeta{
				TableName:   tt.fields.TableName,
				Columns:     tt.fields.Columns,
				Indexs:      tt.fields.Indexs,
				ColumnNames: tt.fields.ColumnNames,
			}
			got, err := m.GetPrimaryKeyType()
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "GetPrimaryKeyType()")
		})
	}
}

func TestTableMetaCache_GetTableMeta(t *testing.T) {
	cache := map[string]*TableMeta{
		"table1": &TableMeta{
			TableName: "table1",
			Columns: map[string]ColumnMeta{
				"id": {
					Schema:     "test",
					Table:      "table1",
					ColumnName: "id",
					ColumnType: "int",
				},
			},
		},
	}

	// Test existing table
	meta := cache["table1"]
	assert.NotNil(t, meta)
	assert.Equal(t, "table1", meta.TableName)

	// Test non-existing table
	meta = cache["table2"]
	assert.Nil(t, meta)
}

func TestColumnType_DatabaseTypeName(t *testing.T) {
	meta := ColumnType{DatabaseType: "varchar(255)"}
	assert.Equal(t, "varchar(255)", meta.DatabaseTypeName())
}

func TestTableMeta_IsEmpty(t *testing.T) {
	t.Run("empty table", func(t *testing.T) {
		meta := &TableMeta{}
		assert.True(t, meta.IsEmpty())
	})

	t.Run("table with columns", func(t *testing.T) {
		meta := &TableMeta{
			TableName: "test_table",
			Columns: map[string]ColumnMeta{
				"id": {ColumnName: "id"},
			},
		}
		assert.False(t, meta.IsEmpty())
	})
}

func TestTableMeta_GetPrimaryKeyMap(t *testing.T) {
	meta := &TableMeta{
		Indexs: map[string]IndexMeta{
			"primary": {
				Name:  "primary",
				IType: IndexTypePrimaryKey,
				Columns: []ColumnMeta{
					{ColumnName: "id"},
					{ColumnName: "user_id"},
				},
			},
			"normal": {
				Name:  "normal",
				IType: IndexTypeNull,
				Columns: []ColumnMeta{
					{ColumnName: "name"},
				},
			},
		},
	}

	primaryKeys := meta.GetPrimaryKeyMap()
	assert.Len(t, primaryKeys, 2)
	assert.Contains(t, primaryKeys, "id")
	assert.Contains(t, primaryKeys, "user_id")
}

func TestTableMeta_GetPrimaryKeyOnlyName(t *testing.T) {
	meta := &TableMeta{
		Indexs: map[string]IndexMeta{
			"primary": {
				Name:  "primary",
				IType: IndexTypePrimaryKey,
				Columns: []ColumnMeta{
					{ColumnName: "id"},
					{ColumnName: "user_id"},
				},
			},
		},
	}

	primaryKeys := meta.GetPrimaryKeyOnlyName()
	assert.Len(t, primaryKeys, 2)
	assert.Contains(t, primaryKeys, "id")
	assert.Contains(t, primaryKeys, "user_id")
}
