package transaction

import (
	"context"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/rm/tcc/api"
)

type ContextVariable struct {
	Xid                   string
	Status                *GlobalStatus
	Role                  *GlobalTransactionRole
	BusinessActionContext *api.BusinessActionContext
}

func InitSeataContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, common.CONTEXT_VARIABLE, &ContextVariable{})
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

func GetTransactionRole(ctx context.Context) *GlobalTransactionRole {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable == nil {
		return nil
	}
	return variable.(*ContextVariable).Role
}

func SetTransactionRole(ctx context.Context, role GlobalTransactionRole) {
	variable := ctx.Value(common.CONTEXT_VARIABLE)
	if variable != nil {
		variable.(*ContextVariable).Role = &role
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
