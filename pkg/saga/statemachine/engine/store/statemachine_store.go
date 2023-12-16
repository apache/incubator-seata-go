package store

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"io"
)

type StateLogRepository interface {
	GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateInstance, error)

	GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error)

	GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error)
}

type StateLogStore interface {
	RecordStateMachineStarted(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error

	RecordStateMachineFinished(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error

	RecordStateMachineRestarted(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error

	RecordStateStarted(ctx context.Context, stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error

	RecordStateFinished(ctx context.Context, stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error

	GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateInstance, error)

	GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error)

	GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error)
}

type StateMachineRepository interface {
	GetStateMachineById(stateMachineId string) (statelang.StateMachine, error)

	GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	RegistryStateMachine(statelang.StateMachine) error

	RegistryStateMachineByReader(reader io.Reader) error
}

type StateLangStore interface {
	GetStateMachineById(stateMachineId string) (statelang.StateMachine, error)

	GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	StoreStateMachine(stateMachine statelang.StateMachine) error
}
