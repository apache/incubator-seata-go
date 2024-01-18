package engine

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

// ProcessContextBuilder process_ctrl builder
type ProcessContextBuilder struct {
	processContext process_ctrl.ProcessContext
}

func NewProcessContextBuilder() *ProcessContextBuilder {
	processContextImpl := &process_ctrl.ProcessContextImpl{}
	return &ProcessContextBuilder{processContextImpl}
}

func (p *ProcessContextBuilder) WithProcessType(processType process_ctrl.ProcessType) *ProcessContextBuilder {
	p.processContext.SetVariable(constant.VarNameProcessType, processType)
	return p
}

func (p *ProcessContextBuilder) WithOperationName(operationName string) *ProcessContextBuilder {
	p.processContext.SetVariable(constant.VarNameOperationName, operationName)
	return p
}

func (p *ProcessContextBuilder) WithAsyncCallback(callBack CallBack) *ProcessContextBuilder {
	if callBack != nil {
		p.processContext.SetVariable(constant.VarNameAsyncCallback, callBack)
	}

	return p
}

func (p *ProcessContextBuilder) WithInstruction(instruction process_ctrl.Instruction) *ProcessContextBuilder {
	if instruction != nil {
		p.processContext.SetInstruction(instruction)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineInstance(stateMachineInstance statelang.StateMachineInstance) *ProcessContextBuilder {
	if stateMachineInstance != nil {
		p.processContext.SetVariable(constant.VarNameStateMachineInst, stateMachineInstance)
		p.processContext.SetVariable(constant.VarNameStateMachine, stateMachineInstance.StateMachine())
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineEngine(stateMachineEngine StateMachineEngine) *ProcessContextBuilder {
	if stateMachineEngine != nil {
		p.processContext.SetVariable(constant.VarNameStateMachineEngine, stateMachineEngine)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineConfig(stateMachineConfig StateMachineConfig) *ProcessContextBuilder {
	if stateMachineConfig != nil {
		p.processContext.SetVariable(constant.VarNameStateMachineConfig, stateMachineConfig)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineContextVariables(contextMap map[string]interface{}) *ProcessContextBuilder {
	if contextMap != nil {
		p.processContext.SetVariable(constant.VarNameStateMachineContext, contextMap)
	}

	return p
}

func (p *ProcessContextBuilder) WithIsAsyncExecution(async bool) *ProcessContextBuilder {
	p.processContext.SetVariable(constant.VarNameIsAsyncExecution, async)

	return p
}

func (p *ProcessContextBuilder) Build() process_ctrl.ProcessContext {
	return p.processContext
}
