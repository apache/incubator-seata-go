package repository

import (
	"io"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
)

type StateMachineRepositoryImpl struct {
}

func (s StateMachineRepositoryImpl) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) RegistryStateMachine(machine statelang.StateMachine) error {
	//TODO implement me
	panic("implement me")
}

func (s StateMachineRepositoryImpl) RegistryStateMachineByReader(reader io.Reader) error {
	//TODO implement me
	panic("implement me")
}
