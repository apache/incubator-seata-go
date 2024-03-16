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
	"fmt"

	"strconv"
	"strings"

	"github.com/arana-db/parser/mysql"

	"seata.apache.org/seata-go/pkg/compressor"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/factor"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/parser"
	"seata.apache.org/seata-go/pkg/util/collection"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	compressorTypeKey       = "compressorTypeKey"
	serializerKey           = "serializerKey"
	defaultUndoLogTableName = " undo_log "
)

func getUndoLogTableName() string {
	if undo.UndoConfig.LogTable != "" {
		return undo.UndoConfig.LogTable
	}
	return defaultUndoLogTableName
}

func getCheckUndoLogTableExistSql() string {
	return "SELECT 1 FROM " + getUndoLogTableName() + " LIMIT 1"
}

func getInsertUndoLogSql() string {
	return "INSERT INTO " + getUndoLogTableName() + "(branch_id,xid,context,rollback_info,log_status,log_created,log_modified) VALUES (?, ?, ?, ?, ?, now(6), now(6))"
}

func getSelectUndoLogSql() string {
	return "SELECT `branch_id`,`xid`,`context`,`rollback_info`,`log_status` FROM " + getUndoLogTableName() + " WHERE branch_id = ? AND xid = ? FOR UPDATE"
}

func getDeleteUndoLogSql() string {
	return "DELETE FROM " + getUndoLogTableName() + " WHERE branch_id = ? AND xid = ?"
}

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
	log.Infof("begin to insert undo log, xid %v, branch id %v", record.XID, record.BranchID)
	stmt, err := conn.Prepare(getInsertUndoLogSql())
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
	stmt, err := conn.PrepareContext(ctx, getInsertUndoLogSql())
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
	stmt, err := conn.PrepareContext(ctx, getDeleteUndoLogSql())
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

	size := len(beforeImages)
	if size < len(afterImages) {
		size = len(afterImages)
	}

	for i := 0; i < size; i++ {
		var (
			tableName   string
			sqlType     types.SQLType
			beforeImage *types.RecordImage
			afterImage  *types.RecordImage
		)

		if i < len(beforeImages) && beforeImages[i] != nil {
			tableName = beforeImages[i].TableName
			sqlType = beforeImages[i].SQLType
		} else if i < len(afterImages) && afterImages[i] != nil {
			tableName = afterImages[i].TableName
			sqlType = afterImages[i].SQLType
		} else {
			continue
		}

		if i < len(beforeImages) {
			beforeImage = beforeImages[i]
		}
		if i < len(afterImages) {
			afterImage = afterImages[i]
		}

		undoLog := undo.SQLUndoLog{
			SQLType:     sqlType,
			TableName:   tableName,
			BeforeImage: beforeImage,
			AfterImage:  afterImage,
		}
		sqlUndoLogs = append(sqlUndoLogs, undoLog)
	}

	branchUndoLog := undo.BranchUndoLog{
		Xid:      tranCtx.XID,
		BranchID: tranCtx.BranchID,
		Logs:     sqlUndoLogs,
	}

	parseContext := make(map[string]string, 0)
	parseContext[serializerKey] = "json"
	parseContext[compressorTypeKey] = undo.UndoConfig.CompressConfig.Type
	undoLogContent := m.encodeUndoLogCtx(parseContext)
	rollbackInfo, err := m.serializeBranchUndoLog(&branchUndoLog, parseContext[serializerKey])
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

	stmt, err := conn.PrepareContext(ctx, getSelectUndoLogSql())
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

		var logCtx map[string]string
		if record.Context != nil && string(record.Context) != "" {
			logCtx = m.decodeUndoLogCtx(record.Context)
		}

		if logCtx == nil {
			return fmt.Errorf("undo log context not exist in record %+v", record)
		}

		rollbackInfo, err := m.getRollbackInfo(record.RollbackInfo, logCtx)
		if err != nil {
			return err
		}

		var branchUndoLog *undo.BranchUndoLog
		if branchUndoLog, err = m.deserializeBranchUndoLog(rollbackInfo, logCtx); err != nil {
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

			undoLog.SetTableMeta(tableMeta)

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
	parseContext[serializerKey] = "json"
	parseContext[compressorTypeKey] = undo.UndoConfig.CompressConfig.Type
	undoLogContent := m.encodeUndoLogCtx(parseContext)

	logParse, err := parser.GetCache().Load(parseContext[serializerKey])
	if err != nil {
		return err
	}

	rbInfo := logParse.GetDefaultContent()

	record := undo.UndologRecord{
		BranchID:     branchID,
		XID:          xid,
		RollbackInfo: rbInfo,
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
	if _, err = conn.QueryContext(ctx, getCheckUndoLogTableExistSql()); err != nil {
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
		return "", fmt.Errorf("xid or branch_id can't nil")
	}

	var undoLogDeleteSql strings.Builder
	undoLogDeleteSql.WriteString(" DELETE FROM ")
	undoLogDeleteSql.WriteString(getUndoLogTableName())
	undoLogDeleteSql.WriteString(" WHERE branch_id IN ")
	m.appendInParam(len(branchID), &undoLogDeleteSql)
	undoLogDeleteSql.WriteString(" AND xid IN ")
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
		return "", fmt.Errorf("param type is fault")
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

func (m *BaseUndoLogManager) UnmarshalContext(undoContext []byte) (map[string]string, error) {
	res := make(map[string]string)

	if err := json.Unmarshal(undoContext, &res); err != nil {
		return nil, err
	}

	return res, nil
}

// getRollbackInfo parser rollback info
func (m *BaseUndoLogManager) getRollbackInfo(rollbackInfo []byte, undoContext map[string]string) ([]byte, error) {
	var err error
	res := rollbackInfo
	// get compress type
	if v, ok := undoContext[compressorTypeKey]; ok {
		res, err = compressor.CompressorType(v).GetCompressor().Decompress(rollbackInfo)
		if err != nil {
			log.Errorf("[getRollbackInfo] decompress fail, err: %+v", err)
			return nil, err
		}
	}

	return res, nil
}

// getSerializer get serializer from undo context
func (m *BaseUndoLogManager) getSerializer(undoLogContext map[string]string) (serializer string) {
	if undoLogContext == nil {
		return
	}
	serializer, _ = undoLogContext[serializerKey]
	return
}

func (m *BaseUndoLogManager) deserializeBranchUndoLog(rbInfo []byte, logCtx map[string]string) (*undo.BranchUndoLog, error) {
	var (
		err       error
		logParser parser.UndoLogParser
	)

	if serialzerType := m.getSerializer(logCtx); serialzerType != "" {
		if logParser, err = parser.GetCache().Load(serialzerType); err != nil {
			return nil, err
		}
	}

	var branchUndoLog *undo.BranchUndoLog
	if branchUndoLog, err = logParser.Decode(rbInfo); err != nil {
		return nil, err
	}

	return branchUndoLog, nil
}

func (m *BaseUndoLogManager) serializeBranchUndoLog(log *undo.BranchUndoLog, serializerType string) ([]byte, error) {
	logParser, err := parser.GetCache().Load(serializerType)
	if err != nil {
		return nil, err
	}

	return logParser.Encode(log)
}

func (m *BaseUndoLogManager) encodeUndoLogCtx(undoLogCtx map[string]string) []byte {
	return collection.EncodeMap(undoLogCtx)
}

func (m *BaseUndoLogManager) decodeUndoLogCtx(undoLogCtx []byte) map[string]string {
	return collection.DecodeMap(undoLogCtx)
}
