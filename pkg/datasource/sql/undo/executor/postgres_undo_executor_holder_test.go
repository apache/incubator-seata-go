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
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	serr "seata.apache.org/seata-go/v2/pkg/util/errors"
)

func TestPostgresUpdateUndoPreservesColumnIdentity(t *testing.T) {
	previous := undo.UndoConfig
	undo.UndoConfig.DataValidation = true
	t.Cleanup(func() { undo.UndoConfig = previous })
	for _, tc := range []struct {
		name        string
		columns     []string
		metaColumns []string
		primaryKey  []string
		before      []driver.Value
		after       []driver.Value
		dirtyIndex  int
		checkSQL    string
		checkArgs   []driver.Value
		undoSQL     string
		undoArgs    []driver.Value
	}{
		{
			name:    "ordinary column differs from primary key by case",
			columns: []string{"ID", "id"}, primaryKey: []string{"id"},
			before: []driver.Value{int64(10), int64(1)}, after: []driver.Value{int64(20), int64(1)},
			checkSQL:  `SELECT "ID", "id" FROM "public"."users" WHERE ("id") IN (($1)) FOR UPDATE`,
			checkArgs: []driver.Value{int64(1)},
			undoSQL:   `UPDATE "public"."users" SET "ID" = $1 WHERE "id" = $2`,
			undoArgs:  []driver.Value{int64(10), int64(1)},
		},
		{
			name:    "all columns include case distinct ordinary columns",
			columns: []string{"balance", "Balance", "id"}, primaryKey: []string{"id"},
			before: []driver.Value{int64(100), int64(10), int64(1)}, after: []driver.Value{int64(100), int64(20), int64(1)},
			checkSQL:  `SELECT "balance", "Balance", "id" FROM "public"."users" WHERE ("id") IN (($1)) FOR UPDATE`,
			checkArgs: []driver.Value{int64(1)},
			undoSQL:   `UPDATE "public"."users" SET "balance" = $1 , "Balance" = $2 WHERE "id" = $3`,
			undoArgs:  []driver.Value{int64(100), int64(10), int64(1)},
		},
		{
			name:    "case distinct composite primary keys keep metadata order",
			columns: []string{"ID", "balance", "id"}, primaryKey: []string{"id", "ID"},
			metaColumns: []string{"id", "ID", "balance"},
			before:      []driver.Value{int64(2), int64(10), int64(1)}, after: []driver.Value{int64(2), int64(20), int64(1)}, dirtyIndex: 1,
			checkSQL:  `SELECT "ID", "balance", "id" FROM "public"."users" WHERE ("id","ID") IN (($1,$2)) FOR UPDATE`,
			checkArgs: []driver.Value{int64(1), int64(2)},
			undoSQL:   `UPDATE "public"."users" SET "balance" = $1 WHERE "id" = $2 and "ID" = $3`,
			undoArgs:  []driver.Value{int64(10), int64(1), int64(2)},
		},
	} {
		for _, state := range []string{"after image", "already rolled back", "dirty data"} {
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
				require.NoError(t, err)
				defer db.Close()
				conn, err := db.Conn(context.Background())
				require.NoError(t, err)
				defer conn.Close()

				key := &types.TableMetaKey{Schema: "public", TableName: "users"}
				pkColumns := make([]types.ColumnMeta, 0, len(tc.primaryKey))
				for _, name := range tc.primaryKey {
					pkColumns = append(pkColumns, types.ColumnMeta{ColumnName: name})
				}
				meta := &types.TableMeta{TableName: "public.users", ColumnNames: tc.columns,
					Indexs: map[string]types.IndexMeta{"PRIMARY": {IType: types.IndexTypePrimaryKey, Columns: pkColumns}},
				}
				if tc.metaColumns != nil {
					meta.ColumnNames = tc.metaColumns
				}
				image := func(values []driver.Value) *types.RecordImage {
					columns := make([]types.ColumnImage, 0, len(tc.columns))
					for i, name := range tc.columns {
						column := types.ColumnImage{ColumnName: name, ColumnType: types.JDBCTypeBigInt, Value: values[i]}
						for _, pk := range tc.primaryKey {
							if name == pk {
								column.KeyType = types.IndexTypePrimaryKey
							}
						}
						columns = append(columns, column)
					}
					return &types.RecordImage{TableName: "public.users", TableMetaKey: key, TableMeta: meta,
						Rows: []types.RowImage{{Columns: columns}}, SQLType: types.SQLTypeUpdate}
				}
				log := undo.SQLUndoLog{TableName: "users", TableMetaKey: key,
					BeforeImage: image(tc.before), AfterImage: image(tc.after), SQLType: types.SQLTypeUpdate}
				current := append([]driver.Value(nil), tc.after...)
				if state == "already rolled back" {
					current = tc.before
				} else if state == "dirty data" {
					current[tc.dirtyIndex] = int64(999)
				}
				columnDefs := make([]*sqlmock.Column, 0, len(tc.columns))
				for _, name := range tc.columns {
					columnDefs = append(columnDefs, sqlmock.NewColumn(name).OfType("BIGINT", int64(0)))
				}
				mock.ExpectQuery(tc.checkSQL).WithArgs(tc.checkArgs...).
					WillReturnRows(sqlmock.NewRowsWithColumnDefinition(columnDefs...).AddRow(current...))
				if state == "after image" {
					mock.ExpectPrepare(tc.undoSQL).WillBeClosed().ExpectExec().WithArgs(tc.undoArgs...).
						WillReturnResult(sqlmock.NewResult(0, 1))
				}

				err = NewPostgreSQLUndoExecutorHolder().GetUpdateExecutor(log).
					ExecuteOn(context.Background(), types.DBTypePostgreSQL, conn)
				if state == "dirty data" {
					var undoErr *serr.SeataError
					require.ErrorAs(t, err, &undoErr)
					require.Equal(t, serr.SQLUndoDirtyError, undoErr.Code)
				} else {
					require.NoError(t, err)
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}
}
