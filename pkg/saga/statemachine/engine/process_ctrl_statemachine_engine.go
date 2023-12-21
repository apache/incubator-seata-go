package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type ProcessCtrlStateMachineEngine struct {
	StateMachineConfig StateMachineConfig
}

func (p ProcessCtrlStateMachineEngine) Start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	defer func() {
		// Cleanup logic here.
	}()

	if tenantId == "" {
		tenantId = p.StateMachineConfig.DefaultTenantId()
	}

	stateMachineInstance := createMachineInstance(stateMachineName, tenantId, "", startParams)

	// Build the process context.
	processContextBuilder := NewProcessContextBuilder().WithProcessType(StateLang).WithOperationName(OperationNameStart).WithAsyncCallback(nil).WithInstruction(NewStateInstruction(stateMachineName, tenantId)).WithStateMachineInstance(stateMachineInstance).WithStateMachineConfig(p.StateMachineConfig).WithStateMachineEngine(p)

	if startParams == nil {

	} else {

	}

	processContextBuilder.Build()

	return stateMachineInstance, nil
}

func createMachineInstance(name string, id string, businessKey string, params map[string]interface{}) statelang.StateMachineInstance {
	return nil
}

func NewProcessCtrlStateMachineEngine(stateMachineConfig StateMachineConfig) *ProcessCtrlStateMachineEngine {
	return &ProcessCtrlStateMachineEngine{StateMachineConfig: stateMachineConfig}
}
