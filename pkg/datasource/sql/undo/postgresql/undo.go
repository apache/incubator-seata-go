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

package postgresql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"seata.apache.org/seata-go/pkg/compressor"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/base"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/factor"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/parser"
	serr "seata.apache.org/seata-go/pkg/util/errors"
	"seata.apache.org/seata-go/pkg/util/log"
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

// getUndoLogTableName returns the undo log table name
func getUndoLogTableName() string {
	if undo.UndoConfig.LogTable != "" {
		return undo.UndoConfig.LogTable
	}
	return " undo_log "
}

// getInsertUndoLogSql returns PostgreSQL-specific INSERT SQL
func getInsertUndoLogSql() string {
	return "INSERT INTO " + getUndoLogTableName() + "(branch_id,xid,context,rollback_info,log_status,log_created,log_modified) VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
}

// InsertUndoLog inserts undo log record using driver.Conn
func (m *undoLogManager) InsertUndoLog(record undo.UndologRecord, conn driver.Conn) error {
	log.Infof("begin to insert undo log, xid %v, branch id %v", record.XID, record.BranchID)

	// Use ExecerContext if available (preferred for PostgreSQL)
	if execer, ok := conn.(driver.ExecerContext); ok {
		_, err := execer.ExecContext(context.Background(), getInsertUndoLogSql(), []driver.NamedValue{
			{Ordinal: 1, Value: record.BranchID},
			{Ordinal: 2, Value: record.XID},
			{Ordinal: 3, Value: record.Context},
			{Ordinal: 4, Value: record.RollbackInfo},
			{Ordinal: 5, Value: int64(record.LogStatus)},
		})
		if err != nil {
			log.Errorf("insert undo log via ExecerContext failed: %v", err)
			return err
		}
		return nil
	}

	// Fallback to PrepareContext + StmtExecContext
	if prepCtx, ok := conn.(driver.ConnPrepareContext); ok {
		stmt, err := prepCtx.PrepareContext(context.Background(), getInsertUndoLogSql())
		if err != nil {
			log.Errorf("prepare context failed: %v", err)
			return err
		}
		defer stmt.Close()

		if stmtExecCtx, ok := stmt.(driver.StmtExecContext); ok {
			_, err = stmtExecCtx.ExecContext(context.Background(), []driver.NamedValue{
				{Ordinal: 1, Value: record.BranchID},
				{Ordinal: 2, Value: record.XID},
				{Ordinal: 3, Value: record.Context},
				{Ordinal: 4, Value: record.RollbackInfo},
				{Ordinal: 5, Value: int64(record.LogStatus)},
			})
			if err != nil {
				log.Errorf("exec context failed: %v", err)
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("driver.Conn does not support ExecerContext or PrepareContext/StmtExecContext")
}

// InsertUndoLogWithSqlConn inserts undo log record using sql.Conn
func (m *undoLogManager) InsertUndoLogWithSqlConn(ctx context.Context, record undo.UndologRecord, conn *sql.Conn) error {
	stmt, err := conn.PrepareContext(ctx, getInsertUndoLogSql())
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(record.BranchID, record.XID, record.Context, record.RollbackInfo, int64(record.LogStatus))
	if err != nil {
		return err
	}
	return nil
}

// FlushUndoLog flush undo log
func (m *undoLogManager) FlushUndoLog(tranCtx *types.TransactionContext, conn driver.Conn) error {
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
	const (
		serializerKey     = "serializerKey"
		compressorTypeKey = "compressorTypeKey"
	)
	parseContext[serializerKey] = undo.UndoConfig.LogSerialization
	parseContext[compressorTypeKey] = undo.UndoConfig.CompressConfig.Type

	undoLogContent := m.Base.EncodeUndoLogCtx(parseContext)
	rollbackInfo, err := m.Base.SerializeBranchUndoLog(&branchUndoLog, parseContext[serializerKey])
	if err != nil {
		return err
	}

	// Call PostgreSQL's InsertUndoLog method
	return m.InsertUndoLog(undo.UndologRecord{
		BranchID:     tranCtx.BranchID,
		XID:          tranCtx.XID,
		Context:      undoLogContent,
		RollbackInfo: rollbackInfo,
		LogStatus:    undo.UndoLogStatueNormnal,
	}, conn)
}

// getSelectUndoLogSql returns PostgreSQL-specific SELECT SQL
func getSelectUndoLogSql() string {
	return "SELECT branch_id, xid, context, rollback_info, log_status FROM " + getUndoLogTableName() + " WHERE branch_id = $1 AND xid = $2 FOR UPDATE"
}

// getDeleteUndoLogSql returns PostgreSQL-specific DELETE SQL
func getDeleteUndoLogSql() string {
	return "DELETE FROM " + getUndoLogTableName() + " WHERE branch_id = $1 AND xid = $2"
}

// DeleteUndoLog exec delete single undo log operate with PostgreSQL syntax
func (m *undoLogManager) DeleteUndoLog(ctx context.Context, xid string, branchID int64, conn *sql.Conn) error {
	stmt, err := conn.PrepareContext(ctx, getDeleteUndoLogSql())
	if err != nil {
		log.Errorf("[DeleteUndoLog] prepare sql fail, err: %v", err)
		return err
	}
	defer stmt.Close()
	if _, err = stmt.Exec(branchID, xid); err != nil {
		log.Errorf("[DeleteUndoLog] exec delete undo log fail, err: %v", err)
		return err
	}

	return nil
}

// BatchDeleteUndoLog implements PostgreSQL-specific batch deletion
func (m *undoLogManager) BatchDeleteUndoLog(xid []string, branchID []int64, conn *sql.Conn) error {
	if len(xid) == 0 || len(branchID) == 0 {
		return fmt.Errorf("xid or branch_id can't be nil")
	}

	ctx := context.Background()

	// Build PostgreSQL-specific DELETE SQL with dynamic placeholders
	// DELETE FROM undo_log WHERE branch_id IN ($1, $2, ...) AND xid IN (...)
	var deleteSql string
	var args []interface{}

	deleteSql = fmt.Sprintf("DELETE FROM %s WHERE branch_id IN (", getUndoLogTableName())

	// Add branch_id placeholders
	paramIndex := 1
	for i := range branchID {
		if i > 0 {
			deleteSql += ","
		}
		deleteSql += fmt.Sprintf("$%d", paramIndex)
		args = append(args, branchID[i])
		paramIndex++
	}

	deleteSql += ") AND xid IN ("

	// Add xid placeholders
	for i := range xid {
		if i > 0 {
			deleteSql += ","
		}
		deleteSql += fmt.Sprintf("$%d", paramIndex)
		args = append(args, xid[i])
		paramIndex++
	}

	deleteSql += ")"

	stmt, err := conn.PrepareContext(ctx, deleteSql)
	if err != nil {
		log.Errorf("prepare sql fail, err: %v", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, args...)
	if err != nil {
		log.Errorf("exec delete undo log fail, err: %v", err)
		return err
	}

	return nil
}

// RunUndo undo sql
func (m *undoLogManager) RunUndo(ctx context.Context, xid string, branchID int64, db *sql.DB, dbName string) error {
	// Use PostgreSQL-specific Undo implementation
	return m.Undo(ctx, xid, branchID, db, dbName)
}

// Undo implements PostgreSQL-specific undo logic
func (m *undoLogManager) Undo(ctx context.Context, xid string, branchID int64, db *sql.DB, dbName string) (err error) {
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
				log.Errorf("rollback fail, xid: %s, branchID:%d err:%v", xid, branchID, err)
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
			log.Errorf("stmt close fail, xid: %s, branchID:%d err:%v", xid, branchID, err)
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
			log.Errorf("rows close fail, xid: %s, branchID:%d err:%v", xid, branchID, err)
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
	if err := rows.Err(); err != nil {
		log.Errorf("read rows next fail, xid: %s, branchID:%d err:%v", xid, branchID, err)
		return err
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
			var err error
			logCtx, err = m.Base.UnmarshalContext(record.Context)
			if err != nil {
				return err
			}
		}

		if logCtx == nil {
			return fmt.Errorf("undo log context not exist in record %+v", record)
		}

		// Use helper methods to process rollback info
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
			tableMeta, err := datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, dbName, undoLog.TableName)
			if err != nil {
				log.Errorf("get table meta fail, err: %v", err)
				return err
			}

			undoLog.SetTableMeta(tableMeta)

			undoExecutor, err := factor.GetUndoExecutor(types.DBTypePostgreSQL, undoLog)
			if err != nil {
				log.Errorf("get undo executor, err: %v", err)
				return err
			}

			if err = undoExecutor.ExecuteOn(ctx, types.DBTypePostgreSQL, conn); err != nil {
				log.Errorf("execute on fail, err: %v", err)
				if undoErr, ok := err.(*serr.SeataError); ok && undoErr.Code == serr.SQLUndoDirtyError {
					log.Errorf("Branch session rollback failed because of dirty undo log, please delete the relevant undolog after manually calibrating the data. xid = %s branchId = %d: %v", xid, branchID, undoErr)
					return serr.New(serr.TransactionErrorCodeBranchRollbackFailedUnretriable, "dirty undo log, manual cleanup required", nil)
				}
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
		return err
	}
	return nil
}

func (m *undoLogManager) insertUndoLogWithGlobalFinished(ctx context.Context, xid string, branchID uint64, conn *sql.Conn) error {
	// Use Base implementation but with PostgreSQL-specific insert
	parseContext := make(map[string]string, 0)
	const (
		serializerKey     = "serializerKey"
		compressorTypeKey = "compressorTypeKey"
	)
	parseContext[serializerKey] = undo.UndoConfig.LogSerialization
	parseContext[compressorTypeKey] = undo.UndoConfig.CompressConfig.Type
	undoLogContent := m.Base.EncodeUndoLogCtx(parseContext)

	// Get default content
	rbInfo := []byte("{}")

	record := undo.UndologRecord{
		BranchID:     branchID,
		XID:          xid,
		RollbackInfo: rbInfo,
		LogStatus:    1, // UndoLogStatusGlobalFinished
		Context:      undoLogContent,
	}
	err := m.InsertUndoLogWithSqlConn(ctx, record, conn)
	if err != nil {
		log.Errorf("insert undo log fail, err: %v", err)
		return err
	}
	return nil
}

// DBType
func (m *undoLogManager) DBType() types.DBType {
	return types.DBTypePostgreSQL
}

// HasUndoLogTable check undo log table if exist
func (m *undoLogManager) HasUndoLogTable(ctx context.Context, conn *sql.Conn) (bool, error) {
	return m.Base.HasUndoLogTable(ctx, conn)
}

// Helper methods for undo log processing

const (
	serializerKey     = "serializerKey"
	compressorTypeKey = "compressorTypeKey"
)

// getRollbackInfo decompresses rollback info if needed
func (m *undoLogManager) getRollbackInfo(rollbackInfo []byte, undoContext map[string]string) ([]byte, error) {
	var err error
	res := rollbackInfo
	// get compress type
	if v, ok := undoContext[compressorTypeKey]; ok && v != "" {
		res, err = compressor.CompressorType(v).GetCompressor().Decompress(rollbackInfo)
		if err != nil {
			log.Errorf("[getRollbackInfo] decompress fail, err: %+v", err)
			return nil, err
		}
	}

	return res, nil
}

// getSerializer gets serializer from undo context
func (m *undoLogManager) getSerializer(undoLogContext map[string]string) (serializer string) {
	if undoLogContext == nil {
		return
	}
	serializer, _ = undoLogContext[serializerKey]
	return
}

// deserializeBranchUndoLog deserializes branch undo log
func (m *undoLogManager) deserializeBranchUndoLog(rbInfo []byte, logCtx map[string]string) (*undo.BranchUndoLog, error) {
	var (
		err       error
		logParser parser.UndoLogParser
	)

	if serializerType := m.getSerializer(logCtx); serializerType != "" {
		if logParser, err = parser.GetCache().Load(serializerType); err != nil {
			return nil, err
		}
	}

	var branchUndoLog *undo.BranchUndoLog
	if branchUndoLog, err = logParser.Decode(rbInfo); err != nil {
		return nil, err
	}

	return branchUndoLog, nil
}