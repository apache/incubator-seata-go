package mysql

import (
	"context"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/internal"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type DeleteExecutor struct {
	internal.DeleteExecutor
}

func NewDeleteExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) internal.Executor {
	return &DeleteExecutor{
		DeleteExecutor: internal.DeleteExecutor{
			BaseExecutor: internal.BaseExecutor{Hooks: hooks, ParserCtx: parserCtx, ExecCtx: execContext},
		},
	}
}

func (d *DeleteExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	return d.DeleteExecutor.ExecContext(ctx, f)
}
