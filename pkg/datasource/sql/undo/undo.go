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
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sync"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	undoLogManagerMap = map[types.DBType]*undoLogMgrHolder{}
	builders          = map[types.ExecutorType]func() UndoLogBuilder{}
)

type undoLogMgrHolder struct {
	once sync.Once
	mgr  UndoLogManager
}

func RegisterUndoLogManager(m UndoLogManager) error {
	if _, exist := undoLogManagerMap[m.DBType()]; exist {
		return nil
	}

	undoLogManagerMap[m.DBType()] = &undoLogMgrHolder{
		mgr:  m,
		once: sync.Once{},
	}
	return nil
}

func RegisterUndoLogBuilder(executorType types.ExecutorType, fun func() UndoLogBuilder) {
	if _, ok := builders[executorType]; !ok {
		builders[executorType] = fun
	}
}

func GetUndologBuilder(sqlType types.ExecutorType) UndoLogBuilder {
	if f, ok := builders[sqlType]; ok {
		return f()
	}
	return nil
}

// UndoLogManager
type UndoLogManager interface {
	Init()
	// DeleteUndoLog
	DeleteUndoLog(ctx context.Context, xid string, branchID int64, conn *sql.Conn) error
	// BatchDeleteUndoLog
	BatchDeleteUndoLog(xid []string, branchID []int64, conn *sql.Conn) error
	//FlushUndoLog
	FlushUndoLog(tranCtx *types.TransactionContext, conn driver.Conn) error
	// RunUndo
	RunUndo(ctx context.Context, xid string, branchID int64, conn *sql.DB, dbName string) error
	// DBType
	DBType() types.DBType
	// HasUndoLogTable
	HasUndoLogTable(ctx context.Context, conn *sql.Conn) (bool, error)
}

// GetUndoLogManager
func GetUndoLogManager(d types.DBType) (UndoLogManager, error) {
	v, ok := undoLogManagerMap[d]

	if !ok {
		return nil, fmt.Errorf("not found UndoLogManager")
	}

	v.once.Do(func() {
		v.mgr.Init()
	})

	return v.mgr, nil
}

type UndoLogStatue int8

const (
	UndoLogStatueNormnal        UndoLogStatue = 0
	UndoLogStatueGlobalFinished UndoLogStatue = 1
)

type UndologRecord struct {
	BranchID     uint64        `json:"branchId"`
	XID          string        `json:"xid"`
	Context      []byte        `json:"context"`
	RollbackInfo []byte        `json:"rollbackInfo"`
	LogStatus    UndoLogStatue `json:"logStatus"`
	LogCreated   []byte        `json:"logCreated"`
	LogModified  []byte        `json:"logModified"`
}

func (u *UndologRecord) CanUndo() bool {
	return u.LogStatus == UndoLogStatueNormnal
}

// BranchUndoLog
type BranchUndoLog struct {
	// Xid
	Xid string `json:"xid"`
	// BranchID
	BranchID uint64 `json:"branchId"`
	// Logs
	Logs []SQLUndoLog `json:"sqlUndoLogs"`
}

// Marshal
func (b *BranchUndoLog) Marshal() []byte {
	return nil
}

func (b *BranchUndoLog) Reverse() {
	if len(b.Logs) == 0 {
		return
	}

	left, right := 0, len(b.Logs)-1

	for left < right {
		b.Logs[left], b.Logs[right] = b.Logs[right], b.Logs[left]
		left++
		right--
	}
}

// SQLUndoLog
type SQLUndoLog struct {
	SQLType     types.SQLType      `json:"sqlType"`
	TableName   string             `json:"tableName"`
	BeforeImage *types.RecordImage `json:"beforeImage"`
	AfterImage  *types.RecordImage `json:"afterImage"`
}

func (s SQLUndoLog) SetTableMeta(tableMeta *types.TableMeta) {
	if s.BeforeImage != nil {
		s.BeforeImage.TableMeta = tableMeta
		s.BeforeImage.TableName = tableMeta.TableName
	}
	if s.AfterImage != nil {
		s.AfterImage.TableMeta = tableMeta
		s.AfterImage.TableName = tableMeta.TableName
	}
}

type UndoLogBuilder interface {
	BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error)
	AfterImage(ctx context.Context, execCtx *types.ExecContext, beforImages []*types.RecordImage) ([]*types.RecordImage, error)
	GetExecutorType() types.ExecutorType
}
