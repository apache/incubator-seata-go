package manager

import (
	"database/sql"
	"time"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/proxy_tx"
)

type UndoLogManager interface {
	FlushUndoLogs(tx *proxy_tx.ProxyTx) error

	Undo(db *sql.DB, xid string, branchID int64, resourceID string) error

	DeleteUndoLog(db *sql.DB, xid string, branchID int64) error

	BatchDeleteUndoLog(db *sql.DB, xids []string, branchIDs []int64) error

	DeleteUndoLogByLogCreated(db *sql.DB, logCreated time.Time, limitRows int) (sql.Result, error)
}
