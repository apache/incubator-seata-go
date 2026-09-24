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
	"context"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
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

func TestBuildSelectPKSQL_PostgreSQL(t *testing.T) {
	e := selectForUpdateExecutor{
		execContext: &types.ExecContext{DBType: types.DBTypePostgreSQL},
	}
	sql := "select name, order_id from t_user where age > $1 for update"

	ctx, err := parser.DoParser(sql)
	assert.Nil(t, err)

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

	selSQL, err := e.buildSelectPKSQL(ctx.SelectStmt, &metaData)
	assert.Nil(t, err)
	assert.Contains(t, []string{
		"SELECT id,order_id FROM t_user WHERE age>$1 FOR UPDATE",
		"SELECT order_id,id FROM t_user WHERE age>$1 FOR UPDATE",
	}, selSQL)
	assert.NotContains(t, selSQL, "SQL_NO_CACHE")
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

func TestPrepareFallbackRowsCloseStatement(t *testing.T) {
	ctx := context.Background()
	updateSQL := "UPDATE t_user SET name = ? WHERE id = ?"
	deleteSQL := "DELETE FROM t_user WHERE id = ?"
	updateParser, err := parser.DoParser(updateSQL)
	assert.NoError(t, err)
	deleteParser, err := parser.DoParser(deleteSQL)
	assert.NoError(t, err)

	meta := &types.TableMeta{
		TableName:   "t_user",
		ColumnNames: []string{"id", "name"},
		Columns: map[string]types.ColumnMeta{
			"id":   {ColumnName: "id", DatabaseTypeString: "BIGINT"},
			"name": {ColumnName: "name", DatabaseTypeString: "VARCHAR"},
		},
		Indexs: map[string]types.IndexMeta{
			"PRIMARY": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "id",
				Columns:    []types.ColumnMeta{{ColumnName: "id"}},
			},
		},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})
	updateArgs := []driver.NamedValue{{Ordinal: 1, Value: "updated"}, {Ordinal: 2, Value: int64(1)}}
	deleteArgs := []driver.NamedValue{{Ordinal: 1, Value: int64(1)}}
	beforeImage := types.RecordImage{Rows: []types.RowImage{{Columns: []types.ColumnImage{{ColumnName: "id", Value: int64(1)}}}}}

	tests := []struct {
		name string
		run  func(*prepareFallbackConn) error
	}{
		{name: "insert query", run: func(conn *prepareFallbackConn) error {
			rows, err := (&insertExecutor{execContext: &types.ExecContext{Conn: conn}}).queryRows(ctx, "SELECT 1", nil)
			if err != nil {
				return err
			}
			return rows.Close()
		}},
		{name: "select for update", run: func(conn *prepareFallbackConn) error {
			rows, err := (&selectForUpdateExecutor{execContext: &types.ExecContext{Conn: conn}}).exec(ctx, "SELECT 1 FOR UPDATE", nil, nil)
			if err != nil {
				return err
			}
			return rows.Close()
		}},
		{name: "update before image", run: func(conn *prepareFallbackConn) error {
			executor := &updateExecutor{parserCtx: updateParser, execContext: &types.ExecContext{
				Conn: conn, Query: updateSQL, NamedValues: updateArgs, TxCtx: types.NewTxCtx(),
			}}
			_, err := executor.beforeImage(ctx)
			return err
		}},
		{name: "update after image", run: func(conn *prepareFallbackConn) error {
			executor := &updateExecutor{parserCtx: updateParser, execContext: &types.ExecContext{Conn: conn}}
			_, err := executor.afterImage(ctx, beforeImage)
			return err
		}},
		{name: "delete before image", run: func(conn *prepareFallbackConn) error {
			executor := &deleteExecutor{parserCtx: deleteParser, execContext: &types.ExecContext{
				Conn: conn, Query: deleteSQL, NamedValues: deleteArgs, TxCtx: types.NewTxCtx(),
			}}
			_, err := executor.beforeImage(ctx)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &prepareFallbackConn{}
			assert.NoError(t, tt.run(conn))
			assert.True(t, conn.rowsClosed)
			assert.True(t, conn.stmtClosed)
		})
	}
}

type prepareFallbackConn struct {
	rowsClosed bool
	stmtClosed bool
}

func (c *prepareFallbackConn) Prepare(string) (driver.Stmt, error) {
	return &prepareFallbackStmt{conn: c}, nil
}

func (*prepareFallbackConn) Close() error              { return nil }
func (*prepareFallbackConn) Begin() (driver.Tx, error) { return nil, nil }
func (*prepareFallbackConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, driver.ErrSkip
}

type prepareFallbackStmt struct{ conn *prepareFallbackConn }

func (s *prepareFallbackStmt) Close() error { s.conn.stmtClosed = true; return nil }
func (*prepareFallbackStmt) NumInput() int  { return -1 }
func (*prepareFallbackStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, driver.ErrSkip
}
func (s *prepareFallbackStmt) Query([]driver.Value) (driver.Rows, error) {
	return &prepareFallbackRows{conn: s.conn}, nil
}
func (s *prepareFallbackStmt) QueryContext(context.Context, []driver.NamedValue) (driver.Rows, error) {
	return &prepareFallbackRows{conn: s.conn}, nil
}

type prepareFallbackRows struct{ conn *prepareFallbackConn }

func (*prepareFallbackRows) Columns() []string         { return nil }
func (r *prepareFallbackRows) Close() error            { r.conn.rowsClosed = true; return nil }
func (*prepareFallbackRows) Next([]driver.Value) error { return io.EOF }

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
