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
	"database/sql/driver"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	index     = 0
	rowValues = [][]interface{}{
		{1, "oid11"},
		{2, "oid22"},
		{3, "oid33"},
	}
)

func TestBuildSelectPKSQL(t *testing.T) {
	e := selectForUpdateExecutor{}
	sql := "select name, order_id from t_user where age > ? for update"

	ctx, err := parser.DoParser(sql)

	metaData := types.TableMeta{
		TableName:   "t_user",
		ColumnNames: []string{"id", "order_id", "age"},
		Indexs: map[string]types.IndexMeta{
			"id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "id",
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
			"order_id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "order_id",
				Columns: []types.ColumnMeta{
					{ColumnName: "order_id"},
				},
			},
			"age": {
				IType:      types.IndexTypeNull,
				ColumnName: "age",
				Columns: []types.ColumnMeta{
					{ColumnName: "age"},
				},
			},
		},
	}

	assert.Nil(t, err)
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.SelectStmt)

	selSQL, err := e.buildSelectPKSQL(ctx.SelectStmt, &metaData)
	assert.Nil(t, err)
	assert.Equal(t, "SELECT SQL_NO_CACHE id,order_id FROM t_user WHERE age>? FOR UPDATE" == selSQL, true)
}

func TestBuildLockKey(t *testing.T) {
	e := selectForUpdateExecutor{}

	metaData := types.TableMeta{
		TableName: "t_user",
		Indexs: map[string]types.IndexMeta{
			"id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "id",
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
			"order_id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "order_id",
				Columns: []types.ColumnMeta{
					{ColumnName: "order_id"},
				},
			},
			"age": {
				IType:      types.IndexTypeNull,
				ColumnName: "age",
				Columns: []types.ColumnMeta{
					{ColumnName: "age"},
				},
			},
		},
		Columns: map[string]types.ColumnMeta{
			"id": {
				DatabaseTypeString: "INT",
				ColumnName:         "id",
			},
			"order_id": {
				DatabaseTypeString: "VARCHAR",
				ColumnName:         "order_id",
			},
			"age": {
				DatabaseTypeString: "INT",
				ColumnName:         "age",
			},
		},
		ColumnNames: []string{"id", "order_id", "age"},
	}
	rows := mockRows{}
	lockKey := e.buildLockKey(rows, &metaData)
	assert.Equal(t, "t_user:1_oid11,2_oid22,3_oid33", lockKey)
}

type mockRows struct{}

func (m mockRows) Columns() []string {
	return []string{"id", "order_id"}
}

func (m mockRows) Close() error {
	//TODO implement me
	panic("implement me")
}

func (m mockRows) Next(dest []driver.Value) error {
	if index == len(rowValues) {
		return io.EOF
	}

	if len(dest) >= 1 {
		dest[0] = rowValues[index][0]
		dest[1] = rowValues[index][1]
		index++
	}

	return nil
}
