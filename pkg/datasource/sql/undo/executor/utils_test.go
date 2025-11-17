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
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestIsRecordsEquals_BothNil(t *testing.T) {
	equal, err := IsRecordsEquals(nil, nil)
	assert.NoError(t, err)
	assert.True(t, equal)
}

func TestIsRecordsEquals_OneNil(t *testing.T) {
	image := &types.RecordImage{
		TableName: "test",
		Rows:      []types.RowImage{},
	}

	equal, err := IsRecordsEquals(image, nil)
	assert.NoError(t, err)
	assert.False(t, equal)

	equal, err = IsRecordsEquals(nil, image)
	assert.NoError(t, err)
	assert.False(t, equal)
}

func TestIsRecordsEquals_DifferentTableNames(t *testing.T) {
	image1 := &types.RecordImage{
		TableName: "table1",
		Rows:      []types.RowImage{},
	}
	image2 := &types.RecordImage{
		TableName: "table2",
		Rows:      []types.RowImage{},
	}

	equal, err := IsRecordsEquals(image1, image2)
	assert.NoError(t, err)
	assert.False(t, equal)
}

func TestIsRecordsEquals_DifferentRowCounts(t *testing.T) {
	image1 := &types.RecordImage{
		TableName: "test",
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{{ColumnName: "id", Value: 1}}},
		},
	}
	image2 := &types.RecordImage{
		TableName: "test",
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{{ColumnName: "id", Value: 1}}},
			{Columns: []types.ColumnImage{{ColumnName: "id", Value: 2}}},
		},
	}

	equal, err := IsRecordsEquals(image1, image2)
	assert.NoError(t, err)
	assert.False(t, equal)
}

func TestIsRecordsEquals_EmptyRows(t *testing.T) {
	image1 := &types.RecordImage{
		TableName: "test",
		Rows:      []types.RowImage{},
	}
	image2 := &types.RecordImage{
		TableName: "test",
		Rows:      []types.RowImage{},
	}

	equal, err := IsRecordsEquals(image1, image2)
	assert.NoError(t, err)
	assert.True(t, equal)
}

func TestIsRecordsEquals_SameData(t *testing.T) {
	tableMeta := &types.TableMeta{
		TableName: "test_table",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				Name:    "PRIMARY",
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
				IType:   types.IndexTypePrimaryKey,
			},
		},
	}

	image1 := &types.RecordImage{
		TableName: "test_table",
		TableMeta: tableMeta,
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{
				{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
				{ColumnName: "name", Value: "test"},
			}},
		},
	}
	image2 := &types.RecordImage{
		TableName: "test_table",
		TableMeta: tableMeta,
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{
				{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
				{ColumnName: "name", Value: "test"},
			}},
		},
	}

	equal, err := IsRecordsEquals(image1, image2)
	assert.NoError(t, err)
	assert.True(t, equal)
}

func TestIsRecordsEquals_DifferentData(t *testing.T) {
	tableMeta := &types.TableMeta{
		TableName: "test_table",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				Name:    "PRIMARY",
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
				IType:   types.IndexTypePrimaryKey,
			},
		},
	}

	image1 := &types.RecordImage{
		TableName: "test_table",
		TableMeta: tableMeta,
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{
				{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
				{ColumnName: "name", Value: "test1"},
			}},
		},
	}
	image2 := &types.RecordImage{
		TableName: "test_table",
		TableMeta: tableMeta,
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{
				{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
				{ColumnName: "name", Value: "test2"},
			}},
		},
	}

	equal, err := IsRecordsEquals(image1, image2)
	assert.NoError(t, err)
	assert.False(t, equal)
}

func TestCompareRows_DifferentPrimaryKeys(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "test_table",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				Name:    "PRIMARY",
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
				IType:   types.IndexTypePrimaryKey,
			},
		},
	}

	oldRows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test"},
		}},
	}
	newRows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 2, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test"},
		}},
	}

	equal, err := compareRows(tableMeta, oldRows, newRows)
	assert.NoError(t, err)
	assert.False(t, equal)
}

