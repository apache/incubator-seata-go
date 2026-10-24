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
	"fmt"
	"regexp"
	"strings"
	"testing"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		name, query, alias, fields, from, schema string
		args                                     []driver.Value
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
			name: "qualified target", schema: "sales", alias: "o", fields: "o.status,o.id", from: "sales.t_order AS o",
			query: "UPDATE sales.t_order o JOIN t_item i ON o.id=i.order_id SET o.status=1",
		},
		{
			name: "qualified right target", schema: "sales", alias: "o", fields: "o.status,o.id", from: "sales.t_order AS o",
			query: "UPDATE t_item i JOIN sales.t_order o ON o.id=i.order_id SET o.status=1",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.DoParser(tt.query)
			require.NoError(t, err)
			u := NewUpdateJoinExecutor(parsed, &types.ExecContext{
				DBType: types.DBTypeMySQL, DbVersion: "8.0.36", NamedValues: util.ValueToNamedValue(tt.args),
			}, nil).(*updateJoinExecutor)
			query, args, err := u.buildAfterImageSQL(context.Background(), before, meta, joinTableRef{ref: types.TableRef{Qualifier: tt.schema, TableName: meta.TableName}, alias: tt.alias})
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
		_, _, err = u.buildAfterImageSQL(context.Background(), before, &withoutPK, joinTableRef{ref: types.TableRef{TableName: meta.TableName}, alias: "o"})
		require.ErrorContains(t, err, "primary key metadata is empty")

		missingPK := types.RecordImage{Rows: []types.RowImage{
			{Columns: []types.ColumnImage{{ColumnName: "status", Value: int64(0)}}},
		}}
		_, _, err = u.buildAfterImageSQL(context.Background(), missingPK, meta, joinTableRef{ref: types.TableRef{TableName: meta.TableName}, alias: "o"})
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
		query, args, err := u.buildAfterImageSQL(context.Background(), before, meta, joinTableRef{ref: types.TableRef{TableName: meta.TableName}, alias: "o"})
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
			query, args, err := u.buildAfterImageSQL(context.Background(), before, meta, joinTableRef{ref: types.TableRef{TableName: meta.TableName}, alias: "o"})
			require.NoError(t, err)
			fields := "o.*"
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
			query, args, err := u.buildBeforeImageSQL(context.Background(), meta, joinTableRef{ref: types.TableRef{TableName: meta.TableName}, alias: tt.alias}, nil)
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

func TestUpdateJoinSelectFieldsUnqualifiedColumns(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	for _, onlyCareUpdateColumns := range []bool{true, false} {
		undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: onlyCareUpdateColumns})
		for _, tt := range []struct {
			name, assignment, metaColumn string
			wantColumn                   string
		}{
			{name: "column exists", assignment: "balance=1", metaColumn: "balance", wantColumn: "balance"},
			{name: "case insensitive", assignment: "`BaLaNcE`=1", metaColumn: "BALANCE", wantColumn: "BaLaNcE"},
			{name: "escaped metadata", assignment: "`balance`=1", metaColumn: "`BALANCE`", wantColumn: "balance"},
			{name: "quote in identifier", assignment: "`display``name`=1", metaColumn: "display`name", wantColumn: "display`name"},
			{name: "only matching assignment", assignment: "balance=1,enabled=0", metaColumn: "balance", wantColumn: "balance"},
			{name: "column belongs to another table", assignment: "balance=1", metaColumn: "enabled"},
			{name: "qualified column belongs to another table", assignment: "d.balance=1", metaColumn: "balance"},
		} {
			t.Run(tt.name, func(t *testing.T) {
				parsed, err := parser.DoParser("UPDATE account a JOIN detail d ON a.id=d.account_id SET " + tt.assignment)
				require.NoError(t, err)
				meta := &types.TableMeta{
					TableName: "account", ColumnNames: []string{"id", tt.metaColumn},
					Columns: map[string]types.ColumnMeta{"id": {ColumnName: "id"}, tt.metaColumn: {ColumnName: tt.metaColumn}},
					Indexs: map[string]types.IndexMeta{"PRIMARY": {
						IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
					}},
				}
				u := NewUpdateJoinExecutor(parsed, &types.ExecContext{DBType: types.DBTypeMySQL, DbVersion: "8.0.36"}, nil).(*updateJoinExecutor)
				fields, err := u.buildSelectFields(context.Background(), meta, types.TableRef{TableName: meta.TableName}, "a", parsed.UpdateStmt.List)
				require.NoError(t, err)
				if tt.wantColumn == "" {
					assert.Empty(t, fields, "an unused table must not become a target just because it has a primary key")
					query, _, err := u.buildBeforeImageSQL(context.Background(), meta, joinTableRef{ref: types.TableRef{TableName: meta.TableName}, alias: "a"}, nil)
					require.NoError(t, err)
					assert.Empty(t, query)
					return
				}
				if onlyCareUpdateColumns {
					require.Len(t, fields, 2)
					assert.Equal(t, tt.wantColumn, fields[0].Expr.(*ast.ColumnNameExpr).Name.Name.O)
					assert.Equal(t, "id", fields[1].Expr.(*ast.ColumnNameExpr).Name.Name.O)
				} else {
					require.Len(t, fields, 1)
					assert.Equal(t, "*", fields[0].Expr.(*ast.ColumnNameExpr).Name.Name.O)
				}
			})
		}
	}
}

