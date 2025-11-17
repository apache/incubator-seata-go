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

package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestNewUndoLogManager(t *testing.T) {
	manager := NewUndoLogManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.Base)
}

func TestUndoLogManager_Init(t *testing.T) {
	manager := NewUndoLogManager()
	// Init should not panic
	manager.Init()
}

func TestUndoLogManager_DBType(t *testing.T) {
	manager := NewUndoLogManager()
	dbType := manager.DBType()
	assert.Equal(t, types.DBTypeMySQL, dbType)
}

func TestUndoLogManager_DeleteUndoLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()
	ctx := context.Background()
	xid := "test-xid-123"
	branchID := int64(456)

	// Mock the connection
	mock.ExpectBegin()
	mock.ExpectPrepare("DELETE FROM (.+) WHERE branch_id = \\? AND xid = \\?").
		ExpectExec().
		WithArgs(branchID, xid).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Start transaction to get the conn in proper state
	tx, err := conn.BeginTx(ctx, nil)
	require.NoError(t, err)

	err = manager.DeleteUndoLog(ctx, xid, branchID, conn)
	assert.NoError(t, err)

	tx.Commit()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_DeleteUndoLog_PrepareError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()
	ctx := context.Background()
	xid := "test-xid-123"
	branchID := int64(456)

	// Mock connection and prepare error
	mock.ExpectBegin()
	mock.ExpectPrepare("DELETE FROM (.+) WHERE branch_id = \\? AND xid = \\?").
		WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	tx, err := conn.BeginTx(ctx, nil)
	require.NoError(t, err)

	err = manager.DeleteUndoLog(ctx, xid, branchID, conn)
	assert.Error(t, err)

	tx.Rollback()
}

func TestUndoLogManager_BatchDeleteUndoLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()
	ctx := context.Background()
	xids := []string{"xid-1", "xid-2"}
	branchIDs := []int64{100, 200}

	// Mock the batch delete
	mock.ExpectBegin()
	mock.ExpectPrepare("DELETE FROM (.+) WHERE branch_id IN (.+) AND xid IN (.+)").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	tx, err := conn.BeginTx(ctx, nil)
	require.NoError(t, err)

	err = manager.BatchDeleteUndoLog(xids, branchIDs, conn)
	assert.NoError(t, err)

	tx.Commit()
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_BatchDeleteUndoLog_EmptySlices(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()
	ctx := context.Background()

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Test with empty slices
	err = manager.BatchDeleteUndoLog([]string{}, []int64{}, conn)
	assert.Error(t, err)
}

func TestUndoLogManager_FlushUndoLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()

	// Create a mock driver.Conn
	driverConn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer driverConn.Close()

	// Create transaction context with empty round images
	tranCtx := &types.TransactionContext{
		XID:         "test-xid",
		BranchID:    123,
		RoundImages: &types.RoundRecordImage{},
	}

	// Get raw driver connection
	var rawConn driver.Conn
	err = driverConn.Raw(func(dc interface{}) error {
		rawConn = dc.(driver.Conn)
		return nil
	})
	require.NoError(t, err)

	// FlushUndoLog should return nil for empty images
	err = manager.FlushUndoLog(tranCtx, rawConn)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_FlushUndoLog_WithImages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Save original config and restore after test
	originalSerialization := undo.UndoConfig.LogSerialization
	originalCompressType := undo.UndoConfig.CompressConfig.Type
	defer func() {
		undo.UndoConfig.LogSerialization = originalSerialization
		undo.UndoConfig.CompressConfig.Type = originalCompressType
	}()

	undo.UndoConfig.LogSerialization = "json"
	undo.UndoConfig.CompressConfig.Type = "none"

	manager := NewUndoLogManager()

	// Create a mock driver.Conn
	driverConn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer driverConn.Close()

	// Create transaction context with images
	roundImages := &types.RoundRecordImage{}

	// Add before images
	beforeImage := &types.RecordImage{
		TableName: "test_table",
		SQLType:   types.SQLTypeInsert,
		Rows:      []types.RowImage{},
	}
	roundImages.AppendBeofreImage(beforeImage)

	// Add after images
	afterImage := &types.RecordImage{
		TableName: "test_table",
		SQLType:   types.SQLTypeInsert,
		Rows: []types.RowImage{
			{
				Columns: []types.ColumnImage{
					{ColumnName: "id", Value: 1},
				},
			},
		},
	}
	roundImages.AppendAfterImage(afterImage)

	tranCtx := &types.TransactionContext{
		XID:         "test-xid",
		BranchID:    123,
		RoundImages: roundImages,
	}

	// Mock the insert operation
	mock.ExpectPrepare("INSERT INTO (.+) VALUES").
		ExpectExec().
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Get raw driver connection
	var rawConn driver.Conn
	err = driverConn.Raw(func(dc interface{}) error {
		rawConn = dc.(driver.Conn)
		return nil
	})
	require.NoError(t, err)

	// FlushUndoLog should succeed
	err = manager.FlushUndoLog(tranCtx, rawConn)
	// Error expected as we're using sqlmock which doesn't fully implement all driver features
	// The test verifies the method is callable and handles the input
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_RunUndo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Save original config and restore after test
	originalSerialization := undo.UndoConfig.LogSerialization
	defer func() {
		undo.UndoConfig.LogSerialization = originalSerialization
	}()

	undo.UndoConfig.LogSerialization = "json"

	manager := NewUndoLogManager()
	ctx := context.Background()
	xid := "test-xid"
	branchID := int64(123)
	dbName := "test_db"

	// Mock the connection for Undo operation - expect full flow
	mock.ExpectBegin()
	mock.ExpectPrepare("SELECT (.+) FROM (.+) WHERE branch_id = \\? AND xid = \\? FOR UPDATE").
		ExpectQuery().
		WithArgs(branchID, xid).
		WillReturnRows(sqlmock.NewRows([]string{"branch_id", "xid", "context", "rollback_info", "log_status"}))
	// Expect INSERT for undo with global finished status
	mock.ExpectPrepare("INSERT INTO undo_log").
		ExpectExec().
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = manager.RunUndo(ctx, xid, branchID, db, dbName)
	assert.NoError(t, err)
	// The method will execute and attempt database operations
	// We're testing that it's callable and delegates to Base.Undo
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_HasUndoLogTable(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()
	ctx := context.Background()

	// Mock successful query (table exists)
	mock.ExpectQuery("SELECT 1 FROM (.+) LIMIT 1").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	exists, err := manager.HasUndoLogTable(ctx, conn)
	assert.NoError(t, err)
	assert.True(t, exists)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_InterfaceImplementation(t *testing.T) {
	var _ undo.UndoLogManager = (*undoLogManager)(nil)
	manager := NewUndoLogManager()
	assert.Implements(t, (*undo.UndoLogManager)(nil), manager)
}
