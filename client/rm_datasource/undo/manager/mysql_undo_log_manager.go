package manager

import (
	"database/sql"
	"fmt"
	"github.com/dk-lockdown/seata-golang/client/rm_datasource/sql/struct/cache"
	"github.com/dk-lockdown/seata-golang/client/rm_datasource/undo"
	parser2 "github.com/dk-lockdown/seata-golang/client/rm_datasource/undo/parser"
	"github.com/pkg/errors"
	"strings"
	"time"
)

import (
	"github.com/dk-lockdown/seata-golang/client/rm_datasource/tx"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
)

const (
	DeleteUndoLogSql = "DELETE FROM undo_log WHERE xid = ? and branch_id = ?"
	DeleteUndoLogByCreateSql = "DELETE FROM undo_log WHERE log_created <= ? LIMIT ?"
	InsertUndoLogSql = `INSERT INTO undo_log (branch_id, xid, context, rollback_info, log_status, log_created, 
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

func (manager MysqlUndoLogManager) FlushUndoLogs(tx *tx.ProxyTx) error {
	defer func() {
		if err := recover(); err != nil {
			logging.Logger.Error(err)
		}
	}()
	ctx := tx.Context
	xid := ctx.Xid
	branchId := ctx.BranchId

	branchUndoLog := &undo.BranchUndoLog{
		Xid: xid,
		BranchId:branchId,
		SqlUndoLogs:ctx.SqlUndoItemsBuffer,
	}

	parser := parser2.GetUndoLogParser()
	undoLogContent := parser.Encode(branchUndoLog)
	logging.Logger.Debugf("Flushing UNDO LOG: %s",string(undoLogContent))

	return manager.insertUndoLogWithNormal(tx.Tx,xid,branchId,buildContext(parser.GetName()),undoLogContent)
}

func (manager MysqlUndoLogManager) Undo(db *sql.DB,xid string,branchId int64,resourceId string) error {
	tx,err := db.Begin()
	if err != nil {
		return err
	}

	rows,err := tx.Query(SelectUndoLogSql,branchId,xid)
	if err != nil {
		return err
	}

	exists := false
	for rows.Next() {
		exists = true

		var branchId int64
		var xid,context string
		var rollbackInfo []byte
		var state int32

		rows.Scan(&branchId,&xid,&context,&rollbackInfo,&state)

		if State(state) != Normal {
			logging.Logger.Infof("xid %s branch %d, ignore %s undo_log", xid, branchId, State(state).String())
			return nil
		}

		//serializer := getSerializer(context)
		parser := parser2.GetUndoLogParser()
		branchUndoLog := parser.Decode(rollbackInfo)

		sqlUndoLogs := branchUndoLog.SqlUndoLogs
		for _,sqlUndoLog := range sqlUndoLogs {
			tableMeta,err := cache.GetTableMetaCache().GetTableMeta(tx,sqlUndoLog.TableName,resourceId)
			if err != nil {
				tx.Rollback()
				return errors.WithStack(err)
			}

			sqlUndoLog.SetTableMeta(tableMeta)
			NewMysqlUndoExecutor(*sqlUndoLog).Execute(tx)
		}
	}

	if exists {
		_,err := tx.Exec(DeleteUndoLogSql,xid,branchId)
		if err != nil {
			tx.Rollback()
			return errors.WithStack(err)
		}
		logging.Logger.Infof("xid %s branch %d, undo_log deleted with %s", xid, branchId,
			GlobalFinished.String())
		tx.Commit()
	} else {
		manager.insertUndoLogWithGlobalFinished(tx,xid,branchId,
			buildContext(parser2.GetUndoLogParser().GetName()),parser2.GetUndoLogParser().GetDefaultContent())
		tx.Commit()
	}
	return nil
}

func (manager MysqlUndoLogManager) DeleteUndoLog(db *sql.DB,xid string,branchId int64) error {
	result,err := db.Exec(DeleteUndoLogSql,xid,branchId)
	if err != nil {
		return err
	}
	logging.Logger.Infof("%d undo log deleted by xid:%s and branchId:%d",result.RowsAffected(),xid,branchId)
	return nil
}

func (manager MysqlUndoLogManager) BatchDeleteUndoLog(db *sql.DB,xids []string,branchIds []int64) error {
	if xids == nil || branchIds == nil || len(xids) == 0 || len(branchIds) == 0 {
		return nil
	}
	xidSize := len(xids)
	branchIdSize := len(branchIds)
	batchDeleteSql := toBatchDeleteUndoLogSql(xidSize,branchIdSize)
	var args = make([]interface{},0,xidSize+branchIdSize)
	for _,xid := range xids {
		args = append(args,xid)
	}
	for _,branchId := range branchIds {
		args = append(args,branchId)
	}
	result,err := db.Exec(batchDeleteSql,args...)
	if err != nil {
		return err
	}
	logging.Logger.Infof("%d undo log deleted by xids:%v and branchIds:%v",result.RowsAffected(),xids,branchIds)
	return nil
}

func (manager MysqlUndoLogManager) DeleteUndoLogByLogCreated(db *sql.DB,logCreated time.Time,limitRows int) (sql.Result,error) {
	result,err := db.Exec(DeleteUndoLogByCreateSql,logCreated,limitRows)
	return result,err
}

func toBatchDeleteUndoLogSql(xidSize int,branchIdSize int) string {
	var sb strings.Builder
	fmt.Fprint(&sb,"DELETE FROM undo_log WHERE xid in ")
	fmt.Fprint(&sb,appendInParam(xidSize))
	fmt.Fprint(&sb," AND branch_id in ")
	fmt.Fprint(&sb,appendInParam(branchIdSize))
	return sb.String()
}

func appendInParam(size int) string {
	var sb strings.Builder
	fmt.Fprintf(&sb,"(")
	for i :=0; i < size; i++ {
		fmt.Fprintf(&sb,"?")
		if i < size -1 {
			fmt.Fprint(&sb,",")
		}
	}
	fmt.Fprintf(&sb,")")
	return sb.String()
}

func (manager MysqlUndoLogManager) insertUndoLogWithNormal(tx *sql.Tx,xid string,branchId int64,
	rollbackCtx string,undoLogContent []byte) error {
	return manager.insertUndoLog(tx,xid,branchId,rollbackCtx,undoLogContent,Normal)
}

func (manager MysqlUndoLogManager) insertUndoLogWithGlobalFinished(tx *sql.Tx,xid string,branchId int64,
	rollbackCtx string,undoLogContent []byte) error {
	return manager.insertUndoLog(tx,xid,branchId,rollbackCtx,undoLogContent,GlobalFinished)
}

func (manager MysqlUndoLogManager) insertUndoLog(tx *sql.Tx,xid string,branchId int64,
	rollbackCtx string,undoLogContent []byte,state State) error {
	_,err := tx.Exec(InsertUndoLogSql,branchId,xid,rollbackCtx,undoLogContent,state)
	return err
}

func buildContext(serializer string) string {
	return fmt.Sprintf("serializer=%s",serializer)
}

func getSerializer(context string) string {
	return context[10:]
}