func TestUpdateJoinRejectsSelfJoinsBeforeExecution(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() {
		undo.UndoConfig = originalUndoConfig
	})
	for _, onlyCareUpdateColumns := range []bool{true, false} {
		undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: onlyCareUpdateColumns})
		for _, tt := range []struct {
			name, query, dbName, table, aliases string
		}{
			{name: "two aliases", query: "UPDATE t a JOIN t b ON a.id=b.id SET a.balance=1", table: "t", aliases: "a and b"},
			{name: "quoted table", query: "UPDATE `t` a JOIN t b ON a.id=b.id SET a.balance=1", table: "t", aliases: "a and b"},
			{name: "unaliased table", query: "UPDATE t JOIN t b ON t.id=b.id SET t.balance=1", table: "t", aliases: "t and b"},
			{name: "nested left join", query: "UPDATE t a JOIN detail d ON a.id=d.id JOIN t b ON a.id=b.id SET d.balance=1", table: "t", aliases: "a and b"},
			{name: "nested right join", query: "UPDATE detail d JOIN (t a JOIN t b ON a.id=b.id) ON d.id=a.id SET d.balance=1", table: "t", aliases: "a and b"},
			{name: "same schema", query: "UPDATE db1.t a JOIN db1.t b ON a.id=b.id SET a.balance=1", table: "db1.t", aliases: "a and b"},
			{name: "default schema", query: "UPDATE t a JOIN db1.t b ON a.id=b.id SET a.balance=1", dbName: "db1", table: "db1.t", aliases: "a and b"},
		} {
			t.Run(tt.name, func(t *testing.T) {
				parsed, err := parser.DoParser(tt.query)
				require.NoError(t, err)
				ctrl := gomock.NewController(t)
				reader := mock.NewMockTableMetaCache(ctrl)
				for _, table := range parseTableName(parsed, parsed.UpdateStmt.TableRefs.TableRefs) {
					dbName := table.ref.Qualifier
					if dbName == "" {
						dbName = tt.dbName
					}
					key := types.TableMetaKey{DBName: dbName, TableName: table.ref.TableName}
					reader.EXPECT().ResolveTableMetaKey(gomock.Any(), gomock.Any(), table.ref).Return(key, nil).MaxTimes(1)
				}
				// No metadata or image-query expectations: rejection must precede beforeImage.
				txCtx := types.NewTxCtx()
				u := NewUpdateJoinExecutor(parsed, &types.ExecContext{
					Query: tt.query, DBName: tt.dbName, DBType: types.DBTypeMySQL,
					DbVersion: "8.0.36", Conn: mock.NewMockTestDriverConn(ctrl), TxCtx: txCtx, TableMetaReader: reader,
				}, nil)
				called := false
				result, err := u.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
					called = true
					return types.NewResult(types.WithResult(driver.RowsAffected(1))), nil
				})
				require.EqualError(t, err, "UPDATE JOIN self-joins are not supported: table "+tt.table+" appears as aliases "+tt.aliases)
				assert.False(t, called, "the business callback must not execute for self-joins")
				assert.Nil(t, result)
				assert.True(t, txCtx.RoundImages.IsEmpty())
				assert.Empty(t, txCtx.LockKeys)
			})
		}
	}
}

