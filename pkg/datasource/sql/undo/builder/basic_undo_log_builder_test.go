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
	metaData := types.TableMeta{
		TableName: "test_name",
		Indexs: map[string]types.IndexMeta{
			"PRIMARY_KEY": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}, {ColumnName: "userId"}}},
		},
	}

	records := types.RecordImage{
		TableName: "test_name",
		Rows: []types.RowImage{
			{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 1}, {KeyType: types.IndexTypePrimaryKey, ColumnName: "userId", Value: "one"}}},
			{Columns: []types.ColumnImage{{KeyType: types.IndexTypePrimaryKey, ColumnName: "id", Value: 2}, {KeyType: types.IndexTypePrimaryKey, ColumnName: "userId", Value: "two"}}},
		},
	}

	builder := BasicUndoLogBuilder{}
	lockKeys := builder.buildLockKey2(&records, metaData)
	assert.Equal(t, "test_name:1_one,2_two", lockKeys)
}
