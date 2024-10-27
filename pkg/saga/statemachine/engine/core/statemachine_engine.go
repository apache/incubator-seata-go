package core

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineEngine interface {
	Start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error)
	Compensate(ctx context.Context, stateMachineInstId string, replaceParams map[string]any) (statelang.StateMachineInstance, error)
}

type CallBack interface {
	OnFinished(ctx context.Context, context ProcessContext, stateMachineInstance statelang.StateMachineInstance)
	OnError(ctx context.Context, context ProcessContext, stateMachineInstance statelang.StateMachineInstance, err error)
}