func TestUpdateJoinSelfJoinDetectionDistinguishesSchemas(t *testing.T) {
	for _, query := range []string{
		"UPDATE db1.t a JOIN db2.t b ON a.id=b.id SET a.balance=1",
		"UPDATE t a JOIN db2.t b ON a.id=b.id SET a.balance=1",
	} {
		parsed, err := parser.DoParser(query)
		require.NoError(t, err)
		reader := mock.NewMockTableMetaCache(gomock.NewController(t))
		for _, table := range parseTableName(parsed, parsed.UpdateStmt.TableRefs.TableRefs) {
			dbName := table.ref.Qualifier
			if dbName == "" {
				dbName = "db1"
			}
			reader.EXPECT().ResolveTableMetaKey(gomock.Any(), gomock.Any(), table.ref).
				Return(types.TableMetaKey{DBName: dbName, TableName: table.ref.TableName}, nil)
		}
		metaErr := errors.New("metadata read reached")
		reader.EXPECT().GetTableMeta(gomock.Any(), types.TableMetaKey{DBName: "db1", TableName: "t"}).Return(nil, metaErr)
		u := NewUpdateJoinExecutor(parsed, &types.ExecContext{DBName: "db1", TableMetaReader: reader}, nil)
		_, err = u.ExecContext(context.Background(), func(context.Context, string, []driver.NamedValue) (types.ExecResult, error) {
			t.Fatal("the business callback must not execute after a metadata read failure")
			return nil, nil
		})
		require.ErrorIs(t, err, metaErr, "different schemas must pass self-join detection and reach the metadata read")
	}
}

func TestUpdateJoinAfterImages(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() {
		undo.UndoConfig = originalUndoConfig
	})
	resultErr := errors.New("affected rows unavailable")
	closeErr := errors.New("query rows close failed")
	for _, tt := range []struct {
		name, detailPKType              string
		assignment                      string
		beforeAccount, beforeDetail     [][]driver.Value
		afterAccount, afterDetail       [][]driver.Value
		wantAccountRows, wantDetailRows int
		allColumns                      bool
		result                          func(*gomock.Controller) types.ExecResult
		wantErr                         string
		wantCause                       error
	}{
		{
			name: "unqualified column belongs to one table", assignment: "balance=balance+1",
			beforeAccount: [][]driver.Value{{int64(10), int64(42)}},
			afterAccount:  [][]driver.Value{{int64(11), int64(42)}}, wantAccountRows: 1,
		},
		{
			name: "unqualified column with all columns", assignment: "balance=balance+1", allColumns: true,
			beforeAccount: [][]driver.Value{{int64(10), int64(42)}},
			afterAccount:  [][]driver.Value{{int64(11), int64(42)}}, wantAccountRows: 1,
		},
		{
			name: "unused join table with all columns", assignment: "a.balance=a.balance+1", allColumns: true,
			beforeAccount: [][]driver.Value{{int64(10), int64(42)}},
			afterAccount:  [][]driver.Value{{int64(11), int64(42)}}, wantAccountRows: 1,
		},
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
			reader := mock.NewMockTableMetaCache(ctrl)
			metas := make(map[string]*types.TableMeta)
			const from = "account AS a LEFT JOIN detail AS d ON a.id=d.account_id"
			assignment := tt.assignment
			if assignment == "" {
				assignment = "a.balance=a.balance+1,d.balance=d.balance+1"
			}
			query := "UPDATE account a LEFT JOIN detail d ON a.id=d.account_id SET " + assignment + " WHERE a.balance=?"
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
				key := types.TableMetaKey{TableName: table.name}
				reader.EXPECT().ResolveTableMetaKey(gomock.Any(), gomock.Any(), types.TableRef{TableName: table.name}).Return(key, nil)
				reader.EXPECT().GetTableMeta(gomock.Any(), key).Return(meta, nil).AnyTimes()
				metas[table.name] = meta
				if tt.assignment != "" && table.name == "detail" {
					delete(meta.Columns, "balance")
					meta.ColumnNames = []string{"id"}
					continue // Unused JOIN tables must not trigger either image query.
				}
				fields := table.alias + ".balance," + table.alias + ".id"
				if tt.assignment == "balance=balance+1" {
					fields = "balance," + table.alias + ".id"
				}
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
					fields = table.alias + ".*"
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
				DBType: types.DBTypeMySQL, DbVersion: "8.0.36", TableMetaReader: reader,
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
				before.TableMeta = metas[before.TableName]
				prepared := rollbackMock.ExpectPrepare("UPDATE `" + before.TableName + "` SET")
				for _, row := range originalRows {
					if row[1] != nil {
						prepared.ExpectExec().WithArgs(row[0], row[1]).WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
				rollback := undoexecutor.NewMySQLUndoExecutorHolder().GetUpdateExecutor(undo.SQLUndoLog{
					SQLType: before.SQLType, TableName: before.TableName, TableMetaKey: before.TableMetaKey,
					BeforeImage: before, AfterImage: afterImages[i],
				})
				require.NoError(t, rollback.ExecuteOn(context.Background(), types.DBTypeMySQL, rollbackConn))
			}
			require.NoError(t, rollbackMock.ExpectationsWereMet())
		})
	}
}

