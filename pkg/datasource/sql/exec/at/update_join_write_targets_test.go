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
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	undoexecutor "seata.apache.org/seata-go/v2/pkg/datasource/sql/undo/executor"
)

func TestUpdateJoinRecordsOnlyWriteTargets(t *testing.T) {
	previous := undo.UndoConfig
	t.Cleanup(func() { undo.UndoConfig = previous })
	type source struct {
		database string
		table    string
		alias    string
		written  bool
	}
	for _, tc := range []struct {
		name    string
		query   string
		sources []source
	}{
		{
			name: "foreign read on right",
			query: "UPDATE db_a.account a JOIN db_b.source b ON a.id=b.id " +
				"SET a.balance=b.balance WHERE a.id=?",
			sources: []source{{"db_a", "account", "a", true}, {"db_b", "source", "b", false}},
		},
		{
			name: "foreign read on left",
			query: "UPDATE db_b.source b JOIN db_a.account a ON a.id=b.id " +
				"SET a.balance=b.balance WHERE a.id=?",
			sources: []source{{"db_b", "source", "b", false}, {"db_a", "account", "a", true}},
		},
		{
			name: "multiple local write targets",
			query: "UPDATE db_a.account a JOIN db_b.source b ON a.id=b.id " +
				"JOIN db_a.audit c ON a.id=c.id SET a.balance=b.balance,c.balance=b.balance WHERE a.id=?",
			sources: []source{{"db_a", "account", "a", true}, {"db_b", "source", "b", false}, {"db_a", "audit", "c", true}},
		},
	} {
		for _, onlyCareUpdateColumns := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/onlyCareUpdateColumns=%t", tc.name, onlyCareUpdateColumns), func(t *testing.T) {
				undo.UndoConfig = undo.Config{OnlyCareUpdateColumns: onlyCareUpdateColumns, DataValidation: false}
				ctx := context.Background()
				db, sqlMock, err := sqlmock.New()
				require.NoError(t, err)
				defer db.Close()
				conn, err := db.Conn(ctx)
				require.NoError(t, err)
				defer conn.Close()

				reader := mock.NewMockTableMetaCache(gomock.NewController(t))
				var targets []source
				var metas []*types.TableMeta
				wantLocks := map[string]struct{}{}
				for _, table := range tc.sources {
					key := types.TableMetaKey{DBName: table.database, TableName: table.table}
					ref := types.TableRef{Qualifier: table.database, TableName: table.table}
					reader.EXPECT().ResolveTableMetaKey(gomock.Any(), gomock.Any(), ref).Return(key, nil)
					if !table.written {
						// Any metadata or image read for this source is an unexpected call.
						continue
					}
					meta := &types.TableMeta{
						TableName: table.table, ColumnNames: []string{"id", "balance"},
						Columns: map[string]types.ColumnMeta{
							"id":      {ColumnName: "id", DatabaseTypeString: "BIGINT"},
							"balance": {ColumnName: "balance", DatabaseTypeString: "BIGINT"},
						},
						Indexs: map[string]types.IndexMeta{"PRIMARY": {
							IType: types.IndexTypePrimaryKey, Columns: []types.ColumnMeta{{ColumnName: "id"}},
						}},
					}
					reader.EXPECT().GetTableMeta(gomock.Any(), key).Return(meta, nil).Times(2)
					targets = append(targets, table)
					metas = append(metas, meta)
					wantLocks[strings.ToUpper(table.table)+":1"] = struct{}{}
				}

				imageQuery := func(table source, before bool) string {
					fields := table.alias + ".*"
					if onlyCareUpdateColumns {
						fields = table.alias + ".balance," + table.alias + ".id"
					}
					query := "^" + regexp.QuoteMeta("SELECT SQL_NO_CACHE "+fields+" FROM ") +
						".* GROUP BY " + regexp.QuoteMeta(table.alias+".id")
					if before {
						query += " FOR UPDATE"
					}
					return query + "$"
				}
				imageRows := func(balance int64) *sqlmock.Rows {
					if onlyCareUpdateColumns {
						return sqlmock.NewRows([]string{"balance", "id"}).AddRow(balance, int64(1))
					}
					return sqlmock.NewRows([]string{"id", "balance"}).AddRow(int64(1), balance)
				}
				for _, table := range targets {
					sqlMock.ExpectQuery(imageQuery(table, true)).WithArgs(int64(1)).WillReturnRows(imageRows(10))
				}
				sqlMock.ExpectExec(regexp.QuoteMeta(tc.query)).WithArgs(int64(1)).WillReturnResult(sqlmock.NewResult(0, int64(len(targets))))
				for _, table := range targets {
					sqlMock.ExpectQuery(imageQuery(table, false)).WithArgs(int64(1)).WillReturnRows(imageRows(30))
				}

				txCtx := types.NewTxCtx()
				txCtx.TransactionMode = types.ATMode
				err = conn.Raw(func(raw any) error {
					driverConn := raw.(driver.Conn)
					execCtx := &types.ExecContext{
						Query: tc.query, DBName: "db_a", DBType: types.DBTypeMySQL, DbVersion: "8.0.29",
						Conn: driverConn, TableMetaReader: reader, TxCtx: txCtx,
						NamedValues: []driver.NamedValue{{Ordinal: 1, Value: int64(1)}},
					}
					_, err := (&ATExecutor{}).ExecWithNamedValue(ctx, execCtx,
						func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
							result, err := driverConn.(driver.ExecerContext).ExecContext(ctx, query, args)
							return types.NewResult(types.WithResult(result)), err
						})
					return err
				})
				require.NoError(t, err)
				require.Equal(t, wantLocks, txCtx.LockKeys)
				before, after := txCtx.RoundImages.BeofreImages(), txCtx.RoundImages.AfterImages()
				require.Len(t, before, len(targets))
				require.Len(t, after, len(targets))
				for index := len(targets) - 1; index >= 0; index-- {
					key := &types.TableMetaKey{DBName: "db_a", TableName: targets[index].table}
					require.Equal(t, key, before[index].TableMetaKey)
					require.Equal(t, key, after[index].TableMetaKey)
					log := undo.SQLUndoLog{SQLType: types.SQLTypeUpdate, TableName: key.TableName,
						TableMetaKey: key, BeforeImage: before[index], AfterImage: after[index]}
					log.SetTableMeta(metas[index])
					query := "UPDATE `db_a`.`" + key.TableName + "` SET balance = ? WHERE id = ?"
					sqlMock.ExpectPrepare("^"+regexp.QuoteMeta(query)+"$").WillBeClosed().
						ExpectExec().WithArgs(int64(10), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
					rollback := undoexecutor.NewMySQLUndoExecutorHolder().GetUpdateExecutor(log)
					require.NoError(t, rollback.ExecuteOn(ctx, types.DBTypeMySQL, conn))
				}
				require.NoError(t, sqlMock.ExpectationsWereMet())
			})
		}
	}
}
