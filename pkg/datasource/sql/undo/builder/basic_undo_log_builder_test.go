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

package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestBuildWhereConditionByPKs(t *testing.T) {
	builder := BasicUndoLogBuilder{}
	tests := []struct {
		name       string
		pkNameList []string
		rowSize    int
		maxInSize  int
		expectSQL  string
	}{
		{"test1", []string{"id", "name"}, 1, 1, "(`id`,`name`) IN ((?,?))"},
		{"test1", []string{"id", "name"}, 3, 2, "(`id`,`name`) IN ((?,?),(?,?)) OR (`id`,`name`) IN ((?,?))"},
		{"test1", []string{"id", "name"}, 3, 1, "(`id`,`name`) IN ((?,?)) OR (`id`,`name`) IN ((?,?)) OR (`id`,`name`) IN ((?,?))"},
		{"test1", []string{"id", "name"}, 4, 2, "(`id`,`name`) IN ((?,?),(?,?)) OR (`id`,`name`) IN ((?,?),(?,?))"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// todo add dbType param
			sql := builder.buildWhereConditionByPKs(test.pkNameList, test.rowSize, "", test.maxInSize)
			assert.Equal(t, test.expectSQL, sql)
		})
	}
}

func TestBuildLockKey(t *testing.T) {
	var builder BasicUndoLogBuilder

	tests := []struct {
		name     string
		metaData types.TableMeta
		records  types.RecordImage
		expected string
	}{
		{
			name: "Two Primary Keys",
			metaData: types.TableMeta{
				TableName: "test_name",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}, {ColumnName: "userId"}}},
				},
			},
			records: types.RecordImage{
				TableName: "test_name",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 1}, {KeyType: types.IndexTypePrimaryKey, ColumnName: "userId", Value: "one"}}},
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 2}, {KeyType: types.IndexTypePrimaryKey, ColumnName: "userId", Value: "two"}}},
				},
			},
			expected: "test_name:1_one,2_two",
		},
		{
			name: "Single Primary Key",
			metaData: types.TableMeta{
				TableName: "single_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
				},
			},
			records: types.RecordImage{
				TableName: "single_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 100}}},
				},
			},
			expected: "single_key:100",
		},
		{
			name: "Mixed Type Keys",
			metaData: types.TableMeta{
				TableName: "mixed_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "name"}, {ColumnName: "age"}}},
				},
			},
			records: types.RecordImage{
				TableName: "mixed_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "name", Value: "Alice"}, {KeyType: types.IndexTypePrimaryKey, ColumnName: "age", Value: 25}}},
				},
			},
			expected: "mixed_key:Alice_25",
		},
		{
			name: "Empty Records",
			metaData: types.TableMeta{
				TableName: "empty",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
				},
			},
			records:  types.RecordImage{TableName: "empty"},
			expected: "empty:",
		},
		{
			name: "Duplicate Primary Keys",
			metaData: types.TableMeta{
				TableName: "dupes",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
				},
			},
			records: types.RecordImage{
				TableName: "dupes",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 1}}},
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 1}}},
				},
			},
			expected: "dupes:1,1",
		},
		{
			name: "Special Characters",
			metaData: types.TableMeta{
				TableName: "special",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
				},
			},
			records: types.RecordImage{
				TableName: "special",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: "a,b_c"}}},
				},
			},
			expected: "special:a,b_c",
		},
		{
			name: "Non-existent Key Name",
			metaData: types.TableMeta{
				TableName: "error_key",
				Indexs: map[string]types.IndexMeta{
					"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "non_existent"}}},
				},
			},
			records: types.RecordImage{
				TableName: "error_key",
				Rows: []types.RowImage{
					{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 1}}},
				},
			},
			expected: "error_key:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := builder.buildLockKey2(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}
