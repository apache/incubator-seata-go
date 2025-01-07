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

type StateInstance interface {
	ID() string

	SetID(id string)

	Name() string

	SetName(name string)

	Type() string

	SetType(typeName string)

	ServiceName() string

	SetServiceName(serviceName string)

	ServiceMethod() string

	SetServiceMethod(serviceMethod string)

	ServiceType() string

	SetServiceType(serviceType string)

	BusinessKey() string

	SetBusinessKey(businessKey string)

	StartedTime() time.Time

	SetStartedTime(startedTime time.Time)

	UpdatedTime() time.Time

	SetUpdatedTime(updateTime time.Time)

	EndTime() time.Time

	SetEndTime(endTime time.Time)

	IsForUpdate() bool

	SetForUpdate(forUpdate bool)

	Error() error

	SetError(err error)

	InputParams() interface{}

	SetInputParams(inputParams interface{})

	OutputParams() interface{}

	SetOutputParams(outputParams interface{})

	Status() ExecutionStatus

	SetStatus(status ExecutionStatus)

	StateIDCompensatedFor() string

	SetStateIDCompensatedFor(stateIdCompensatedFor string)

	StateIDRetriedFor() string

	SetStateIDRetriedFor(stateIdRetriedFor string)

	CompensationState() StateInstance

	SetCompensationState(compensationState StateInstance)

	StateMachineInstance() StateMachineInstance

	MachineInstanceID() string

	SetStateMachineInstance(stateMachineInstance StateMachineInstance)

	IsIgnoreStatus() bool

	SetIgnoreStatus(ignoreStatus bool)

	IsForCompensation() bool

	SerializedInputParams() interface{}

	SetSerializedInputParams(serializedInputParams interface{})

	SerializedOutputParams() interface{}

	SetSerializedOutputParams(serializedOutputParams interface{})

	SerializedError() interface{}

	SetSerializedError(serializedErr interface{})

	CompensationStatus() ExecutionStatus
}

type StateInstanceImpl struct {
	id                     string
	machineInstanceId      string
	name                   string
	typeName               string
	serviceName            string
	serviceMethod          string
	serviceType            string
	businessKey            string
	startedTime            time.Time
	updatedTime            time.Time
	endTime                time.Time
	isForUpdate            bool
	err                    error
	serializedErr          interface{}
	inputParams            interface{}
	serializedInputParams  interface{}
	outputParams           interface{}
	serializedOutputParams interface{}
	status                 ExecutionStatus
	stateIdCompensatedFor  string
	stateIdRetriedFor      string
	compensationState      StateInstance
	stateMachineInstance   StateMachineInstance
	ignoreStatus           bool
}

func NewStateInstanceImpl() *StateInstanceImpl {
	return &StateInstanceImpl{}
}

func (s *StateInstanceImpl) ID() string {
	return s.id
}

func (s *StateInstanceImpl) SetID(id string) {
	s.id = id
}

func (s *StateInstanceImpl) MachineInstanceID() string {
	return s.machineInstanceId
}

func (s *StateInstanceImpl) SetMachineInstanceID(machineInstanceId string) {
	s.machineInstanceId = machineInstanceId
}

func (s *StateInstanceImpl) Name() string {
	return s.name
}

func (s *StateInstanceImpl) SetName(name string) {
	s.name = name
}

func (s *StateInstanceImpl) Type() string {
	return s.typeName
}

func (s *StateInstanceImpl) SetType(typeName string) {
	s.typeName = typeName
}

func (s *StateInstanceImpl) ServiceName() string {
	return s.serviceName
}

func (s *StateInstanceImpl) SetServiceName(serviceName string) {
	s.serviceName = serviceName
}

func (s *StateInstanceImpl) ServiceMethod() string {
	return s.serviceMethod
}

func (s *StateInstanceImpl) SetServiceMethod(serviceMethod string) {
	s.serviceMethod = serviceMethod
}

func (s *StateInstanceImpl) ServiceType() string {
	return s.serviceType
}

func (s *StateInstanceImpl) SetServiceType(serviceType string) {
	s.serviceType = serviceType
}

func (s *StateInstanceImpl) BusinessKey() string {
	return s.businessKey
}

