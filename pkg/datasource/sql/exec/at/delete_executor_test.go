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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

func TestNewDeleteExecutor(t *testing.T) {
	executor := NewDeleteExecutor(nil, nil, nil)
	_, ok := executor.(*deleteExecutor)
	assert.Equalf(t, true, ok, "should be *deleteExecutor")
}

func Test_deleteExecutor_buildBeforeImageSQL(t *testing.T) {
	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "delete from t_user where id = ?",
			sourceQueryArgs: []driver.Value{100},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
		{
			sourceQuery:     "delete from t_user where id = ? and name = 'Jack' and age between ? and ?",
			sourceQueryArgs: []driver.Value{100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "delete from t_user where id = ? and name = 'Jack' and age in (?,?)",
			sourceQueryArgs: []driver.Value{100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "delete from t_user where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc limit ?",
			sourceQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE kk BETWEEN ? AND ? AND id=? AND addr IN (?,?) AND age>? ORDER BY name DESC LIMIT ? FOR UPDATE",
			expectQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery)
			assert.Nil(t, err)
			executor := NewDeleteExecutor(c, &types.ExecContext{Values: tt.sourceQueryArgs, NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs)}, []exec.SQLHook{})
			query, args, err := executor.(*deleteExecutor).buildBeforeImageSQL(tt.sourceQuery, util.ValueToNamedValue(tt.sourceQueryArgs))
			assert.Nil(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}
}

func Test_deleteExecutor_buildBeforeImageSQL_PostgreSQL(t *testing.T) {
	sourceQueryArgs := []driver.Value{100, 18, 28}
	c, err := parser.DoParser("delete from t_user where id = $1 and name = 'Jack' and age between $2 and $3")
	assert.Nil(t, err)

	executor := NewDeleteExecutor(c, &types.ExecContext{
		DBType:      types.DBTypePostgreSQL,
		Values:      sourceQueryArgs,
		NamedValues: util.ValueToNamedValue(sourceQueryArgs),
	}, []exec.SQLHook{})
	query, args, err := executor.(*deleteExecutor).buildBeforeImageSQL("delete from t_user where id = $1 and name = 'Jack' and age between $2 and $3", util.ValueToNamedValue(sourceQueryArgs))
	assert.Nil(t, err)
	assert.Equal(t, "SELECT * FROM t_user WHERE id=$1 AND name='Jack' AND age BETWEEN $2 AND $3 FOR UPDATE", query)
	assert.Equal(t, sourceQueryArgs, util.NamedValueToValue(args))
	assert.NotContains(t, query, "SQL_NO_CACHE")
	assert.NotContains(t, query, "`")
}

func TestDeleteExecutorAccumulatesOnlyEffectiveBatchItems(t *testing.T) {
	meta := &types.TableMeta{
		TableName:   "t_order",
		ColumnNames: []string{"tenant_id", "id", "value"},
		Columns: map[string]types.ColumnMeta{
			"tenant_id": {ColumnName: "tenant_id", DatabaseTypeString: "BIGINT"},
			"id":        {ColumnName: "id", DatabaseTypeString: "BIGINT"},
			"value":     {ColumnName: "value", DatabaseTypeString: "VARCHAR"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{
				{ColumnName: "tenant_id"},
				{ColumnName: "id"},
			},
		}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})

	ctrl := gomock.NewController(t)
	conn := mock.NewMockTestDriverConn(ctrl)
	query := "delete from t_order where tenant_id = ? order by id limit 2"
	gomock.InOrder(
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames,
			[]driver.Value{int64(10), int64(1), "a"},
			[]driver.Value{int64(10), int64(2), "b"},
		), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames,
			[]driver.Value{int64(20), int64(3), "c"},
		), nil),
	)

	txCtx := types.NewTxCtx()
	for _, tenantID := range []driver.Value{int64(10), int64(10), int64(20)} {
		parserCtx, err := parser.DoParser(query)
		assert.NoError(t, err)
		namedValues := util.ValueToNamedValue([]driver.Value{tenantID})
		executor := NewDeleteExecutor(parserCtx, &types.ExecContext{
			Query: query, NamedValues: namedValues, Conn: conn, TxCtx: txCtx, DBType: types.DBTypeMySQL,
		}, nil)
		_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
			return types.NewResult(types.WithResult(driver.ResultNoRows)), nil
		})
		assert.NoError(t, err)
	}

	assert.Len(t, txCtx.RoundImages.BeofreImages(), 2)
	assert.Len(t, txCtx.RoundImages.AfterImages(), 2)
	assert.Len(t, txCtx.RoundImages.BeofreImages()[0].Rows, 2)
	assert.Len(t, txCtx.RoundImages.BeofreImages()[1].Rows, 1)
	assert.Contains(t, txCtx.LockKeys, "T_ORDER:10_1,10_2")
	assert.Contains(t, txCtx.LockKeys, "T_ORDER:20_3")
}

func TestDeleteExecutorDoesNotAppendArtifactsOnFailure(t *testing.T) {
	meta := &types.TableMeta{
		TableName:   "t_user",
		ColumnNames: []string{"id"},
		Columns: map[string]types.ColumnMeta{
			"id": {ColumnName: "id", DatabaseTypeString: "BIGINT"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})

	for _, tt := range []struct {
		name          string
		beforeRows    driver.Rows
		beforeErr     error
		businessErr   error
		wantCallbacks int
	}{
		{name: "before query fails", beforeErr: assert.AnError},
		{name: "business delete fails", beforeRows: newDeleteRows(meta.ColumnNames, []driver.Value{int64(1)}), businessErr: assert.AnError, wantCallbacks: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			conn := mock.NewMockTestDriverConn(ctrl)
			conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.beforeRows, tt.beforeErr)

			query := "delete from t_user where id = ?"
			parserCtx, err := parser.DoParser(query)
			assert.NoError(t, err)
			txCtx := types.NewTxCtx()
			executor := NewDeleteExecutor(parserCtx, &types.ExecContext{
				Query: query, NamedValues: util.ValueToNamedValue([]driver.Value{int64(1)}), Conn: conn, TxCtx: txCtx,
			}, nil)

			callbacks := 0
			_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
				callbacks++
				return nil, tt.businessErr
			})

			assert.Error(t, err)
			assert.Equal(t, tt.wantCallbacks, callbacks)
			assert.Empty(t, txCtx.RoundImages.BeofreImages())
			assert.Empty(t, txCtx.RoundImages.AfterImages())
			assert.Empty(t, txCtx.LockKeys)
		})
	}
}

type deleteRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

func newDeleteRows(columns []string, rows ...[]driver.Value) *deleteRows {
	return &deleteRows{columns: columns, rows: rows}
}

func (r *deleteRows) Columns() []string { return r.columns }
func (r *deleteRows) Close() error      { return nil }
func (r *deleteRows) Next(dest []driver.Value) error {
	if r.index == len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.index])
	r.index++
	return nil
}
