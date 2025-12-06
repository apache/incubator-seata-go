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

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/base"
)

var _ undo.UndoLogManager = (*undoLogManager)(nil)

type undoLogManager struct {
	Base *base.BaseUndoLogManager
}

func NewUndoLogManager() *undoLogManager {
	return &undoLogManager{Base: base.NewBaseUndoLogManager()}
}

// Init init
func (m *undoLogManager) Init() {
}

// DeleteUndoLog
func (m *undoLogManager) DeleteUndoLog(ctx context.Context, xid string, branchID int64, conn *sql.Conn) error {
	return m.Base.DeleteUndoLog(ctx, xid, branchID, conn)
}

// BatchDeleteUndoLog
func (m *undoLogManager) BatchDeleteUndoLog(xid []string, branchID []int64, conn *sql.Conn) error {
	return m.Base.BatchDeleteUndoLog(xid, branchID, conn)
}

// FlushUndoLog
func (m *undoLogManager) FlushUndoLog(tranCtx *types.TransactionContext, conn driver.Conn) error {
	return m.Base.FlushUndoLog(tranCtx, conn)
}

// RunUndo undo sql
func (m *undoLogManager) RunUndo(ctx context.Context, xid string, branchID int64, db *sql.DB, dbName string) error {
	return m.Base.Undo(ctx, m.DBType(), xid, branchID, db, dbName)
}

// DBType
func (m *undoLogManager) DBType() types.DBType {
	return types.DBTypeMySQL
}

// HasUndoLogTable check undo log table if exist
func (m *undoLogManager) HasUndoLogTable(ctx context.Context, conn *sql.Conn) (bool, error) {
	return m.Base.HasUndoLogTable(ctx, conn)
}
