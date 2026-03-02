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
	"fmt"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestIsRecordsEquals(t *testing.T) {
	tests := []struct {
		name         string
		beforeImage  *types.RecordImage
		afterImage   *types.RecordImage
		expectResult bool
		expectErr    bool
	}{
		{
			name:         "both images are nil",
			beforeImage:  nil,
			afterImage:   nil,
			expectResult: true,
			expectErr:    false,
		},
		{
			name:        "before image is nil, after image is not nil",
			beforeImage: nil,
			afterImage: &types.RecordImage{
				TableName: "test_table",
				Rows:      []types.RowImage{},
			},
			expectResult: false,
			expectErr:    false,
		},
		{
			name: "after image is nil, before image is not nil",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows:      []types.RowImage{},
			},
			afterImage:   nil,
			expectResult: false,
			expectErr:    false,
		},
		{
			name: "different table names",
			beforeImage: &types.RecordImage{
				TableName: "test_table1",
				Rows:      []types.RowImage{},
			},
			afterImage: &types.RecordImage{
				TableName: "test_table2",
				Rows:      []types.RowImage{},
			},
			expectResult: false,
			expectErr:    false,
		},
		{
			name: "different row counts",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{ColumnName: "id", Value: 1}}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "test_table",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{ColumnName: "id", Value: 1}}},
					{Columns: []types.ColumnImage{{ColumnName: "id", Value: 2}}},
				},
			},
			expectResult: false,
			expectErr:    false,
		},
		{
			name: "both images have empty rows",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				Rows:      []types.RowImage{},
			},
			afterImage: &types.RecordImage{
				TableName: "test_table",
				Rows:      []types.RowImage{},
			},
			expectResult: true,
			expectErr:    false,
		},
		{
			name: "same table name and row count",
			beforeImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
						},
					},
				},
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{ColumnName: "id", Value: 1}}},
				},
			},
			afterImage: &types.RecordImage{
				TableName: "test_table",
				TableMeta: &types.TableMeta{
					TableName: "test_table",
					Indexs: map[string]types.IndexMeta{
						"PRIMARY": {
							IType:      types.IndexTypePrimaryKey,
							ColumnName: "id",
						},
					},
				},
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{ColumnName: "id", Value: 1}}},
				},
			},
			expectResult: true,
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock compareRows function for the case where we need to compare actual rows
			if tt.beforeImage != nil && tt.afterImage != nil &&
				tt.beforeImage.TableName == tt.afterImage.TableName &&
				len(tt.beforeImage.Rows) == len(tt.afterImage.Rows) &&
				len(tt.beforeImage.Rows) > 0 {

				patches := gomonkey.ApplyFunc(compareRows, func(tableMeta types.TableMeta, oldRows []types.RowImage, newRows []types.RowImage) (bool, error) {
					return tt.expectResult, nil
				})
				defer patches.Reset()
			}

			result, err := IsRecordsEquals(tt.beforeImage, tt.afterImage)

			assert.Equal(t, tt.expectResult, result)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompareRows(t *testing.T) {
	tests := []struct {
		name         string
		tableMeta    types.TableMeta
		oldRows      []types.RowImage
		newRows      []types.RowImage
		expectResult bool
		expectErr    bool
	}{
		{
			name: "identical rows",
			tableMeta: types.TableMeta{
				TableName: "test_table",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "id",
					},
				},
			},
			oldRows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "test"},
					},
				},
			},
			newRows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "test"},
					},
				},
			},
			expectResult: true,
			expectErr:    false,
		},
		{
			name: "different values",
			tableMeta: types.TableMeta{
				TableName: "test_table",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "id",
					},
				},
			},
			oldRows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "old_name"},
					},
				},
			},
			newRows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "new_name"},
					},
				},
			},
			expectResult: false,
			expectErr:    false,
		},
		{
			name: "missing row in new data",
			tableMeta: types.TableMeta{
				TableName: "test_table",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "id",
					},
				},
			},
			oldRows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "test"},
					},
				},
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 2},
						{ColumnName: "name", Value: "test2"},
					},
				},
			},
			newRows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "test"},
					},
				},
			},
			expectResult: false,
			expectErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock TableMeta.GetPrimaryKeyOnlyName
			patches := gomonkey.ApplyMethod(&tt.tableMeta, "GetPrimaryKeyOnlyName", func() []string {
				if primaryIndex, exists := tt.tableMeta.Indexs["PRIMARY"]; exists {
					return []string{primaryIndex.ColumnName}
				}
				return []string{"id"}
			})
			defer patches.Reset()

			// Mock datasource.DeepEqual
			patches.ApplyFunc(datasource.DeepEqual, func(a, b interface{}) bool {
				return a == b
			})

			result, err := compareRows(tt.tableMeta, tt.oldRows, tt.newRows)

			assert.Equal(t, tt.expectResult, result)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRowListToMap(t *testing.T) {
	tests := []struct {
		name           string
		rows           []types.RowImage
		primaryKeyList []string
		expectedCount  int
	}{
		{
			name: "single primary key",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "test"},
					},
				},
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 2},
						{ColumnName: "name", Value: "test2"},
					},
				},
			},
			primaryKeyList: []string{"id"},
			expectedCount:  2,
		},
		{
			name: "composite primary key",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "user_id", Value: 1},
						{ColumnName: "order_id", Value: 100},
						{ColumnName: "amount", Value: 99.99},
					},
				},
			},
			primaryKeyList: []string{"user_id", "order_id"},
			expectedCount:  1,
		},
		{
			name:           "empty rows",
			rows:           []types.RowImage{},
			primaryKeyList: []string{"id"},
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rowListToMap(tt.rows, tt.primaryKeyList)

			assert.Len(t, result, tt.expectedCount)

			if len(tt.rows) > 0 {
				for _, row := range tt.rows {
					// Verify that each row can be found in the map by constructing expected key
					var expectedKey string
					var firstUnderline bool
					for _, column := range row.Columns {
						for i, key := range tt.primaryKeyList {
							if column.ColumnName == key {
								if firstUnderline && i > 0 {
									expectedKey += "_##$$_"
								}
								expectedKey += fmt.Sprintf("%v", column.GetActualValue())
								firstUnderline = true
							}
						}
					}

					rowData, exists := result[expectedKey]
					assert.True(t, exists, "Row should exist in map with key: %s", expectedKey)

					if exists {
						for _, column := range row.Columns {
							value, ok := rowData[strings.ToUpper(column.ColumnName)]
							assert.True(t, ok, "Column %s should exist in row data", column.ColumnName)
							assert.Equal(t, column.Value, value, "Column %s value should match", column.ColumnName)
						}
					}
				}
			}
		})
	}
}