func TestBuildSelectSQLByUpdateJoin(t *testing.T) {
	tableMetas := map[string]*types.TableMeta{
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
			tableNames := parseTableName(c, c.UpdateStmt.TableRefs.TableRefs)
			for _, table := range tableNames {
				tbName := table.ref.TableName
				query, args, err := executor.(*updateJoinExecutor).buildBeforeImageSQL(context.Background(), tableMetas[tbName], table, util.ValueToNamedValue(tt.sourceQueryArgs))
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

func TestUpdateJoinKeepsSameNamedTablesSeparate(t *testing.T) {
	parsed, err := parser.DoParser("UPDATE one.users a JOIN two.users b ON a.id=b.id SET a.id=1")
	assert.NoError(t, err)
	tables := parseTableName(parsed, parsed.UpdateStmt.TableRefs.TableRefs)
	assert.Equal(t, []joinTableRef{
		{ref: types.TableRef{Qualifier: "one", TableName: "users"}, alias: "a"},
		{ref: types.TableRef{Qualifier: "two", TableName: "users"}, alias: "b"},
	}, tables)
}

func TestUpdateJoinSameNamedTablesWithoutAliasesUsesQualifiedColumns(t *testing.T) {
	originalUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = originalUndoConfig })
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})
	parsed, err := parser.DoParser("UPDATE one.users JOIN two.users ON one.users.id=two.users.id SET one.users.name='A',two.users.name='B'")
	if !assert.NoError(t, err) {
		return
	}
	executor := NewUpdateJoinExecutor(parsed, &types.ExecContext{}, nil).(*updateJoinExecutor)
	tables := parseTableName(parsed, parsed.UpdateStmt.TableRefs.TableRefs)
	assert.Len(t, tables, 2)
	meta := &types.TableMeta{TableName: "users", ColumnNames: []string{"id", "name"}, Indexs: map[string]types.IndexMeta{
		"id": {IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}}},
	}}
	for _, table := range tables {
		query, _, err := executor.buildBeforeImageSQL(context.Background(), meta, table, nil)
		assert.NoError(t, err)
		qualified := table.ref.Qualifier + ".users"
		assert.Contains(t, query, "SELECT SQL_NO_CACHE "+qualified+".name,"+qualified+".id FROM")
		assert.Contains(t, query, " GROUP BY "+qualified+".name,"+qualified+".id FOR UPDATE")
		assert.NotContains(t, query, "SELECT SQL_NO_CACHE one.users.name,two.users.name")

		before := types.RecordImage{Rows: []types.RowImage{{Columns: []types.ColumnImage{{ColumnName: "id", Value: int64(1)}}}}}
		afterQuery, _, err := executor.buildAfterImageSQL(context.Background(), before, meta, table)
		assert.NoError(t, err)
		assert.Contains(t, afterQuery, qualified+".name,"+qualified+".id FROM")
		assert.Contains(t, afterQuery, " FROM "+qualified+" WHERE (`id`) IN ((?))")
		assert.NotContains(t, afterQuery, " JOIN ")
		assert.NotContains(t, afterQuery, " GROUP BY ")
	}

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})
	for _, table := range tables {
		query, _, err := executor.buildBeforeImageSQL(context.Background(), meta, table, nil)
		assert.NoError(t, err)
		qualified := "`" + table.ref.Qualifier + "`.`users`"
		assert.Contains(t, query, "SELECT SQL_NO_CACHE "+qualified+".`id`,"+qualified+".`name` FROM")
	}
}

