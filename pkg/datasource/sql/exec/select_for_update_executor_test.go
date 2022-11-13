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

package exec

import (
	"database/sql/driver"
	"io"
	"testing"
)

var (
	index   = 0
	rowVals = [][]interface{}{
		{1, "oid11"},
		{2, "oid22"},
		{3, "oid33"},
	}
)

func TestBuildSelectPKSQL(t *testing.T) {
	// Todo Fix CI fault , pls solve it
	/*e := SelectForUpdateExecutor{BasicUndoLogBuilder: builder.BasicUndoLogBuilder{}}
	sql := "select name, order_id from t_user where age > ?"

	ctx, err := parser.DoParser(sql)

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
	}

	assert.Nil(t, err)
	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.SelectStmt)

	selSQL, err := e.buildSelectPKSQL(ctx.SelectStmt, metaData)
	assert.Nil(t, err)
	assert.Equal(t, "SELECT SQL_NO_CACHE id,order_id FROM t_user WHERE age>?", selSQL)*/
}

func TestBuildLockKey(t *testing.T) {
	// Todo pls solve panic
	/*e := SelectForUpdateExecutor{BasicUndoLogBuilder: builder.BasicUndoLogBuilder{}}
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
	}
	rows := mockRows{}
	lockkey := e.buildLockKey(rows, metaData)
	assert.Equal(t, "t_user:1_oid11,2_oid22,3_oid33", lockkey)*/
}

type mockRows struct{}

func (m mockRows) Columns() []string {
	//TODO implement me
	panic("implement me")
}

func (m mockRows) Close() error {
	//TODO implement me
	panic("implement me")
}

func (m mockRows) Next(dest []driver.Value) error {
	if index == len(rowVals) {
		return io.EOF
	}

	if len(dest) >= 1 {
		dest[0] = rowVals[index][0]
		dest[1] = rowVals[index][1]
		index++
	}

	return nil
}
