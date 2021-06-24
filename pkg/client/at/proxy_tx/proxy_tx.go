package proxy_tx

import (
	"database/sql"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/client/at/undo"
)

type ProxyTx struct {
	*sql.Tx
	DBType     string
	DSN        string
	ResourceID string
	Context    *TxContext
}

func (tx *ProxyTx) Bind(xid string) {
	tx.Context.Bind(xid)
}

func (tx *ProxyTx) SetGlobalLockRequire(isLock bool) {
	tx.Context.IsGlobalLockRequire = isLock
}

func (tx *ProxyTx) IsGlobalLockRequire() bool {
	return tx.Context.IsGlobalLockRequire
}

func (tx *ProxyTx) AppendUndoLog(undoLog *undo.SqlUndoLog) {
	tx.Context.AppendUndoItem(undoLog)
}

func (tx *ProxyTx) AppendLockKey(lockKey string) {
	tx.Context.AppendLockKey(lockKey)
}

func (tx *ProxyTx) GetResourceID() string {
	return tx.ResourceID
}
