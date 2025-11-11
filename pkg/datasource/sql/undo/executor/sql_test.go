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
	"log"
	"testing"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"

	"github.com/stretchr/testify/assert"
)

// TestDelEscape
func TestDelEscape(t *testing.T) {
	strSlice := []string{`"scheme"."id"`, "`scheme`.`id`", `"scheme".id`, `scheme."id"`, `scheme."id"`, "scheme.`id`"}

	for k, v := range strSlice {
		res := DelEscape(v, types.DBTypeMySQL)
		log.Printf("val_%d: %s, res_%d: %s\n", k, v, k, res)
		assert.Equal(t, "scheme.id", res)
	}
}

// TestAddEscape
func TestAddEscape(t *testing.T) {
	strSlice := []string{`"scheme".id`, "`scheme`.id", `scheme."id"`, "scheme.`id`"}

	for k, v := range strSlice {
		res := AddEscape(v, types.DBTypeMySQL)
		log.Printf("val_%d: %s, res_%d: %s\n", k, v, k, res)
		assert.Equal(t, v, res)
	}

	strSlice1 := []string{"ALTER", "ANALYZE"}
	for k, v := range strSlice1 {
		res := AddEscape(v, types.DBTypeMySQL)
		log.Printf("val_%d: %s, res_%d: %s\n", k, v, k, res)
		assert.Equal(t, "`"+v+"`", res)
	}
}

func TestDelEscapeEmptyString(t *testing.T) {
	result := DelEscape("", types.DBTypeMySQL)
	assert.Equal(t, "", result)
}

func TestDelEscapeNoEscape(t *testing.T) {
	result := DelEscape("scheme.id", types.DBTypeMySQL)
	assert.Equal(t, "scheme.id", result)
}

func TestDelEscapePartialEscapeSchemaOnly(t *testing.T) {
	result := DelEscape(`"scheme".id`, types.DBTypeMySQL)
	assert.Equal(t, "scheme.id", result)

	result = DelEscape("`scheme`.id", types.DBTypeMySQL)
	assert.Equal(t, "scheme.id", result)
}

func TestDelEscapePartialEscapeColumnOnly(t *testing.T) {
	result := DelEscape(`scheme."id"`, types.DBTypeMySQL)
	assert.Equal(t, "scheme.id", result)

	result = DelEscape("scheme.`id`", types.DBTypeMySQL)
	assert.Equal(t, "scheme.id", result)
}

func TestDelEscapeSingleColumn(t *testing.T) {
	result := DelEscape(`"id"`, types.DBTypeMySQL)
	assert.Equal(t, "id", result)

	result = DelEscape("`id`", types.DBTypeMySQL)
	assert.Equal(t, "id", result)
}

func TestAddEscapeEmptyString(t *testing.T) {
	result := AddEscape("", types.DBTypeMySQL)
	assert.Equal(t, "", result)
}

func TestAddEscapeAlreadyEscaped(t *testing.T) {
	result := AddEscape("`ALTER`", types.DBTypeMySQL)
	assert.Equal(t, "`ALTER`", result)

	result = AddEscape(`"ALTER"`, types.DBTypePostgreSQL)
	assert.Equal(t, `"ALTER"`, result)
}

func TestAddEscapeNonKeyword(t *testing.T) {
	result := AddEscape("my_column", types.DBTypeMySQL)
	assert.Equal(t, "my_column", result)

	result = AddEscape("user_name", types.DBTypeMySQL)
	assert.Equal(t, "user_name", result)
}

func TestAddEscapeSchemaColumnKeyword(t *testing.T) {
	result := AddEscape("ALTER", types.DBTypeMySQL)
	assert.Equal(t, "`ALTER`", result)

	result = AddEscape("SELECT", types.DBTypeMySQL)
	assert.Equal(t, "`SELECT`", result)
}

func TestAddEscapePostgreSQL(t *testing.T) {
	result := AddEscape("user", types.DBTypePostgreSQL)
	assert.Equal(t, `"user"`, result)

	result = AddEscape("my_table.user", types.DBTypePostgreSQL)
	assert.Equal(t, `"my_table"."user"`, result)
}

