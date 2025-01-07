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
	"time"
)

type StateMachineStatus string

const (
	Active   StateMachineStatus = "Active"
	Inactive StateMachineStatus = "Inactive"
)

// RecoverStrategy : Recover Strategy
type RecoverStrategy string

const (
	//Compensate stateMachine
	Compensate RecoverStrategy = "Compensate"
	// Forward  stateMachine
	Forward RecoverStrategy = "Forward"
)

func ValueOfRecoverStrategy(recoverStrategy string) (RecoverStrategy, bool) {
	switch recoverStrategy {
	case "Compensate":
		return Compensate, true
	case "Forward":
		return Forward, true
	default:
		var recoverStrategy RecoverStrategy
		return recoverStrategy, false
	}
}

type StateMachine interface {
	ID() string

	SetID(id string)

	Name() string

	SetName(name string)

	Comment() string

	SetComment(comment string)

	StartState() string

	SetStartState(startState string)

	Version() string

	SetVersion(version string)

	States() map[string]State

	State(stateName string) State

	TenantId() string

	SetTenantId(tenantId string)

	AppName() string

	SetAppName(appName string)

	Type() string

	SetType(typeName string)

	Status() StateMachineStatus

	SetStatus(status StateMachineStatus)

	RecoverStrategy() RecoverStrategy

	SetRecoverStrategy(recoverStrategy RecoverStrategy)

	IsPersist() bool

	SetPersist(persist bool)

	IsRetryPersistModeUpdate() bool

	SetRetryPersistModeUpdate(retryPersistModeUpdate bool)

	IsCompensatePersistModeUpdate() bool

	SetCompensatePersistModeUpdate(compensatePersistModeUpdate bool)

	Content() string

	SetContent(content string)

	CreateTime() time.Time

	SetCreateTime(createTime time.Time)
}

type StateMachineImpl struct {
	id                          string
	tenantId                    string
	appName                     string
	name                        string
	comment                     string
	version                     string
	startState                  string
	status                      StateMachineStatus
	recoverStrategy             RecoverStrategy
	persist                     bool
	retryPersistModeUpdate      bool
	compensatePersistModeUpdate bool
	typeName                    string
	content                     string
	createTime                  time.Time
	states                      map[string]State
}

func NewStateMachineImpl() *StateMachineImpl {
	stateMap := make(map[string]State)
	return &StateMachineImpl{
		appName:  "SEATA",
		status:   Active,
		typeName: "STATE_LANG",
		states:   stateMap,
	}
}

func (s *StateMachineImpl) ID() string {
	return s.id
}

func (s *StateMachineImpl) SetID(id string) {
	s.id = id
}

func (s *StateMachineImpl) Name() string {
	return s.name
}

func (s *StateMachineImpl) SetName(name string) {
	s.name = name
}

func (s *StateMachineImpl) SetComment(comment string) {
	s.comment = comment
}

func (s *StateMachineImpl) Comment() string {
	return s.comment
}

func (s *StateMachineImpl) StartState() string {
	return s.startState
}

func (s *StateMachineImpl) SetStartState(startState string) {
	s.startState = startState
}

func (s *StateMachineImpl) Version() string {
	return s.version
}

func (s *StateMachineImpl) SetVersion(version string) {
	s.version = version
}

func (s *StateMachineImpl) States() map[string]State {
	return s.states
}

func (s *StateMachineImpl) State(stateName string) State {
	if s.states == nil {
		return nil
	}

	return s.states[stateName]
}

func (s *StateMachineImpl) TenantId() string {
	return s.tenantId
}

func (s *StateMachineImpl) SetTenantId(tenantId string) {
	s.tenantId = tenantId
}

func (s *StateMachineImpl) AppName() string {
	return s.appName
}

func (s *StateMachineImpl) SetAppName(appName string) {
	s.appName = appName
}

func (s *StateMachineImpl) Type() string {
	return s.typeName
}

func (s *StateMachineImpl) SetType(typeName string) {
	s.typeName = typeName
}

func (s *StateMachineImpl) Status() StateMachineStatus {
	return s.status
}

func (s *StateMachineImpl) SetStatus(status StateMachineStatus) {
	s.status = status
}

func (s *StateMachineImpl) RecoverStrategy() RecoverStrategy {
	return s.recoverStrategy
}

func (s *StateMachineImpl) SetRecoverStrategy(recoverStrategy RecoverStrategy) {
	s.recoverStrategy = recoverStrategy
}

func (s *StateMachineImpl) IsPersist() bool {
	return s.persist
}

func (s *StateMachineImpl) SetPersist(persist bool) {
	s.persist = persist
}

func (s *StateMachineImpl) IsRetryPersistModeUpdate() bool {
	return s.retryPersistModeUpdate
}

func (s *StateMachineImpl) SetRetryPersistModeUpdate(retryPersistModeUpdate bool) {
	s.retryPersistModeUpdate = retryPersistModeUpdate
}

func (s *StateMachineImpl) IsCompensatePersistModeUpdate() bool {
	return s.compensatePersistModeUpdate
}

func (s *StateMachineImpl) SetCompensatePersistModeUpdate(compensatePersistModeUpdate bool) {
	s.compensatePersistModeUpdate = compensatePersistModeUpdate
}

func (s *StateMachineImpl) Content() string {
	return s.content
}

func (s *StateMachineImpl) SetContent(content string) {
	s.content = content
}

func (s *StateMachineImpl) CreateTime() time.Time {
	return s.createTime
}

func (s *StateMachineImpl) SetCreateTime(createTime time.Time) {
	s.createTime = createTime
}
