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

package core

import (
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"sync"
)

const (
	DefaultTransOperTimeout     = 60000 * 30
	DefaultServiceInvokeTimeout = 60000 * 5
)

type DefaultStateMachineConfig struct {
	// Configuration
	transOperationTimeout int
	serviceInvokeTimeout  int
	charset               string
	defaultTenantId       string

	// Components

	// Event publisher
	syncProcessCtrlEventPublisher  EventPublisher
	asyncProcessCtrlEventPublisher EventPublisher

	// Store related components
	stateLogRepository     StateLogRepository
	stateLogStore          StateLogStore
	stateLangStore         StateLangStore
	stateMachineRepository StateMachineRepository

	// Expression related components
	expressionFactoryManager expr.ExpressionFactoryManager
	expressionResolver       expr.ExpressionResolver

	// Invoker related components
	serviceInvokerManager invoker.ServiceInvokerManager
	scriptInvokerManager  invoker.ScriptInvokerManager

	// Other components
	statusDecisionStrategy StatusDecisionStrategy
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

func (c *DefaultStateMachineConfig) SetSyncProcessCtrlEventPublisher(syncProcessCtrlEventPublisher EventPublisher) {
	c.syncProcessCtrlEventPublisher = syncProcessCtrlEventPublisher
}

func (c *DefaultStateMachineConfig) SetAsyncProcessCtrlEventPublisher(asyncProcessCtrlEventPublisher EventPublisher) {
	c.asyncProcessCtrlEventPublisher = asyncProcessCtrlEventPublisher
}

func (c *DefaultStateMachineConfig) SetStateLogRepository(stateLogRepository StateLogRepository) {
	c.stateLogRepository = stateLogRepository
}

func (c *DefaultStateMachineConfig) SetStateLogStore(stateLogStore StateLogStore) {
	c.stateLogStore = stateLogStore
}

func (c *DefaultStateMachineConfig) SetStateLangStore(stateLangStore StateLangStore) {
	c.stateLangStore = stateLangStore
}

func (c *DefaultStateMachineConfig) SetStateMachineRepository(stateMachineRepository StateMachineRepository) {
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

func (c *DefaultStateMachineConfig) SetStatusDecisionStrategy(statusDecisionStrategy StatusDecisionStrategy) {
	c.statusDecisionStrategy = statusDecisionStrategy
}

func (c *DefaultStateMachineConfig) SetSeqGenerator(seqGenerator sequence.SeqGenerator) {
	c.seqGenerator = seqGenerator
}

func (c *DefaultStateMachineConfig) StateLogRepository() StateLogRepository {
	return c.stateLogRepository
}

func (c *DefaultStateMachineConfig) StateMachineRepository() StateMachineRepository {
	return c.stateMachineRepository
}

func (c *DefaultStateMachineConfig) StateLogStore() StateLogStore {
	return c.stateLogStore
}

func (c *DefaultStateMachineConfig) StateLangStore() StateLangStore {
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

func (c *DefaultStateMachineConfig) StatusDecisionStrategy() StatusDecisionStrategy {
	return c.statusDecisionStrategy
}

func (c *DefaultStateMachineConfig) EventPublisher() EventPublisher {
	return c.syncProcessCtrlEventPublisher
}

func (c *DefaultStateMachineConfig) AsyncEventPublisher() EventPublisher {
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

func NewDefaultStateMachineConfig() *DefaultStateMachineConfig {
	c := &DefaultStateMachineConfig{
		transOperationTimeout: DefaultTransOperTimeout,
		serviceInvokeTimeout:  DefaultServiceInvokeTimeout,
		charset:               "UTF-8",
		defaultTenantId:       "000001",
		componentLock:         &sync.Mutex{},
	}

	// TODO: init config
	return c
}
