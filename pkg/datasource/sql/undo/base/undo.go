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
	"database/sql"
	"database/sql/driver"
	"github.com/seata/seata-go/pkg/common/log"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
)

var _ undo.UndoLogManager = (*BaseUndoLogManager)(nil)

// Todo 等 delete undo 合并完成后放到 constant 目录下面
const CheckUndoLogTableExistSql = "SELECT 1 FROM " + " undo_log " + " LIMIT 1"

// BaseUndoLogManager
type BaseUndoLogManager struct{}

// Init
func (m *BaseUndoLogManager) Init() {
}

// InsertUndoLog
func (m *BaseUndoLogManager) InsertUndoLog(l []undo.BranchUndoLog, tx driver.Tx) error {
	return nil
}

// DeleteUndoLog
func (m *BaseUndoLogManager) DeleteUndoLogs(xid []string, branchID []int64, conn *sql.Conn) error {
	return nil
}

// FlushUndoLog
func (m *BaseUndoLogManager) FlushUndoLog(txCtx *types.TransactionContext, tx driver.Tx) error {
	return nil
}

// RunUndo
func (m *BaseUndoLogManager) RunUndo(xid string, branchID int64, conn *sql.Conn) error {
	return nil
}

// DBType
func (m *BaseUndoLogManager) DBType() types.DBType {
	panic("implement me")
}

// HasUndoLogTable check undo log table if exist
func (m *BaseUndoLogManager) HasUndoLogTable(ctx context.Context, conn *sql.Conn) (err error) {
	stmt, err := conn.PrepareContext(ctx, CheckUndoLogTableExistSql)
	if err != nil {
		log.Errorf("[HasUndoLogTable] prepare sql fail, err: %v", err)
		return
	}

	if _, err = stmt.ExecContext(ctx); err != nil {
		log.Errorf("[HasUndoLogTable] exec sql fail, err: %v", err)
		return
	}

	return
}
