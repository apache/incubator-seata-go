package engine

import (
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

	GetInstruction() Instruction

	SetInstruction(instruction Instruction)
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

type SafeMap struct {
	mu sync.RWMutex
	mp map[string]interface{}
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		mp: make(map[string]interface{}),
	}
}

func (sm *SafeMap) Set(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.mp[key] = value
}

func (sm *SafeMap) Get(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	value, exists := sm.mp[key]
	return value, exists
}

func (sm *SafeMap) Delete(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.mp, key)
}

func (sm *SafeMap) Remove(key string) interface{} {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	value, _ := sm.mp[key]
	delete(sm.mp, key)
	return value
}

func (sm *SafeMap) Range(f func(key string, value interface{}) bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for k, v := range sm.mp {
		if !f(k, v) {
			break
		}
	}
}

func (sm *SafeMap) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for k, _ := range sm.mp {
		delete(sm.mp, k)
	}
}

type ProcessContextImpl struct {
	parent      ProcessContext
	safaMap     SafeMap
	instruction Instruction
}

func (p *ProcessContextImpl) GetVariable(name string) interface{} {
	value, ok := p.safaMap.Get(name)
	if ok {
		return value
	}

	if p.parent != nil {
		return p.parent.GetVariable(name)
	}

	return nil
}

func (p *ProcessContextImpl) SetVariable(name string, value interface{}) {
	_, ok := p.safaMap.Get(name)
	if ok {
		p.safaMap.Set(name, value)
	} else {
		if p.parent != nil {
			p.parent.SetVariable(name, value)
		} else {
			p.safaMap.Set(name, value)
		}
	}
}

func (p *ProcessContextImpl) GetVariables() map[string]interface{} {
	newVariablesMap := make(map[string]interface{})
	if p.parent != nil {
		variables := p.parent.GetVariables()
		for k, v := range variables {
			newVariablesMap[k] = v
		}
	}

	p.safaMap.Range(func(key string, value interface{}) bool {
		newVariablesMap[key] = value
		return true
	})
	return newVariablesMap
}

func (p *ProcessContextImpl) SetVariables(variables map[string]interface{}) {
	for k, v := range variables {
		p.SetVariable(k, v)
	}
}

func (p *ProcessContextImpl) RemoveVariable(name string) interface{} {
	value, ok := p.safaMap.Get(name)
	if ok {
		p.safaMap.Delete(name)
		return value
	}

	if p.parent != nil {
		return p.parent.RemoveVariable(name)
	}

	return nil
}

func (p *ProcessContextImpl) HasVariable(name string) bool {
	_, ok := p.safaMap.Get(name)
	if ok {
		return true
	}

	if p.parent != nil {
		return p.parent.HasVariable(name)
	}

	return false
}

func (p *ProcessContextImpl) GetInstruction() Instruction {
	return p.instruction
}

func (p *ProcessContextImpl) SetInstruction(instruction Instruction) {
	p.instruction = instruction
}

func (p *ProcessContextImpl) GetVariableLocally(name string) interface{} {
	value, _ := p.safaMap.Get(name)
	return value
}

func (p *ProcessContextImpl) SetVariableLocally(name string, value interface{}) {
	p.safaMap.Set(name, value)
}

func (p *ProcessContextImpl) GetVariablesLocally() map[string]interface{} {
	newVariablesMap := make(map[string]interface{})
	p.safaMap.Range(func(key string, value interface{}) bool {
		newVariablesMap[key] = value
		return true
	})
	return newVariablesMap
}

func (p *ProcessContextImpl) SetVariablesLocally(variables map[string]interface{}) {
	for k, v := range variables {
		p.SetVariableLocally(k, v)
	}
}

func (p *ProcessContextImpl) RemoveVariableLocally(name string) interface{} {
	return p.safaMap.Remove(name)
}

func (p *ProcessContextImpl) HasVariableLocally(name string) bool {
	_, ok := p.safaMap.Get(name)
	return ok
}

func (p *ProcessContextImpl) ClearLocally() {
	p.safaMap.Clear()
}

//ProcessContextBuilder process builder
type ProcessContextBuilder struct {
	processContext ProcessContext
}

func NewProcessContextBuilder() *ProcessContextBuilder {
	processContextImpl := &ProcessContextImpl{}
	return &ProcessContextBuilder{processContextImpl}
}

func (p *ProcessContextBuilder) WithProcessType(processType ProcessType) *ProcessContextBuilder {
	p.processContext.SetVariable(VarNameProcessType, processType)
	return p
}

func (p *ProcessContextBuilder) WithOperationName(operationName string) *ProcessContextBuilder {
	p.processContext.SetVariable(VarNameOperationName, operationName)
	return p
}

func (p *ProcessContextBuilder) WithAsyncCallback(callBack CallBack) *ProcessContextBuilder {
	if callBack != nil {
		p.processContext.SetVariable(VarNameAsyncCallback, callBack)
	}

	return p
}

func (p *ProcessContextBuilder) WithInstruction(instruction Instruction) *ProcessContextBuilder {
	if instruction != nil {
		p.processContext.SetInstruction(instruction)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineInstance(stateMachineInstance statelang.StateMachineInstance) *ProcessContextBuilder {
	if stateMachineInstance != nil {
		p.processContext.SetVariable(VarNameStateMachineInst, stateMachineInstance)
		p.processContext.SetVariable(VarNameStateMachine, stateMachineInstance.StateMachine())
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineEngine(stateMachineEngine StateMachineEngine) *ProcessContextBuilder {
	if stateMachineEngine != nil {
		p.processContext.SetVariable(VarNameStateMachineEngine, stateMachineEngine)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineConfig(stateMachineConfig StateMachineConfig) *ProcessContextBuilder {
	if stateMachineConfig != nil {
		p.processContext.SetVariable(VarNameStateMachineConfig, stateMachineConfig)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineContextVariables() *ProcessContextBuilder {

	return p
}

func (p *ProcessContextBuilder) WithIsAsyncExecution(async bool) *ProcessContextBuilder {
	p.processContext.SetVariable(VarNameIsAsyncExecution, async)

	return p
}

func (p *ProcessContextBuilder) Build() ProcessContext {
	return p.processContext
}
