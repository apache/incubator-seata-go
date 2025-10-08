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
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"

	"github.com/stretchr/testify/assert"
)

type TestStateMachineRepositoryMock struct {
}

func (m *TestStateMachineRepositoryMock) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	return nil, errors.New("mock repository error")
}

func (m *TestStateMachineRepositoryMock) GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return nil, errors.New("mock repository error")
}

func (m *TestStateMachineRepositoryMock) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return nil, errors.New("mock repository error")
}

func (m *TestStateMachineRepositoryMock) RegistryStateMachine(stateMachine statelang.StateMachine) error {
	return errors.New("mock registration error")
}

func (m *TestStateMachineRepositoryMock) RegistryStateMachineByReader(reader io.Reader) error {
	return errors.New("mock registration error")
}

func TestDefaultStateMachineConfig_LoadValidJSON(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	testFile := filepath.Join("testdata", "saga_config.json")
	config, err := NewDefaultStateMachineConfig(WithConfigPath(testFile))
	assert.NoError(t, err, "Failed to initialize config")
	assert.NotNil(t, config, "config is nil")

	smo, err := config.stateMachineRepository.GetStateMachineByNameAndTenantId("OrderSaga", config.GetDefaultTenantId())
	assert.NoError(t, err)
	assert.NotNil(t, smo, "State machine definition should not be nil")

	assert.Equal(t, "CreateOrder", smo.StartState(), "The start state should be correct")
	assert.Contains(t, smo.States(), "CreateOrder", "The state node should exist")
	assert.Equal(t, 30000, config.transOperationTimeout, "The timeout should be read correctly")
}

func TestDefaultStateMachineConfig_LoadValidYAML(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	testFile := filepath.Join("testdata", "saga_config.yaml")
	config, err := NewDefaultStateMachineConfig(WithConfigPath(testFile))
	assert.NoError(t, err, "Failed to initialize config")
	assert.NotNil(t, config, "config is nil")

	smo, err := config.stateMachineRepository.GetStateMachineByNameAndTenantId("OrderSaga", config.GetDefaultTenantId())
	assert.NoError(t, err)
	assert.NotNil(t, smo, "State machine definition should not be nil (YAML)")

	assert.Equal(t, "CreateOrder", smo.StartState(), "The start state should be correct (YAML)")
	assert.Contains(t, smo.States(), "CreateOrder", "The state node should exist (YAML)")
	assert.Equal(t, 30000, config.transOperationTimeout, "The timeout should be read correctly (YAML)")
}

