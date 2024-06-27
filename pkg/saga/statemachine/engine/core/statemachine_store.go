package core

import (
	"context"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"io"
)

type StateLogRepository interface {
	GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateMachineInstance, error)

	GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateInstance, error)

	GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error)

	GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error)
}

type StateLogStore interface {
	RecordStateMachineStarted(ctx context.Context, machineInstance statelang.StateMachineInstance, context ProcessContext) error

	RecordStateMachineFinished(ctx context.Context, machineInstance statelang.StateMachineInstance, context ProcessContext) error

	RecordStateMachineRestarted(ctx context.Context, machineInstance statelang.StateMachineInstance, context ProcessContext) error

	RecordStateStarted(ctx context.Context, stateInstance statelang.StateInstance, context ProcessContext) error

	RecordStateFinished(ctx context.Context, stateInstance statelang.StateInstance, context ProcessContext) error

	GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateMachineInstance, error)

	GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateMachineInstance, error)

	GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error)

	GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error)

	GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error)
}

type StateMachineRepository interface {
	GetStateMachineById(stateMachineId string) (statelang.StateMachine, error)

	GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	RegistryStateMachine(statelang.StateMachine) error

	RegistryStateMachineByReader(reader io.Reader) error
}

type StateLangStore interface {
	GetStateMachineById(stateMachineId string) (statelang.StateMachine, error)

	GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error)

	StoreStateMachine(stateMachine statelang.StateMachine) error
}
