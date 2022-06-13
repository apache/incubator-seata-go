package seatactx

import (
	"context"
)

import (
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/protocol/transaction"
	"github.com/seata/seata-go/pkg/rm/tcc/api"
)

type ContextVariable struct {
	TxName                string
	Xid                   string
	Status                *transaction.GlobalStatus
	TxRole                *transaction.GlobalTransactionRole
	BusinessActionContext *api.BusinessActionContext
	TxStatus              *transaction.GlobalStatus
}

func InitSeataContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, common.CONTEXT_VARIABLE, &ContextVariable{})
}

func GetTxStatus(ctx context.Context) *transaction.GlobalStatus {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return nil
	}
	return variable.(*ContextVariable).TxStatus
}

func SetTxStatus(ctx context.Context, status transaction.GlobalStatus) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).TxStatus = &status
	}
}

func GetTxName(ctx context.Context) string {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return ""
	}
	return variable.(*ContextVariable).TxName
}

func SetTxName(ctx context.Context, name string) {
	variable := ctx.Value(common.TccBusinessActionContext)
	if variable != nil {
		variable.(*ContextVariable).TxName = name
	}
}

func IsSeataContext(ctx context.Context) bool {
	return ctx.Value(common.CONTEXT_VARIABLE) != nil
}

func GetBusinessActionContext(ctx context.Context) *api.BusinessActionContext {
	variable := ctx.Value(common.TccBusinessActionContext)
	if variable == nil {
		return nil
	}
	return variable.(*api.BusinessActionContext)
}

func SetBusinessActionContext(ctx context.Context, businessActionContext *api.BusinessActionContext) {
	variable := ctx.Value(common.TccBusinessActionContext)
	if variable != nil {
		variable.(*ContextVariable).BusinessActionContext = businessActionContext
	}
}

func GetTransactionRole(ctx context.Context) *transaction.GlobalTransactionRole {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return nil
	}
	return variable.(*ContextVariable).TxRole
}

func SetTransactionRole(ctx context.Context, role transaction.GlobalTransactionRole) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).TxRole = &role
	}
}

func GetXID(ctx context.Context) string {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return ""
	}
	return variable.(*ContextVariable).Xid
}

func HasXID(ctx context.Context) bool {
	return GetXID(ctx) != ""
}

func SetXID(ctx context.Context, xid string) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).Xid = xid
	}
}

func UnbindXid(ctx context.Context) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).Xid = ""
	}
}
