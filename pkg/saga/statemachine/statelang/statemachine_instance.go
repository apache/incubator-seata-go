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

package statelang

import (
	"sync"
	"time"
)

type ExecutionStatus string

const (
	// RU Running
	RU ExecutionStatus = "RU"
	// SU Succeed
	SU ExecutionStatus = "SU"
	// FA Failed
	FA ExecutionStatus = "FA"
	// UN Unknown
	UN ExecutionStatus = "UN"
	// SK Skipped
	SK ExecutionStatus = "SK"
)

type StateMachineInstance struct {
	mu                    sync.RWMutex
	ID                    string
	MachineID             string
	TenantID              string
	ParentID              string
	StartedTime           time.Time
	EndTime               time.Time
	UpdatedTime           time.Time
	Status                ExecutionStatus
	CompensationStatus    ExecutionStatus
	IsRunning             bool
	BusinessKey           string
	Exception             error
	StartParams           map[string]interface{}
	EndParams             map[string]interface{}
	Context               map[string]interface{}
	StateMachine          StateMachine
	SerializedStartParams interface{}
	SerializedEndParams   interface{}
	SerializedError       interface{}
	stateMap              map[string]*StateInstance
	stateList             []*StateInstance
}

func NewStateMachineInstance() *StateMachineInstance {
	return &StateMachineInstance{
		StartParams: make(map[string]interface{}),
		EndParams:   make(map[string]interface{}),
		stateList:   make([]*StateInstance, 0),
		stateMap:    make(map[string]*StateInstance),
	}
}

func (s *StateMachineInstance) PutState(stateId string, stateInstance *StateInstance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stateInstance.StateMachineInstance = s
	s.stateMap[stateId] = stateInstance
	s.stateList = append(s.stateList, stateInstance)
}

func (s *StateMachineInstance) State(stateId string) *StateInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stateMap[stateId]
}

func (s *StateMachineInstance) StateList() []*StateInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stateList
}

func (s *StateMachineInstance) StateMap() map[string]*StateInstance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stateMap
}

func (s *StateMachineInstance) SetStateMap(stateMap map[string]*StateInstance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stateMap = stateMap
}

func (s *StateMachineInstance) PutContext(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Context[key] = value
}

func (s *StateMachineInstance) SetStateMachineAndID(stateMachine StateMachine) {
	s.StateMachine = stateMachine
	s.MachineID = stateMachine.ID()
}
