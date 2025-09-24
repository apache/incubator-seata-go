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

func TestBuildLockKey(t *testing.T) {
	// Test with empty records
	meta := types.TableMeta{
		TableName: "test_table",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id"},
			"name": {ColumnName: "name"},
		},
		Indexs: map[string]types.IndexMeta{
			"id": {
				Name:    "id",
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
				IType:   types.IndexTypePrimaryKey,
			},
		},
	}

	records := &types.RecordImage{
		Rows: []types.RowImage{},
	}

	result := BuildLockKey(records, meta)
	assert.Equal(t, "TEST_TABLE:", result)

	// Test with single row record
	meta2 := types.TableMeta{
		TableName: "user_table",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id"},
			"name": {ColumnName: "name"},
		},
		Indexs: map[string]types.IndexMeta{
			"id": {
				Name:    "id",
				Columns: []types.ColumnMeta{{ColumnName: "id"}},
				IType:   types.IndexTypePrimaryKey,
			},
		},
	}

	records2 := &types.RecordImage{
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 123},
					{ColumnName: "name", Value: "test"},
				},
			},
		},
	}

	result2 := BuildLockKey(records2, meta2)
	assert.Equal(t, "USER_TABLE:123", result2)

	// Test with multiple row records
	records3 := &types.RecordImage{
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 123},
					{ColumnName: "name", Value: "test"},
				},
			},
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 456},
					{ColumnName: "name", Value: "test2"},
				},
			},
		},
	}

	result3 := BuildLockKey(records3, meta2)
	assert.Equal(t, "USER_TABLE:123,456", result3)
}