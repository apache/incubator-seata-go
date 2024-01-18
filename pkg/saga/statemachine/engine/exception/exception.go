package exception

import "github.com/seata/seata-go/pkg/util/errors"

type EngineExecutionException struct {
	errors.SeataError
	stateName              string
	stateMachineName       string
	stateMachineInstanceId string
	stateInstanceId        string
}

func NewEngineExecutionException(code errors.TransactionErrorCode, msg string, parent error) *EngineExecutionException {
	seataError := errors.New(code, msg, parent)
	return &EngineExecutionException{
		SeataError: *seataError,
	}
}

func (e *EngineExecutionException) StateName() string {
	return e.stateName
}

func (e *EngineExecutionException) SetStateName(stateName string) {
	e.stateName = stateName
}

func (e *EngineExecutionException) StateMachineName() string {
	return e.stateMachineName
}

func (e *EngineExecutionException) SetStateMachineName(stateMachineName string) {
	e.stateMachineName = stateMachineName
}

func (e *EngineExecutionException) StateMachineInstanceId() string {
	return e.stateMachineInstanceId
}

func (e *EngineExecutionException) SetStateMachineInstanceId(stateMachineInstanceId string) {
	e.stateMachineInstanceId = stateMachineInstanceId
}

func (e *EngineExecutionException) StateInstanceId() string {
	return e.stateInstanceId
}

func (e *EngineExecutionException) SetStateInstanceId(stateInstanceId string) {
	e.stateInstanceId = stateInstanceId
}

type ForwardInvalidException struct {
	EngineExecutionException
}
