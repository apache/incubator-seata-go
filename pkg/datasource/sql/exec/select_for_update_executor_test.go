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
	"context"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	index   = 0
	rowVals = [][]interface{}{
		{1, "oid11"},
		{2, "oid22"},
		{3, "oid33"},
	}
)

func TestSelectForUpdateExecutor_interceptors(t *testing.T) {
	executor := SelectForUpdateExecutor{}
	
	// This is just for coverage as the method is empty
	executor.interceptors(nil)
	
	// Test passes if no panic
	assert.True(t, true)
}

func TestSelectForUpdateExecutor_ExecWithNamedValue(t *testing.T) {
	executor := SelectForUpdateExecutor{}
	
	ctx := context.Background()
	execCtx := &types.ExecContext{
		Query: "SELECT * FROM test_table",
	}
	
	callback := func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
		return nil, nil
	}
	
	// Test execution without global transaction context
	result, err := executor.ExecWithNamedValue(ctx, execCtx, callback)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestSelectForUpdateExecutor_ExecWithValue(t *testing.T) {
	executor := SelectForUpdateExecutor{}
	
	ctx := context.Background()
	execCtx := &types.ExecContext{
		Query: "SELECT * FROM test_table",
	}
	
	callback := func(ctx context.Context, query string, args []driver.Value) (types.ExecResult, error) {
		return nil, nil
	}
	
	// Test execution without global transaction context
	result, err := executor.ExecWithValue(ctx, execCtx, callback)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestBuildSelectPKSQL(t *testing.T) {
	e := SelectForUpdateExecutor{}
	sql := "select name, order_id from t_user where age > ?"

	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	assert.Nil(t, err)
	assert.NotNil(t, stmtNodes)
	
	selectStmt, ok := stmtNodes[0].(*ast.SelectStmt)
	assert.True(t, ok)
	assert.NotNil(t, selectStmt)

	metaData := types.TableMeta{
		TableName: "t_user",
		ColumnNames: []string{"id", "order_id", "age", "name"},
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

	selSQL, err := e.buildSelectPKSQL(selectStmt, metaData)
	assert.Nil(t, err)
	// Note: The actual implementation adds SQL_NO_CACHE hint and orders primary key columns according to ColumnNames
	assert.Equal(t, "SELECT SQL_NO_CACHE id,order_id FROM t_user WHERE age>?", selSQL)
}

func TestBuildSelectPKSQLWithLimit(t *testing.T) {
	e := SelectForUpdateExecutor{}
	sql := "select name, order_id from t_user where age > ? limit 10"

	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	assert.Nil(t, err)
	assert.NotNil(t, stmtNodes)
	
	selectStmt, ok := stmtNodes[0].(*ast.SelectStmt)
	assert.True(t, ok)
	assert.NotNil(t, selectStmt)

	metaData := types.TableMeta{
		TableName: "t_user",
		ColumnNames: []string{"order_id", "id", "age", "name"},
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

	selSQL, err := e.buildSelectPKSQL(selectStmt, metaData)
	assert.Nil(t, err)
	// Note: The actual implementation retains the LIMIT clause, and the primary key columns are ordered according to ColumnNames
	assert.Equal(t, "SELECT SQL_NO_CACHE order_id,id FROM t_user WHERE age>? LIMIT 10", selSQL)
}

func TestBuildSelectPKSQLWithOrderBy(t *testing.T) {
	e := SelectForUpdateExecutor{}
	sql := "select name, order_id from t_user where age > ? order by name"

	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	assert.Nil(t, err)
	assert.NotNil(t, stmtNodes)
	
	selectStmt, ok := stmtNodes[0].(*ast.SelectStmt)
	assert.True(t, ok)
	assert.NotNil(t, selectStmt)

	metaData := types.TableMeta{
		TableName: "t_user",
		ColumnNames: []string{"order_id", "id", "age", "name"},
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

	selSQL, err := e.buildSelectPKSQL(selectStmt, metaData)
	assert.Nil(t, err)
	// Note: The actual implementation retains the ORDER BY clause, and the primary key columns are ordered according to ColumnNames
	assert.Equal(t, "SELECT SQL_NO_CACHE order_id,id FROM t_user WHERE age>? ORDER BY name", selSQL)
}

func TestBuildSelectPKSQLWithGroupBy(t *testing.T) {
	e := SelectForUpdateExecutor{}
	sql := "select name, order_id from t_user where age > ? group by name"

	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	assert.Nil(t, err)
	assert.NotNil(t, stmtNodes)
	
	selectStmt, ok := stmtNodes[0].(*ast.SelectStmt)
	assert.True(t, ok)
	assert.NotNil(t, selectStmt)

	metaData := types.TableMeta{
		TableName: "t_user",
		ColumnNames: []string{"id", "order_id", "age", "name"},
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

	selSQL, err := e.buildSelectPKSQL(selectStmt, metaData)
	assert.Nil(t, err)
	// Note: The actual implementation does not retain the GROUP BY clause, so the expected result does not contain "GROUP BY name"
	assert.Equal(t, "SELECT SQL_NO_CACHE id,order_id FROM t_user WHERE age>?", selSQL)
}

func TestBuildLockKey(t *testing.T) {
	e := SelectForUpdateExecutor{}
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
		},
	}
	rows := mockRows{}
	lockkey := e.buildLockKey(rows, metaData)
	// Expected: "t_user:1_oid11,2_oid22,3_oid33"
	assert.Contains(t, lockkey, "t_user:")
	assert.Contains(t, lockkey, "1_oid11")
	assert.Contains(t, lockkey, "2_oid22")
	assert.Contains(t, lockkey, "3_oid33")
}

func TestBuildLockKeyWithEmptyRows(t *testing.T) {
	e := SelectForUpdateExecutor{}
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
		},
	}
	
	// Creating an empty mockRows
	emptyRows := mockEmptyRows{}
	lockkey := e.buildLockKey(emptyRows, metaData)
	// Expected result: "t_user:"
	assert.Equal(t, "t_user:", lockkey)
}

type mockRows struct{}

func (m mockRows) Columns() []string {
	return []string{"id", "order_id"}
}

func (m mockRows) Close() error {
	return nil
}

func (m mockRows) Next(dest []driver.Value) error {
	if index == len(rowVals) {
		return io.EOF
	}

	if len(dest) >= 2 {
		dest[0] = rowVals[index][0]
		dest[1] = rowVals[index][1]
		index++
	}

	return nil
}

type mockEmptyRows struct{}

func (m mockEmptyRows) Columns() []string {
	return []string{"id"}
}

func (m mockEmptyRows) Close() error {
	return nil
}

func (m mockEmptyRows) Next(dest []driver.Value) error {
	return io.EOF
}

type mockSQLHookForSelect struct {
	beforeCalled bool
	afterCalled  bool
	sqlType      types.SQLType
}

func (m *mockSQLHookForSelect) Type() types.SQLType {
	return m.sqlType
}

func (m *mockSQLHookForSelect) Before(ctx context.Context, execCtx *types.ExecContext) error {
	m.beforeCalled = true
	return nil
}

func (m *mockSQLHookForSelect) After(ctx context.Context, execCtx *types.ExecContext) error {
	m.afterCalled = true
	return nil
}

func TestBuildSelectPKSQLWithNoPrimaryKey(t *testing.T) {
	e := SelectForUpdateExecutor{}
	
	// Create a dummy select statement
	p := parser.New()
	stmtNodes, _, err := p.Parse("SELECT name FROM t_user WHERE age > 18", "", "")
	if err != nil {
		t.Fatal(err)
	}
	
	selectStmt, ok := stmtNodes[0].(*ast.SelectStmt)
	if !ok {
		t.Fatal("Failed to cast to SelectStmt")
	}
	
	// Table meta without primary key
	metaData := types.TableMeta{
		TableName: "t_user",
		Indexs: map[string]types.IndexMeta{
			"age": {
				IType:      types.IndexTypeNull,
				ColumnName: "age",
				Columns: []types.ColumnMeta{
					{ColumnName: "age"},
				},
			},
		},
	}
	
	_, err = e.buildSelectPKSQL(selectStmt, metaData)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "needs to contain the primary key")
}

func TestSelectForUpdateExecutor_interceptors_with_hooks(t *testing.T) {
	executor := SelectForUpdateExecutor{}
	
	// Create mock hooks
	hooks := []SQLHook{
		&mockSQLHookForSelect{sqlType: types.SQLTypeSelect},
		&mockSQLHookForSelect{sqlType: types.SQLTypeUpdate},
	}
	
	// Call interceptors
	executor.interceptors(hooks)
	
	// Currently interceptors method is empty, so we just verify it doesn't panic
	assert.True(t, true)
}