func TestUpdateJoinCaseSensitiveIdentityKeepsImagesAndUndoSeparate(t *testing.T) {
	previousUndoConfig := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previousUndoConfig })

	cases := []struct {
		name      string
		tables    string
		selectors [2]string
		refs      [2]types.TableRef
	}{
		{
			name: "aliases", tables: "t1 a JOIN t2 A ON a.id=A.id",
			selectors: [2]string{"a", "A"},
			refs:      [2]types.TableRef{{TableName: "t1"}, {TableName: "t2"}},
		},
		{
			name: "tables", tables: "users JOIN Users ON users.id=Users.id",
			selectors: [2]string{"users", "Users"},
			refs:      [2]types.TableRef{{TableName: "users"}, {TableName: "Users"}},
		},
		{
			name: "databases", tables: "one.users JOIN ONE.users ON one.users.id=ONE.users.id",
			selectors: [2]string{"one.users", "ONE.users"},
			refs:      [2]types.TableRef{{Qualifier: "one", TableName: "users"}, {Qualifier: "ONE", TableName: "users"}},
		},
	}
	for _, tc := range cases {
		for _, validate := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/dataValidation=%t", tc.name, validate), func(t *testing.T) {
				undo.UndoConfig = undo.Config{OnlyCareUpdateColumns: true, DataValidation: validate}
				ctx := context.Background()
				db, sqlMock, err := sqlmock.New()
				require.NoError(t, err)
				defer db.Close()
				conn, err := db.Conn(ctx)
				require.NoError(t, err)
				defer conn.Close()

				reader := mock.NewMockTableMetaCache(gomock.NewController(t))
				var keys [2]types.TableMetaKey
				var metas [2]*types.TableMeta
				oldBalances, newBalances := [2]int64{10, 20}, [2]int64{100, 200}
				for i, ref := range tc.refs {
					dbName := ref.Qualifier
					if dbName == "" {
						dbName = "app"
					}
					keys[i] = types.TableMetaKey{DBName: dbName, TableName: ref.TableName}
					metas[i] = &types.TableMeta{
						TableName: ref.TableName, ColumnNames: []string{"id", "balance"},
						Columns: map[string]types.ColumnMeta{
							"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
							"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
						},
						Indexs: map[string]types.IndexMeta{"PRIMARY": {
							IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
						}},
					}
					reader.EXPECT().ResolveTableMetaKey(gomock.Any(), gomock.Any(), ref).Return(keys[i], nil)
					reader.EXPECT().GetTableMeta(gomock.Any(), keys[i]).Return(metas[i], nil).Times(2)
					prefix := "SELECT SQL_NO_CACHE " + tc.selectors[i] + ".balance," + tc.selectors[i] + ".id FROM "
					sqlMock.ExpectQuery("^" + regexp.QuoteMeta(prefix) + ".* FOR UPDATE$").WithArgs(int64(1)).
						WillReturnRows(sqlmock.NewRows([]string{"balance", "id"}).AddRow(oldBalances[i], int64(1)))
				}
				query := fmt.Sprintf("UPDATE %s SET %s.balance=100,%s.balance=200 WHERE %s.id=?",
					tc.tables, tc.selectors[0], tc.selectors[1], tc.selectors[0])
				sqlMock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(int64(1)).WillReturnResult(sqlmock.NewResult(0, 2))
				for i, selector := range tc.selectors {
					from := tc.refs[i].TableName
					if tc.refs[i].Qualifier != "" {
						from = tc.refs[i].Qualifier + "." + from
					}
					if tc.name == "aliases" {
						from += " AS " + selector
					}
					afterSQL := "SELECT SQL_NO_CACHE " + selector + ".balance," + selector + ".id FROM " + from + " WHERE (`id`) IN ((?))"
					sqlMock.ExpectQuery("^" + regexp.QuoteMeta(afterSQL) + "$").
						WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"balance", "id"}).AddRow(newBalances[i], int64(1)))
				}

				parsed, err := parser.DoParser(query)
				require.NoError(t, err)
				txCtx := types.NewTxCtx()
				err = conn.Raw(func(raw interface{}) error {
					driverConn := raw.(driver.Conn)
					execCtx := &types.ExecContext{
						Query: query, NamedValues: []driver.NamedValue{{Ordinal: 1, Value: int64(1)}},
						DBType: types.DBTypeMySQL, DbVersion: "8.0.29", Conn: driverConn,
						TxCtx: txCtx, TableMetaReader: reader,
					}
					_, err := NewUpdateJoinExecutor(parsed, execCtx, nil).ExecContext(ctx,
						func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
							result, err := driverConn.(driver.ExecerContext).ExecContext(ctx, query, args)
							return types.NewResult(types.WithResult(result)), err
						})
					return err
				})
				require.NoError(t, err)
				before, after := txCtx.RoundImages.BeofreImages(), txCtx.RoundImages.AfterImages()
				require.Len(t, before, 2)
				require.Len(t, after, 2)
				for i := len(before) - 1; i >= 0; i-- {
					require.Equal(t, &keys[i], before[i].TableMetaKey)
					require.Equal(t, &keys[i], after[i].TableMetaKey)
					require.Len(t, before[i].Rows, 1)
					require.Len(t, after[i].Rows, 1)
					require.Len(t, before[i].Rows[0].Columns, 2)
					require.Len(t, after[i].Rows[0].Columns, 2)
					require.Equal(t, oldBalances[i], before[i].Rows[0].Columns[0].GetActualValue())
					require.Equal(t, newBalances[i], after[i].Rows[0].Columns[0].GetActualValue())

					log := undo.SQLUndoLog{SQLType: types.SQLTypeUpdate, TableName: before[i].TableName,
						TableMetaKey: before[i].TableMetaKey, BeforeImage: before[i], AfterImage: after[i]}
					log.SetTableMeta(metas[i])
					qualified := "`" + keys[i].DBName + "`.`" + keys[i].TableName + "`"
					if validate {
						sqlMock.ExpectQuery("^" + regexp.QuoteMeta("SELECT balance, id FROM "+qualified+" WHERE ") + ".* FOR UPDATE$").
							WithArgs(int64(1)).WillReturnRows(sqlmock.NewRowsWithColumnDefinition(
							sqlmock.NewColumn("balance").OfType("BIGINT", int64(0)),
							sqlmock.NewColumn("id").OfType("BIGINT", int64(0)),
						).AddRow(newBalances[i], int64(1)))
					}
					sqlMock.ExpectPrepare("^"+regexp.QuoteMeta("UPDATE "+qualified+" SET balance = ? WHERE id = ?")+"$").
						WillBeClosed().ExpectExec().WithArgs(oldBalances[i], int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
					rollback := undoexecutor.NewMySQLUndoExecutorHolder().GetUpdateExecutor(log)
					require.NoError(t, rollback.ExecuteOn(ctx, types.DBTypeMySQL, conn))
				}
				require.NoError(t, sqlMock.ExpectationsWereMet())
			})
		}
	}
}

