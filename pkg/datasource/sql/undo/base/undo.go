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
	"encoding/json"
	"strconv"
	"strings"

	"github.com/arana-db/parser/mysql"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/factor"
	"github.com/seata/seata-go/pkg/util/log"
)

// checkUndoLogTableExistSql check undo log if exist
var (
	ErrorDeleteUndoLogParamsFault = errors.New("xid or branch_id can't nil")
)

var (
	checkUndoLogTableExistSql = "SELECT 1 FROM " + constant.UndoLogTableName + " LIMIT 1"
	insertUndoLogSql          = "INSERT INTO " + constant.UndoLogTableName + "(branch_id,xid,context,rollback_info,log_status,log_created,log_modified) VALUES (?, ?, ?, ?, ?, now(6), now(6))"
	selectUndoLogSql          = "SELECT `branch_id`,`xid`,`context`,`rollback_info`,`log_status` FROM " + constant.UndoLogTableName + " WHERE " + constant.UndoLogBranchXid + " = ? AND " + constant.UndoLogXid + " = ? FOR UPDATE" // todo 替换成常量吧，不用使用变量来表示字段名
)

const (
	PairSplit = "&"
	KvSplit   = "="

	CompressorTypeKey = "compressorTypeKey"
	SerializerKey     = "serializerKey"

	// CheckUndoLogTableExistSql check undo log if exist
	CheckUndoLogTableExistSql = "SELECT 1 FROM " + constant.UndoLogTableName + " LIMIT 1"
	// DeleteUndoLogSql delete undo log
	DeleteUndoLogSql = constant.DeleteFrom + constant.UndoLogTableName + " WHERE " + constant.UndoLogBranchXid + " = ? AND " + constant.UndoLogXid + " = ?"
)

// undo log status
const (
	// UndoLogStatusNormal This state can be properly rolled back by services
	UndoLogStatusNormal = iota
	// UndoLogStatusGlobalFinished This state prevents the branch transaction from inserting undo_log after the global transaction is rolled back.
	UndoLogStatusGlobalFinished
)

// BaseUndoLogManager
type BaseUndoLogManager struct{}

func NewBaseUndoLogManager() *BaseUndoLogManager {
	return &BaseUndoLogManager{}
}

// Init
func (m *BaseUndoLogManager) Init() {
}

// InsertUndoLog
func (m *BaseUndoLogManager) InsertUndoLog(record undo.UndologRecord, conn driver.Conn) error {
	stmt, err := conn.Prepare(insertUndoLogSql)
	if err != nil {
		return err
	}
	_, err = stmt.Exec([]driver.Value{record.BranchID, record.XID, record.Context, record.RollbackInfo, int64(record.LogStatus)})
	if err != nil {
		return err
	}
	return nil
}

