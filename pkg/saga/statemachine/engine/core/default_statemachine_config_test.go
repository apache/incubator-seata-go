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
	"errors"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
)

func TestDefaultStateMachineConfig_LoadValidJSON(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "order_saga.json")

	err := config.LoadConfig(testFile)
	assert.NoError(t, err, "Loading JSON configuration should succeed")

	smo := config.GetStateMachineDefinition("OrderSaga")
	assert.NotNil(t, smo, "State machine definition should not be nil")
	assert.Equal(t, "CreateOrder", smo.StartState, "The start state should be correct")
	assert.Contains(t, smo.States, "CreateOrder", "The state node should exist")

	assert.Equal(t, 30000, config.transOperationTimeout, "The timeout should be read correctly")
}

func TestDefaultStateMachineConfig_LoadValidYAML(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "order_saga.yaml")

	err := config.LoadConfig(testFile)
	assert.NoError(t, err, "Loading YAML configuration should succeed")

	smo := config.GetStateMachineDefinition("OrderSaga")
	assert.NotNil(t, smo)
}

func TestLoadNonExistentFile(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	err := config.LoadConfig("non_existent.json")
	assert.Error(t, err, "Loading a non-existent file should report an error")
	assert.Contains(t, err.Error(), "failed to read config file", "The error message should contain file read failure")
}

func TestGetStateMachineDefinition_Exists(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	_ = config.LoadConfig(filepath.Join("testdata", "order_saga.json"))

	smo := config.GetStateMachineDefinition("OrderSaga")
	assert.NotNil(t, smo)
	assert.Equal(t, "1.0", smo.Version, "The version number should be correct")
}

func TestGetNonExistentStateMachine(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	smo := config.GetStateMachineDefinition("NonExistent")
	assert.Nil(t, smo, "An unloaded state machine should return nil")
}

func TestLoadDuplicateStateMachine(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "order_saga.json")

	err := config.LoadConfig(testFile)
	assert.NoError(t, err)

	err = config.LoadConfig(testFile)
	assert.Error(t, err, "Duplicate loading should trigger a name conflict")
	assert.Contains(t, err.Error(), "already exists", "The error message should contain a conflict prompt")
}

func TestRuntimeConfig_OverrideDefaults(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	assert.Equal(t, "UTF-8", config.charset, "The default character set should be UTF-8")

	_ = config.LoadConfig(filepath.Join("testdata", "order_saga.json"))
	assert.Equal(t, "UTF-8", config.charset, "If the configuration does not specify, the default value should be used")

	customConfig := &ConfigFileParams{
		Charset: "GBK",
	}
	config.applyConfigFileParams(customConfig)
	assert.Equal(t, "GBK", config.charset, "Runtime parameters should be correctly overridden")
}

func TestGetDefaultExpressionFactory(t *testing.T) {
	config := NewDefaultStateMachineConfig()

	err := config.Init()
	assert.NoError(t, err, "Init should not return error")

	factory := config.GetExpressionFactory("el")
	assert.NotNil(t, factory, "The default EL factory should exist")

	unknownFactory := config.GetExpressionFactory("unknown")
	assert.Nil(t, unknownFactory, "An unknown expression type should return nil")
}

func TestGetServiceInvoker(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	if err := config.Init(); err != nil {
		t.Fatalf("init config failed: %v", err)
	}

	invoker := config.GetServiceInvoker("local")
	if invoker == nil {
		t.Errorf("expected non-nil invoker, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "invalid.json")

	err := config.LoadConfig(testFile)
	assert.Error(t, err, "Loading an invalid JSON configuration should report an error")
	assert.Contains(t, err.Error(), "failed to parse state machine definition", "The error message should contain parsing failure")
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	testFile := filepath.Join("testdata", "invalid.yaml")

	err := config.LoadConfig(testFile)
	assert.Error(t, err, "Loading an invalid YAML configuration should report an error")
	assert.Contains(t, err.Error(), "failed to parse state machine definition", "The error message should contain parsing failure")
}

func TestRegisterStateMachineDef_Fail(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	invalidResource := []string{"invalid_path.json"}

	err := config.RegisterStateMachineDef(invalidResource)
	assert.Error(t, err, "Registering an invalid resource should report an error")
	assert.Contains(t, err.Error(), "open resource file failed", "The error message should contain file opening failure")
}

func TestInit_ExpressionFactoryManagerNil(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.expressionFactoryManager = nil

	err := config.Init()
	assert.NoError(t, err, "Initialization should succeed when the expression factory manager is nil")
}

func TestInit_ServiceInvokerManagerNil(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.serviceInvokerManager = nil

	err := config.Init()
	assert.NoError(t, err, "Initialization should succeed when the service invoker manager is nil")
}

func TestInit_StateMachineRepositoryNil(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.stateMachineRepository = nil

	err := config.Init()
	assert.NoError(t, err, "Initialization should succeed when the state machine repository is nil")
}

func TestApplyRuntimeConfig_BoundaryValues(t *testing.T) {
	config := NewDefaultStateMachineConfig()
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

type TestStateMachineRepositoryMock struct{}

func (m *TestStateMachineRepositoryMock) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	return nil, errors.New("get state machine by id failed")
}

func (m *TestStateMachineRepositoryMock) GetStateMachineByNameAndTenantId(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return nil, errors.New("get state machine by name and tenant id failed")
}

func (m *TestStateMachineRepositoryMock) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	return nil, errors.New("get last version state machine failed")
}

func (m *TestStateMachineRepositoryMock) RegistryStateMachine(machine statelang.StateMachine) error {
	return errors.New("registry state machine failed")
}

func (m *TestStateMachineRepositoryMock) RegistryStateMachineByReader(reader io.Reader) error {
	return errors.New("registry state machine by reader failed")
}

func TestRegisterStateMachineDef_RepositoryError(t *testing.T) {
	config := NewDefaultStateMachineConfig()
	config.stateMachineRepository = &TestStateMachineRepositoryMock{}
	resource := []string{"testdata/test.json"}

	err := config.RegisterStateMachineDef(resource)
	assert.Error(t, err, "Registration should fail when the state machine repository reports an error")
	assert.Contains(t, err.Error(), "register state machine from file failed", "The error message should contain registration failure")
}
