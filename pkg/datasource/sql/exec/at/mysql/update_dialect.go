package mysql

import (
	"context"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/internal"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type UpdateExecutor struct {
	internal.UpdateExecutor
}

// NewUpdateExecutor get update Executor
func NewUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) *UpdateExecutor {
	return &UpdateExecutor{
		UpdateExecutor: *internal.NewUpdateExecutor(parserCtx, execContent, hooks),
	}
}

func (u *UpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	return u.UpdateExecutor.ExecContext(ctx, f)
}
