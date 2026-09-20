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
	"errors"
	"strings"
	"testing"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/arana-db/parser/ast"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	undoexecutor "seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/executor"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	_ "seata.apache.org/seata-go/v2/pkg/util/log"
)

func TestUpdateJoinBuildAfterImageSQL(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	meta := &types.TableMeta{
		TableName: "t_order", ColumnNames: []string{"id", "status"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	before := types.RecordImage{TableName: meta.TableName, Rows: []types.RowImage{
		{Columns: []types.ColumnImage{{ColumnName: "status", Value: int64(0)}, {ColumnName: "id", Value: int64(42)}}},
	}}
	for _, tt := range []struct {
		name, query, alias, fields, from string
		args                             []driver.Value
	}{
		{
			name: "where argument count", alias: "o", fields: "o.status,o.id", from: "t_order AS o",
			query: "UPDATE t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1 WHERE o.status=? AND i.enabled=?",
			args:  []driver.Value{int64(0), int64(1)},
		},
		{
			name: "where argument value", alias: "o", fields: "o.status,o.id", from: "t_order AS o",
			query: "UPDATE t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1 WHERE o.status=?",
			args:  []driver.Value{int64(0)},
		},
		{
			name: "updated where column", alias: "o", fields: "o.status,o.id", from: "t_order AS o",
			query: "UPDATE t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1 WHERE o.status=0",
		},
		{
			name: "updated join column", alias: "o", fields: "o.status,o.id", from: "t_order AS o",
			query: "UPDATE t_order o JOIN t_item i ON o.id=i.order_id AND o.status=0 SET o.status=1 WHERE o.id=?",
			args:  []driver.Value{int64(42)},
		},
		{
			name: "without alias", fields: "t_order.status,t_order.id", from: "t_order",
			query: "UPDATE t_order JOIN t_item ON t_order.id=t_item.order_id SET t_order.status=1 WHERE t_item.enabled=?",
			args:  []driver.Value{int64(1)},
		},
		{
			name: "order and limit", alias: "o", fields: "o.status,o.id", from: "t_order AS o",
			query: "UPDATE t_order o JOIN t_item i ON o.id=i.order_id SET o.status=? WHERE i.enabled=? ORDER BY i.id LIMIT ?",
			args:  []driver.Value{int64(1), int64(1), int64(2)},
		},
		{
			name: "qualified target", alias: "o", fields: "o.status,o.id", from: "sales.t_order AS o",
			query: "UPDATE sales.t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1",
		},
		{
			name: "qualified right target", alias: "o", fields: "o.status,o.id", from: "sales.t_order AS o",
			query: "UPDATE t_item i JOIN sales.t_order o ON o.id=i.order_id SET o.status=1",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.DoParser(tt.query)
			require.NoError(t, err)
			u := NewUpdateJoinExecutor(parsed, &types.ExecContext{
				DBType: types.DBTypeMySQL, DbVersion: "8.0.36", NamedValues: util.ValueToNamedValue(tt.args),
			}, nil).(*updateJoinExecutor)
			query, args, err := u.buildAfterImageSQL(context.Background(), before, meta, tt.alias)
			require.NoError(t, err)
			assert.Equal(t, "SELECT SQL_NO_CACHE "+tt.fields+" FROM "+tt.from+" WHERE (`id`) IN ((?))", query)
			assert.Equal(t, []driver.NamedValue{{Ordinal: 1, Value: int64(42)}}, args)
			assert.Equal(t, len(args), strings.Count(query, "?"))
		})
	}

	t.Run("primary key validation", func(t *testing.T) {
		parsed, err := parser.DoParser("UPDATE t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1")
		require.NoError(t, err)
		u := NewUpdateJoinExecutor(parsed, &types.ExecContext{DBType: types.DBTypeMySQL}, nil).(*updateJoinExecutor)
		withoutPK := *meta
		withoutPK.Indexs = nil
		_, _, err = u.buildAfterImageSQL(context.Background(), before, &withoutPK, "o")
		require.ErrorContains(t, err, "primary key metadata is empty")

		missingPK := types.RecordImage{Rows: []types.RowImage{
			{Columns: []types.ColumnImage{{ColumnName: "status", Value: int64(0)}}},
		}}
		_, _, err = u.buildAfterImageSQL(context.Background(), missingPK, meta, "o")
		require.ErrorContains(t, err, "incomplete primary keys")
	})

	t.Run("primary key batches", func(t *testing.T) {
		parsed, err := parser.DoParser("UPDATE t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1")
		require.NoError(t, err)
		u := NewUpdateJoinExecutor(parsed, &types.ExecContext{DBType: types.DBTypeMySQL}, nil).(*updateJoinExecutor)
		before := types.RecordImage{Rows: make([]types.RowImage, maxInSize+1)}
		for i := range before.Rows {
			before.Rows[i].Columns = []types.ColumnImage{{ColumnName: "id", Value: int64(i + 1)}}
		}
		query, args, err := u.buildAfterImageSQL(context.Background(), before, meta, "o")
		require.NoError(t, err)
		require.Len(t, args, len(before.Rows))
		assert.Equal(t, len(args), strings.Count(query, "?"))
		assert.Equal(t, 2, strings.Count(query, "(`id`) IN ("))
		assert.Contains(t, query, " OR (`id`) IN ((?))")
		for i, arg := range args {
			assert.Equal(t, driver.NamedValue{Ordinal: i + 1, Value: int64(i + 1)}, arg)
		}
	})

	t.Run("composite keys and all columns", func(t *testing.T) {
		meta := &types.TableMeta{
			TableName: "t_order", ColumnNames: []string{"tenant_id", "id", "status"},
			Indexs: map[string]types.IndexMeta{"PRIMARY": {
				IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}, {ColumnName: "tenant_id"}},
			}},
		}
		before := types.RecordImage{Rows: []types.RowImage{
			{Columns: []types.ColumnImage{{ColumnName: "id", Value: int64(42)}, {ColumnName: "tenant_id", Value: int64(1)}}},
			{Columns: []types.ColumnImage{{ColumnName: "tenant_id", Value: int64(2)}, {ColumnName: "id", Value: int64(43)}}},
		}}
		parsed, err := parser.DoParser("UPDATE t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1 WHERE i.enabled=?")
		require.NoError(t, err)
		u := NewUpdateJoinExecutor(parsed, &types.ExecContext{DBType: types.DBTypeMySQL, DbVersion: "8.0.36"}, nil).(*updateJoinExecutor)
		for _, onlyCareUpdateColumns := range []bool{true, false} {
			undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: onlyCareUpdateColumns})
			query, args, err := u.buildAfterImageSQL(context.Background(), before, meta, "o")
			require.NoError(t, err)
			fields := "*"
			if onlyCareUpdateColumns {
				fields = "o.status,o.tenant_id,o.id"
			}
			assert.Equal(t, "SELECT SQL_NO_CACHE "+fields+" FROM t_order AS o WHERE (`tenant_id`,`id`) IN ((?,?),(?,?))", query)
			assert.Equal(t, []driver.NamedValue{
				{Ordinal: 1, Value: int64(1)}, {Ordinal: 2, Value: int64(42)},
				{Ordinal: 3, Value: int64(2)}, {Ordinal: 4, Value: int64(43)},
			}, args)
		}
	})
}

