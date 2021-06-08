package manager

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/proxy_tx"
	"github.com/transaction-wg/seata-golang/pkg/client/at/sql/schema/cache"
	"github.com/transaction-wg/seata-golang/pkg/client/at/undo"
	parser2 "github.com/transaction-wg/seata-golang/pkg/client/at/undo/parser"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	sql2 "github.com/transaction-wg/seata-golang/pkg/util/sql"
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
			log.Error(err)
		}
	}()
	ctx := tx.Context
	xid := ctx.XID
	branchID := ctx.BranchID

	branchUndoLog := &undo.BranchUndoLog{
		Xid:         xid,
		BranchID:    branchID,
		SqlUndoLogs: ctx.SqlUndoItemsBuffer,
	}

	parser := parser2.GetUndoLogParser()
	undoLogContent := parser.Encode(branchUndoLog)
	log.Debugf("Flushing UNDO LOG: %s", string(undoLogContent))

	return manager.insertUndoLogWithNormal(tx.Tx, xid, branchID, buildContext(parser.GetName()), undoLogContent)
}

func (manager MysqlUndoLogManager) Undo(db *sql.DB, xid string, branchID int64, resourceID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	rows, err := tx.Query(SelectUndoLogSql, branchID, xid)
	if err != nil {
		return err
	}

	exists := false

	undoLogs := make([]*undo.BranchUndoLog, 0)
	for rows.Next() {
		exists = true

		var branchID int64
		var xid, context string
		var rollbackInfo []byte
		var state int32

		rows.Scan(&branchID, &xid, &context, &rollbackInfo, &state)

		if State(state) != Normal {
			log.Infof("xid %s branch %d, ignore %s undo_log", xid, branchID, State(state).String())
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
			tableMeta, err := cache.GetTableMetaCache("mysql").GetTableMeta(tx, sqlUndoLog.TableName, resourceID)
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
		_, err := tx.Exec(DeleteUndoLogSql, xid, branchID)
		if err != nil {
			tx.Rollback()
			return errors.WithStack(err)
		}
		log.Infof("xid %s branch %d, undo_log deleted with %s", xid, branchID,
			GlobalFinished.String())
		tx.Commit()
	} else {
		manager.insertUndoLogWithGlobalFinished(tx, xid, branchID,
			buildContext(parser2.GetUndoLogParser().GetName()), parser2.GetUndoLogParser().GetDefaultContent())
		tx.Commit()
	}
	return nil
}

func (manager MysqlUndoLogManager) DeleteUndoLog(db *sql.DB, xid string, branchID int64) error {
	result, err := db.Exec(DeleteUndoLogSql, xid, branchID)
	if err != nil {
		return err
	}
	affectCount, _ := result.RowsAffected()
	log.Infof("%d undo log deleted by xid:%s and branchID:%d", affectCount, xid, branchID)
	return nil
}

func (manager MysqlUndoLogManager) BatchDeleteUndoLog(db *sql.DB, xids []string, branchIDs []int64) error {
	if xids == nil || branchIDs == nil || len(xids) == 0 || len(branchIDs) == 0 {
		return nil
	}
	xidSize := len(xids)
	branchIDSize := len(branchIDs)
	batchDeleteSql := toBatchDeleteUndoLogSql(xidSize, branchIDSize)
	var args = make([]interface{}, 0, xidSize+branchIDSize)
	for _, xid := range xids {
		args = append(args, xid)
	}
	for _, branchID := range branchIDs {
		args = append(args, branchID)
	}
	result, err := db.Exec(batchDeleteSql, args...)
	if err != nil {
		return err
	}
	affectCount, _ := result.RowsAffected()
	log.Infof("%d undo log deleted by xids:%v and branchIDs:%v", affectCount, xids, branchIDs)
	return nil
}

func (manager MysqlUndoLogManager) DeleteUndoLogByLogCreated(db *sql.DB, logCreated time.Time, limitRows int) (sql.Result, error) {
	result, err := db.Exec(DeleteUndoLogByCreateSql, logCreated, limitRows)
	return result, err
}

func toBatchDeleteUndoLogSql(xidSize int, branchIDSize int) string {
	var sb strings.Builder
	fmt.Fprint(&sb, "DELETE FROM undo_log WHERE xid in ")
	fmt.Fprint(&sb, sql2.AppendInParam(xidSize))
	fmt.Fprint(&sb, " AND branch_id in ")
	fmt.Fprint(&sb, sql2.AppendInParam(branchIDSize))
	return sb.String()
}

func (manager MysqlUndoLogManager) insertUndoLogWithNormal(tx *sql.Tx, xid string, branchID int64,
	rollbackCtx string, undoLogContent []byte) error {
	return manager.insertUndoLog(tx, xid, branchID, rollbackCtx, undoLogContent, Normal)
}

func (manager MysqlUndoLogManager) insertUndoLogWithGlobalFinished(tx *sql.Tx, xid string, branchID int64,
	rollbackCtx string, undoLogContent []byte) error {
	return manager.insertUndoLog(tx, xid, branchID, rollbackCtx, undoLogContent, GlobalFinished)
}

func (manager MysqlUndoLogManager) insertUndoLog(tx *sql.Tx, xid string, branchID int64,
	rollbackCtx string, undoLogContent []byte, state State) error {
	_, err := tx.Exec(InsertUndoLogSql, branchID, xid, rollbackCtx, undoLogContent, state)
	return err
}

func buildContext(serializer string) string {
	return fmt.Sprintf("serializer=%s", serializer)
}

func getSerializer(context string) string {
	return context[10:]
}
