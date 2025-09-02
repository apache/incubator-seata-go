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
	"context"
	"encoding/json"
	"fmt"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/repo/repository"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/strategy"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/expr"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/invoker"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/repo"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/store"
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
	stateMachineResources           []string

	// Components
	processController process_ctrl.ProcessController

	// Event Bus
	syncEventBus  process_ctrl.EventBus
	asyncEventBus process_ctrl.EventBus

	// Event publisher
	syncProcessCtrlEventPublisher  process_ctrl.EventPublisher
	asyncProcessCtrlEventPublisher process_ctrl.EventPublisher

	// Store related components
	stateLogRepository     repo.StateLogRepository
	stateLogStore          store.StateLogStore
	stateLangStore         store.StateLangStore
	stateMachineRepository repo.StateMachineRepository

	// Expression related components
	expressionFactoryManager *expr.ExpressionFactoryManager
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

func (c *DefaultStateMachineConfig) SetSyncEventBus(syncEventBus process_ctrl.EventBus) {
	c.syncEventBus = syncEventBus
}

func (c *DefaultStateMachineConfig) SetAsyncEventBus(asyncEventBus process_ctrl.EventBus) {
	c.asyncEventBus = asyncEventBus
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

func (c *DefaultStateMachineConfig) SetExpressionFactoryManager(expressionFactoryManager *expr.ExpressionFactoryManager) {
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

func (c *DefaultStateMachineConfig) ExpressionFactoryManager() *expr.ExpressionFactoryManager {
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

func (c *DefaultStateMachineConfig) SyncEventBus() process_ctrl.EventBus {
	return c.syncEventBus
}

func (c *DefaultStateMachineConfig) AsyncEventBus() process_ctrl.EventBus {
	return c.asyncEventBus
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

func (c *DefaultStateMachineConfig) GetDefaultTenantId() string {
	return c.defaultTenantId
}

func (c *DefaultStateMachineConfig) GetTransOperationTimeout() int {
	return c.transOperationTimeout
}

func (c *DefaultStateMachineConfig) GetServiceInvokeTimeout() int {
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

func (c *DefaultStateMachineConfig) GetStateMachineConfig() engine.StateMachineConfig {
	return c
}

func (c *DefaultStateMachineConfig) GetExpressionFactory(expressionType string) expr.ExpressionFactory {
	return c.expressionFactoryManager.GetExpressionFactory(expressionType)
}

func (c *DefaultStateMachineConfig) GetServiceInvoker(serviceType string) (invoker.ServiceInvoker, error) {
	if serviceType == "" {
		serviceType = "local"
	}

	invoker := c.serviceInvokerManager.ServiceInvoker(serviceType)
	if invoker == nil {
		return nil, fmt.Errorf("service invoker not found for type: %s", serviceType)
	}

	return invoker, nil
}

func (c *DefaultStateMachineConfig) RegisterStateMachineDef(resources []string) error {
	var allFiles []string

	for _, pattern := range resources {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("failed to expand glob pattern: pattern=%s, err=%w", pattern, err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("open resource file failed: pattern=%s", pattern)
		}
		allFiles = append(allFiles, matches...)
	}

	for _, realPath := range allFiles {
		file, err := os.Open(realPath)
		if err != nil {
			return fmt.Errorf("open resource file failed: path=%s, err=%w", realPath, err)
		}
		defer file.Close()

		if err := c.stateMachineRepository.RegistryStateMachineByReader(file); err != nil {
			return fmt.Errorf("register state machine from file failed: path=%s, err=%w", realPath, err)
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

type ConfigFileParams struct {
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
	if c.seqGenerator == nil {
		c.seqGenerator = sequence.NewUUIDSeqGenerator()
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: path=%s, error=%w", configPath, err)
	}

	var configFileParams ConfigFileParams
	ext := strings.ToLower(filepath.Ext(configPath))

	switch ext {
	case ".json":
		if err := json.Unmarshal(content, &configFileParams); err != nil {
			return fmt.Errorf("failed to unmarshal config file as JSON: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(content, &configFileParams); err != nil {
			return fmt.Errorf("failed to unmarshal config file as YAML: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file type: path=%s, ext=%s (only .json/.yaml/.yml are supported)", configPath, ext)
	}

	c.applyConfigFileParams(&configFileParams)
	return nil
}

func (c *DefaultStateMachineConfig) applyConfigFileParams(rc *ConfigFileParams) {
	if rc.TransOperationTimeout > 0 {
		c.transOperationTimeout = rc.TransOperationTimeout
	}
	if rc.ServiceInvokeTimeout > 0 {
		c.serviceInvokeTimeout = rc.ServiceInvokeTimeout
	}
	if rc.Charset != "" {
		c.charset = rc.Charset
	}
	if rc.DefaultTenantId != "" {
		c.defaultTenantId = rc.DefaultTenantId
	}
	c.sagaRetryPersistModeUpdate = rc.SagaRetryPersistModeUpdate
	c.sagaCompensatePersistModeUpdate = rc.SagaCompensatePersistModeUpdate
	c.sagaBranchRegisterEnable = rc.SagaBranchRegisterEnable
	c.rmReportSuccessEnable = rc.RmReportSuccessEnable
	if len(rc.StateMachineResources) > 0 {
		c.stateMachineResources = rc.StateMachineResources
	}
}

func (c *DefaultStateMachineConfig) registerEventConsumers() error {
	if c.processController == nil {
		return fmt.Errorf("ProcessController is not initialized")
	}

	pcImpl, ok := c.processController.(*process_ctrl.ProcessControllerImpl)
	if !ok {
		return fmt.Errorf("ProcessController is not an instance of ProcessControllerImpl")
	}

	if pcImpl.BusinessProcessor() == nil {
		return fmt.Errorf("BusinessProcessor in ProcessController is not initialized")
	}

	consumer := process_ctrl.NewProcessCtrlEventConsumer(c.processController)

	c.syncEventBus.RegisterEventConsumer(consumer)
	c.asyncEventBus.RegisterEventConsumer(consumer)

	return nil
}

func (c *DefaultStateMachineConfig) Init() error {
	if err := c.initExpressionComponents(); err != nil {
		return fmt.Errorf("initialize expression components failed: %w", err)
	}

	if err := c.initServiceInvokers(); err != nil {
		return fmt.Errorf("initialize service invokers failed: %w", err)
	}

	if err := c.registerEventConsumers(); err != nil {
		return fmt.Errorf("register event consumers failed: %w", err)
	}

	if c.stateMachineRepository != nil && len(c.stateMachineResources) > 0 {
		if err := c.RegisterStateMachineDef(c.stateMachineResources); err != nil {
			return fmt.Errorf("register state machine def failed: %w", err)
		}
	}

	if err := c.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	return nil
}

func (c *DefaultStateMachineConfig) initExpressionComponents() error {
	if c.expressionFactoryManager == nil {
		c.expressionFactoryManager = expr.NewExpressionFactoryManager()
	}

	defaultType := expr.DefaultExpressionType
	if defaultType == "" {
		defaultType = "Default"
	}

	if factory := c.expressionFactoryManager.GetExpressionFactory(defaultType); factory == nil {
		c.RegisterExpressionFactory(defaultType, expr.NewCELExpressionFactory())
	}

	if factory := c.expressionFactoryManager.GetExpressionFactory("CEL"); factory == nil {
		c.RegisterExpressionFactory("CEL", expr.NewCELExpressionFactory())
	}

	if factory := c.expressionFactoryManager.GetExpressionFactory("el"); factory == nil {
		c.RegisterExpressionFactory("el", expr.NewCELExpressionFactory())
	}

	if c.seqGenerator != nil {
		sequenceFactory := expr.NewSequenceExpressionFactory(c.seqGenerator)
		c.RegisterExpressionFactory("SEQUENCE", sequenceFactory)
		c.RegisterExpressionFactory("SEQ", sequenceFactory)
	}

	if c.expressionResolver == nil {
		resolver := &expr.DefaultExpressionResolver{}
		resolver.SetExpressionFactoryManager(*c.expressionFactoryManager)
		c.expressionResolver = resolver
	}

	return nil
}

func (c *DefaultStateMachineConfig) initServiceInvokers() error {
	if c.serviceInvokerManager == nil {
		c.serviceInvokerManager = invoker.NewServiceInvokerManagerImpl()
	}

	if existing := c.serviceInvokerManager.ServiceInvoker("local"); existing == nil {
		c.RegisterServiceInvoker("local", invoker.NewLocalServiceInvoker())
	}

	if existing := c.serviceInvokerManager.ServiceInvoker("http"); existing == nil {
		c.RegisterServiceInvoker("http", invoker.NewHTTPInvoker())
	}

	if existing := c.serviceInvokerManager.ServiceInvoker("grpc"); existing == nil {
		c.RegisterServiceInvoker("grpc", invoker.NewGRPCInvoker())
	}

	if existing := c.serviceInvokerManager.ServiceInvoker("func"); existing == nil {
		c.RegisterServiceInvoker("func", invoker.NewFuncInvoker())
	}

	if c.scriptInvokerManager == nil {
		c.scriptInvokerManager = invoker.NewScriptInvokerManager()
	}

	if jsInvoker, err := c.scriptInvokerManager.GetInvoker("javascript"); err != nil || jsInvoker == nil {
		c.scriptInvokerManager.RegisterInvoker(invoker.NewJavaScriptScriptInvoker())
	}

	return nil
}

func (c *DefaultStateMachineConfig) Validate() error {
	var errs []error

	if c.expressionFactoryManager == nil {
		errs = append(errs, fmt.Errorf("expression factory manager is nil"))
	}
	if c.expressionResolver == nil {
		errs = append(errs, fmt.Errorf("expression resolver is nil"))
	}
	if c.serviceInvokerManager == nil {
		errs = append(errs, fmt.Errorf("service invoker manager is nil"))
	}

	if c.scriptInvokerManager == nil {
		errs = append(errs, fmt.Errorf("script invoker manager is nil"))
	}

	if c.transOperationTimeout <= 0 {
		errs = append(errs, fmt.Errorf("invalid trans operation timeout: %d", c.transOperationTimeout))
	}
	if c.serviceInvokeTimeout <= 0 {
		errs = append(errs, fmt.Errorf("invalid service invoke timeout: %d", c.serviceInvokeTimeout))
	}
	if c.charset == "" {
		errs = append(errs, fmt.Errorf("charset is empty"))
	}

	if c.stateMachineRepository != nil {
		if c.stateLogStore == nil {
			errs = append(errs, fmt.Errorf("state log store is nil"))
		}
		if c.stateLangStore == nil {
			errs = append(errs, fmt.Errorf("state lang store is nil"))
		}
		if c.stateLogRepository == nil {
			errs = append(errs, fmt.Errorf("state log repository is nil"))
		}
	}

	if c.statusDecisionStrategy == nil {
		errs = append(errs, fmt.Errorf("status decision strategy is nil"))
	}
	if c.syncEventBus == nil {
		errs = append(errs, fmt.Errorf("sync event bus is nil"))
	}
	if c.asyncEventBus == nil {
		errs = append(errs, fmt.Errorf("async event bus is nil"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed with %d errors: %v", len(errs), errs)
	}
	return nil
}

func (c *DefaultStateMachineConfig) EvaluateExpression(expressionStr string, context any) (any, error) {
	if c.expressionResolver == nil {
		return nil, fmt.Errorf("expression resolver not initialized")
	}

	expression := c.expressionResolver.Expression(expressionStr)
	if expression == nil {
		return nil, fmt.Errorf("failed to parse expression: %s", expressionStr)
	}

	var result any
	var evalErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				evalErr = fmt.Errorf("expression evaluation panicked: %v", r)
			}
		}()

		result = expression.Value(context)
	}()

	if evalErr != nil {
		return nil, evalErr
	}

	if err, ok := result.(error); ok {
		return nil, fmt.Errorf("expression evaluation returned error: %w", err)
	}

	return result, nil
}

func NewDefaultStateMachineConfig(opts ...Option) (*DefaultStateMachineConfig, error) {
	ctx := context.Background()
	defaultBP := process_ctrl.NewBusinessProcessor()

	c := &DefaultStateMachineConfig{
		transOperationTimeout:           DefaultTransOperTimeout,
		serviceInvokeTimeout:            DefaultServiceInvokeTimeout,
		charset:                         "UTF-8",
		defaultTenantId:                 "000001",
		stateMachineResources:           parseEnvResources(),
		sagaRetryPersistModeUpdate:      DefaultClientSagaRetryPersistModeUpdate,
		sagaCompensatePersistModeUpdate: DefaultClientSagaCompensatePersistModeUpdate,
		sagaBranchRegisterEnable:        DefaultClientSagaBranchRegisterEnable,
		rmReportSuccessEnable:           DefaultClientReportSuccessEnable,
		componentLock:                   &sync.Mutex{},
		seqGenerator:                    sequence.NewUUIDSeqGenerator(),
		statusDecisionStrategy:          strategy.NewDefaultStatusDecisionStrategy(),
		processController: func() process_ctrl.ProcessController {
			pc := &process_ctrl.ProcessControllerImpl{}
			pc.SetBusinessProcessor(defaultBP)
			return pc
		}(),
		syncEventBus:  process_ctrl.NewDirectEventBus(),
		asyncEventBus: process_ctrl.NewAsyncEventBus(ctx, 1000, 5),

		syncProcessCtrlEventPublisher:  nil,
		asyncProcessCtrlEventPublisher: nil,

		stateLogStore:  &NoopStateLogStore{},
		stateLangStore: &NoopStateLangStore{},
	}

	c.stateMachineRepository = repository.GetStateMachineRepositoryImpl()
	c.stateLogRepository = repository.NewStateLogRepositoryImpl()

	c.syncProcessCtrlEventPublisher = process_ctrl.NewProcessCtrlEventPublisher(c.syncEventBus)
	c.asyncProcessCtrlEventPublisher = process_ctrl.NewProcessCtrlEventPublisher(c.asyncEventBus)

	for _, opt := range opts {
		opt(c)
	}

	if err := c.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize state machine config: %w", err)
	}

	return c, nil
}

func parseEnvResources() []string {
	if env := os.Getenv("SEATA_STATE_MACHINE_RESOURCES"); env != "" {
		parts := strings.Split(env, ",")
		var res []string
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				res = append(res, p)
			}
		}
		return res
	}
	return nil
}

type Option func(*DefaultStateMachineConfig)

func WithStatusDecisionStrategy(strategy engine.StatusDecisionStrategy) Option {
	return func(c *DefaultStateMachineConfig) {
		c.statusDecisionStrategy = strategy
	}
}

func WithSeqGenerator(gen sequence.SeqGenerator) Option {
	return func(c *DefaultStateMachineConfig) {
		c.seqGenerator = gen
	}
}

func WithProcessController(ctrl process_ctrl.ProcessController) Option {
	return func(c *DefaultStateMachineConfig) {
		c.processController = ctrl
	}
}

func WithBusinessProcessor(bp process_ctrl.BusinessProcessor) Option {
	return func(c *DefaultStateMachineConfig) {
		if pc, ok := c.processController.(*process_ctrl.ProcessControllerImpl); ok {
			pc.SetBusinessProcessor(bp)
		} else {
			log.Printf("ProcessController is not of type *ProcessControllerImpl, unable to set BusinessProcessor")
		}
	}
}

func WithStateMachineResources(paths []string) Option {
	return func(c *DefaultStateMachineConfig) {
		if len(paths) > 0 {
			c.stateMachineResources = paths
		}
	}
}

func WithStateLogRepository(logRepo repo.StateLogRepository) Option {
	return func(c *DefaultStateMachineConfig) {
		c.stateLogRepository = logRepo
	}
}

func WithStateLogStore(logStore store.StateLogStore) Option {
	return func(c *DefaultStateMachineConfig) {
		c.stateLogStore = logStore
	}
}

func WithStateLangStore(langStore store.StateLangStore) Option {
	return func(c *DefaultStateMachineConfig) {
		c.stateLangStore = langStore
	}
}

func WithStateMachineRepository(machineRepo repo.StateMachineRepository) Option {
	return func(c *DefaultStateMachineConfig) {
		c.stateMachineRepository = machineRepo
	}
}

func WithConfigPath(path string) Option {
	return func(c *DefaultStateMachineConfig) {
		if path == "" {
			return
		}
		if err := c.LoadConfig(path); err != nil {
			log.Printf("Failed to load config from %s: %v", path, err)
		} else {
			log.Printf("Successfully loaded config from %s", path)
		}
	}
}

func WithScriptInvokerManager(scriptManager invoker.ScriptInvokerManager) Option {
	return func(c *DefaultStateMachineConfig) {
		c.scriptInvokerManager = scriptManager
	}
}
