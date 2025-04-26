package mysql

import (
	"context"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/internal"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type MultiExecutor struct {
	internal.MultiExecutor
}

// NewMultiExecutor get new multi Executor
func NewMultiExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) *MultiExecutor {
	return &MultiExecutor{
		MultiExecutor: *internal.NewMultiExecutor(parserCtx, execContext, hooks),
	}
}

func (m *MultiExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	return m.MultiExecutor.ExecContext(ctx, f)
}