func TestAddEscapeOracle(t *testing.T) {
	result := AddEscape("user", types.DBTypeOracle)
	assert.Equal(t, `"user"`, result)
}

func TestAddEscapeSQLServer(t *testing.T) {
	result := AddEscape("user", types.DBTypeSQLServer)
	assert.Equal(t, `"user"`, result)
}

func TestCheckEscapeMySQLKeywords(t *testing.T) {
	keywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "ALTER", "CREATE", "DROP", "TABLE"}
	for _, keyword := range keywords {
		result := checkEscape(keyword, types.DBTypeMySQL)
		assert.True(t, result, "Expected %s to be a MySQL keyword", keyword)

		lowerKeyword := keyword
		result = checkEscape(lowerKeyword, types.DBTypeMySQL)
		assert.True(t, result, "Expected %s to be a MySQL keyword (case-insensitive)", lowerKeyword)
	}
}

func TestCheckEscapeNonKeyword(t *testing.T) {
	result := checkEscape("my_column", types.DBTypeMySQL)
	assert.False(t, result)

	result = checkEscape("user_name", types.DBTypeMySQL)
	assert.False(t, result)
}

func TestCheckEscapeNonMySQL(t *testing.T) {
	result := checkEscape("anything", types.DBTypePostgreSQL)
	assert.True(t, result)

	result = checkEscape("anything", types.DBTypeOracle)
	assert.True(t, result)

	result = checkEscape("anything", types.DBTypeSQLServer)
	assert.True(t, result)
}

func TestGetOrderedPkListSinglePK(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "t_user",
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id"},
			"name": {ColumnName: "name"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	image := &types.RecordImage{
		TableName: "t_user",
		TableMeta: &tableMeta,
	}

	row := types.RowImage{
		Columns: []types.ColumnImage{
			{ColumnName: "id", Value: 1, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "name", Value: "test", KeyType: types.IndexTypeNull},
		},
	}

	result, err := GetOrderedPkList(image, row, types.DBTypeMySQL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "id", result[0].ColumnName)
	assert.Equal(t, 1, result[0].Value)
}

func TestGetOrderedPkListCompositePK(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "t_order",
		Columns: map[string]types.ColumnMeta{
			"order_id": {ColumnName: "order_id"},
			"user_id":  {ColumnName: "user_id"},
			"amount":   {ColumnName: "amount"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "order_id"},
					{ColumnName: "user_id"},
				},
			},
		},
	}

	image := &types.RecordImage{
		TableName: "t_order",
		TableMeta: &tableMeta,
	}

	row := types.RowImage{
		Columns: []types.ColumnImage{
			{ColumnName: "user_id", Value: 1, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "order_id", Value: 100, KeyType: types.IndexTypePrimaryKey},
			{ColumnName: "amount", Value: 99.99, KeyType: types.IndexTypeNull},
		},
	}

	result, err := GetOrderedPkList(image, row, types.DBTypeMySQL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, "order_id", result[0].ColumnName)
	assert.Equal(t, 100, result[0].Value)
	assert.Equal(t, "user_id", result[1].ColumnName)
	assert.Equal(t, 1, result[1].Value)
}

func TestGetOrderedPkListWithEscapedNames(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "t_user",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	image := &types.RecordImage{
		TableName: "t_user",
		TableMeta: &tableMeta,
	}

	row := types.RowImage{
		Columns: []types.ColumnImage{
			{ColumnName: "`id`", Value: 1, KeyType: types.IndexTypePrimaryKey},
		},
	}

	result, err := GetOrderedPkList(image, row, types.DBTypeMySQL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "id", result[0].ColumnName)
}

func TestGetOrderedPkListEmptyRow(t *testing.T) {
	tableMeta := types.TableMeta{
		TableName: "t_user",
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}

	image := &types.RecordImage{
		TableName: "t_user",
		TableMeta: &tableMeta,
	}

	row := types.RowImage{
		Columns: []types.ColumnImage{},
	}

	result, err := GetOrderedPkList(image, row, types.DBTypeMySQL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}