func TestLoadNonExistentFile(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig(WithConfigPath("non_existent.json"))
	err := config.LoadConfig("non_existent.json")
	assert.Error(t, err, "Loading a non-existent file should report an error")
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestGetStateMachineDefinition_Exists(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig(WithConfigPath(filepath.Join("testdata", "saga_config.json")))

	smo, err := config.stateMachineRepository.GetStateMachineByNameAndTenantId("OrderSaga", config.GetDefaultTenantId())
	assert.NoError(t, err)
	assert.NotNil(t, smo)
	assert.Equal(t, "1.0", smo.Version(), "The version number should be correct")
}

func TestGetNonExistentStateMachine(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	smo, err := config.stateMachineRepository.GetStateMachineByNameAndTenantId("NonExistent", config.GetDefaultTenantId())
	assert.Error(t, err)
	assert.True(t, smo == nil || reflect.ValueOf(smo).IsZero(), "An unloaded state machine should return nil/zero")
}

func TestLoadDuplicateStateMachine(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	testFile := filepath.Join("testdata", "saga_config.json")
	config, err := NewDefaultStateMachineConfig(WithConfigPath(testFile))
	assert.NoError(t, err)

	err = config.LoadConfig(testFile)
	assert.NoError(t, err, "First load failed")

	err = config.LoadConfig(testFile)
	assert.NoError(t, err, "Duplicate load of identical config should not error")

	v2File := filepath.Join("testdata", "saga_config_v2.json")
	err = config.LoadConfig(v2File)
	assert.NoError(t, err, "Load of different version should succeed")
}

func TestRuntimeConfig_OverrideDefaults(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	assert.Equal(t, "UTF-8", config.charset, "The default character set should be UTF-8")

	_, _ = NewDefaultStateMachineConfig(WithConfigPath(filepath.Join("testdata", "order_saga.json")))
	assert.Equal(t, "UTF-8", config.charset, "If the configuration does not specify, the default value should be used")

	customConfig := &ConfigFileParams{
		Charset: "GBK",
	}
	config.applyConfigFileParams(customConfig)
	assert.Equal(t, "GBK", config.charset, "Runtime parameters should be correctly overridden")
}

func TestGetDefaultExpressionFactory(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()

	factory := config.GetExpressionFactory("el")
	assert.NotNil(t, factory, "The default EL factory should exist")

	unknownFactory := config.GetExpressionFactory("unknown")
	assert.Nil(t, unknownFactory, "An unknown expression type should return nil")
}

func TestGetServiceInvoker(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig(WithConfigPath(filepath.Join("testdata", "saga_config.json")))

	invoker, _ := config.GetServiceInvoker("local")
	if invoker == nil {
		t.Errorf("expected non-nil invoker, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "invalid.json")

	err := config.LoadConfig(testFile)
	assert.Error(t, err, "Loading an invalid JSON configuration should report an error")
	assert.Contains(t, err.Error(), "JSON", "The error message should indicate JSON parsing failure")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "invalid.yaml")

	err := config.LoadConfig(testFile)
	assert.Error(t, err, "Loading an invalid YAML configuration should report an error")
	assert.Contains(t, err.Error(), "yaml", "The error message should indicate YAML parsing failure")
}

func TestRegisterStateMachineDef_Fail(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	invalidResource := []string{"invalid_path.json"}

	err := config.RegisterStateMachineDef(invalidResource)
	assert.Error(t, err, "Registering an invalid resource should report an error")
	assert.Contains(t, err.Error(), "open resource file failed", "The error message should contain file opening failure")
}

func TestInit_ExpressionFactoryManagerNil(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	config.expressionFactoryManager = nil

	err := config.Init()
	assert.NoError(t, err, "Initialization should succeed when the expression factory manager is nil")
}

func TestInit_ServiceInvokerManagerNil(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	config.serviceInvokerManager = nil

	err := config.Init()
	assert.NoError(t, err, "Initialization should succeed when the service invoker manager is nil")
}

func TestInit_StateMachineRepositoryNil(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	config.stateMachineRepository = nil

	err := config.Init()
	assert.NoError(t, err, "Initialization should succeed when the state machine repository is nil")
}

func TestApplyRuntimeConfig_BoundaryValues(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig()
	customConfig := &ConfigFileParams{
		TransOperationTimeout: 1,
		ServiceInvokeTimeout:  1,
	}
	config.applyConfigFileParams(customConfig)
	assert.Equal(t, 1, config.transOperationTimeout, "The minimum transaction operation timeout should be correctly applied")
	assert.Equal(t, 1, config.serviceInvokeTimeout, "The minimum service invocation timeout should be correctly applied")

	maxTimeout := int(^uint(0) >> 1)
	customConfig = &ConfigFileParams{
		TransOperationTimeout: maxTimeout,
		ServiceInvokeTimeout:  maxTimeout,
	}
	config.applyConfigFileParams(customConfig)
	assert.Equal(t, maxTimeout, config.transOperationTimeout, "The maximum transaction operation timeout should be correctly applied")
	assert.Equal(t, maxTimeout, config.serviceInvokeTimeout, "The maximum service invocation timeout should be correctly applied")
}

func TestRegisterStateMachineDef_RepositoryError(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, _ := NewDefaultStateMachineConfig(WithStateMachineRepository(&TestStateMachineRepositoryMock{}))
	resource := []string{filepath.Join("testdata", "order_saga.json")}

	err := config.RegisterStateMachineDef(resource)
	assert.Error(t, err, "Registration should fail when the state machine repository reports an error")
	assert.Contains(t, err.Error(), "register state machine from file failed", "The error message should contain registration failure")
}

func TestAllComponentsInitialization(t *testing.T) {
	os.Unsetenv("SEATA_STATE_MACHINE_RESOURCES")

	config, err := NewDefaultStateMachineConfig()
	assert.NoError(t, err, "Failed to create config instance")
	assert.NotNil(t, config, "Config instance is nil")

	err = config.Init()
	assert.NoError(t, err, "Failed to initialize config")

	t.Run("ProcessController", func(t *testing.T) {
		assert.NotNil(t, config.processController, "Process controller is not initialized")
	})

	t.Run("EventBusAndPublisher", func(t *testing.T) {
		assert.NotNil(t, config.syncEventBus, "Sync event bus is not initialized")
		assert.NotNil(t, config.asyncEventBus, "Async event bus is not initialized")
		assert.NotNil(t, config.syncProcessCtrlEventPublisher, "Sync event publisher is not initialized")
		assert.NotNil(t, config.asyncProcessCtrlEventPublisher, "Async event publisher is not initialized")
	})

	t.Run("StoreComponents", func(t *testing.T) {
		assert.NotNil(t, config.stateLogRepository, "State log repository is not initialized")
		assert.NotNil(t, config.stateLogStore, "State log store is not initialized")
		assert.NotNil(t, config.stateLangStore, "State language store is not initialized")
		assert.NotNil(t, config.stateMachineRepository, "State machine repository is not initialized")
	})

	t.Run("ExpressionComponents", func(t *testing.T) {
		assert.NotNil(t, config.expressionFactoryManager, "Expression factory manager is not initialized")
		assert.NotNil(t, config.expressionResolver, "Expression resolver is not initialized")

		elFactory := config.expressionFactoryManager.GetExpressionFactory("el")
		assert.NotNil(t, elFactory, "EL expression factory is not registered")
	})

	t.Run("InvokerComponents", func(t *testing.T) {
		assert.NotNil(t, config.serviceInvokerManager, "Service invoker manager is not initialized")
		assert.NotNil(t, config.scriptInvokerManager, "Script invoker manager is not initialized")

		localInvoker := config.serviceInvokerManager.ServiceInvoker("local")
		assert.NotNil(t, localInvoker, "Local service invoker is not registered")
	})

	t.Run("OtherCoreComponents", func(t *testing.T) {
		assert.NotNil(t, config.statusDecisionStrategy, "Status decision strategy is not initialized")
		assert.NotNil(t, config.seqGenerator, "Sequence generator is not initialized")
		assert.NotNil(t, config.componentLock, "Component lock is not initialized")

		testID := config.seqGenerator.GenerateId("test-machine", "test-tenant")
		assert.NotEmpty(t, testID, "Sequence generator failed to generate ID")
	})
}
