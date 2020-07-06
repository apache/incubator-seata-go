package manager

import (
	"database/sql"
	"time"
)

import (
	"github.com/dk-lockdown/seata-golang/client/at/proxy_tx"
)

type UndoLogManager interface {
	FlushUndoLogs(tx *proxy_tx.ProxyTx) error

	Undo(db *sql.DB, xid string, branchId int64, resourceId string) error

	DeleteUndoLog(db *sql.DB, xid string, branchId int64) error

	BatchDeleteUndoLog(db *sql.DB, xids []string, branchIds []int64) error

	DeleteUndoLogByLogCreated(db *sql.DB, logCreated time.Time, limitRows int) (sql.Result, error)
}

var undoLogManager UndoLogManager

func GetUndoLogManager() UndoLogManager {
	return MysqlUndoLogManager{}
}
