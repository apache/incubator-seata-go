package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineEngine interface {
	Start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error)
}

type CallBack interface {
	OnFinished(context ProcessContext, stateMachineInstance statelang.StateMachineInstance)
	OnError(context ProcessContext, stateMachineInstance statelang.StateMachineInstance, err error)
}
