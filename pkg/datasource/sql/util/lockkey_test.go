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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestBuildLockKey_SinglePrimaryKey(t *testing.T) {
	// Create table meta with single primary key
	tableMeta := types.TableMeta{
		TableName: "users",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id", DatabaseType: 4},
			"name": {ColumnName: "name", DatabaseType: 12},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"id", "name"},
	}

	// Create record image
	recordImage := &types.RecordImage{
		TableName: "users",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: int64(1), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "name", Value: "Alice", KeyType: types.IndexTypeNull},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "USERS:1", lockKey)
}

func TestBuildLockKey_CompositePrimaryKey(t *testing.T) {
	// Create table meta with composite primary key
	tableMeta := types.TableMeta{
		TableName: "order_items",
		Columns: map[string]types.ColumnMeta{
			"order_id": {ColumnName: "order_id", DatabaseType: 4},
			"item_id":  {ColumnName: "item_id", DatabaseType: 4},
			"quantity": {ColumnName: "quantity", DatabaseType: 4},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "order_id", DatabaseType: 4},
					{ColumnName: "item_id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"order_id", "item_id", "quantity"},
	}

	// Create record image
	recordImage := &types.RecordImage{
		TableName: "order_items",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "order_id", Value: int64(100), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "item_id", Value: int64(5), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "quantity", Value: int64(3), KeyType: types.IndexTypeNull},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "ORDER_ITEMS:100_5", lockKey)
}

func TestBuildLockKey_MultipleRows(t *testing.T) {
	// Create table meta
	tableMeta := types.TableMeta{
		TableName: "products",
		Columns: map[string]types.ColumnMeta{
			"id":    {ColumnName: "id", DatabaseType: 4},
			"name":  {ColumnName: "name", DatabaseType: 12},
			"price": {ColumnName: "price", DatabaseType: 8},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"id", "name", "price"},
	}

	// Create record image with multiple rows
	recordImage := &types.RecordImage{
		TableName: "products",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: int64(1), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "name", Value: "Product A", KeyType: types.IndexTypeNull},
					{ColumnName: "price", Value: float64(19.99), KeyType: types.IndexTypeNull},
				},
			},
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: int64(2), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "name", Value: "Product B", KeyType: types.IndexTypeNull},
					{ColumnName: "price", Value: float64(29.99), KeyType: types.IndexTypeNull},
				},
			},
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: int64(3), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "name", Value: "Product C", KeyType: types.IndexTypeNull},
					{ColumnName: "price", Value: float64(39.99), KeyType: types.IndexTypeNull},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "PRODUCTS:1,2,3", lockKey)
}

func TestBuildLockKey_EmptyRows(t *testing.T) {
	// Create table meta
	tableMeta := types.TableMeta{
		TableName: "users",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id", DatabaseType: 4},
			"name": {ColumnName: "name", DatabaseType: 12},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"id", "name"},
	}

	// Create record image with no rows
	recordImage := &types.RecordImage{
		TableName: "users",
		SQLType:   types.SQLTypeUpdate,
		Rows:      []types.RowImage{},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "USERS:", lockKey)
}

func TestBuildLockKey_NilValue(t *testing.T) {
	// Create table meta
	tableMeta := types.TableMeta{
		TableName: "users",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id", DatabaseType: 4},
			"name": {ColumnName: "name", DatabaseType: 12},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"id", "name"},
	}

	// Create record image with nil primary key value
	recordImage := &types.RecordImage{
		TableName: "users",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: nil, KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "name", Value: "Alice", KeyType: types.IndexTypeNull},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "USERS:", lockKey)
}

func TestBuildLockKey_StringPrimaryKey(t *testing.T) {
	// Create table meta with string primary key
	tableMeta := types.TableMeta{
		TableName: "categories",
		Columns: map[string]types.ColumnMeta{
			"code": {ColumnName: "code", DatabaseType: 12},
			"name": {ColumnName: "name", DatabaseType: 12},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "code", DatabaseType: 12},
				},
			},
		},
		ColumnNames: []string{"code", "name"},
	}

	// Create record image
	recordImage := &types.RecordImage{
		TableName: "categories",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "code", Value: "CAT001", KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "name", Value: "Electronics", KeyType: types.IndexTypeNull},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "CATEGORIES:CAT001", lockKey)
}

func TestBuildLockKey_CompositeKeyMultipleRows(t *testing.T) {
	// Create table meta with composite primary key
	tableMeta := types.TableMeta{
		TableName: "user_roles",
		Columns: map[string]types.ColumnMeta{
			"user_id": {ColumnName: "user_id", DatabaseType: 4},
			"role_id": {ColumnName: "role_id", DatabaseType: 4},
			"granted": {ColumnName: "granted", DatabaseType: 91},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "user_id", DatabaseType: 4},
					{ColumnName: "role_id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"user_id", "role_id", "granted"},
	}

	// Create record image with multiple rows
	recordImage := &types.RecordImage{
		TableName: "user_roles",
		SQLType:   types.SQLTypeInsert,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "user_id", Value: int64(10), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "role_id", Value: int64(1), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "granted", Value: "2023-01-01", KeyType: types.IndexTypeNull},
				},
			},
			{
				Columns: []types.ColumnImage{
					{ColumnName: "user_id", Value: int64(10), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "role_id", Value: int64(2), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "granted", Value: "2023-01-02", KeyType: types.IndexTypeNull},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "USER_ROLES:10_1,10_2", lockKey)
}

