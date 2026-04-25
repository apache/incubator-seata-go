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

import "time"

type StateInstance struct {
	ID                     string
	MachineInstanceID      string
	Name                   string
	TypeName               string
	ServiceName            string
	ServiceMethod          string
	ServiceType            string
	BusinessKey            string
	StartedTime            time.Time
	UpdatedTime            time.Time
	EndTime                time.Time
	IsForUpdate            bool
	Err                    error
	SerializedErr          interface{}
	InputParams            interface{}
	SerializedInputParams  interface{}
	OutputParams           interface{}
	SerializedOutputParams interface{}
	Status                 ExecutionStatus
	StateIDCompensatedFor  string
	StateIDRetriedFor      string
	CompensationState      *StateInstance
	StateMachineInstance   *StateMachineInstance
	IgnoreStatus           bool
}

func NewStateInstance() *StateInstance {
	return &StateInstance{}
}

func (s *StateInstance) IsForCompensation() bool {
	// A state instance is considered a compensation execution if it points
	// back to an original forward state via StateIDCompensatedFor.
	// When this field is non-empty, this instance is a compensation.
	return s.StateIDCompensatedFor != ""
}

func (s *StateInstance) CompensationStatus() ExecutionStatus {
	if s.CompensationState != nil {
		return s.CompensationState.Status
	}

	//return nil ExecutionStatus
	var status ExecutionStatus
	return status
}