func TestBuildWhereConditionByPKs(t *testing.T) {
	tests := []struct {
		name        string
		pkNameList  []string
		rowSize     int
		maxInSize   int
		expectedSQL string
	}{
		{
			name:        "single PK single row",
			pkNameList:  []string{"id"},
			rowSize:     1,
			maxInSize:   1000,
			expectedSQL: "(`id`) IN ((?)",
		},
		{
			name:        "single PK multiple rows within limit",
			pkNameList:  []string{"id"},
			rowSize:     3,
			maxInSize:   1000,
			expectedSQL: "(`id`) IN ((?),(?),(?)",
		},
		{
			name:        "composite PK single row",
			pkNameList:  []string{"user_id", "order_id"},
			rowSize:     1,
			maxInSize:   1000,
			expectedSQL: "(`user_id`,`order_id`) IN ((?,?)",
		},
		{
			name:        "single PK with batching",
			pkNameList:  []string{"id"},
			rowSize:     3,
			maxInSize:   2,
			expectedSQL: "(`id`) IN ((?),(?)) OR (`id`) IN ((?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildWhereConditionByPKs(tt.pkNameList, tt.rowSize, tt.maxInSize)

			assert.Contains(t, result, tt.expectedSQL)
			// Verify structure contains proper closing parentheses
			assert.Contains(t, result, ")")
		})
	}
}

func TestBuildPKParams(t *testing.T) {
	tests := []struct {
		name         string
		rows         []types.RowImage
		pkNameList   []string
		expectedLen  int
		expectedVals []interface{}
	}{
		{
			name: "single PK",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 1},
						{ColumnName: "name", Value: "test"},
					},
				},
				{
					Columns: []types.ColumnImage{
						{ColumnName: "id", Value: 2},
						{ColumnName: "name", Value: "test2"},
					},
				},
			},
			pkNameList:   []string{"id"},
			expectedLen:  2,
			expectedVals: []interface{}{1, 2},
		},
		{
			name: "composite PK",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "user_id", Value: 1},
						{ColumnName: "order_id", Value: 100},
						{ColumnName: "amount", Value: 99.99},
					},
				},
			},
			pkNameList:   []string{"user_id", "order_id"},
			expectedLen:  2,
			expectedVals: []interface{}{1, 100},
		},
		{
			name:         "empty rows",
			rows:         []types.RowImage{},
			pkNameList:   []string{"id"},
			expectedLen:  0,
			expectedVals: []interface{}{},
		},
		{
			name: "missing PK column",
			rows: []types.RowImage{
				{
					Columns: []types.ColumnImage{
						{ColumnName: "name", Value: "test"},
					},
				},
			},
			pkNameList:   []string{"id"},
			expectedLen:  0,
			expectedVals: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPKParams(tt.rows, tt.pkNameList)

			assert.Len(t, result, tt.expectedLen)
			assert.Equal(t, tt.expectedVals, result)
		})
	}
}
