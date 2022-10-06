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
	"strconv"
	"strings"

	"github.com/arana-db/parser/mysql"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/parser"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/common/util"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/util/log"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

var _ undo.UndoLogManager = (*BaseUndoLogManager)(nil)

var ErrorDeleteUndoLogParamsFault = errors.New("xid or branch_id can't nil")

// CheckUndoLogTableExistSql check undo log if exist
const CheckUndoLogTableExistSql = "SELECT 1 FROM " + constant.UndoLogTableName + " LIMIT 1"

// BaseUndoLogManager
type BaseUndoLogManager struct{}

// Init
func (m *BaseUndoLogManager) Init() {
}

// InsertUndoLog
func (m *BaseUndoLogManager) InsertUndoLog(ctx context.Context, l []undo.BranchUndoLog, conn *sql.Conn) error {

	tx, err := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Errorf("[InsertUndoLog] prepare sql fail, err: %v ", err)
		return err
	}
	size := len(l)

	for i := 0; i < size; i++ {
		 parser := parser.BaseParser{}
		 //TODO Serialized undolog protocol

		tx.ExecContext(ctx, constant.InsertUndoLogSql , l[i].BranchID, l[i].Xid, l[i].)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

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

	branchIDStr, err := Int64Slice2Str(branchID, ",")
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

// FlushUndoLog flush undo log
func (m *BaseUndoLogManager) FlushUndoLog(txCtx *types.TransactionContext, tx driver.Conn) error {
	if !txCtx.HasUndoLog() {
		return nil
	}
	logs := []undo.SQLUndoLog{
		{
			SQLType:   types.SQLTypeInsert,
			TableName: constant.UndoLogTableName,
			Images:    *txCtx.RoundImages,
		},
	}
	branchUndoLogs := []undo.BranchUndoLog{
		{
			Xid:      txCtx.XaID,
			BranchID: strconv.FormatUint(txCtx.BranchID, 10),
			Logs:     logs,
		},
	}
	return m.InsertUndoLog(branchUndoLogs, tx)
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
func (m *BaseUndoLogManager) HasUndoLogTable(ctx context.Context, conn *sql.Conn) (res bool, err error) {
	if _, err = conn.QueryContext(ctx, CheckUndoLogTableExistSql); err != nil {
		// 1146 mysql table not exist fault code
		if e, ok := err.(*mysql.SQLError); ok && e.Code == mysql.ErrNoSuchTable {
			return false, nil
		}
		log.Errorf("[HasUndoLogTable] query sql fail, err: %v", err)
		return
	}

	return true, nil
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

// Int64Slice2Str
func Int64Slice2Str(values interface{}, sep string) (string, error) {
	v, ok := values.([]int64)
	if !ok {
		return "", errors.New("param type is fault")
	}

	var valuesText []string

	for i := range v {
		text := strconv.FormatInt(v[i], 10)
		valuesText = append(valuesText, text)
	}

	return strings.Join(valuesText, sep), nil
}
