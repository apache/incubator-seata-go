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
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	_ "seata.apache.org/seata-go/v2/pkg/util/log"
)

type stubTableMetaCache struct {
	meta *types.TableMeta
}

func (s *stubTableMetaCache) Init(ctx context.Context, conn *sql.DB) error {
	return nil
}

func (s *stubTableMetaCache) GetTableMeta(ctx context.Context, dbName, table string) (*types.TableMeta, error) {
	return s.meta, nil
}

func (s *stubTableMetaCache) Destroy() error {
	return nil
}

func TestBuildSelectSQLByUpdate(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() {
		undo.UndoConfig = originalUndoConfig
	})

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{
		meta: &types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
		},
	})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age between ? and ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age in (?,?)",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE id=? AND name=_UTF8MB4Jack AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			sourceQuery:     "update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc limit ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, 2},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,id FROM t_user WHERE kk BETWEEN ? AND ? AND id=? AND addr IN (?,?) AND age>? ORDER BY name DESC LIMIT ? FOR UPDATE",
			expectQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
		},
		{
			sourceQuery:     "update t_user set id = id where id = ?",
			sourceQueryArgs: []driver.Value{100},
			expectQuery:     "SELECT SQL_NO_CACHE id FROM t_user WHERE id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery)
			assert.Nil(t, err)
			executor := NewUpdateExecutor(c, &types.ExecContext{Values: tt.sourceQueryArgs, NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs)}, []exec.SQLHook{})
			query, args, err := executor.(*updateExecutor).buildBeforeImageSQL(context.Background(), util.ValueToNamedValue(tt.sourceQueryArgs))
			assert.Nil(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}

	compositeMeta := &types.TableMeta{
		TableName:   "t_order",
		ColumnNames: []string{"tenant_id", "id", "name"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "tenant_id"}, {ColumnName: "id"}},
		}},
	}
	rows := []types.RowImage{
		{Columns: []types.ColumnImage{{ColumnName: "name", Value: "A"}, {ColumnName: "id", Value: 1}, {ColumnName: "tenant_id", Value: 10}}},
		{Columns: []types.ColumnImage{{ColumnName: "name", Value: "B"}, {ColumnName: "id", Value: 2}, {ColumnName: "tenant_id", Value: 20}}},
	}
	executor := &updateExecutor{execContext: &types.ExecContext{DBType: types.DBTypeMySQL}}
	query, args := executor.buildAfterImageSQL(types.RecordImage{Rows: rows}, compositeMeta)
	assert.Contains(t, query, "(`tenant_id`,`id`) IN ((?,?),(?,?))")
	assert.Equal(t, 1, strings.Count(query, "name,id,tenant_id"))
	assert.Equal(t, []driver.Value{10, 1, 20, 2}, util.NamedValueToValue(args))
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})
	query, _ = executor.buildAfterImageSQL(types.RecordImage{Rows: rows}, compositeMeta)
	assert.Contains(t, query, "SELECT * FROM")
}

