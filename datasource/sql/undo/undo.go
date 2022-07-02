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

package undo

import (
	"database/sql"
	"errors"
	"sync"

	"github.com/seata/seata-go-datasource/sql/types"
)

var (
	solts = map[types.DBType]*undoLogMgrHolder{}

	_ UndoLogManager = (*BaseUndoLogManager)(nil)
)

type undoLogMgrHolder struct {
	once sync.Once
	mgr  UndoLogManager
}

func Register(m UndoLogManager) error {
	if _, exist := solts[m.DBType()]; exist {
		return nil
	}

	solts[m.DBType()] = &undoLogMgrHolder{
		mgr:  m,
		once: sync.Once{},
	}
	return nil
}

// UndoLogManager
type UndoLogManager interface {
	Init(b *BaseUndoLogManager)
	// InsertUndoLog
	InsertUndoLog(l []UndoLog, conn *sql.Conn) error
	// DeleteUndoLog
	DeleteUndoLogs(xid, branchID []string, conn *sql.Conn) error
	// FlushUndoLog
	FlushUndoLog(txCtx *types.TransactionContext, tx *sql.Tx) error
	// RunUndo
	RunUndo(xid, branchID string, conn *sql.Conn) error
	// DBType
	DBType() types.DBType
}

// GetUndoLogManager
func GetUndoLogManager(d types.DBType) (UndoLogManager, error) {
	v, ok := solts[d]

	if !ok {
		return nil, errors.New("not found UndoLogManager")
	}

	v.once.Do(func() {
		v.mgr.Init(&BaseUndoLogManager{})
	})

	return v.mgr, nil
}

// UndoLog
type UndoLog struct {
	// Xid
	Xid string
	// BranchID
	BranchID string
	// Content
	Content []byte
}

// BaseUndoLogManager
type BaseUndoLogManager struct {
}

// Init
func (m *BaseUndoLogManager) Init(b *BaseUndoLogManager) {
	panic("implement me")
}

// InsertUndoLog
func (m *BaseUndoLogManager) InsertUndoLog(l []UndoLog, conn *sql.Conn) error {
	return nil
}

// DeleteUndoLog
func (m *BaseUndoLogManager) DeleteUndoLogs(xid, branchID []string, conn *sql.Conn) error {
	return nil
}

// FlushUndoLog
func (m *BaseUndoLogManager) FlushUndoLog(txCtx *types.TransactionContext, tx *sql.Tx) error {
	return nil
}

// RunUndo
func (m *BaseUndoLogManager) RunUndo(xid, branchID string, conn *sql.Conn) error {
	return nil
}

// DBType
func (m *BaseUndoLogManager) DBType() types.DBType {
	panic("implement me")
}
