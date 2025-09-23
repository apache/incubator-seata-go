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
					{[]types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "one")}},
					{[]types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "two")}},
				},
			},
			"TEST_NAME:1_one,2_two",
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
					{[]types.ColumnImage{getColumnImage("id", 1), getColumnImage("userId", "one"), getColumnImage("age", "11")}},
					{[]types.ColumnImage{getColumnImage("id", 2), getColumnImage("userId", "two"), getColumnImage("age", "22")}},
					{[]types.ColumnImage{getColumnImage("id", 3), getColumnImage("userId", "three"), getColumnImage("age", "33")}},
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
					{Columns: []types.ColumnImage{getColumnImage("name", "Alice"), getColumnImage("age", 25)}},
				},
			},
			expected: "MIXED_KEY:Alice_25",
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
					{Columns: []types.ColumnImage{getColumnImage("id", "a,b_c")}},
				},
			},
			expected: "SPECIAL:a,b_c",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lockKeys := builder.buildLockKey2(&tt.records, tt.metaData)
			assert.Equal(t, tt.expected, lockKeys)
		})
	}
}
