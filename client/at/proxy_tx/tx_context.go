package proxy_tx

import (
	"fmt"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/model"
	"github.com/dk-lockdown/seata-golang/client/at/undo"
	"github.com/dk-lockdown/seata-golang/client/context"
)

type TxContext struct {
	*context.RootContext
	Xid                 string
	BranchId            int64
	IsGlobalLockRequire bool

	LockKeysBuffer     *model.Set
	SqlUndoItemsBuffer []*undo.SqlUndoLog
}

func NewTxContext(ctx *context.RootContext) *TxContext {
	txContext := &TxContext{
		RootContext:        ctx,
		LockKeysBuffer:     model.NewSet(),
		SqlUndoItemsBuffer: make([]*undo.SqlUndoLog, 0),
	}
	txContext.Xid = ctx.GetXID()
	return txContext
}

func (ctx *TxContext) InGlobalTransaction() bool {
	return ctx.Xid != ""
}

func (ctx *TxContext) AppendLockKey(lockKey string) {
	if ctx.LockKeysBuffer == nil {
		ctx.LockKeysBuffer = model.NewSet()
	}
	ctx.LockKeysBuffer.Add(lockKey)
}

func (ctx *TxContext) AppendUndoItem(sqlUndoLog *undo.SqlUndoLog) {
	if ctx.SqlUndoItemsBuffer == nil {
		ctx.SqlUndoItemsBuffer = make([]*undo.SqlUndoLog, 0)
	}
	ctx.SqlUndoItemsBuffer = append(ctx.SqlUndoItemsBuffer, sqlUndoLog)
}

func (ctx *TxContext) Bind(xid string) {
	if xid == "" {
		panic(errors.New("xid should not be null"))
	}
	if !ctx.InGlobalTransaction() {
		ctx.Xid = xid
	} else {
		if ctx.Xid != xid {
			panic(errors.New("should never happen"))
		}
	}
}

func (ctx *TxContext) HasUndoLog() bool {
	return ctx.LockKeysBuffer != nil &&
		ctx.LockKeysBuffer.Len() > 0
}

func (ctx *TxContext) BuildLockKeys() string {
	if ctx.LockKeysBuffer == nil ||
		ctx.LockKeysBuffer.Len() == 0 {
		return ""
	}
	var sb strings.Builder
	lockKeys := ctx.LockKeysBuffer.List()
	for _, lockKey := range lockKeys {
		fmt.Fprintf(&sb, "%s;", lockKey)
	}
	return sb.String()
}

func (ctx *TxContext) IsBranchRegistered() bool {
	return ctx.BranchId > 0
}

func (ctx *TxContext) Reset() {
	ctx.Xid = ""
}
