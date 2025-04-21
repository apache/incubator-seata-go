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
	"encoding/json"
	"fmt"
	"github.com/seata/seata-go/pkg/saga/statemachine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/parser"
	"gopkg.in/yaml.v3"
	"os"
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
	TransOperationTimeout           int      `json:"trans_operation_timeout" yaml:"trans_operation_timeout"`
	ServiceInvokeTimeout            int      `json:"service_invoke_timeout" yaml:"service_invoke_timeout"`
	Charset                         string   `json:"charset" yaml:"charset"`
	DefaultTenantId                 string   `json:"default_tenant_id" yaml:"default_tenant_id"`
	SagaRetryPersistModeUpdate      bool     `json:"saga_retry_persist_mode_update" yaml:"saga_retry_persist_mode_update"`
	SagaCompensatePersistModeUpdate bool     `json:"saga_compensate_persist_mode_update" yaml:"saga_compensate_persist_mode_update"`
	SagaBranchRegisterEnable        bool     `json:"saga_branch_register_enable" yaml:"saga_branch_register_enable"`
	RmReportSuccessEnable           bool     `json:"rm_report_success_enable" yaml:"rm_report_success_enable"`
	StateMachineResources           []string `json:"state_machine_resources" yaml:"state_machine_resources"`

	// State machine definitions
	stateMachineDefs map[string]*statemachine.StateMachineObject

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
	expressionFactoryManager expr.ExpressionFactoryManagerInterface
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
	c.TransOperationTimeout = transOperationTimeout
}

func (c *DefaultStateMachineConfig) SetServiceInvokeTimeout(serviceInvokeTimeout int) {
	c.ServiceInvokeTimeout = serviceInvokeTimeout
}

func (c *DefaultStateMachineConfig) SetCharset(charset string) {
	c.Charset = charset
}

func (c *DefaultStateMachineConfig) SetDefaultTenantId(defaultTenantId string) {
	c.DefaultTenantId = defaultTenantId
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

func (c *DefaultStateMachineConfig) SetExpressionFactoryManager(expressionFactoryManager expr.ExpressionFactoryManagerInterface) {
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

func (c *DefaultStateMachineConfig) ExpressionFactoryManager() expr.ExpressionFactoryManagerInterface {
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
	return c.Charset
}

func (c *DefaultStateMachineConfig) SetCharSet(charset string) {
	c.Charset = charset
}

func (c *DefaultStateMachineConfig) GetDefaultTenantId() string {
	return c.DefaultTenantId
}

func (c *DefaultStateMachineConfig) GetTransOperationTimeout() int {
	return c.TransOperationTimeout
}

func (c *DefaultStateMachineConfig) GetServiceInvokeTimeout() int {
	return c.ServiceInvokeTimeout
}

func (c *DefaultStateMachineConfig) IsSagaRetryPersistModeUpdate() bool {
	return c.SagaRetryPersistModeUpdate
}

func (c *DefaultStateMachineConfig) SetSagaRetryPersistModeUpdate(sagaRetryPersistModeUpdate bool) {
	c.SagaRetryPersistModeUpdate = sagaRetryPersistModeUpdate
}

func (c *DefaultStateMachineConfig) IsSagaCompensatePersistModeUpdate() bool {
	return c.SagaCompensatePersistModeUpdate
}

func (c *DefaultStateMachineConfig) SetSagaCompensatePersistModeUpdate(sagaCompensatePersistModeUpdate bool) {
	c.SagaCompensatePersistModeUpdate = sagaCompensatePersistModeUpdate
}

func (c *DefaultStateMachineConfig) IsSagaBranchRegisterEnable() bool {
	return c.SagaBranchRegisterEnable
}

func (c *DefaultStateMachineConfig) SetSagaBranchRegisterEnable(sagaBranchRegisterEnable bool) {
	c.SagaBranchRegisterEnable = sagaBranchRegisterEnable
}

func (c *DefaultStateMachineConfig) IsRmReportSuccessEnable() bool {
	return c.RmReportSuccessEnable
}

func (c *DefaultStateMachineConfig) SetRmReportSuccessEnable(rmReportSuccessEnable bool) {
	c.RmReportSuccessEnable = rmReportSuccessEnable
}

func (c *DefaultStateMachineConfig) GetStateMachineDefinition(name string) *statemachine.StateMachineObject {
	return c.stateMachineDefs[name]
}

func (c *DefaultStateMachineConfig) GetExpressionFactory(expressionType string) expr.ExpressionFactory {
	return c.expressionFactoryManager.GetExpressionFactory(expressionType)
}

func (c *DefaultStateMachineConfig) GetServiceInvoker(serviceType string) invoker.ServiceInvoker {
	return c.serviceInvokerManager.ServiceInvoker(serviceType)
}

func (c *DefaultStateMachineConfig) RegisterStateMachineDef(resources []string) error {
	for _, resourcePath := range resources {
		file, err := os.Open(resourcePath)
		if err != nil {
			return fmt.Errorf("open resource file failed: path=%s, err=%w", resourcePath, err)
		}
		defer file.Close()

		if err := c.stateMachineRepository.RegistryStateMachineByReader(file); err != nil {
			return fmt.Errorf("register state machine from file failed: path=%s, err=%w", resourcePath, err)
		}
	}
	return nil
}

func (c *DefaultStateMachineConfig) RegisterExpressionFactory(expressionType string, factory expr.ExpressionFactory) {
	c.expressionFactoryManager.PutExpressionFactory(expressionType, factory)
}

func (c *DefaultStateMachineConfig) RegisterServiceInvoker(serviceType string, invoker invoker.ServiceInvoker) {
	c.serviceInvokerManager.PutServiceInvoker(serviceType, invoker)
}

type RuntimeConfig struct {
	TransOperationTimeout           int      `json:"trans_operation_timeout" yaml:"trans_operation_timeout"`
	ServiceInvokeTimeout            int      `json:"service_invoke_timeout" yaml:"service_invoke_timeout"`
	Charset                         string   `json:"charset" yaml:"charset"`
	DefaultTenantId                 string   `json:"default_tenant_id" yaml:"default_tenant_id"`
	SagaRetryPersistModeUpdate      bool     `json:"saga_retry_persist_mode_update" yaml:"saga_retry_persist_mode_update"`
	SagaCompensatePersistModeUpdate bool     `json:"saga_compensate_persist_mode_update" yaml:"saga_compensate_persist_mode_update"`
	SagaBranchRegisterEnable        bool     `json:"saga_branch_register_enable" yaml:"saga_branch_register_enable"`
	RmReportSuccessEnable           bool     `json:"rm_report_success_enable" yaml:"rm_report_success_enable"`
	StateMachineResources           []string `json:"state_machine_resources" yaml:"state_machine_resources"`
}

func (c *DefaultStateMachineConfig) LoadConfig(configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: path=%s, error=%w", configPath, err)
	}

	parser := parser.NewStateMachineConfigParser()
	smo, err := parser.Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse state machine definition: path=%s, error=%w", configPath, err)
	}

	var runtimeConfig RuntimeConfig
	if err := json.Unmarshal(content, &runtimeConfig); err != nil {
		if err := yaml.Unmarshal(content, &runtimeConfig); err != nil {
		} else {
			c.applyRuntimeConfig(&runtimeConfig)
		}
	} else {
		c.applyRuntimeConfig(&runtimeConfig)
	}

	if _, exists := c.stateMachineDefs[smo.Name]; exists {
		return fmt.Errorf("state machine definition with name %s already exists", smo.Name)
	}
	c.stateMachineDefs[smo.Name] = smo

	return nil
}

