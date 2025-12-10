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
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/base"
)

func TestNewUndoLogManager(t *testing.T) {
	manager := NewUndoLogManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.Base)
	assert.IsType(t, &base.BaseUndoLogManager{}, manager.Base)
}

func TestUndoLogManager_Init(t *testing.T) {
	manager := NewUndoLogManager()
	assert.NotPanics(t, func() {
		manager.Init()
	})
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

	// Mock the DELETE query that Base.DeleteUndoLog would execute
	mock.ExpectPrepare("DELETE FROM (.+) WHERE").
		ExpectExec().
		WithArgs(int64(123), "test-xid").
		WillReturnResult(sqlmock.NewResult(0, 1))

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	err = manager.DeleteUndoLog(ctx, "test-xid", 123, conn)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_BatchDeleteUndoLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewUndoLogManager()

	// Mock the batch DELETE query that Base.BatchDeleteUndoLog would execute
	// Parameters: branchIDStr (comma-separated), xidStr (comma-separated)
	mock.ExpectPrepare("DELETE FROM (.+) WHERE").
		ExpectExec().
		WithArgs("123,456", "xid1,xid2").
		WillReturnResult(sqlmock.NewResult(0, 2))

	conn, err := db.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()

	err = manager.BatchDeleteUndoLog([]string{"xid1", "xid2"}, []int64{123, 456}, conn)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUndoLogManager_FlushUndoLog(t *testing.T) {
	manager := NewUndoLogManager()

	// Test with empty RoundImages (should return nil)
	tranCtx := &types.TransactionContext{
		XID:         "test-xid",
		BranchID:    123,
		RoundImages: &types.RoundRecordImage{},
	}

	err := manager.FlushUndoLog(tranCtx, nil)
	// Should return nil for empty images without needing database connection
	assert.NoError(t, err)
}

func TestUndoLogManager_RunUndo(t *testing.T) {
	// Skip test since it requires a real database connection
	t.Skip("Skipping test that requires database connection")

	manager := NewUndoLogManager()
	assert.NotPanics(t, func() {
		manager.RunUndo(context.Background(), "test-xid", 123, nil, "test_db")
	})
}

func TestUndoLogManager_HasUndoLogTable(t *testing.T) {
	t.Run("table exists", func(t *testing.T) {
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
	})
}

func TestUndoLogManager_ImplementsInterface(t *testing.T) {
	manager := NewUndoLogManager()
	assert.Implements(t, (*undo.UndoLogManager)(nil), manager)
}
