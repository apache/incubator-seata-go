package engine

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type ProcessCtrlStateMachineEngine struct {
	StateMachineConfig StateMachineConfig
}

func (p ProcessCtrlStateMachineEngine) Start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	return p.startInternal(ctx, stateMachineName, tenantId, "", startParams, false, nil)
}

func (p ProcessCtrlStateMachineEngine) startInternal(ctx context.Context, stateMachineName string, tenantId string, businessKey string, startParams map[string]interface{}, async bool, callback CallBack) (statelang.StateMachineInstance, error) {
	defer func() {
		// Cleanup logic here.
	}()

	if tenantId == "" {
		tenantId = p.StateMachineConfig.DefaultTenantId()
	}

	stateMachineInstance := p.createMachineInstance(stateMachineName, tenantId, businessKey, startParams)

	// Build the process context.
	processContextBuilder := NewProcessContextBuilder().
		WithProcessType(StateLang).
		WithOperationName(OperationNameStart).
		WithAsyncCallback(callback).
		WithInstruction(NewStateInstruction(stateMachineName, tenantId)).
		WithStateMachineInstance(stateMachineInstance).
		WithStateMachineConfig(p.StateMachineConfig).
		WithStateMachineEngine(p).
		WithIsAsyncExecution(async)

	contextMap := p.deepCopy(startParams)

	stateMachineInstance.SetContext(contextMap)

	processContext := processContextBuilder.WithStateMachineContextVariables(contextMap).Build()

	if stateMachineInstance.StateMachine().IsPersist() && p.StateMachineConfig.StateLogStore() != nil {

	}

	if stateMachineInstance.ID() == "" {
		stateMachineInstance.SetID(p.StateMachineConfig.SeqGenerator().GenerateId(SeqEntityStateMachineInst, ""))
	}

	if async {
		_, err := p.StateMachineConfig.AsyncEventPublisher().PushEvent(ctx, processContext)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := p.StateMachineConfig.EventPublisher().PushEvent(ctx, processContext)
		if err != nil {
			return nil, err
		}
	}

	return stateMachineInstance, nil
}

func (p ProcessCtrlStateMachineEngine) deepCopy(startParams map[string]interface{}) map[string]interface{} {
	copyMap := make(map[string]interface{}, len(startParams))
	for k, v := range startParams {
		copyMap[k] = v
	}
	return copyMap
}

func (p ProcessCtrlStateMachineEngine) createMachineInstance(name string, id string, businessKey string, params map[string]interface{}) statelang.StateMachineInstance {
	return nil
}

func NewProcessCtrlStateMachineEngine(stateMachineConfig StateMachineConfig) *ProcessCtrlStateMachineEngine {
	return &ProcessCtrlStateMachineEngine{StateMachineConfig: stateMachineConfig}
}