func TestRowListToMap(t *testing.T) {
	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test"},
		}},
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 2, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test2"},
		}},
	}

	primaryKeyList := []string{"id"}
	rowMap := rowListToMap(rows, primaryKeyList)

	assert.NotNil(t, rowMap)
	assert.Equal(t, 2, len(rowMap))
	assert.NotNil(t, rowMap["1"])
	assert.NotNil(t, rowMap["2"])
	assert.Equal(t, "test", rowMap["1"]["NAME"])
	assert.Equal(t, "test2", rowMap["2"]["NAME"])
}

func TestRowListToMap_CompositePrimaryKey(t *testing.T) {
	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "user_id", Value: 100, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test"},
		}},
	}

	primaryKeyList := []string{"id", "user_id"}
	rowMap := rowListToMap(rows, primaryKeyList)

	assert.NotNil(t, rowMap)
	assert.Equal(t, 1, len(rowMap))
	// The key should be composite: "1_##$$_100"
	var foundKey string
	for k := range rowMap {
		foundKey = k
		break
	}
	assert.Contains(t, foundKey, "1")
	assert.Contains(t, foundKey, "100")
}

func TestBuildWhereConditionByPKs_SingleBatch(t *testing.T) {
	pkNameList := []string{"id"}
	rowSize := 5
	maxInSize := 1000

	whereSQL := buildWhereConditionByPKs(pkNameList, rowSize, maxInSize)

	assert.NotEmpty(t, whereSQL)
	assert.Contains(t, whereSQL, "(`id`) IN")
	assert.Contains(t, whereSQL, "(?)")     // Should have 5 placeholders
	assert.NotContains(t, whereSQL, " OR ") // Single batch, no OR
}

func TestBuildWhereConditionByPKs_MultipleBatches(t *testing.T) {
	pkNameList := []string{"id"}
	rowSize := 2500
	maxInSize := 1000

	whereSQL := buildWhereConditionByPKs(pkNameList, rowSize, maxInSize)

	assert.NotEmpty(t, whereSQL)
	assert.Contains(t, whereSQL, "(`id`) IN")
	assert.Contains(t, whereSQL, " OR ") // Multiple batches
}

func TestBuildWhereConditionByPKs_CompositePK(t *testing.T) {
	pkNameList := []string{"id", "user_id"}
	rowSize := 3
	maxInSize := 1000

	whereSQL := buildWhereConditionByPKs(pkNameList, rowSize, maxInSize)

	assert.NotEmpty(t, whereSQL)
	assert.Contains(t, whereSQL, "(`id`,`user_id`) IN")
	assert.Contains(t, whereSQL, "(?,?)")
}

func TestBuildWhereConditionByPKs_ExactMultiple(t *testing.T) {
	pkNameList := []string{"id"}
	rowSize := 2000
	maxInSize := 1000

	whereSQL := buildWhereConditionByPKs(pkNameList, rowSize, maxInSize)

	assert.NotEmpty(t, whereSQL)
	assert.Contains(t, whereSQL, " OR ") // Should have 2 batches
}

func TestBuildPKParams(t *testing.T) {
	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test1"},
		}},
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 2, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test2"},
		}},
	}

	pkNameList := []string{"id"}
	params := buildPKParams(rows, pkNameList)

	assert.NotNil(t, params)
	assert.Equal(t, 2, len(params))
	assert.Equal(t, 1, params[0])
	assert.Equal(t, 2, params[1])
}

func TestBuildPKParams_CompositePK(t *testing.T) {
	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "user_id", Value: 100, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test"},
		}},
	}

	pkNameList := []string{"id", "user_id"}
	params := buildPKParams(rows, pkNameList)

	assert.NotNil(t, params)
	assert.Equal(t, 2, len(params))
	assert.Equal(t, 1, params[0])
	assert.Equal(t, 100, params[1])
}

func TestBuildPKParams_NoPK(t *testing.T) {
	rows := []types.RowImage{
		{Columns: []types.ColumnImage{
			{ColumnName: "name", Value: "test"},
		}},
	}

	pkNameList := []string{"id"}
	params := buildPKParams(rows, pkNameList)

	assert.NotNil(t, params)
	assert.Equal(t, 0, len(params))
}