func TestUpdateJoinBuildBeforeImageSQLAllColumns(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})
	meta := &types.TableMeta{
		TableName: "account", ColumnNames: []string{"id", "balance", "memo", "display-name", "display name", "display`name"},
		Indexs: map[string]types.IndexMeta{"PRIMARY": {
			IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
		}},
	}
	for _, tt := range []struct {
		name, query, alias, from, fields, groupBy, version string
	}{
		{
			name: "alias", version: "8.0.36", alias: "a", fields: "`a`.`id`,`a`.`balance`,`a`.`memo`,`a`.`display-name`,`a`.`display name`,`a`.`display``name`", groupBy: "`a`.`id`",
			query: "UPDATE account a LEFT JOIN detail d ON a.id=d.account_id SET a.balance=1",
			from:  "`account` AS `a` LEFT JOIN `detail` AS `d` ON `a`.`id`=`d`.`account_id`",
		},
		{
			name: "no alias", version: "8.0.36", fields: "`account`.`id`,`account`.`balance`,`account`.`memo`,`account`.`display-name`,`account`.`display name`,`account`.`display``name`", groupBy: "`account`.`id`",
			query: "UPDATE account LEFT JOIN detail ON account.id=detail.account_id SET account.balance=1",
			from:  "`account` LEFT JOIN `detail` ON `account`.`id`=`detail`.`account_id`",
		},
		{
			name: "group by all target columns", version: "5.6.0", alias: "a", fields: "`a`.`id`,`a`.`balance`,`a`.`memo`,`a`.`display-name`,`a`.`display name`,`a`.`display``name`", groupBy: "`a`.`id`,`a`.`balance`,`a`.`memo`,`a`.`display-name`,`a`.`display name`,`a`.`display``name`",
			query: "UPDATE account a LEFT JOIN detail d ON a.id=d.account_id SET a.balance=1",
			from:  "`account` AS `a` LEFT JOIN `detail` AS `d` ON `a`.`id`=`d`.`account_id`",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.DoParser(tt.query)
			require.NoError(t, err)
			u := NewUpdateJoinExecutor(parsed, &types.ExecContext{DBType: types.DBTypeMySQL, DbVersion: tt.version}, nil).(*updateJoinExecutor)
			u.sqlMode = "ONLY_FULL_GROUP_BY"
			query, args, err := u.buildBeforeImageSQL(context.Background(), meta, tt.alias, nil)
			require.NoError(t, err)
			assert.Equal(t, "SELECT SQL_NO_CACHE "+tt.fields+" FROM "+tt.from+" GROUP BY "+tt.groupBy+" FOR UPDATE", query)
			assert.Empty(t, args)
			imageQuery, err := parser.DoParser(query)
			require.NoError(t, err)
			require.Len(t, imageQuery.SelectStmt.Fields.Fields, len(meta.ColumnNames))
			for i, field := range imageQuery.SelectStmt.Fields.Fields {
				column, ok := field.Expr.(*ast.ColumnNameExpr)
				require.True(t, ok, "image field must remain a column reference")
				assert.Equal(t, meta.ColumnNames[i], column.Name.Name.O)
			}
		})
	}
}

