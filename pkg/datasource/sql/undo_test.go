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

package sql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/undo/base"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/mysql"
)

// TestBatchDeleteUndoLogs
func TestBatchDeleteUndoLogs(t *testing.T) {
	// local test can annotation t.SkipNow()
	t.SkipNow()

	testBatchDeleteUndoLogs := func() {
		db, err := sql.Open(SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_order?multiStatements=true")
		assert.Nil(t, err)

		sqlConn, err := db.Conn(context.Background())
		assert.Nil(t, err)

		undoLogManager := new(base.BaseUndoLogManager)

		err = undoLogManager.BatchDeleteUndoLog([]string{"1"}, []int64{1}, sqlConn)
		assert.Nil(t, err)
	}

	t.Run("test_batch_delete_undo_logs", func(t *testing.T) {
		testBatchDeleteUndoLogs()
	})
}

func TestDeleteUndoLogs(t *testing.T) {
	// local test can annotation t.SkipNow()
	t.SkipNow()

	testDeleteUndoLogs := func() {
		/*db, err := sql.Open(SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_order?multiStatements=true")
		assert.Nil(t, err)

		ctx := context.Background()
		sqlConn, err := db.Conn(ctx)
		assert.Nil(t, err)

		undoLogManager := new(base.BaseUndoLogManager)

		err = undoLogManager.DeleteUndoLog(ctx, "1", 1, sqlConn)
		assert.Nil(t, err)*/
	}

	t.Run("test_delete_undo_logs", func(t *testing.T) {
		testDeleteUndoLogs()
	})
}

// TestHasUndoLogTable
func TestHasUndoLogTable(t *testing.T) {
	// local test can annotation t.SkipNow()
	t.SkipNow()

	testHasUndoLogTable := func() {
		db, err := sql.Open(SeataATMySQLDriver, "root:12345678@tcp(127.0.0.1:3306)/seata_order?multiStatements=true")
		assert.Nil(t, err)

		ctx := context.Background()
		sqlConn, err := db.Conn(ctx)
		assert.Nil(t, err)

		undoLogManager := new(base.BaseUndoLogManager)

		res, err := undoLogManager.HasUndoLogTable(ctx, sqlConn)
		assert.Nil(t, err)
		assert.True(t, res)
	}

	t.Run("test_has_undo_log_table", func(t *testing.T) {
		testHasUndoLogTable()
	})
}

func TestUndo(t *testing.T) {
	// Todo TestUndo update
	// Todo TestUndo delete
	// local test can annotation t.SkipNow()
	t.SkipNow()
	testUndoLog := func() {
		manager := mysql.NewUndoLogManager()

		db, err := sql.Open(SeataATMySQLDriver, "root:123456@tcp(127.0.0.1:3306)/seata_order?multiStatements=true")
		assert.Nil(t, err)

		ctx := context.Background()
		sqlConn, err := db.Conn(ctx)
		assert.Nil(t, err)

		defer func() {
			_ = sqlConn.Close()
		}()

		if err = manager.RunUndo(ctx, "36375516866494489", 28, db, "seata_order"); err != nil {
			t.Logf("%+v", err)
		}

		assert.Nil(t, err)
	}

	t.Run("test_undo_log", func(t *testing.T) {
		testUndoLog()
	})
}
