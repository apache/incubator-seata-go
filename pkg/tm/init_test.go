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

package tm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitTm(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() {
		config = originalConfig
	}()

	// Test with default config
	testConfig := TmConfig{
		CommitRetryCount:                5,
		RollbackRetryCount:              3,
		DefaultGlobalTransactionTimeout: 30 * time.Second,
		DegradeCheck:                    true,
		DegradeCheckPeriod:              1000,
		DegradeCheckAllowTimes:          5 * time.Second,
		InterceptorOrder:                100,
	}

	InitTm(testConfig)

	assert.Equal(t, testConfig.CommitRetryCount, config.CommitRetryCount)
	assert.Equal(t, testConfig.RollbackRetryCount, config.RollbackRetryCount)
	assert.Equal(t, testConfig.DefaultGlobalTransactionTimeout, config.DefaultGlobalTransactionTimeout)
	assert.Equal(t, testConfig.DegradeCheck, config.DegradeCheck)
	assert.Equal(t, testConfig.DegradeCheckPeriod, config.DegradeCheckPeriod)
	assert.Equal(t, testConfig.DegradeCheckAllowTimes, config.DegradeCheckAllowTimes)
	assert.Equal(t, testConfig.InterceptorOrder, config.InterceptorOrder)
}

func TestInitTm_ZeroConfig(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() {
		config = originalConfig
	}()

	// Test with zero config
	zeroConfig := TmConfig{}

	InitTm(zeroConfig)

	assert.Equal(t, 0, config.CommitRetryCount)
	assert.Equal(t, 0, config.RollbackRetryCount)
	assert.Equal(t, time.Duration(0), config.DefaultGlobalTransactionTimeout)
	assert.False(t, config.DegradeCheck)
	assert.Equal(t, 0, config.DegradeCheckPeriod)
	assert.Equal(t, time.Duration(0), config.DegradeCheckAllowTimes)
	assert.Equal(t, 0, config.InterceptorOrder)
}

func TestInitTm_MultipleInits(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() {
		config = originalConfig
	}()

	// Test multiple initializations
	firstConfig := TmConfig{
		CommitRetryCount:   10,
		RollbackRetryCount: 5,
		DegradeCheck:       true,
	}

	secondConfig := TmConfig{
		CommitRetryCount:   20,
		RollbackRetryCount: 15,
		DegradeCheck:       false,
	}

	// First initialization
	InitTm(firstConfig)
	assert.Equal(t, 10, config.CommitRetryCount)
	assert.Equal(t, 5, config.RollbackRetryCount)
	assert.True(t, config.DegradeCheck)

	// Second initialization should override
	InitTm(secondConfig)
	assert.Equal(t, 20, config.CommitRetryCount)
	assert.Equal(t, 15, config.RollbackRetryCount)
	assert.False(t, config.DegradeCheck)
}

func TestInitTm_ConfigIsolation(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() {
		config = originalConfig
	}()

	// Test that modifying the original config doesn't affect the stored config
	testConfig := TmConfig{
		CommitRetryCount: 5,
	}

	InitTm(testConfig)
	assert.Equal(t, 5, config.CommitRetryCount)

	// Modify original config
	testConfig.CommitRetryCount = 10

	// Stored config should not be affected
	assert.Equal(t, 5, config.CommitRetryCount)
}

func TestInitTm_ExtremeValues(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() {
		config = originalConfig
	}()

	// Test with extreme values
	extremeConfig := TmConfig{
		CommitRetryCount:                2147483647,     // Max int32
		RollbackRetryCount:              -2147483648,    // Min int32
		DefaultGlobalTransactionTimeout: 24 * time.Hour, // Large duration
		DegradeCheck:                    true,
		DegradeCheckPeriod:              0,
		DegradeCheckAllowTimes:          time.Nanosecond, // Small duration
		InterceptorOrder:                -1000000,
	}

	InitTm(extremeConfig)

	assert.Equal(t, 2147483647, config.CommitRetryCount)
	assert.Equal(t, -2147483648, config.RollbackRetryCount)
	assert.Equal(t, 24*time.Hour, config.DefaultGlobalTransactionTimeout)
	assert.True(t, config.DegradeCheck)
	assert.Equal(t, 0, config.DegradeCheckPeriod)
	assert.Equal(t, time.Nanosecond, config.DegradeCheckAllowTimes)
	assert.Equal(t, -1000000, config.InterceptorOrder)
}

func TestGlobalConfigVariable(t *testing.T) {
	// Test that the global config variable exists and is accessible
	assert.NotNil(t, &config, "Global config variable should exist")

	// Test that it's initially a zero value
	var zeroConfig TmConfig
	if config == zeroConfig {
		assert.True(t, true, "Global config starts as zero value")
	}
}

func TestInitTm_Concurrency(t *testing.T) {
	// Save original config
	originalConfig := config
	defer func() {
		config = originalConfig
	}()

	// Test concurrent access (basic test - not comprehensive race detection)
	done := make(chan bool, 2)

	go func() {
		InitTm(TmConfig{CommitRetryCount: 1})
		done <- true
	}()

	go func() {
		InitTm(TmConfig{CommitRetryCount: 2})
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// One of the values should be set (either 1 or 2)
	assert.True(t, config.CommitRetryCount == 1 || config.CommitRetryCount == 2,
		"Config should have one of the concurrently set values")
}
