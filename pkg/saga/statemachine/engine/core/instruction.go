package core

import (
	"errors"
	"fmt"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

type Instruction interface {
}

type StateInstruction struct {
	stateName        string
	stateMachineName string
	tenantId         string
	end              bool
	temporaryState   statelang.State
}

func NewStateInstruction(stateMachineName string, tenantId string) *StateInstruction {
	return &StateInstruction{stateMachineName: stateMachineName, tenantId: tenantId}
}

func (s *StateInstruction) StateName() string {
	return s.stateName
}

func (s *StateInstruction) SetStateName(stateName string) {
	s.stateName = stateName
}

func (s *StateInstruction) StateMachineName() string {
	return s.stateMachineName
}

func (s *StateInstruction) SetStateMachineName(stateMachineName string) {
	s.stateMachineName = stateMachineName
}

func (s *StateInstruction) TenantId() string {
	return s.tenantId
}

func (s *StateInstruction) SetTenantId(tenantId string) {
	s.tenantId = tenantId
}

func (s *StateInstruction) End() bool {
	return s.end
}

func (s *StateInstruction) SetEnd(end bool) {
	s.end = end
}

func (s *StateInstruction) TemporaryState() statelang.State {
	return s.temporaryState
}

func (s *StateInstruction) SetTemporaryState(temporaryState statelang.State) {
	s.temporaryState = temporaryState
}

func (s *StateInstruction) GetState(context ProcessContext) (statelang.State, error) {
	if s.temporaryState != nil {
		return s.temporaryState, nil
	}

	if s.stateMachineName == "" {
		return nil, errors.New("stateMachineName is required")
	}

	stateMachineConfig, ok := context.GetVariable(constant.VarNameStateMachineConfig).(StateMachineConfig)
	if !ok {
		return nil, errors.New("stateMachineConfig is required in context")
	}
	stateMachine, err := stateMachineConfig.StateMachineRepository().GetLastVersionStateMachine(s.stateMachineName, s.tenantId)
	if err != nil {
		return nil, errors.New("get stateMachine in state machine repository error")
	}
	if stateMachine == nil {
		return nil, errors.New(fmt.Sprintf("stateMachine [%s] is not exist", s.stateMachineName))
	}

	if s.stateName == "" {
		s.stateName = stateMachine.StartState()
	}

	state := stateMachine.States()[s.stateName]
	if state == nil {
		return nil, errors.New(fmt.Sprintf("state [%s] is not exist", s.stateName))
	}

	return state, nil
}