func TestUpdateExecutorAccumulatesOnlyEffectiveBatchItems(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	meta := &types.TableMeta{
		TableName:   "account",
		ColumnNames: []string{"id", "balance"},
		Columns: map[string]types.ColumnMeta{
			"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
			"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})

	ctrl := gomock.NewController(t)
	conn := mock.NewMockTestDriverConn(ctrl)
	gomock.InOrder(
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), int64(100)}), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), int64(110)}), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), int64(110)}), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), int64(110)}), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), int64(110)}), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), int64(120)}), nil),
		conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(newDeleteRows(meta.ColumnNames), nil),
	)

	const query = "update account set id = id, balance = ? where id = ? order by id limit 1"
	txCtx := types.NewTxCtx()
	txCtx.TransactionMode = types.ATMode
	callbacks := 0
	for _, item := range []struct {
		args         []driver.Value
		rowsAffected int64
	}{
		{args: []driver.Value{int64(110), int64(1)}, rowsAffected: 1},
		{args: []driver.Value{int64(110), int64(1)}},
		{args: []driver.Value{int64(120), int64(1)}, rowsAffected: 1},
		{args: []driver.Value{int64(999), int64(2)}},
	} {
		parserCtx, err := parser.DoParser(query)
		assert.NoError(t, err)
		namedValues := util.ValueToNamedValue(item.args)
		executor := NewUpdateExecutor(parserCtx, &types.ExecContext{
			Query: query, NamedValues: namedValues, Conn: conn, TxCtx: txCtx, DBType: types.DBTypeMySQL,
		}, nil)
		_, err = executor.ExecContext(context.Background(), func(_ context.Context, businessQuery string, businessArgs []driver.NamedValue) (types.ExecResult, error) {
			callbacks++
			assert.Equal(t, query, businessQuery)
			assert.Equal(t, namedValues, businessArgs)
			return types.NewResult(types.WithResult(driver.RowsAffected(item.rowsAffected))), nil
		})
		assert.NoError(t, err)
	}

	beforeImages := txCtx.RoundImages.BeofreImages()
	afterImages := txCtx.RoundImages.AfterImages()
	assert.Len(t, beforeImages, 2)
	assert.Len(t, afterImages, 2)
	assert.EqualValues(t, 100, beforeImages[0].Rows[0].GetColumnMap()["balance"].Value)
	assert.EqualValues(t, 110, afterImages[0].Rows[0].GetColumnMap()["balance"].Value)
	assert.EqualValues(t, 110, beforeImages[1].Rows[0].GetColumnMap()["balance"].Value)
	assert.EqualValues(t, 120, afterImages[1].Rows[0].GetColumnMap()["balance"].Value)
	assert.Equal(t, 4, callbacks)
	assert.Equal(t, map[string]struct{}{"ACCOUNT:1": {}}, txCtx.LockKeys)
}

func TestUpdateExecutorDoesNotAppendArtifactsOnFailure(t *testing.T) {
	meta := &types.TableMeta{
		TableName:   "account",
		ColumnNames: []string{"id", "balance"},
		Columns: map[string]types.ColumnMeta{
			"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
			"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
		},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: meta})

	for _, tt := range []struct {
		name          string
		beforeErr     error
		businessErr   error
		afterErr      error
		wantCallbacks int
	}{
		{name: "before query fails", beforeErr: assert.AnError},
		{name: "business update fails", businessErr: assert.AnError, wantCallbacks: 1},
		{name: "after query fails", afterErr: assert.AnError, wantCallbacks: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			conn := mock.NewMockTestDriverConn(ctrl)
			beforeRows := newDeleteRows(meta.ColumnNames, []driver.Value{int64(1), int64(100)})
			beforeCall := conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(beforeRows, tt.beforeErr)
			if tt.afterErr != nil {
				afterCall := conn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, tt.afterErr)
				gomock.InOrder(beforeCall, afterCall)
			}

			const query = "update account set balance = ? where id = ?"
			parserCtx, err := parser.DoParser(query)
			assert.NoError(t, err)
			txCtx := types.NewTxCtx()
			executor := NewUpdateExecutor(parserCtx, &types.ExecContext{
				Query: query, NamedValues: util.ValueToNamedValue([]driver.Value{int64(110), int64(1)}), Conn: conn, TxCtx: txCtx,
			}, nil)

			callbacks := 0
			_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
				callbacks++
				if tt.businessErr != nil {
					return nil, tt.businessErr
				}
				return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
			})

			assert.ErrorIs(t, err, assert.AnError)
			assert.Equal(t, tt.wantCallbacks, callbacks)
			assert.Empty(t, txCtx.RoundImages.BeofreImages())
			assert.Empty(t, txCtx.RoundImages.AfterImages())
			assert.Empty(t, txCtx.LockKeys)
		})
	}
}

