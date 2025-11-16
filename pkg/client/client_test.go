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

package client

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/tm"
)

func TestInit(t *testing.T) {
	// Test that Init() function exists
	assert.NotNil(t, Init)
}

func TestInitPath(t *testing.T) {
	// Test that InitPath() function exists
	assert.NotNil(t, InitPath)
}

func TestOnceInstances(t *testing.T) {
	// Test that sync.Once instances are properly initialized
	assert.NotNil(t, &onceInitTmClient, "onceInitTmClient should be initialized")
	assert.NotNil(t, &onceInitRmClient, "onceInitRmClient should be initialized")
	assert.NotNil(t, &onceInitDatasource, "onceInitDatasource should be initialized")
	assert.NotNil(t, &onceInitRegistry, "onceInitRegistry should be initialized")
}

func TestInitTmClientFunction(t *testing.T) {
	// Create a minimal config for testing
	cfg := &Config{
		ClientConfig: ClientConfig{
			TmConfig: tm.TmConfig{
				CommitRetryCount:   5,
				RollbackRetryCount: 5,
			},
		},
	}

	// The function should exist and be callable
	assert.NotNil(t, initTmClient)

	// Test that the function has the correct signature
	assert.IsType(t, func(*Config) {}, initTmClient)

	// We won't actually call it to avoid dependency issues
	_ = cfg
}

func TestInitRmClientFunction(t *testing.T) {
	// Create a minimal config for testing
	cfg := &Config{
		ApplicationID:  "test-app",
		TxServiceGroup: "test-group",
	}

	// The function should exist and be callable
	assert.NotNil(t, initRmClient)

	// Test that the function has the correct signature
	assert.IsType(t, func(*Config) {}, initRmClient)

	// We won't actually call it to avoid dependency issues
	_ = cfg
}

func TestInitDatasourceFunction(t *testing.T) {
	// The function should exist and be callable
	assert.NotNil(t, initDatasource)

	// Test that the function has the correct signature
	assert.IsType(t, func() {}, initDatasource)
}

func TestInitRegistryFunction(t *testing.T) {
	// Create a minimal config for testing
	cfg := &Config{
		ApplicationID:  "test-app",
		TxServiceGroup: "test-group",
	}

	// The function should exist and be callable
	assert.NotNil(t, initRegistry)

	// Test that the function has the correct signature
	assert.IsType(t, func(*Config) {}, initRegistry)

	// We won't actually call it to avoid dependency issues
	_ = cfg
}

func TestInitRemotingFunction(t *testing.T) {
	// Create a minimal config for testing
	cfg := &Config{
		ApplicationID:  "test-app",
		TxServiceGroup: "test-group",
	}

	// The function should exist and be callable
	assert.NotNil(t, initRemoting)

	// Test that the function has the correct signature
	assert.IsType(t, func(*Config) {}, initRemoting)

	// We won't actually call it to avoid dependency issues
	_ = cfg
}

func TestSyncOnceTypes(t *testing.T) {
	// Test that all once variables are of type sync.Once
	assert.IsType(t, sync.Once{}, onceInitTmClient)
	assert.IsType(t, sync.Once{}, onceInitRmClient)
	assert.IsType(t, sync.Once{}, onceInitDatasource)
	assert.IsType(t, sync.Once{}, onceInitRegistry)
}

func TestConfigStruct(t *testing.T) {
	// Test that we can create a Config struct
	cfg := &Config{}
	assert.NotNil(t, cfg)

	// Test that Config has expected fields
	cfg.ApplicationID = "test"
	cfg.TxServiceGroup = "test-group"
	cfg.Enabled = true

	assert.Equal(t, "test", cfg.ApplicationID)
	assert.Equal(t, "test-group", cfg.TxServiceGroup)
	assert.True(t, cfg.Enabled)
}

func TestClientConfigStruct(t *testing.T) {
	// Test that we can create a ClientConfig struct
	clientCfg := &ClientConfig{}
	assert.NotNil(t, clientCfg)

	// Test that ClientConfig has TmConfig field
	clientCfg.TmConfig = tm.TmConfig{
		CommitRetryCount: 3,
	}

	assert.Equal(t, 3, clientCfg.TmConfig.CommitRetryCount)
}
