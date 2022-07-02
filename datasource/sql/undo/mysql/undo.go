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
	"database/sql"

	"github.com/seata/seata-go-datasource/sql/types"
	"github.com/seata/seata-go-datasource/sql/undo"
)

var (
	_ undo.UndoLogManager = (*undoLogManager)(nil)
)

type undoLogManager struct {
	Base *undo.BaseUndoLogManager
}

func (m *undoLogManager) Init(b *undo.BaseUndoLogManager) {
	m.Base = b
}

// InsertUndoLog
func (m *undoLogManager) InsertUndoLog(l []undo.UndoLog, conn *sql.Conn) error {
	return m.Base.InsertUndoLog(l, conn)
}

// DeleteUndoLog
func (m *undoLogManager) DeleteUndoLogs(xid, branchID []string, conn *sql.Conn) error {
	return m.Base.DeleteUndoLogs(xid, branchID, conn)
}

// FlushUndoLog
func (m *undoLogManager) FlushUndoLog(xid, branchID string, logs []undo.UndoLog, tx *sql.Tx) error {
	return m.Base.FlushUndoLog(tx)
}

// RunUndo
func (m *undoLogManager) RunUndo(xid, branchID string, conn *sql.Conn) error {
	return m.Base.RunUndo(xid, branchID, conn)
}

// DBType
func (m *undoLogManager) DBType() types.DBType {
	return types.MySQL
}