func TestBuildSelectSQLByUpdate_PostgreSQL(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() {
		undo.UndoConfig = originalUndoConfig
	})

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	datasource.RegisterTableCache(types.DBTypePostgreSQL, &stubTableMetaCache{
		meta: &types.TableMeta{
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
		},
	})

	sourceQueryArgs := []driver.Value{"Jack", 1, 100, 18, 28}
	c, err := parser.DoParser("update t_user set name = $1, age = $2 where id = $3 and name = 'Jack' and age between $4 and $5")
	assert.Nil(t, err)

	executor := NewUpdateExecutor(c, &types.ExecContext{
		DBType:      types.DBTypePostgreSQL,
		DBName:      "public",
		Values:      sourceQueryArgs,
		NamedValues: util.ValueToNamedValue(sourceQueryArgs),
	}, []exec.SQLHook{})

	query, args, err := executor.(*updateExecutor).buildBeforeImageSQL(context.Background(), util.ValueToNamedValue(sourceQueryArgs))
	assert.Nil(t, err)
	assert.Equal(t, "SELECT name,age,id FROM t_user WHERE id=$1 AND name='Jack' AND age BETWEEN $2 AND $3 FOR UPDATE", query)
	assert.Equal(t, []driver.Value{100, 18, 28}, util.NamedValueToValue(args))

	meta := &types.TableMeta{
		TableName: "t_user",
		Indexs: map[string]types.IndexMeta{
			"id": {
				IType: types.IndexTypePrimaryKey,
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
		},
	}
	beforeImage := types.RecordImage{Rows: []types.RowImage{
		{Columns: []types.ColumnImage{{ColumnName: "name", Value: "Jack"}, {ColumnName: "age", Value: 1}, {ColumnName: "id", Value: 100}}},
		{Columns: []types.ColumnImage{{ColumnName: "name", Value: "Jill"}, {ColumnName: "age", Value: 2}, {ColumnName: "id", Value: 101}}},
	}}
	afterSQL, afterArgs := executor.(*updateExecutor).buildAfterImageSQL(beforeImage, meta)
	assert.NotContains(t, afterSQL, "SQL_NO_CACHE")
	assert.NotContains(t, afterSQL, "`")
	assert.Contains(t, afterSQL, `("id") IN (($1),($2))`)
	assert.Equal(t, 1, strings.Count(afterSQL, "name,age,id"))
	assert.Equal(t, []driver.Value{100, 101}, util.NamedValueToValue(afterArgs))

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})
	afterSQL, _ = executor.(*updateExecutor).buildAfterImageSQL(beforeImage, meta)
	assert.Contains(t, afterSQL, "SELECT * FROM")
}

func TestBuildSelectSQLByUpdateRejectsPrimaryKeyChange(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	datasource.RegisterTableCache(types.DBTypeMySQL, &stubTableMetaCache{meta: &types.TableMeta{
		TableName:   "t_user",
		ColumnNames: []string{"tenant_id", "id", "name"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType:   types.IndexTypePrimaryKey,
			Columns: []types.ColumnMeta{{ColumnName: "tenant_id"}, {ColumnName: "id"}},
		}},
	}})

	for _, query := range []string{
		"update t_user set id = ? where tenant_id = ? and id = ?",
		"update t_user set tenant_id = tenant_id, id = id + 0 where tenant_id = ? and id = ?",
	} {
		parserCtx, err := parser.DoParser(query)
		assert.NoError(t, err)
		executor := NewUpdateExecutor(parserCtx, &types.ExecContext{
			Query: query, NamedValues: util.ValueToNamedValue([]driver.Value{1, 2, 3}), TxCtx: types.NewTxCtx(), DBType: types.DBTypeMySQL,
		}, nil)
		callbacks := 0
		_, err = executor.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
			callbacks++
			return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
		})
		assert.ErrorContains(t, err, "updating primary key column")
		assert.Zero(t, callbacks)
	}
}
