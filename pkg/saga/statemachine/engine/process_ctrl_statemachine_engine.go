package engine

import (
	"context"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/events"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/process_ctrl/instruction"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"time"
)

type ProcessCtrlStateMachineEngine struct {
	StateMachineConfig StateMachineConfig
}

func (p ProcessCtrlStateMachineEngine) Start(ctx context.Context, stateMachineName string, tenantId string, startParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	return p.startInternal(ctx, stateMachineName, tenantId, "", startParams, false, nil)
}

func (p ProcessCtrlStateMachineEngine) startInternal(ctx context.Context, stateMachineName string, tenantId string, businessKey string, startParams map[string]interface{}, async bool, callback CallBack) (statelang.StateMachineInstance, error) {
	if tenantId == "" {
		tenantId = p.StateMachineConfig.DefaultTenantId()
	}

	stateMachineInstance, err := p.createMachineInstance(stateMachineName, tenantId, businessKey, startParams)
	if err != nil {
		return nil, err
	}

	// Build the process_ctrl context.
	processContextBuilder := process_ctrl.NewProcessContextBuilder().
		WithProcessType(process_ctrl.StateLang).
		WithOperationName(OperationNameStart).
		WithAsyncCallback(callback).
		WithInstruction(instruction.NewStateInstruction(stateMachineName, tenantId)).
		WithStateMachineInstance(stateMachineInstance).
		WithStateMachineConfig(p.StateMachineConfig).
		WithStateMachineEngine(p).
		WithIsAsyncExecution(async)

	contextMap := p.copyMap(startParams)

	stateMachineInstance.SetContext(contextMap)

	processContext := processContextBuilder.WithStateMachineContextVariables(contextMap).Build()

	if stateMachineInstance.StateMachine().IsPersist() && p.StateMachineConfig.StateLogStore() != nil {
		err := p.StateMachineConfig.StateLogStore().RecordStateMachineStarted(ctx, stateMachineInstance, processContext)
		if err != nil {
			return nil, err
		}
	}

	if stateMachineInstance.ID() == "" {
		stateMachineInstance.SetID(p.StateMachineConfig.SeqGenerator().GenerateId(SeqEntityStateMachineInst, ""))
	}

	var eventPublisher events.EventPublisher
	if async {
		eventPublisher = p.StateMachineConfig.AsyncEventPublisher()
	} else {
		eventPublisher = p.StateMachineConfig.EventPublisher()
	}

	_, err = eventPublisher.PushEvent(ctx, processContext)
	if err != nil {
		return nil, err
	}

	return stateMachineInstance, nil
}

// copyMap not deep copy, so best practice: Don’t pass by reference
func (p ProcessCtrlStateMachineEngine) copyMap(startParams map[string]interface{}) map[string]interface{} {
	copyMap := make(map[string]interface{}, len(startParams))
	for k, v := range startParams {
		copyMap[k] = v
	}
	return copyMap
}

func (p ProcessCtrlStateMachineEngine) createMachineInstance(stateMachineName string, tenantId string, businessKey string, startParams map[string]interface{}) (statelang.StateMachineInstance, error) {
	stateMachine, err := p.StateMachineConfig.StateMachineRepository().GetLastVersionStateMachine(stateMachineName, tenantId)
	if err != nil {
		return nil, err
	}

	if stateMachine == nil {
		return nil, errors.New("StateMachine [" + stateMachineName + "] is not exists")
	}

	stateMachineInstance := statelang.NewStateMachineInstanceImpl()
	stateMachineInstance.SetStateMachine(stateMachine)
	stateMachineInstance.SetTenantID(tenantId)
	stateMachineInstance.SetBusinessKey(businessKey)
	stateMachineInstance.SetStartParams(startParams)
	if startParams != nil {
		if businessKey != "" {
			startParams[VarNameBusinesskey] = businessKey
		}

		if startParams[VarNameParentId] != nil {
			parentId, ok := startParams[VarNameParentId].(string)
			if !ok {

			}
			stateMachineInstance.SetParentID(parentId)
			delete(startParams, VarNameParentId)
		}
	}

	stateMachineInstance.SetStatus(statelang.RU)
	stateMachineInstance.SetRunning(true)

	now := time.Now()
	stateMachineInstance.SetStartedTime(now)
	stateMachineInstance.SetUpdatedTime(now)
	return stateMachineInstance, nil
}

func NewProcessCtrlStateMachineEngine(stateMachineConfig StateMachineConfig) *ProcessCtrlStateMachineEngine {
	return &ProcessCtrlStateMachineEngine{StateMachineConfig: stateMachineConfig}
}
