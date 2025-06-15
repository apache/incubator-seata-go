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

package pcext

import (
	"errors"
	"fmt"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

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

func (s *StateInstruction) GetState(context process_ctrl.ProcessContext) (statelang.State, error) {
	if s.temporaryState != nil {
		return s.temporaryState, nil
	}

	if s.stateMachineName == "" {
		return nil, errors.New("stateMachineName is required")
	}

	stateMachineConfig, ok := context.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)
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
