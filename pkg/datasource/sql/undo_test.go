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
	"github.com/seata/seata-go/pkg/datasource/sql/undo/base"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestDeleteUndoLogs
func TestDeleteUndoLogs(t *testing.T) {
	// local test can annotation t.SkipNow()
	t.SkipNow()

	testDeleteUndoLogs := func() {
		db, err := sql.Open("seata-mysql", "root:12345678@tcp(127.0.0.1:3306)/seata_order?multiStatements=true")
		assert.Nil(t, err)

		sqlConn, err := db.Conn(context.Background())
		assert.Nil(t, err)

		undoLogManager := new(base.BaseUndoLogManager)

		err = undoLogManager.DeleteUndoLogs([]string{"1"}, []int64{1}, sqlConn)
		assert.Nil(t, err)
	}

	t.Run("test_delete_undo_logs", func(t *testing.T) {
		testDeleteUndoLogs()
	})
}
