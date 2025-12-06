/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package process_ctrl

import (
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

type ProcessContextImpl struct {
	parent      ProcessContext
	mu          sync.RWMutex
	mp          map[string]interface{}
	instruction Instruction
}

func NewProcessContextImpl() *ProcessContextImpl {
	return &ProcessContextImpl{
		mp: make(map[string]interface{}),
	}
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

func (p *ProcessContextImpl) GetInstruction() Instruction {
	return p.instruction
}

func (p *ProcessContextImpl) SetInstruction(instruction Instruction) {
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