func TestBuildLockKey_LowercaseTableName(t *testing.T) {
	// Create table meta with lowercase table name
	tableMeta := types.TableMeta{
		TableName: "my_table",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id", DatabaseType: 4},
			"data": {ColumnName: "data", DatabaseType: 12},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"id", "data"},
	}

	// Create record image
	recordImage := &types.RecordImage{
		TableName: "my_table",
		SQLType:   types.SQLTypeDelete,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: int64(999), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "data", Value: "test", KeyType: types.IndexTypeNull},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "MY_TABLE:999", lockKey)
}

func TestBuildLockKey_MixedNilValues(t *testing.T) {
	// Create table meta with composite key
	tableMeta := types.TableMeta{
		TableName: "test_table",
		Columns: map[string]types.ColumnMeta{
			"pk1": {ColumnName: "pk1", DatabaseType: 4},
			"pk2": {ColumnName: "pk2", DatabaseType: 4},
			"pk3": {ColumnName: "pk3", DatabaseType: 4},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "pk1", DatabaseType: 4},
					{ColumnName: "pk2", DatabaseType: 4},
					{ColumnName: "pk3", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"pk1", "pk2", "pk3"},
	}

	// Create record image with mixed nil values
	recordImage := &types.RecordImage{
		TableName: "test_table",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "pk1", Value: int64(1), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "pk2", Value: nil, KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "pk3", Value: int64(3), KeyType: types.IndexTypePrimaryKey},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "TEST_TABLE:1__3", lockKey)
}

func TestBuildLockKey_DifferentDataTypes(t *testing.T) {
	// Create table meta
	tableMeta := types.TableMeta{
		TableName: "mixed_types",
		Columns: map[string]types.ColumnMeta{
			"id":      {ColumnName: "id", DatabaseType: 4},
			"code":    {ColumnName: "code", DatabaseType: 12},
			"version": {ColumnName: "version", DatabaseType: 4},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", DatabaseType: 4},
					{ColumnName: "code", DatabaseType: 12},
					{ColumnName: "version", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"id", "code", "version"},
	}

	// Create record image
	recordImage := &types.RecordImage{
		TableName: "mixed_types",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: int64(100), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "code", Value: "ABC", KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "version", Value: int32(5), KeyType: types.IndexTypePrimaryKey},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "MIXED_TYPES:100_ABC_5", lockKey)
}

func TestBuildLockKey_ColumnOrderMatters(t *testing.T) {
	// Create table meta where column order in Columns slice differs from ColumnNames order
	tableMeta := types.TableMeta{
		TableName: "ordered_table",
		Columns: map[string]types.ColumnMeta{
			"pk1": {ColumnName: "pk1", DatabaseType: 4},
			"pk2": {ColumnName: "pk2", DatabaseType: 4},
			"pk3": {ColumnName: "pk3", DatabaseType: 4},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "pk1", DatabaseType: 4},
					{ColumnName: "pk2", DatabaseType: 4},
					{ColumnName: "pk3", DatabaseType: 4},
				},
			},
		},
		// The order in ColumnNames determines the lock key order
		ColumnNames: []string{"pk1", "pk2", "pk3"},
	}

	// Create record image with columns in different order
	recordImage := &types.RecordImage{
		TableName: "ordered_table",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "pk3", Value: int64(3), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "pk1", Value: int64(1), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "pk2", Value: int64(2), KeyType: types.IndexTypePrimaryKey},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	// Should respect the order defined in ColumnNames: pk1, pk2, pk3
	assert.Equal(t, "ORDERED_TABLE:1_2_3", lockKey)
}

func TestBuildLockKey_PartialNilInCompositeKey(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "test_partial_nil",
		Columns: map[string]types.ColumnMeta{
			"pk1": {ColumnName: "pk1", DatabaseType: 4},
			"pk2": {ColumnName: "pk2", DatabaseType: 4},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "pk1", DatabaseType: 4},
					{ColumnName: "pk2", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"pk1", "pk2"},
	}

	recordImage := &types.RecordImage{
		TableName: "test_partial_nil",
		SQLType:   types.SQLTypeUpdate,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "pk1", Value: int64(100), KeyType: types.IndexTypePrimaryKey},
					{ColumnName: "pk2", Value: nil, KeyType: types.IndexTypePrimaryKey},
				},
			},
		},
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Equal(t, "TEST_PARTIAL_NIL:100_", lockKey)
}

func TestBuildLockKey_LargeNumberOfRows(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "large_batch",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id", DatabaseType: 4},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id", DatabaseType: 4},
				},
			},
		},
		ColumnNames: []string{"id"},
	}

	rows := make([]types.RowImage, 100)
	for i := 0; i < 100; i++ {
		rows[i] = types.RowImage{
			Columns: []types.ColumnImage{
				{ColumnName: "id", Value: int64(i + 1), KeyType: types.IndexTypePrimaryKey},
			},
		}
	}

	recordImage := &types.RecordImage{
		TableName: "large_batch",
		SQLType:   types.SQLTypeInsert,
		Rows:      rows,
	}

	lockKey := BuildLockKey(recordImage, tableMeta)
	assert.Contains(t, lockKey, "LARGE_BATCH:")
	assert.Contains(t, lockKey, "1,2,3")
	assert.Contains(t, lockKey, ",99,100")
}