func (m *BaseUndoLogManager) InsertUndoLogWithSqlConn(ctx context.Context, record undo.UndologRecord, conn *sql.Conn) error {
	stmt, err := conn.PrepareContext(ctx, insertUndoLogSql)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(record.BranchID, record.XID, record.Context, record.RollbackInfo, int64(record.LogStatus))
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

	if _, err = stmt.Exec(branchID, xid); err != nil {
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

	if beforeImages.IsEmptyImage() && afterImages.IsEmptyImage() {
		return nil
	}

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
	rollbackInfo, err := json.Marshal(branchUndoLog)
	if err != nil {
		return err
	}

	parseContext := make(map[string]string, 0)
	parseContext[SerializerKey] = "jackson"
	parseContext[CompressorTypeKey] = "NONE"
	undoLogContent, err := json.Marshal(parseContext)
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

// RunUndo undo sql
func (m *BaseUndoLogManager) RunUndo(ctx context.Context, xid string, branchID int64, conn *sql.DB, dbName string) error {
	return nil
}

// Undo undo sql
func (m *BaseUndoLogManager) Undo(ctx context.Context, dbType types.DBType, xid string, branchID int64, db *sql.DB, dbName string) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}

	tx, err := conn.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if err = tx.Rollback(); err != nil {
				log.Errorf("rollback fail, xid: %s, branchID:%s err:%v", xid, branchID, err)
				return
			}
		}
	}()

	stmt, err := conn.PrepareContext(ctx, selectUndoLogSql)
	if err != nil {
		log.Errorf("prepare sql fail, err: %v", err)
		return err
	}
	defer func() {
		if err = stmt.Close(); err != nil {
			log.Errorf("stmt close fail, xid: %s, branchID:%s err:%v", xid, branchID, err)
			return
		}
	}()

	rows, err := stmt.Query(branchID, xid)
	if err != nil {
		log.Errorf("query sql fail, err: %v", err)
		return err
	}
	defer func() {
		if err = rows.Close(); err != nil {
			log.Errorf("rows close fail, xid: %s, branchID:%s err:%v", xid, branchID, err)
			return
		}
	}()

	var undoLogRecords []undo.UndologRecord
	for rows.Next() {
		var record undo.UndologRecord
		err = rows.Scan(&record.BranchID, &record.XID, &record.Context, &record.RollbackInfo, &record.LogStatus)
		if err != nil {
			return err
		}
		undoLogRecords = append(undoLogRecords, record)
	}

	var exists bool
	for _, record := range undoLogRecords {
		exists = true
		if !record.CanUndo() {
			log.Infof("xid %v branch %v, ignore %v undo_log", record.XID, record.BranchID, record.LogStatus)
			return nil
		}

		// todo use serializer and decode
		var branchUndoLog undo.BranchUndoLog
		if err = json.Unmarshal(record.RollbackInfo, &branchUndoLog); err != nil {
			return err
		}

		sqlUndoLogs := branchUndoLog.Logs
		if len(sqlUndoLogs) == 0 {
			return nil
		}
		branchUndoLog.Reverse()

		for _, undoLog := range sqlUndoLogs {
			tableMeta, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, dbName, undoLog.TableName)
			if err != nil {
				log.Errorf("get table meta fail, err: %v", err)
				return err
			}

			undoLog.SetTableMeta(*tableMeta)

			undoExecutor, err := factor.GetUndoExecutor(dbType, undoLog)
			if err != nil {
				log.Errorf("get undo executor, err: %v", err)
				return err
			}

			if err = undoExecutor.ExecuteOn(ctx, dbType, conn); err != nil {
				log.Errorf("execute on fail, err: %v", err)
				return err
			}
		}
	}

	if exists {
		if err = m.DeleteUndoLog(ctx, xid, branchID, conn); err != nil {
			log.Errorf("[Undo] delete undo fail, err: %v", err)
			return err
		}
		log.Infof("xid %v branch %v, undo_log deleted with %v", xid, branchID, undo.UndoLogStatueGlobalFinished)
	} else {
		if err = m.insertUndoLogWithGlobalFinished(ctx, xid, uint64(branchID), conn); err != nil {
			log.Errorf("[Undo] insert undo with global finished fail, err: %v", err)
			return err
		}
		log.Infof("xid %v branch %v, undo_log added with %v", xid, branchID, undo.UndoLogStatueGlobalFinished)
	}

	if err = tx.Commit(); err != nil {
		log.Errorf("[Undo] execute on fail, err: %v", err)
		return nil
	}
	return nil
}

func (m *BaseUndoLogManager) insertUndoLogWithGlobalFinished(ctx context.Context, xid string, branchID uint64, conn *sql.Conn) error {
	// todo use config to replace
	parseContext := make(map[string]string, 0)
	parseContext[SerializerKey] = "jackson"
	parseContext[CompressorTypeKey] = "NONE"
	undoLogContent, err := json.Marshal(parseContext)
	if err != nil {
		return err
	}

	record := undo.UndologRecord{
		BranchID:     branchID,
		XID:          xid,
		RollbackInfo: []byte("{}"),
		LogStatus:    UndoLogStatusGlobalFinished,
		Context:      undoLogContent,
	}
	err = m.InsertUndoLogWithSqlConn(ctx, record, conn)
	if err != nil {
		log.Errorf("insert undo log fail, err: %v", err)
		return err
	}
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

// canUndo check if it can undo
func (m *BaseUndoLogManager) canUndo(state int32) bool {
	return state == UndoLogStatusNormal
}

// parseContext parse undo context
func (m *BaseUndoLogManager) parseContext(str string) map[string]string {
	return m.DecodeMap(str)
}

// DecodeMap Decode undo log context string to map
func (m *BaseUndoLogManager) DecodeMap(str string) map[string]string {
	res := make(map[string]string)

	if str == "" {
		return nil
	}

	strSlice := strings.Split(str, PairSplit)
	if len(strSlice) == 0 {
		return nil
	}

	for key, _ := range strSlice {
		kv := strings.Split(strSlice[key], KvSplit)
		if len(kv) != 2 {
			continue
		}

		res[kv[0]] = kv[1]
	}

	return res
}

// getRollbackInfo parser rollback info
func (m *BaseUndoLogManager) getRollbackInfo(rollbackInfo []byte, undoContext map[string]string) []byte {
	// Todo use compressor
	// get compress type
	/*compressorType, ok := undoContext[constant.CompressorTypeKey]
	if ok {

	}*/

	return rollbackInfo
}

// getSerializer get serializer from undo context
func (m *BaseUndoLogManager) getSerializer(undoLogContext map[string]string) (serializer string) {
	if undoLogContext == nil {
		return
	}
	serializer, _ = undoLogContext[SerializerKey]
	return
}
