package store

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type StateLogStore interface {
	RecordStateMachineStarted(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error

	RecordStateMachineFinished(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error

	RecordStateMachineRestarted(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error

	RecordStateStarted(ctx context.Context, stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error

	RecordStateFinished(ctx context.Context, stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error

	GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateMachineInstance, error)

	GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateMachineInstance, error)

	GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error)

	GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error)

	ClearUp(context process_ctrl.ProcessContext)
}

type StateLangStore interface {
	GetStateMachineById(stateMachineId string) (statelang.StateMachine, error)

	GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	StoreStateMachine(stateMachine statelang.StateMachine) error
}
