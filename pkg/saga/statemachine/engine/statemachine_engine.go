package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineEngine interface {
	start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error)
}
