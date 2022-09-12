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
	"sync"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/impl"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

var solts = map[types.DBType]*undoLogMgrHolder{}

type undoLogMgrHolder struct {
	once sync.Once
	mgr  UndoLogManager
}

func Regis(m UndoLogManager) error {
	if _, exist := solts[m.DBType()]; exist {
		return nil
	}

	solts[m.DBType()] = &undoLogMgrHolder{
		mgr:  m,
		once: sync.Once{},
	}
	return nil
}

// GetUndoLogManager
func GetUndoLogManager(d types.DBType) (UndoLogManager, error) {
	v, ok := solts[d]

	if !ok {
		return nil, errors.New("not found UndoLogManager")
	}

	v.once.Do(func() {
		v.mgr.Init()
	})

	return v.mgr, nil
}

// UndoLogManager
type UndoLogManager interface {
	Init()
	// InsertUndoLog
	InsertUndoLog(l []impl.BranchUndoLog, tx driver.Tx) error
	// DeleteUndoLog
	DeleteUndoLog(ctx context.Context, xid string, branchID int64, conn *sql.Conn) error
	// BatchDeleteUndoLog
	BatchDeleteUndoLog(xid []string, branchID []int64, conn *sql.Conn) error
	// FlushUndoLog
	FlushUndoLog(txCtx *types.TransactionContext, tx driver.Tx) error
	// Undo
	Undo(ctx context.Context, xid string, branchID int64, conn *sql.Conn) error
	// DBType
	DBType() types.DBType
}

// UndoLogParser
type UndoLogParser interface {
	// GetName
	GetName() string
	// GetDefaultContent
	GetDefaultContent() []byte
	// Encode
	Encode(l impl.BranchUndoLog) []byte
	// Decode
	Decode(b []byte) impl.BranchUndoLog
}
