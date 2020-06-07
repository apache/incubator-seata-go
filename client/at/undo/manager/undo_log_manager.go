package manager

import (
	"database/sql"
	"time"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/tx"
)

type IUndoLogManager interface {
	FlushUndoLogs(tx *tx.ProxyTx) error

	Undo(db *sql.DB,xid string,branchId int64,resourceId string) error

	DeleteUndoLog(db *sql.DB,xid string,branchId int64) error

	BatchDeleteUndoLog(db *sql.DB,xids []string,branchIds []int64) error

	DeleteUndoLogByLogCreated(db *sql.DB,logCreated time.Time,limitRows int) (sql.Result,error)
}

var undoLogManager IUndoLogManager

func GetUndoLogManager() IUndoLogManager {
	return MysqlUndoLogManager{}
}