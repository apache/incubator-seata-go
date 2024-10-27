package state

import (
	"github.com/google/uuid"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
)

type SubStateMachine interface {
	TaskState

	StateMachineName() string

	CompensateStateImpl() TaskState
}

type SubStateMachineImpl struct {
	*ServiceTaskStateImpl
	stateMachineName string
	compensateState  TaskState
}

func NewSubStateMachineImpl() *SubStateMachineImpl {
	return &SubStateMachineImpl{
		ServiceTaskStateImpl: NewServiceTaskStateImpl(),
	}
}

func (s *SubStateMachineImpl) StateMachineName() string {
	return s.stateMachineName
}

func (s *SubStateMachineImpl) SetStateMachineName(stateMachineName string) {
	s.stateMachineName = stateMachineName
}

func (s *SubStateMachineImpl) CompensateStateImpl() TaskState {
	return s.compensateState
}

func (s *SubStateMachineImpl) SetCompensateStateImpl(compensateState TaskState) {
	s.compensateState = compensateState
}

type CompensateSubStateMachineState interface {
	ServiceTaskState
}

type CompensateSubStateMachineStateImpl struct {
	*ServiceTaskStateImpl
	hashcode string
}

func NewCompensateSubStateMachineStateImpl() *CompensateSubStateMachineStateImpl {
	uuid := uuid.New()
	c := &CompensateSubStateMachineStateImpl{
		ServiceTaskStateImpl: NewServiceTaskStateImpl(),
		hashcode:             uuid.String(),
	}
	c.SetType(constant.StateTypeCompensateSubMachine)
	return c
}

func (c *CompensateSubStateMachineStateImpl) Hashcode() string {
	return c.hashcode
}

func (c *CompensateSubStateMachineStateImpl) SetHashcode(hashcode string) {
	c.hashcode = hashcode
}
