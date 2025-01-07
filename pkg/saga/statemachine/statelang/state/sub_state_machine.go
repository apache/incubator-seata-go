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