func TestJoinExecutorWriteDatabaseBoundary(t *testing.T) {
	previous := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previous })
	for _, onlyCareUpdateColumns := range []bool{true, false} {
		undo.UndoConfig.OnlyCareUpdateColumns = onlyCareUpdateColumns
		for _, tc := range []struct {
			name       string
			query      string
			currentDB  string
			wantReject bool
		}{
			{"parenthesized join", "UPDATE (db_a.account a JOIN db_b.account b ON a.id=b.id) SET b.balance=1", "", true},
			{"joined write on right", "UPDATE db_a.account a JOIN db_b.account b ON a.id=b.id SET b.balance=1", "", true},
			{"case distinct write alias", "UPDATE db_a.account a JOIN db_b.account A ON a.id=A.id SET A.balance=1", "", true},
			{"case overlapping write aliases", "UPDATE db_a.account a JOIN db_b.other A ON TRUE SET a.only_b=1", "", true},
			{"case overlapping read alias", "UPDATE db_a.account a JOIN db_b.account A ON a.id=A.id SET a.balance=A.balance", "", true},
			{"joined writes on both sides", "UPDATE db_a.account a JOIN db_b.account b ON a.id=b.id SET a.balance=1,b.balance=2", "", true},
			{"qualified joined write", "UPDATE db_a.account JOIN db_b.account ON db_a.account.id=db_b.account.id SET db_b.account.balance=1", "", true},
			{"unqualified joined write", "UPDATE db_a.account a JOIN db_b.other b ON a.id=b.id SET balance=1", "", true},
			{"join reads another database", "UPDATE db_a.account a JOIN db_b.account b ON a.id=b.id SET a.balance=b.balance", "", false},
			{"join reads first foreign table", "UPDATE db_b.account b JOIN db_a.account a ON a.id=b.id SET a.balance=b.balance", "", false},
			{"derived join read", "UPDATE db_a.account a JOIN (SELECT id FROM db_b.account) b ON a.id=b.id SET a.balance=1", "", false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				checkWriteDatabaseBoundary(t, tc.query, tc.currentDB, tc.wantReject)
			})
		}
		t.Run("case overlapping database qualifier", func(t *testing.T) {
			checkWriteDatabaseBoundary(t,
				"UPDATE db_a.account JOIN DB_A.account ON db_a.account.id=DB_A.account.id SET db_a.account.balance=1", "", true, 0)
		})
	}
}