func (c *DefaultStateMachineConfig) applyRuntimeConfig(rc *RuntimeConfig) {
	if rc.TransOperationTimeout > 0 {
		c.TransOperationTimeout = rc.TransOperationTimeout
	}
	if rc.ServiceInvokeTimeout > 0 {
		c.ServiceInvokeTimeout = rc.ServiceInvokeTimeout
	}
	if rc.Charset != "" {
		c.Charset = rc.Charset
	}
	if rc.DefaultTenantId != "" {
		c.DefaultTenantId = rc.DefaultTenantId
	}
	c.SagaRetryPersistModeUpdate = rc.SagaRetryPersistModeUpdate
	c.SagaCompensatePersistModeUpdate = rc.SagaCompensatePersistModeUpdate
	c.SagaBranchRegisterEnable = rc.SagaBranchRegisterEnable
	c.RmReportSuccessEnable = rc.RmReportSuccessEnable
	if len(rc.StateMachineResources) > 0 {
		c.StateMachineResources = rc.StateMachineResources
	}
}

func (c *DefaultStateMachineConfig) Init() error {
	if c.expressionFactoryManager != nil {
		defaultExprType := "el"
		factory := c.expressionFactoryManager.GetExpressionFactory(defaultExprType)
		if factory == nil {
			c.RegisterExpressionFactory(defaultExprType, expr.NewELExpressionFactory())
		}
	}

	if c.serviceInvokerManager != nil {
		defaultServiceType := "local"
		existingInvoker := c.serviceInvokerManager.ServiceInvoker(defaultServiceType)
		if existingInvoker == nil {
			newInvoker := invoker.NewLocalServiceInvoker()
			c.RegisterServiceInvoker(defaultServiceType, newInvoker)
		}
	}

	if c.stateMachineRepository != nil && len(c.StateMachineResources) > 0 {
		if err := c.RegisterStateMachineDef(c.StateMachineResources); err != nil {
			return fmt.Errorf("register state machine def failed: %w", err)
		}
	}

	return nil
}

func NewDefaultStateMachineConfig() *DefaultStateMachineConfig {

	// TODO: Initialize the statemachine_repository, following the implementation of the Java version.

	expressionFactoryManager := expr.NewExpressionFactoryManager()
	expressionFactoryManager.Register("el", expr.NewELExpressionFactory())

	serviceInvokerManager := invoker.NewServiceInvokerManagerImpl()
	serviceInvokerManager.PutServiceInvoker("local", invoker.NewLocalServiceInvoker())

	c := &DefaultStateMachineConfig{
		TransOperationTimeout:           DefaultTransOperTimeout,
		ServiceInvokeTimeout:            DefaultServiceInvokeTimeout,
		Charset:                         "UTF-8",
		DefaultTenantId:                 "000001",
		StateMachineResources:           []string{"classpath*:seata/saga/statelang/**/*.json"},
		SagaRetryPersistModeUpdate:      DefaultClientSagaRetryPersistModeUpdate,
		SagaCompensatePersistModeUpdate: DefaultClientSagaCompensatePersistModeUpdate,
		SagaBranchRegisterEnable:        DefaultClientSagaBranchRegisterEnable,
		RmReportSuccessEnable:           DefaultClientReportSuccessEnable,
		expressionFactoryManager:        expressionFactoryManager,
		serviceInvokerManager:           serviceInvokerManager,

		stateMachineDefs: make(map[string]*statemachine.StateMachineObject),

		componentLock: &sync.Mutex{},
	}
	return c
}
