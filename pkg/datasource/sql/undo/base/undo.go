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
	"strings"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/common/util"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

var _ undo.UndoLogManager = (*BaseUndoLogManager)(nil)

var ErrorDeleteUndoLogParamsFault = errors.New("xid or branch_id can't nil")

// BaseUndoLogManager
type BaseUndoLogManager struct{}

// Init
func (m *BaseUndoLogManager) Init() {
}

// InsertUndoLog
func (m *BaseUndoLogManager) InsertUndoLog(l []undo.BranchUndoLog, tx driver.Tx) error {
	return nil
}

// DeleteUndoLog exec delete single undo log operate
func (m *BaseUndoLogManager) DeleteUndoLog(ctx context.Context, xid string, branchID int64, conn *sql.Conn) error {
	stmt, err := conn.PrepareContext(ctx, constant.DeleteUndoLogSql)
	if err != nil {
		log.Errorf("[DeleteUndoLog] prepare sql fail, err: %v", err)
		return err
	}

	if _, err = stmt.ExecContext(ctx, branchID, xid); err != nil {
		log.Errorf("[DeleteUndoLog] exec delete undo log fail, err: %v", err)
		return err
	}

	return nil
}

// BatchDeleteUndoLog exec delete undo log operate
func (m *BaseUndoLogManager) BatchDeleteUndoLog(xid []string, branchID []int64, conn *sql.Conn) error {
	// build delete undo log sql
	batchDeleteSql, err := m.getBatchDeleteUndoLogSql(xid, branchID)
	if err != nil {
		log.Errorf("get undo sql log fail, err: %v", err)
		return err
	}

	ctx := context.Background()

	// prepare deal sql
	stmt, err := conn.PrepareContext(ctx, batchDeleteSql)
	if err != nil {
		log.Errorf("prepare sql fail, err: %v", err)
		return err
	}

	branchIDStr, err := util.Int64Slice2Str(branchID, ",")
	if err != nil {
		log.Errorf("slice to string transfer fail, err: %v", err)
		return err
	}

	// exec sql stmt
	if _, err = stmt.ExecContext(ctx, branchIDStr, strings.Join(xid, ",")); err != nil {
		log.Errorf("exec delete undo log fail, err: %v", err)
		return err
	}

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
	stmt, err := conn.PrepareContext(ctx, constant.CheckUndoLogTableExistSql)
	if err != nil {
		log.Errorf("[HasUndoLogTable] prepare sql fail, err: %v", err)
		return
	}

	if _, err = stmt.Query(); err != nil {
		log.Errorf("[HasUndoLogTable] exec sql fail, err: %v", err)
		return
	}

	return
}

// getBatchDeleteUndoLogSql build batch delete undo log
func (m *BaseUndoLogManager) getBatchDeleteUndoLogSql(xid []string, branchID []int64) (string, error) {
	if len(xid) == 0 || len(branchID) == 0 {
		return "", ErrorDeleteUndoLogParamsFault
	}

	var undoLogDeleteSql strings.Builder
	undoLogDeleteSql.WriteString(constant.DeleteFrom)
	undoLogDeleteSql.WriteString(constant.UndoLogTableName)
	undoLogDeleteSql.WriteString(" WHERE ")
	undoLogDeleteSql.WriteString(constant.UndoLogBranchXid)
	undoLogDeleteSql.WriteString(" IN ")
	m.appendInParam(len(branchID), &undoLogDeleteSql)
	undoLogDeleteSql.WriteString(" AND ")
	undoLogDeleteSql.WriteString(constant.UndoLogXid)
	undoLogDeleteSql.WriteString(" IN ")
	m.appendInParam(len(xid), &undoLogDeleteSql)

	return undoLogDeleteSql.String(), nil
}

// appendInParam build in param
func (m *BaseUndoLogManager) appendInParam(size int, str *strings.Builder) {
	if size <= 0 {
		return
	}

	str.WriteString(" (")
	for i := 0; i < size; i++ {
		str.WriteString("?")
		if i < size-1 {
			str.WriteString(",")
		}
	}

	str.WriteString(") ")
}