func TestUpdateJoinAfterImages(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	originalCache := datasource.GetTableCache(types.DBTypeMySQL)
	t.Cleanup(func() {
		undo.UndoConfig = originalUndoConfig
		datasource.RegisterTableCache(types.DBTypeMySQL, originalCache)
	})
	resultErr := errors.New("affected rows unavailable")
	closeErr := errors.New("query rows close failed")
	for _, tt := range []struct {
		name, detailPKType              string
		beforeAccount, beforeDetail     [][]driver.Value
		afterAccount, afterDetail       [][]driver.Value
		wantAccountRows, wantDetailRows int
		allColumns                      bool
		result                          func(*gomock.Controller) types.ExecResult
		wantErr                         string
		wantCause                       error
	}{
		{
			name:            "multiple targets and reordered rows",
			beforeAccount:   [][]driver.Value{{int64(10), int64(42)}, {int64(10), int64(43)}},
			beforeDetail:    [][]driver.Value{{int64(20), int64(7)}},
			afterAccount:    [][]driver.Value{{int64(11), int64(43)}, {int64(11), int64(42)}},
			afterDetail:     [][]driver.Value{{int64(21), int64(7)}},
			wantAccountRows: 2, wantDetailRows: 1,
		},
		{name: "empty before images"},
		{name: "empty before images with all columns", allColumns: true},
		{
			name: "empty before images with affected rows",
			result: func(*gomock.Controller) types.ExecResult {
				return types.NewResult(types.WithResult(driver.RowsAffected(1)))
			},
			wantErr: "affected 1 rows with empty before images",
		},
		{
			name: "empty before images with unknown affected rows",
			result: func(*gomock.Controller) types.ExecResult {
				return types.NewResult(types.WithResult(driver.RowsAffected(-1)))
			},
			wantErr: "affected -1 rows with empty before images",
		},
		{
			name: "empty before images with result error",
			result: func(*gomock.Controller) types.ExecResult {
				return types.NewResult(types.WithResult(sqlmock.NewErrorResult(resultErr)))
			},
			wantErr: "cannot determine affected rows", wantCause: resultErr,
		},
		{
			name:    "empty before images without result",
			result:  func(*gomock.Controller) types.ExecResult { return nil },
			wantErr: "result is unavailable",
		},
		{
			name: "empty before images with query result",
			result: func(ctrl *gomock.Controller) types.ExecResult {
				rows := mock.NewMockTestDriverRows(ctrl)
				rows.EXPECT().Close().Return(nil)
				return types.NewResult(types.WithRows(rows))
			},
			wantErr: "result is unavailable",
		},
		{
			name: "empty before images with query rows close error",
			result: func(ctrl *gomock.Controller) types.ExecResult {
				rows := mock.NewMockTestDriverRows(ctrl)
				rows.EXPECT().Close().Return(closeErr)
				return types.NewResult(types.WithRows(rows))
			},
			wantErr: "result is unavailable", wantCause: closeErr,
		},
		{
			name:            "zero primary key",
			beforeAccount:   [][]driver.Value{{int64(10), int64(0)}},
			afterAccount:    [][]driver.Value{{int64(11), int64(0)}},
			wantAccountRows: 1,
		},
		{
			name:            "unmatched binary primary key",
			beforeAccount:   [][]driver.Value{{int64(10), int64(42)}},
			beforeDetail:    [][]driver.Value{{nil, nil}},
			afterAccount:    [][]driver.Value{{int64(11), int64(42)}},
			wantAccountRows: 1, detailPKType: "VARBINARY",
		},
		{
			name:            "unmatched outer join target",
			beforeAccount:   [][]driver.Value{{int64(10), int64(42)}},
			beforeDetail:    [][]driver.Value{{nil, nil}},
			afterAccount:    [][]driver.Value{{int64(11), int64(42)}},
			wantAccountRows: 1,
		},
		{
			name:            "matched and unmatched outer join rows",
			beforeAccount:   [][]driver.Value{{int64(10), int64(42)}, {int64(10), int64(43)}},
			beforeDetail:    [][]driver.Value{{nil, nil}, {int64(20), int64(7)}},
			afterAccount:    [][]driver.Value{{int64(11), int64(42)}, {int64(11), int64(43)}},
			afterDetail:     [][]driver.Value{{int64(21), int64(7)}},
			wantAccountRows: 2, wantDetailRows: 1,
		},
		{
			name:            "all columns with unmatched outer join target",
			allColumns:      true,
			beforeAccount:   [][]driver.Value{{int64(10), int64(42)}},
			beforeDetail:    [][]driver.Value{{nil, nil}},
			afterAccount:    [][]driver.Value{{int64(11), int64(42)}},
			wantAccountRows: 1,
		},
		{
			name:            "all columns with matched and unmatched outer join rows",
			allColumns:      true,
			beforeAccount:   [][]driver.Value{{int64(10), int64(42)}, {int64(10), int64(43)}},
			beforeDetail:    [][]driver.Value{{nil, nil}, {int64(20), int64(7)}},
			afterAccount:    [][]driver.Value{{int64(11), int64(42)}, {int64(11), int64(43)}},
			afterDetail:     [][]driver.Value{{int64(21), int64(7)}},
			wantAccountRows: 2, wantDetailRows: 1,
		},
		{
			name:            "incomplete after image",
			beforeAccount:   [][]driver.Value{{int64(10), int64(42)}, {int64(10), int64(43)}},
			beforeDetail:    [][]driver.Value{{int64(20), int64(7)}},
			afterAccount:    [][]driver.Value{{int64(11), int64(42)}},
			afterDetail:     [][]driver.Value{{int64(21), int64(7)}},
			wantAccountRows: 2, wantDetailRows: 1, wantErr: "account",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: !tt.allColumns, DataValidation: false})
			ctrl := gomock.NewController(t)
			businessResult := types.NewResult(types.WithResult(driver.RowsAffected(tt.wantAccountRows + tt.wantDetailRows)))
			if tt.result != nil {
				businessResult = tt.result(ctrl)
			}
			conn := mock.NewMockTestDriverConn(ctrl)
			cache := mock.NewMockTableMetaCache(ctrl)
			datasource.RegisterTableCache(types.DBTypeMySQL, cache)
			const from = "account AS a LEFT JOIN detail AS d ON a.id=d.account_id"
			const query = "UPDATE account a LEFT JOIN detail d ON a.id=d.account_id SET a.balance=a.balance+1,d.balance=d.balance+1 WHERE a.balance=?"
			businessArgs := util.ValueToNamedValue([]driver.Value{int64(10)})
			updated := false
			for _, table := range []struct {
				name, alias   string
				before, after [][]driver.Value
				wantRows      int
			}{
				{"account", "a", tt.beforeAccount, tt.afterAccount, tt.wantAccountRows},
				{"detail", "d", tt.beforeDetail, tt.afterDetail, tt.wantDetailRows},
			} {
				meta := &types.TableMeta{
					TableName: table.name, ColumnNames: []string{"balance", "id"},
					Columns: map[string]types.ColumnMeta{
						"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
						"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
					},
					Indexs: map[string]types.IndexMeta{"PRIMARY": {
						IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
					}},
				}
				if table.name == "detail" && tt.detailPKType != "" {
					id := meta.Columns["id"]
					id.DatabaseTypeString = tt.detailPKType
					meta.Columns["id"] = id
				}
				cache.EXPECT().GetTableMeta(gomock.Any(), gomock.Any(), table.name).Return(meta, nil).AnyTimes()
				fields := table.alias + ".balance," + table.alias + ".id"
				beforeSQL := "SELECT SQL_NO_CACHE " + fields + " FROM " + from + " WHERE a.balance=? GROUP BY " + table.alias + ".id FOR UPDATE"
				if tt.allColumns {
					beforeSQL = "SELECT SQL_NO_CACHE `" + table.alias + "`.`balance`,`" + table.alias + "`.`id` FROM `account` AS `a` LEFT JOIN `detail` AS `d` ON `a`.`id`=`d`.`account_id` WHERE `a`.`balance`=? GROUP BY `" + table.alias + "`.`id` FOR UPDATE"
				}
				beforeRows := newDeleteRows([]string{"balance", "id"}, table.before...)
				conn.EXPECT().QueryContext(gomock.Any(), beforeSQL, businessArgs).DoAndReturn(
					func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
						assert.False(t, updated)
						return beforeRows, nil
					})
				if table.wantRows == 0 {
					continue // Any after-image query for this table must fail the mock expectation.
				}
				var pkArgs []driver.NamedValue
				var placeholders []string
				for _, row := range table.before {
					if row[1] != nil {
						pkArgs = append(pkArgs, driver.NamedValue{Ordinal: len(pkArgs) + 1, Value: row[1]})
						placeholders = append(placeholders, "(?)")
					}
				}
				if tt.allColumns {
					fields = "*"
				}
				afterSQL := "SELECT SQL_NO_CACHE " + fields + " FROM " + table.name + " AS " + table.alias + " WHERE (`id`) IN (" + strings.Join(placeholders, ",") + ")"
				afterRows := newDeleteRows([]string{"balance", "id"}, table.after...)
				conn.EXPECT().QueryContext(gomock.Any(), afterSQL, pkArgs).DoAndReturn(
					func(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
						assert.True(t, updated)
						return afterRows, nil
					})
			}
			parsed, err := parser.DoParser(query)
			require.NoError(t, err)
			txCtx := types.NewTxCtx()
			u := NewUpdateJoinExecutor(parsed, &types.ExecContext{
				Query: query, NamedValues: businessArgs, Conn: conn, TxCtx: txCtx,
				DBType: types.DBTypeMySQL, DbVersion: "8.0.36",
			}, nil)
			result, err := u.ExecContext(context.Background(), func(_ context.Context, sql string, args []driver.NamedValue) (types.ExecResult, error) {
				assert.Equal(t, query, sql)
				assert.Equal(t, businessArgs, args)
				updated = true
				return businessResult, nil
			})
			require.True(t, updated, "the business UPDATE must execute even when no rows match")
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				if tt.wantCause != nil {
					assert.ErrorIs(t, err, tt.wantCause)
				}
				assert.Nil(t, result)
				assert.Empty(t, txCtx.RoundImages.BeofreImages())
				assert.Empty(t, txCtx.RoundImages.AfterImages())
				if tt.wantAccountRows == 0 && tt.wantDetailRows == 0 {
					assert.Empty(t, txCtx.LockKeys)
				}
				return
			}
			require.NoError(t, err)
			assert.Same(t, businessResult, result)
			beforeImages, afterImages := txCtx.RoundImages.BeofreImages(), txCtx.RoundImages.AfterImages()
			wantImages := 0
			for _, count := range []int{tt.wantAccountRows, tt.wantDetailRows} {
				if count > 0 {
					wantImages++
				}
			}
			require.Len(t, beforeImages, wantImages)
			require.Len(t, afterImages, wantImages)
			assert.Len(t, txCtx.LockKeys, wantImages)
			if wantImages == 0 {
				assert.True(t, txCtx.RoundImages.IsEmpty())
				return
			}

			db, rollbackMock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			rollbackConn, err := db.Conn(context.Background())
			require.NoError(t, err)
			defer rollbackConn.Close()
			for i, before := range beforeImages {
				assert.Equal(t, before.TableName, afterImages[i].TableName)
				wantRows := tt.wantAccountRows
				originalRows := tt.beforeAccount
				if before.TableName == "detail" {
					wantRows = tt.wantDetailRows
					originalRows = tt.beforeDetail
				}
				assert.Len(t, before.Rows, wantRows)
				assert.Len(t, afterImages[i].Rows, wantRows)

				// Exercise the produced images through the real undo executor with validation disabled.
				before.TableMeta, err = cache.GetTableMeta(context.Background(), "", before.TableName)
				require.NoError(t, err)
				prepared := rollbackMock.ExpectPrepare("UPDATE " + before.TableName + " SET")
				for _, row := range originalRows {
					if row[1] != nil {
						prepared.ExpectExec().WithArgs(row[0], row[1]).WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
				rollback := undoexecutor.NewMySQLUndoExecutorHolder().GetUpdateExecutor(undo.SQLUndoLog{
					SQLType: before.SQLType, TableName: before.TableName,
					BeforeImage: before, AfterImage: afterImages[i],
				})
				require.NoError(t, rollback.ExecuteOn(context.Background(), types.DBTypeMySQL, rollbackConn))
			}
			require.NoError(t, rollbackMock.ExpectationsWereMet())
		})
	}
}

