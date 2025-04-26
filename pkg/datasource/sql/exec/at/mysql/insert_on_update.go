package mysql

import (
	"context"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/internal"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type InsertOnUpdateExecutor struct {
	internal.InsertOnUpdateExecutor
}

func NewInsertOnUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) *InsertOnUpdateExecutor {
	return &InsertOnUpdateExecutor{
		InsertOnUpdateExecutor: internal.InsertOnUpdateExecutor{
			BaseExecutor: internal.BaseExecutor{
				Hooks:     hooks,
				ParserCtx: parserCtx,
				ExecCtx:   execContent,
			},
		},
	}
}

func (i *InsertOnUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	return i.InsertOnUpdateExecutor.ExecContext(ctx, f)
}
