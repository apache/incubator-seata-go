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

	"github.com/goccy/go-json"

	"github.com/arana-db/parser/mysql"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/util/log"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

// checkUndoLogTableExistSql check undo log if exist
var (
	ErrorDeleteUndoLogParamsFault = errors.New("xid or branch_id can't nil")
)

const (
	checkUndoLogTableExistSql = "SELECT 1 FROM " + constant.UndoLogTableName + " LIMIT 1"
	insertUndoLogSql          = "INSERT INTO " + constant.UndoLogTableName + "(branch_id,xid,context,rollback_info,log_status,log_created,log_modified) VALUES (?, ?, ?, ?, ?, now(6), now(6))"
)

const (
	serializerKey     = "serializer"
	compressorTypeKey = "compressorType"
)

// BaseUndoLogManager
type BaseUndoLogManager struct{}

// Init
func (m *BaseUndoLogManager) Init() {
}

// InsertUndoLog
func (m *BaseUndoLogManager) InsertUndoLog(record undo.UndologRecord, conn driver.Conn) error {
	stmt, err := conn.Prepare(insertUndoLogSql)
	if err != nil {
		return err
	}
	_, err = stmt.Exec([]driver.Value{record.BranchID, record.XID, record.Context, record.RollbackInfo, record.LogStatus})
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
func (m *BaseUndoLogManager) FlushUndoLog(tranCtx *types.TransactionContext, conn driver.Conn) error {
	if tranCtx.RoundImages.IsEmpty() {
		return nil
	}

	sqlUndoLogs := make([]undo.SQLUndoLog, 0)
	beforeImages := tranCtx.RoundImages.BeofreImages()
	afterImages := tranCtx.RoundImages.AfterImages()

	for i := 0; i < len(beforeImages); i++ {
		var (
			tableName string
			sqlType   types.SQLType
		)

		if beforeImages[i] != nil {
			tableName = beforeImages[i].TableName
			sqlType = beforeImages[i].SQLType
		} else if afterImages[i] != nil {
			tableName = afterImages[i].TableName
			sqlType = afterImages[i].SQLType
		} else {
			continue
		}

		undoLog := undo.SQLUndoLog{
			SQLType:     sqlType,
			TableName:   tableName,
			BeforeImage: beforeImages[i],
			AfterImage:  afterImages[i],
		}
		sqlUndoLogs = append(sqlUndoLogs, undoLog)
	}

	branchUndoLog := undo.BranchUndoLog{
		Xid:      tranCtx.XID,
		BranchID: tranCtx.BranchID,
		Logs:     sqlUndoLogs,
	}

	// use defalut encode
	undoLogContent, err := json.Marshal(branchUndoLog)
	if err != nil {
		return err
	}

	parseContext := make(map[string]string, 0)
	parseContext[serializerKey] = "jackson"
	parseContext[compressorTypeKey] = "NONE"
	rollbackInfo, err := json.Marshal(parseContext)
	if err != nil {
		return err
	}

	return m.InsertUndoLog(undo.UndologRecord{
		BranchID:     tranCtx.BranchID,
		XID:          tranCtx.XID,
		Context:      undoLogContent,
		RollbackInfo: rollbackInfo,
		LogStatus:    undo.UndoLogStatueNormnal,
	}, conn)
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
	if _, err = conn.QueryContext(ctx, checkUndoLogTableExistSql); err != nil {
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