func TestBuildSelectSQLByUpdateJoin(t *testing.T) {
	MetaDataMap := map[string]*types.TableMeta{
		"table1": {
			TableName: "table1",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"name": {
					ColumnDef:  nil,
					ColumnName: "name",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "name", "age"},
		},
		"table2": {
			TableName: "table2",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"name": {
					ColumnDef:  nil,
					ColumnName: "name",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
				"kk": {
					ColumnDef:  nil,
					ColumnName: "kk",
				},
				"addr": {
					ColumnDef:  nil,
					ColumnName: "addr",
				},
			},
			ColumnNames: []string{"id", "name", "age", "kk", "addr"},
		},
		"table3": {
			TableName: "table3",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "age"},
		},
		"table4": {
			TableName: "table4",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "age"},
		},
	}

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     map[string]string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id and t1.age=? set t1.name = 'WILL',t2.name = ?",
			sourceQueryArgs: []driver.Value{18, "Jack"},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id AND t1.age=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id AND t1.age=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{18},
		},
		{
			sourceQuery:     "update table1 AS t1 inner join table2 AS t2 on t1.id = t2.id set t1.name = 'WILL',t2.name = 'WILL' where t1.id=?",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			sourceQuery:     "update table1 AS t1 right join table2 AS t2 on t1.id = t2.id set t1.name = 'WILL',t2.name = 'WILL' where t1.id=?",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 RIGHT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 RIGHT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			sourceQuery:     "update table1 t1 inner join table2 t2 on t1.id = t2.id set t1.name = ?, t1.age = ? where t1.id = ? and t1.name = ? and t2.age between ? and ?",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, "Jack", 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.age,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? AND t1.name=? AND t2.age BETWEEN ? AND ? GROUP BY t1.name,t1.age,t1.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, "Jack", 18, 28},
		},
		{
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id set t1.name = ?, t1.age = ? where t1.id=? and t2.id is null and t1.age IN (?,?)",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.age,t1.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? AND t2.id IS NULL AND t1.age IN (?,?) GROUP BY t1.name,t1.age,t1.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 18, 28},
		},
		{
			sourceQuery:     "update table1 t1 inner join table2 t2 on t1.id = t2.id set t1.name = ?, t2.age = ? where t2.kk between ? and ? and t2.addr in(?,?) and t2.age > ? order by t1.name desc limit ?",
			sourceQueryArgs: []driver.Value{"Jack", 18, 10, 20, "Beijing", "Guangzhou", 18, 2},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t2.kk BETWEEN ? AND ? AND t2.addr IN (?,?) AND t2.age>? GROUP BY t1.name,t1.id ORDER BY t1.name DESC LIMIT ? FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.age,t2.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t2.kk BETWEEN ? AND ? AND t2.addr IN (?,?) AND t2.age>? GROUP BY t2.age,t2.id ORDER BY t1.name DESC LIMIT ? FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{10, 20, "Beijing", "Guangzhou", 18, 2},
		},
		{
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id inner join table3 t3 on t3.id = t2.id right join table4 t4 on t4.id = t2.id set t1.name = ?,t2.name = ? where t1.id=? and t3.age=? and t4.age>30",
			sourceQueryArgs: []driver.Value{"Jack", "WILL", 1, 10},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM ((table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id) JOIN table3 AS t3 ON t3.id=t2.id) RIGHT JOIN table4 AS t4 ON t4.id=t2.id WHERE t1.id=? AND t3.age=? AND t4.age>30 GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM ((table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id) JOIN table3 AS t3 ON t3.id=t2.id) RIGHT JOIN table4 AS t4 ON t4.id=t2.id WHERE t1.id=? AND t3.age=? AND t4.age>30 GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery)
			assert.Nil(t, err)
			executor := NewUpdateJoinExecutor(c, &types.ExecContext{Values: tt.sourceQueryArgs, NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs)}, []exec.SQLHook{})
			tableNames := executor.(*updateJoinExecutor).parseTableName(c.UpdateStmt.TableRefs.TableRefs)
			for tbName, tableAliases := range tableNames {
				query, args, err := executor.(*updateJoinExecutor).buildBeforeImageSQL(context.Background(), MetaDataMap[tbName], tableAliases, util.ValueToNamedValue(tt.sourceQueryArgs))
				assert.Nil(t, err)
				if query == "" {
					continue
				}
				assert.Equal(t, tt.expectQuery[tbName], query)
				assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
			}
		})
	}
}