func (s *StateInstanceImpl) SetBusinessKey(businessKey string) {
	s.businessKey = businessKey
}

func (s *StateInstanceImpl) StartedTime() time.Time {
	return s.startedTime
}

func (s *StateInstanceImpl) SetStartedTime(startedTime time.Time) {
	s.startedTime = startedTime
}

func (s *StateInstanceImpl) UpdatedTime() time.Time {
	return s.updatedTime
}

func (s *StateInstanceImpl) SetUpdatedTime(updatedTime time.Time) {
	s.updatedTime = updatedTime
}

func (s *StateInstanceImpl) EndTime() time.Time {
	return s.endTime
}

func (s *StateInstanceImpl) SetEndTime(endTime time.Time) {
	s.endTime = endTime
}

func (s *StateInstanceImpl) IsForUpdate() bool {
	return s.isForUpdate
}

func (s *StateInstanceImpl) SetForUpdate(forUpdate bool) {
	s.isForUpdate = forUpdate
}

func (s *StateInstanceImpl) Error() error {
	return s.err
}

func (s *StateInstanceImpl) SetError(err error) {
	s.err = err
}

func (s *StateInstanceImpl) InputParams() interface{} {
	return s.inputParams
}

func (s *StateInstanceImpl) SetInputParams(inputParams interface{}) {
	s.inputParams = inputParams
}

func (s *StateInstanceImpl) OutputParams() interface{} {
	return s.outputParams
}

func (s *StateInstanceImpl) SetOutputParams(outputParams interface{}) {
	s.outputParams = outputParams
}

func (s *StateInstanceImpl) Status() ExecutionStatus {
	return s.status
}

func (s *StateInstanceImpl) SetStatus(status ExecutionStatus) {
	s.status = status
}

func (s *StateInstanceImpl) StateIDCompensatedFor() string {
	return s.stateIdCompensatedFor
}

func (s *StateInstanceImpl) SetStateIDCompensatedFor(stateIdCompensatedFor string) {
	s.stateIdCompensatedFor = stateIdCompensatedFor
}

func (s *StateInstanceImpl) StateIDRetriedFor() string {
	return s.stateIdRetriedFor
}

func (s *StateInstanceImpl) SetStateIDRetriedFor(stateIdRetriedFor string) {
	s.stateIdRetriedFor = stateIdRetriedFor
}

func (s *StateInstanceImpl) CompensationState() StateInstance {
	return s.compensationState
}

func (s *StateInstanceImpl) SetCompensationState(compensationState StateInstance) {
	s.compensationState = compensationState
}

func (s *StateInstanceImpl) StateMachineInstance() StateMachineInstance {
	return s.stateMachineInstance
}

func (s *StateInstanceImpl) SetStateMachineInstance(stateMachineInstance StateMachineInstance) {
	s.stateMachineInstance = stateMachineInstance
}

func (s *StateInstanceImpl) IsIgnoreStatus() bool {
	return s.ignoreStatus
}

func (s *StateInstanceImpl) SetIgnoreStatus(ignoreStatus bool) {
	s.ignoreStatus = ignoreStatus
}

func (s *StateInstanceImpl) IsForCompensation() bool {
	return s.stateIdCompensatedFor == ""
}

func (s *StateInstanceImpl) SerializedInputParams() interface{} {
	return s.serializedInputParams
}

func (s *StateInstanceImpl) SetSerializedInputParams(serializedInputParams interface{}) {
	s.serializedInputParams = serializedInputParams
}

func (s *StateInstanceImpl) SerializedOutputParams() interface{} {
	return s.serializedOutputParams
}

func (s *StateInstanceImpl) SetSerializedOutputParams(serializedOutputParams interface{}) {
	s.serializedOutputParams = serializedOutputParams
}

func (s *StateInstanceImpl) SerializedError() interface{} {
	return s.serializedErr
}

func (s *StateInstanceImpl) SetSerializedError(serializedErr interface{}) {
	s.serializedErr = serializedErr
}

func (s *StateInstanceImpl) CompensationStatus() ExecutionStatus {
	if s.compensationState != nil {
		return s.compensationState.Status()
	}

	//return nil ExecutionStatus
	var status ExecutionStatus
	return status
}
