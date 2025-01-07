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

package parser

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
)

type SubStateMachineParser struct {
	*AbstractTaskStateParser
}

func NewSubStateMachineParser() *SubStateMachineParser {
	return &SubStateMachineParser{
		NewAbstractTaskStateParser(),
	}
}

func (s SubStateMachineParser) StateType() string {
	return constant.StateTypeSubStateMachine
}

func (s SubStateMachineParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	subStateMachineImpl := state.NewSubStateMachineImpl()

	err := s.ParseTaskAttributes(stateName, subStateMachineImpl.AbstractTaskState, stateMap)
	if err != nil {
		return nil, err
	}

	stateMachineName, err := s.BaseStateParser.GetString(stateName, stateMap, "StateMachineName")
	if err != nil {
		return nil, err
	}
	subStateMachineImpl.SetName(stateMachineName)

	if subStateMachineImpl.CompensateState() == "" {
		// build default SubStateMachine compensate state
		compensateSubStateMachineStateParser := NewCompensateSubStateMachineStateParser()
		compensateState, err := compensateSubStateMachineStateParser.Parse(stateName, nil)
		if err != nil {
			return nil, err
		}
		compensateStateImpl, ok := compensateState.(state.TaskState)
		if !ok {
			return nil, errors.New(fmt.Sprintf("State [name:%s] has wrong compensateState type", stateName))
		}
		subStateMachineImpl.SetCompensateStateImpl(compensateStateImpl)
		subStateMachineImpl.SetCompensateState(compensateStateImpl.Name())
	}
	return subStateMachineImpl, nil
}

type CompensateSubStateMachineStateParser struct {
	*AbstractTaskStateParser
}

func NewCompensateSubStateMachineStateParser() *CompensateSubStateMachineStateParser {
	return &CompensateSubStateMachineStateParser{
		NewAbstractTaskStateParser(),
	}
}

func (c CompensateSubStateMachineStateParser) StateType() string {
	return constant.StateTypeCompensateSubMachine
}

func (c CompensateSubStateMachineStateParser) Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error) {
	compensateSubStateMachineStateImpl := state.NewCompensateSubStateMachineStateImpl()
	compensateSubStateMachineStateImpl.SetForCompensation(true)

	if stateMap != nil {
		err := c.ParseTaskAttributes(stateName, compensateSubStateMachineStateImpl.ServiceTaskStateImpl.AbstractTaskState, stateMap)
		if err != nil {
			return nil, err
		}
	}
	if compensateSubStateMachineStateImpl.Name() == "" {
		compensateSubStateMachineStateImpl.SetName(constant.CompensateSubMachineStateNamePrefix + compensateSubStateMachineStateImpl.Hashcode())
	}
	return compensateSubStateMachineStateImpl, nil
}
