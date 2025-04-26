package mysql

import (
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at/internal"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type SelectForUpdateExecutor struct {
	*internal.SelectForUpdateExecutor
}

func NewSelectForUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) *SelectForUpdateExecutor {
	return &SelectForUpdateExecutor{
		SelectForUpdateExecutor: internal.NewSelectForUpdateExecutor(parserCtx, execContext, hooks),
	}
}
