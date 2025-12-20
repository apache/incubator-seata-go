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

package utils

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl/process"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

// ProcessContextBuilder process_ctrl builder
type ProcessContextBuilder struct {
	processContext process_ctrl.ProcessContext
}

func NewProcessContextBuilder() *ProcessContextBuilder {
	processContextImpl := process_ctrl.NewProcessContextImpl()
	return &ProcessContextBuilder{processContextImpl}
}

func (p *ProcessContextBuilder) WithProcessType(processType process.ProcessType) *ProcessContextBuilder {
	p.processContext.SetVariable(constant.VarNameProcessType, processType)
	return p
}

func (p *ProcessContextBuilder) WithOperationName(operationName string) *ProcessContextBuilder {
	p.processContext.SetVariable(constant.VarNameOperationName, operationName)
	return p
}

func (p *ProcessContextBuilder) WithAsyncCallback(callBack engine.CallBack) *ProcessContextBuilder {
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

func (p *ProcessContextBuilder) WithStateMachineEngine(stateMachineEngine engine.StateMachineEngine) *ProcessContextBuilder {
	if stateMachineEngine != nil {
		p.processContext.SetVariable(constant.VarNameStateMachineEngine, stateMachineEngine)
	}

	return p
}

func (p *ProcessContextBuilder) WithStateMachineConfig(stateMachineConfig engine.StateMachineConfig) *ProcessContextBuilder {
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

func (p *ProcessContextBuilder) WithStateInstance(state statelang.StateInstance) *ProcessContextBuilder {
	if state != nil {
		p.processContext.SetVariable(constant.VarNameStateInst, state)
	}

	return p
}

func (p *ProcessContextBuilder) Build() process_ctrl.ProcessContext {
	return p.processContext
}
