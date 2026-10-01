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

package base

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
)

type tableMetaReaderStub struct {
	key         types.TableMetaKey
	resolvedKey types.TableMetaKey
	ref         types.TableRef
	err         error
}

func (r *tableMetaReaderStub) ResolveTableMetaKey(_ context.Context, _ driver.Conn, ref types.TableRef) (types.TableMetaKey, error) {
	r.ref = ref
	return r.resolvedKey, nil
}

func (r *tableMetaReaderStub) GetTableMeta(_ context.Context, key types.TableMetaKey) (*types.TableMeta, error) {
	r.key = key
	return nil, r.err
}

func TestFlushUndoLogPersistsTableMetaKey(t *testing.T) {
	key := &types.TableMetaKey{DBName: "tenant_a", TableName: "orders"}
	images := &types.RoundRecordImage{}
	images.AppendBeofreImage(&types.RecordImage{TableName: "orders", TableMetaKey: key, SQLType: types.SQLTypeUpdate, Rows: []types.RowImage{{}}})
	images.AppendAfterImage(&types.RecordImage{TableName: "orders", TableMetaKey: key, SQLType: types.SQLTypeUpdate, Rows: []types.RowImage{{}}})
	ctx := &types.TransactionContext{XID: "xid", BranchID: 1, RoundImages: images}
	previous := undo.UndoConfig.LogSerialization
	undo.UndoConfig.LogSerialization = "json"
	t.Cleanup(func() { undo.UndoConfig.LogSerialization = previous })
	var rollbackInfo []byte
	conn := &mockDriverConn{prepareFunc: func(string) (driver.Stmt, error) {
		return &mockDriverStmt{execFunc: func(args []driver.Value) (driver.Result, error) {
			rollbackInfo = args[3].([]byte)
			return &mockDriverResult{}, nil
		}}, nil
	}}

	require.NoError(t, NewBaseUndoLogManager().FlushUndoLog(ctx, conn))
	var decoded undo.BranchUndoLog
	err := json.Unmarshal(rollbackInfo, &decoded)
	require.NoError(t, err)
	require.Equal(t, key, decoded.Logs[0].TableMetaKey)
}

func TestUndoUsesSuppliedReaderAndStoredKey(t *testing.T) {
	key := types.TableMetaKey{DBName: "tenant_a", TableName: "orders"}
	lookupErr := errors.New("lookup from resource A")
	reader := &tableMetaReaderStub{err: lookupErr}
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	rollbackInfo, err := json.Marshal(&undo.BranchUndoLog{Xid: "xid", BranchID: 1, Logs: []undo.SQLUndoLog{{
		SQLType: types.SQLTypeUpdate, TableName: "orders", TableMetaKey: &key,
	}}})
	require.NoError(t, err)
	ctx := NewBaseUndoLogManager().encodeUndoLogCtx(map[string]string{serializerKey: "json"})
	mock.ExpectBegin()
	mock.ExpectPrepare("SELECT").WillBeClosed().ExpectQuery().WithArgs(int64(1), "xid").WillReturnRows(
		sqlmock.NewRows([]string{"branch_id", "xid", "context", "rollback_info", "log_status"}).AddRow(1, "xid", ctx, rollbackInfo, 0),
	)
	mock.ExpectRollback()

	err = NewBaseUndoLogManager().Undo(context.Background(), types.DBTypeMySQL, "xid", 1, db, "ignored_db", reader)
	require.ErrorIs(t, err, lookupErr)
	require.Equal(t, key, reader.key)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoRejectsMissingReader(t *testing.T) {
	err := NewBaseUndoLogManager().Undo(context.Background(), types.DBTypeMySQL, "xid", 1, nil, "db", nil)
	require.ErrorContains(t, err, "table meta reader")
}

func TestLegacyUndoLogResolvesTableOnRollbackResource(t *testing.T) {
	key := types.TableMetaKey{DBName: "tenant_a", Schema: "Sales", TableName: "Orders"}
	lookupErr := errors.New("lookup from rollback resource")
	reader := &tableMetaReaderStub{resolvedKey: key, err: lookupErr}
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	rollbackInfo, err := json.Marshal(&undo.BranchUndoLog{Xid: "xid", BranchID: 1, Logs: []undo.SQLUndoLog{{
		SQLType: types.SQLTypeUpdate, TableName: `"Sales"."Orders"`,
	}}})
	require.NoError(t, err)
	logCtx := NewBaseUndoLogManager().encodeUndoLogCtx(map[string]string{serializerKey: "json"})
	mock.ExpectBegin()
	mock.ExpectPrepare("SELECT").WillBeClosed().ExpectQuery().WithArgs(int64(1), "xid").WillReturnRows(
		sqlmock.NewRows([]string{"branch_id", "xid", "context", "rollback_info", "log_status"}).AddRow(1, "xid", logCtx, rollbackInfo, 0),
	)
	mock.ExpectRollback()

	err = NewBaseUndoLogManager(types.DBTypePostgreSQL).Undo(context.Background(), types.DBTypePostgreSQL, "xid", 1, db, "ignored_db", reader)
	require.ErrorIs(t, err, lookupErr)
	require.Equal(t, key, reader.key)
	require.Equal(t, types.TableRef{Qualifier: "Sales", TableName: "Orders", QualifierQuoted: true, TableNameQuoted: true}, reader.ref)
	require.NoError(t, mock.ExpectationsWereMet())
}
