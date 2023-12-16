package process_ctrl

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/process_ctrl/instruction"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"sync"
)

type ProcessContext interface {
	GetVariable(name string) interface{}

	SetVariable(name string, value interface{})

	GetVariables() map[string]interface{}

	SetVariables(variables map[string]interface{})

	RemoveVariable(name string) interface{}

	HasVariable(name string) bool

	GetInstruction() instruction.Instruction

	SetInstruction(instruction instruction.Instruction)
}

type HierarchicalProcessContext interface {
	ProcessContext

	GetVariableLocally(name string) interface{}

	SetVariableLocally(name string, value interface{})

	GetVariablesLocally() map[string]interface{}

	SetVariablesLocally(variables map[string]interface{})

	RemoveVariableLocally(name string) interface{}

	HasVariableLocally(name string) bool

	ClearLocally()
}

type ProcessContextImpl struct {
	parent      ProcessContext
	mu          sync.RWMutex
	mp          map[string]interface{}
	instruction instruction.Instruction
}

func (p *ProcessContextImpl) GetVariable(name string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	value, ok := p.mp[name]
	if ok {
		return value
	}

	if p.parent != nil {
		return p.parent.GetVariable(name)
	}

	return nil
}

func (p *ProcessContextImpl) SetVariable(name string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, ok := p.mp[name]
	if ok {
		p.mp[name] = value
	} else {
		if p.parent != nil {
			p.parent.SetVariable(name, value)
		} else {
			p.mp[name] = value
		}
	}
}

func (p *ProcessContextImpl) GetVariables() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	newVariablesMap := make(map[string]interface{})
	if p.parent != nil {
		variables := p.parent.GetVariables()
		for k, v := range variables {
			newVariablesMap[k] = v
		}
	}

	for k, v := range p.mp {
		newVariablesMap[k] = v
	}

	return newVariablesMap
}

func (p *ProcessContextImpl) SetVariables(variables map[string]interface{}) {
	for k, v := range variables {
		p.SetVariable(k, v)
	}
}

func (p *ProcessContextImpl) RemoveVariable(name string) interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	value, ok := p.mp[name]
	if ok {
		delete(p.mp, name)
		return value
	}

	if p.parent != nil {
		return p.parent.RemoveVariable(name)
	}

	return nil
}

func (p *ProcessContextImpl) HasVariable(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, ok := p.mp[name]
	if ok {
		return true
	}

	if p.parent != nil {
		return p.parent.HasVariable(name)
	}

	return false
}

func (p *ProcessContextImpl) GetInstruction() instruction.Instruction {
	return p.instruction
}

func (p *ProcessContextImpl) SetInstruction(instruction instruction.Instruction) {
	p.instruction = instruction
}

func (p *ProcessContextImpl) GetVariableLocally(name string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	value, _ := p.mp[name]
	return value
}

func (p *ProcessContextImpl) SetVariableLocally(name string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mp[name] = value
}

func (p *ProcessContextImpl) GetVariablesLocally() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	newVariablesMap := make(map[string]interface{}, len(p.mp))
	for k, v := range p.mp {
		newVariablesMap[k] = v
	}
	return newVariablesMap
}

func (p *ProcessContextImpl) SetVariablesLocally(variables map[string]interface{}) {
	for k, v := range variables {
		p.SetVariableLocally(k, v)
	}
}

func (p *ProcessContextImpl) RemoveVariableLocally(name string) interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	value, _ := p.mp[name]
	delete(p.mp, name)
	return value
}

func (p *ProcessContextImpl) HasVariableLocally(name string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, ok := p.mp[name]
	return ok
}

func (p *ProcessContextImpl) ClearLocally() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mp = map[string]interface{}{}
}

// ProcessContextBuilder process_ctrl builder
type ProcessContextBuilder struct {
	processContext ProcessContext
}

func NewProcessContextBuilder() *ProcessContextBuilder {
	processContextImpl := &ProcessContextImpl{}
	return &ProcessContextBuilder{processContextImpl}
}

func (p *ProcessContextBuilder) WithProcessType(processType ProcessType) *ProcessContextBuilder {
	p.processContext.SetVariable(engine.VarNameProcessType, processType)
	return p
}

func (p *ProcessContextBuilder) WithOperationName(operationName string) *ProcessContextBuilder {
	p.processContext.SetVariable(engine.VarNameOperationName, operationName)
	return p
}

func (p *ProcessContextBuilder) WithAsyncCallback(callBack engine.CallBack) *ProcessContextBuilder {
	if callBack != nil {
		p.processContext.SetVariable(engine.VarNameAsyncCallback, callBack)
	}

	return p
}

func (p *ProcessContextBuilder) WithInstruction(instruction instruction.Instruction) *ProcessContextBuilder {
	if instruction != nil {
		p.processContext.SetInstruction(instruction)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineInstance(stateMachineInstance statelang.StateMachineInstance) *ProcessContextBuilder {
	if stateMachineInstance != nil {
		p.processContext.SetVariable(engine.VarNameStateMachineInst, stateMachineInstance)
		p.processContext.SetVariable(engine.VarNameStateMachine, stateMachineInstance.StateMachine())
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineEngine(stateMachineEngine engine.StateMachineEngine) *ProcessContextBuilder {
	if stateMachineEngine != nil {
		p.processContext.SetVariable(engine.VarNameStateMachineEngine, stateMachineEngine)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineConfig(stateMachineConfig engine.StateMachineConfig) *ProcessContextBuilder {
	if stateMachineConfig != nil {
		p.processContext.SetVariable(engine.VarNameStateMachineConfig, stateMachineConfig)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineContextVariables(contextMap map[string]interface{}) *ProcessContextBuilder {
	if contextMap != nil {
		p.processContext.SetVariable(engine.VarNameStateMachineContext, contextMap)
	}

	return p
}

func (p *ProcessContextBuilder) WithIsAsyncExecution(async bool) *ProcessContextBuilder {
	p.processContext.SetVariable(engine.VarNameIsAsyncExecution, async)

	return p
}

func (p *ProcessContextBuilder) Build() ProcessContext {
	return p.processContext
}
