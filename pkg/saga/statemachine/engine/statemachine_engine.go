package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineEngine interface {
	Start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error)
}

type CallBack interface {
	OnFinished(ctx context.Context, context process_ctrl.ProcessContext, stateMachineInstance statelang.StateMachineInstance)
	OnError(ctx context.Context, context process_ctrl.ProcessContext, stateMachineInstance statelang.StateMachineInstance, err error)
}
