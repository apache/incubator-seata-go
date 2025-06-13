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

package config

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/repo"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/store"
	"sync"
)

const (
	DefaultTransOperTimeout                      = 60000 * 30
	DefaultServiceInvokeTimeout                  = 60000 * 5
	DefaultClientSagaRetryPersistModeUpdate      = false
	DefaultClientSagaCompensatePersistModeUpdate = false
	DefaultClientReportSuccessEnable             = false
	DefaultClientSagaBranchRegisterEnable        = true
)

type DefaultStateMachineConfig struct {
	// Configuration
	transOperationTimeout           int
	serviceInvokeTimeout            int
	charset                         string
	defaultTenantId                 string
	sagaRetryPersistModeUpdate      bool
	sagaCompensatePersistModeUpdate bool
	sagaBranchRegisterEnable        bool
	rmReportSuccessEnable           bool

	// Components

	// Event publisher
	syncProcessCtrlEventPublisher  process_ctrl.EventPublisher
	asyncProcessCtrlEventPublisher process_ctrl.EventPublisher

	// Store related components
	stateLogRepository     repo.StateLogRepository
	stateLogStore          store.StateLogStore
	stateLangStore         store.StateLangStore
	stateMachineRepository repo.StateMachineRepository

	// Expression related components
	expressionFactoryManager expr.ExpressionFactoryManager
	expressionResolver       expr.ExpressionResolver

	// Invoker related components
	serviceInvokerManager invoker.ServiceInvokerManager
	scriptInvokerManager  invoker.ScriptInvokerManager

	// Other components
	statusDecisionStrategy engine.StatusDecisionStrategy
	seqGenerator           sequence.SeqGenerator
	componentLock          *sync.Mutex
}

func (c *DefaultStateMachineConfig) ComponentLock() *sync.Mutex {
	return c.componentLock
}

func (c *DefaultStateMachineConfig) SetComponentLock(componentLock *sync.Mutex) {
	c.componentLock = componentLock
}

func (c *DefaultStateMachineConfig) SetTransOperationTimeout(transOperationTimeout int) {
	c.transOperationTimeout = transOperationTimeout
}

func (c *DefaultStateMachineConfig) SetServiceInvokeTimeout(serviceInvokeTimeout int) {
	c.serviceInvokeTimeout = serviceInvokeTimeout
}

func (c *DefaultStateMachineConfig) SetCharset(charset string) {
	c.charset = charset
}

func (c *DefaultStateMachineConfig) SetDefaultTenantId(defaultTenantId string) {
	c.defaultTenantId = defaultTenantId
}

func (c *DefaultStateMachineConfig) SetSyncProcessCtrlEventPublisher(syncProcessCtrlEventPublisher process_ctrl.EventPublisher) {
	c.syncProcessCtrlEventPublisher = syncProcessCtrlEventPublisher
}

func (c *DefaultStateMachineConfig) SetAsyncProcessCtrlEventPublisher(asyncProcessCtrlEventPublisher process_ctrl.EventPublisher) {
	c.asyncProcessCtrlEventPublisher = asyncProcessCtrlEventPublisher
}

func (c *DefaultStateMachineConfig) SetStateLogRepository(stateLogRepository repo.StateLogRepository) {
	c.stateLogRepository = stateLogRepository
}

func (c *DefaultStateMachineConfig) SetStateLogStore(stateLogStore store.StateLogStore) {
	c.stateLogStore = stateLogStore
}

func (c *DefaultStateMachineConfig) SetStateLangStore(stateLangStore store.StateLangStore) {
	c.stateLangStore = stateLangStore
}

func (c *DefaultStateMachineConfig) SetStateMachineRepository(stateMachineRepository repo.StateMachineRepository) {
	c.stateMachineRepository = stateMachineRepository
}

func (c *DefaultStateMachineConfig) SetExpressionFactoryManager(expressionFactoryManager expr.ExpressionFactoryManager) {
	c.expressionFactoryManager = expressionFactoryManager
}

func (c *DefaultStateMachineConfig) SetExpressionResolver(expressionResolver expr.ExpressionResolver) {
	c.expressionResolver = expressionResolver
}

func (c *DefaultStateMachineConfig) SetServiceInvokerManager(serviceInvokerManager invoker.ServiceInvokerManager) {
	c.serviceInvokerManager = serviceInvokerManager
}

func (c *DefaultStateMachineConfig) SetScriptInvokerManager(scriptInvokerManager invoker.ScriptInvokerManager) {
	c.scriptInvokerManager = scriptInvokerManager
}

func (c *DefaultStateMachineConfig) SetStatusDecisionStrategy(statusDecisionStrategy engine.StatusDecisionStrategy) {
	c.statusDecisionStrategy = statusDecisionStrategy
}

