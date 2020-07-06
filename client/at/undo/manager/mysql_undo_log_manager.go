package manager

import (
	"database/sql"
	"fmt"
	"github.com/dk-lockdown/seata-golang/client/at/sql/schema/cache"
	"strings"
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/proxy_tx"
	"github.com/dk-lockdown/seata-golang/client/at/undo"
	parser2 "github.com/dk-lockdown/seata-golang/client/at/undo/parser"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	sql2 "github.com/dk-lockdown/seata-golang/pkg/sql"
)

const (
	DeleteUndoLogSql         = "DELETE FROM undo_log WHERE xid = ? and branch_id = ?"
	DeleteUndoLogByCreateSql = "DELETE FROM undo_log WHERE log_created <= ? LIMIT ?"
	InsertUndoLogSql         = `INSERT INTO undo_log (branch_id, xid, context, rollback_info, log_status, log_created, 
		log_modified) VALUES (?, ?, ?, ?, ?, now(), now())`
	SelectUndoLogSql = `SELECT branch_id, xid, context, rollback_info, log_status FROM undo_log 
        WHERE branch_id = ? AND xid = ? FOR UPDATE`
)

type State byte

const (
	Normal State = iota
	GlobalFinished
)

func (state State) String() string {
	switch state {
	case Normal:
		return "Normal"
	case GlobalFinished:
		return "GlobalFinished"
	default:
		return fmt.Sprintf("%d", state)
	}
}

type MysqlUndoLogManager struct {
}

func (manager MysqlUndoLogManager) FlushUndoLogs(tx *proxy_tx.ProxyTx) error {
	defer func() {
		if err := recover(); err != nil {
			logging.Logger.Error(err)
		}
	}()
	ctx := tx.Context
	xid := ctx.Xid
	branchId := ctx.BranchId

	branchUndoLog := &undo.BranchUndoLog{
		Xid:         xid,
		BranchId:    branchId,
		SqlUndoLogs: ctx.SqlUndoItemsBuffer,
	}

	parser := parser2.GetUndoLogParser()
	undoLogContent := parser.Encode(branchUndoLog)
	logging.Logger.Debugf("Flushing UNDO LOG: %s", string(undoLogContent))

	return manager.insertUndoLogWithNormal(tx.Tx, xid, branchId, buildContext(parser.GetName()), undoLogContent)
}

func (manager MysqlUndoLogManager) Undo(db *sql.DB, xid string, branchId int64, resourceId string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	rows, err := tx.Query(SelectUndoLogSql, branchId, xid)
	if err != nil {
		return err
	}

	exists := false

	undoLogs := make([]*undo.BranchUndoLog, 0)
	for rows.Next() {
		exists = true

		var branchId int64
		var xid, context string
		var rollbackInfo []byte
		var state int32

		rows.Scan(&branchId, &xid, &context, &rollbackInfo, &state)

		if State(state) != Normal {
			logging.Logger.Infof("xid %s branch %d, ignore %s undo_log", xid, branchId, State(state).String())
			return nil
		}

		//serializer := getSerializer(context)
		parser := parser2.GetUndoLogParser()
		branchUndoLog := parser.Decode(rollbackInfo)
		undoLogs = append(undoLogs, branchUndoLog)
	}
	rows.Close()

	for _, branchUndoLog := range undoLogs {
		sqlUndoLogs := branchUndoLog.SqlUndoLogs
		for _, sqlUndoLog := range sqlUndoLogs {
			tableMeta, err := cache.GetTableMetaCache().GetTableMeta(tx, sqlUndoLog.TableName, resourceId)
			if err != nil {
				tx.Rollback()
				return errors.WithStack(err)
			}

			sqlUndoLog.SetTableMeta(tableMeta)
			err1 := NewMysqlUndoExecutor(*sqlUndoLog).Execute(tx)
			if err1 != nil {
				tx.Rollback()
				return errors.WithStack(err1)
			}
		}
	}

	if exists {
		_, err := tx.Exec(DeleteUndoLogSql, xid, branchId)
		if err != nil {
			tx.Rollback()
			return errors.WithStack(err)
		}
		logging.Logger.Infof("xid %s branch %d, undo_log deleted with %s", xid, branchId,
			GlobalFinished.String())
		tx.Commit()
	} else {
		manager.insertUndoLogWithGlobalFinished(tx, xid, branchId,
			buildContext(parser2.GetUndoLogParser().GetName()), parser2.GetUndoLogParser().GetDefaultContent())
		tx.Commit()
	}
	return nil
}

func (manager MysqlUndoLogManager) DeleteUndoLog(db *sql.DB, xid string, branchId int64) error {
	result, err := db.Exec(DeleteUndoLogSql, xid, branchId)
	if err != nil {
		return err
	}
	affectCount, _ := result.RowsAffected()
	logging.Logger.Infof("%d undo log deleted by xid:%s and branchId:%d", affectCount, xid, branchId)
	return nil
}

func (manager MysqlUndoLogManager) BatchDeleteUndoLog(db *sql.DB, xids []string, branchIds []int64) error {
	if xids == nil || branchIds == nil || len(xids) == 0 || len(branchIds) == 0 {
		return nil
	}
	xidSize := len(xids)
	branchIdSize := len(branchIds)
	batchDeleteSql := toBatchDeleteUndoLogSql(xidSize, branchIdSize)
	var args = make([]interface{}, 0, xidSize+branchIdSize)
	for _, xid := range xids {
		args = append(args, xid)
	}
	for _, branchId := range branchIds {
		args = append(args, branchId)
	}
	result, err := db.Exec(batchDeleteSql, args...)
	if err != nil {
		return err
	}
	affectCount, _ := result.RowsAffected()
	logging.Logger.Infof("%d undo log deleted by xids:%v and branchIds:%v", affectCount, xids, branchIds)
	return nil
}

func (manager MysqlUndoLogManager) DeleteUndoLogByLogCreated(db *sql.DB, logCreated time.Time, limitRows int) (sql.Result, error) {
	result, err := db.Exec(DeleteUndoLogByCreateSql, logCreated, limitRows)
	return result, err
}

func toBatchDeleteUndoLogSql(xidSize int, branchIdSize int) string {
	var sb strings.Builder
	fmt.Fprint(&sb, "DELETE FROM undo_log WHERE xid in ")
	fmt.Fprint(&sb, sql2.AppendInParam(xidSize))
	fmt.Fprint(&sb, " AND branch_id in ")
	fmt.Fprint(&sb, sql2.AppendInParam(branchIdSize))
	return sb.String()
}

func (manager MysqlUndoLogManager) insertUndoLogWithNormal(tx *sql.Tx, xid string, branchId int64,
	rollbackCtx string, undoLogContent []byte) error {
	return manager.insertUndoLog(tx, xid, branchId, rollbackCtx, undoLogContent, Normal)
}

func (manager MysqlUndoLogManager) insertUndoLogWithGlobalFinished(tx *sql.Tx, xid string, branchId int64,
	rollbackCtx string, undoLogContent []byte) error {
	return manager.insertUndoLog(tx, xid, branchId, rollbackCtx, undoLogContent, GlobalFinished)
}

func (manager MysqlUndoLogManager) insertUndoLog(tx *sql.Tx, xid string, branchId int64,
	rollbackCtx string, undoLogContent []byte, state State) error {
	_, err := tx.Exec(InsertUndoLogSql, branchId, xid, rollbackCtx, undoLogContent, state)
	return err
}

func buildContext(serializer string) string {
	return fmt.Sprintf("serializer=%s", serializer)
}

func getSerializer(context string) string {
	return context[10:]
}
