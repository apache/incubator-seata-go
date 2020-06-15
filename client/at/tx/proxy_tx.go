package tx

import (
	"database/sql"

	"github.com/xiaobudongzhang/seata-golang/client/at/undo"
)

type ProxyTx struct {
	*sql.Tx
	DSN        string
	ResourceId string
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

func (tx *ProxyTx) GetResourceId() string {
	return tx.ResourceId
}