func (c *DefaultStateMachineConfig) SetSeqGenerator(seqGenerator sequence.SeqGenerator) {
	c.seqGenerator = seqGenerator
}

func (c *DefaultStateMachineConfig) StateLogRepository() repo.StateLogRepository {
	return c.stateLogRepository
}

func (c *DefaultStateMachineConfig) StateMachineRepository() repo.StateMachineRepository {
	return c.stateMachineRepository
}

func (c *DefaultStateMachineConfig) StateLogStore() store.StateLogStore {
	return c.stateLogStore
}

func (c *DefaultStateMachineConfig) StateLangStore() store.StateLangStore {
	return c.stateLangStore
}

func (c *DefaultStateMachineConfig) ExpressionFactoryManager() expr.ExpressionFactoryManager {
	return c.expressionFactoryManager
}

func (c *DefaultStateMachineConfig) ExpressionResolver() expr.ExpressionResolver {
	return c.expressionResolver
}

func (c *DefaultStateMachineConfig) SeqGenerator() sequence.SeqGenerator {
	return c.seqGenerator
}

func (c *DefaultStateMachineConfig) StatusDecisionStrategy() engine.StatusDecisionStrategy {
	return c.statusDecisionStrategy
}

func (c *DefaultStateMachineConfig) EventPublisher() process_ctrl.EventPublisher {
	return c.syncProcessCtrlEventPublisher
}

func (c *DefaultStateMachineConfig) AsyncEventPublisher() process_ctrl.EventPublisher {
	return c.asyncProcessCtrlEventPublisher
}

func (c *DefaultStateMachineConfig) ServiceInvokerManager() invoker.ServiceInvokerManager {
	return c.serviceInvokerManager
}

func (c *DefaultStateMachineConfig) ScriptInvokerManager() invoker.ScriptInvokerManager {
	return c.scriptInvokerManager
}

func (c *DefaultStateMachineConfig) CharSet() string {
	return c.charset
}

func (c *DefaultStateMachineConfig) SetCharSet(charset string) {
	c.charset = charset
}

func (c *DefaultStateMachineConfig) DefaultTenantId() string {
	return c.defaultTenantId
}

func (c *DefaultStateMachineConfig) TransOperationTimeout() int {
	return c.transOperationTimeout
}

func (c *DefaultStateMachineConfig) ServiceInvokeTimeout() int {
	return c.serviceInvokeTimeout
}

func (c *DefaultStateMachineConfig) IsSagaRetryPersistModeUpdate() bool {
	return c.sagaRetryPersistModeUpdate
}

func (c *DefaultStateMachineConfig) SetSagaRetryPersistModeUpdate(sagaRetryPersistModeUpdate bool) {
	c.sagaRetryPersistModeUpdate = sagaRetryPersistModeUpdate
}

func (c *DefaultStateMachineConfig) IsSagaCompensatePersistModeUpdate() bool {
	return c.sagaCompensatePersistModeUpdate
}

func (c *DefaultStateMachineConfig) SetSagaCompensatePersistModeUpdate(sagaCompensatePersistModeUpdate bool) {
	c.sagaCompensatePersistModeUpdate = sagaCompensatePersistModeUpdate
}

func (c *DefaultStateMachineConfig) IsSagaBranchRegisterEnable() bool {
	return c.sagaBranchRegisterEnable
}

func (c *DefaultStateMachineConfig) SetSagaBranchRegisterEnable(sagaBranchRegisterEnable bool) {
	c.sagaBranchRegisterEnable = sagaBranchRegisterEnable
}

func (c *DefaultStateMachineConfig) IsRmReportSuccessEnable() bool {
	return c.rmReportSuccessEnable
}

func (c *DefaultStateMachineConfig) SetRmReportSuccessEnable(rmReportSuccessEnable bool) {
	c.rmReportSuccessEnable = rmReportSuccessEnable
}

func NewDefaultStateMachineConfig() *DefaultStateMachineConfig {
	c := &DefaultStateMachineConfig{
		transOperationTimeout:           DefaultTransOperTimeout,
		serviceInvokeTimeout:            DefaultServiceInvokeTimeout,
		charset:                         "UTF-8",
		defaultTenantId:                 "000001",
		sagaRetryPersistModeUpdate:      DefaultClientSagaRetryPersistModeUpdate,
		sagaCompensatePersistModeUpdate: DefaultClientSagaCompensatePersistModeUpdate,
		sagaBranchRegisterEnable:        DefaultClientSagaBranchRegisterEnable,
		rmReportSuccessEnable:           DefaultClientReportSuccessEnable,
		componentLock:                   &sync.Mutex{},
	}

	// TODO: init config
	return c
}
