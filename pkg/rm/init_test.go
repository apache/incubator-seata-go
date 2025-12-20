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

package rm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitRm(t *testing.T) {
	originalConfig := rmConfig
	defer func() {
		rmConfig = originalConfig
	}()

	testConfig := RmConfig{
		Config: Config{
			AsyncCommitBufferLimit:          10000,
			ReportRetryCount:                5,
			TableMetaCheckEnable:            false,
			ReportSuccessEnable:             false,
			SagaBranchRegisterEnable:        false,
			SagaJsonParser:                  "fastjson",
			SagaRetryPersistModeUpdate:      false,
			SagaCompensatePersistModeUpdate: false,
			TccActionInterceptorOrder:       -2147482648,
			SqlParserType:                   "druid",
			LockConfig: LockConfig{
				RetryInterval:                       30 * time.Second,
				RetryTimes:                          10,
				RetryPolicyBranchRollbackOnConflict: true,
			},
		},
		ApplicationID:  "test-app",
		TxServiceGroup: "test-service-group",
	}

	InitRm(testConfig)

	assert.Equal(t, testConfig.AsyncCommitBufferLimit, rmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, testConfig.ReportRetryCount, rmConfig.ReportRetryCount)
	assert.Equal(t, testConfig.TableMetaCheckEnable, rmConfig.TableMetaCheckEnable)
	assert.Equal(t, testConfig.ReportSuccessEnable, rmConfig.ReportSuccessEnable)
	assert.Equal(t, testConfig.SagaBranchRegisterEnable, rmConfig.SagaBranchRegisterEnable)
	assert.Equal(t, testConfig.SagaJsonParser, rmConfig.SagaJsonParser)
	assert.Equal(t, testConfig.SagaRetryPersistModeUpdate, rmConfig.SagaRetryPersistModeUpdate)
	assert.Equal(t, testConfig.SagaCompensatePersistModeUpdate, rmConfig.SagaCompensatePersistModeUpdate)
	assert.Equal(t, testConfig.TccActionInterceptorOrder, rmConfig.TccActionInterceptorOrder)
	assert.Equal(t, testConfig.SqlParserType, rmConfig.SqlParserType)
	assert.Equal(t, testConfig.LockConfig.RetryInterval, rmConfig.LockConfig.RetryInterval)
	assert.Equal(t, testConfig.LockConfig.RetryTimes, rmConfig.LockConfig.RetryTimes)
	assert.Equal(t, testConfig.LockConfig.RetryPolicyBranchRollbackOnConflict, rmConfig.LockConfig.RetryPolicyBranchRollbackOnConflict)
	assert.Equal(t, testConfig.ApplicationID, rmConfig.ApplicationID)
	assert.Equal(t, testConfig.TxServiceGroup, rmConfig.TxServiceGroup)
}

func TestInitRm_ZeroConfig(t *testing.T) {
	originalConfig := rmConfig
	defer func() {
		rmConfig = originalConfig
	}()

	zeroConfig := RmConfig{}

	InitRm(zeroConfig)

	assert.Equal(t, 0, rmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, 0, rmConfig.ReportRetryCount)
	assert.False(t, rmConfig.TableMetaCheckEnable)
	assert.False(t, rmConfig.ReportSuccessEnable)
	assert.False(t, rmConfig.SagaBranchRegisterEnable)
	assert.Equal(t, "", rmConfig.SagaJsonParser)
	assert.False(t, rmConfig.SagaRetryPersistModeUpdate)
	assert.False(t, rmConfig.SagaCompensatePersistModeUpdate)
	assert.Equal(t, 0, rmConfig.TccActionInterceptorOrder)
	assert.Equal(t, "", rmConfig.SqlParserType)
	assert.Equal(t, time.Duration(0), rmConfig.LockConfig.RetryInterval)
	assert.Equal(t, 0, rmConfig.LockConfig.RetryTimes)
	assert.False(t, rmConfig.LockConfig.RetryPolicyBranchRollbackOnConflict)
	assert.Equal(t, "", rmConfig.ApplicationID)
	assert.Equal(t, "", rmConfig.TxServiceGroup)
}

func TestInitRm_MultipleInits(t *testing.T) {
	originalConfig := rmConfig
	defer func() {
		rmConfig = originalConfig
	}()

	firstConfig := RmConfig{
		Config: Config{
			AsyncCommitBufferLimit: 5000,
			ReportRetryCount:       3,
			TableMetaCheckEnable:   true,
		},
		ApplicationID:  "app-1",
		TxServiceGroup: "group-1",
	}

	secondConfig := RmConfig{
		Config: Config{
			AsyncCommitBufferLimit: 15000,
			ReportRetryCount:       10,
			TableMetaCheckEnable:   false,
		},
		ApplicationID:  "app-2",
		TxServiceGroup: "group-2",
	}

	InitRm(firstConfig)
	assert.Equal(t, 5000, rmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, 3, rmConfig.ReportRetryCount)
	assert.True(t, rmConfig.TableMetaCheckEnable)
	assert.Equal(t, "app-1", rmConfig.ApplicationID)
	assert.Equal(t, "group-1", rmConfig.TxServiceGroup)

	InitRm(secondConfig)
	assert.Equal(t, 15000, rmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, 10, rmConfig.ReportRetryCount)
	assert.False(t, rmConfig.TableMetaCheckEnable)
	assert.Equal(t, "app-2", rmConfig.ApplicationID)
	assert.Equal(t, "group-2", rmConfig.TxServiceGroup)
}

func TestInitRm_ConfigIsolation(t *testing.T) {
	originalConfig := rmConfig
	defer func() {
		rmConfig = originalConfig
	}()

	testConfig := RmConfig{
		Config: Config{
			AsyncCommitBufferLimit: 1000,
		},
		ApplicationID: "test-app",
	}

	InitRm(testConfig)
	assert.Equal(t, 1000, rmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, "test-app", rmConfig.ApplicationID)

	testConfig.AsyncCommitBufferLimit = 2000
	testConfig.ApplicationID = "modified-app"

	assert.Equal(t, 1000, rmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, "test-app", rmConfig.ApplicationID)
}

func TestInitRm_ExtremeValues(t *testing.T) {
	originalConfig := rmConfig
	defer func() {
		rmConfig = originalConfig
	}()

	extremeConfig := RmConfig{
		Config: Config{
			AsyncCommitBufferLimit:          2147483647,
			ReportRetryCount:                -2147483648,
			TableMetaCheckEnable:            true,
			ReportSuccessEnable:             true,
			SagaBranchRegisterEnable:        true,
			SagaJsonParser:                  "very-long-json-parser-name-that-exceeds-normal-length",
			SagaRetryPersistModeUpdate:      true,
			SagaCompensatePersistModeUpdate: true,
			TccActionInterceptorOrder:       2147483647,
			SqlParserType:                   "very-long-sql-parser-type-name",
			LockConfig: LockConfig{
				RetryInterval:                       24 * time.Hour,
				RetryTimes:                          2147483647,
				RetryPolicyBranchRollbackOnConflict: true,
			},
		},
		ApplicationID:  "very-long-application-id-that-exceeds-normal-length",
		TxServiceGroup: "very-long-tx-service-group-that-exceeds-normal-length",
	}

	InitRm(extremeConfig)

	assert.Equal(t, 2147483647, rmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, -2147483648, rmConfig.ReportRetryCount)
	assert.True(t, rmConfig.TableMetaCheckEnable)
	assert.True(t, rmConfig.ReportSuccessEnable)
	assert.True(t, rmConfig.SagaBranchRegisterEnable)
	assert.Equal(t, "very-long-json-parser-name-that-exceeds-normal-length", rmConfig.SagaJsonParser)
	assert.True(t, rmConfig.SagaRetryPersistModeUpdate)
	assert.True(t, rmConfig.SagaCompensatePersistModeUpdate)
	assert.Equal(t, 2147483647, rmConfig.TccActionInterceptorOrder)
	assert.Equal(t, "very-long-sql-parser-type-name", rmConfig.SqlParserType)
	assert.Equal(t, 24*time.Hour, rmConfig.LockConfig.RetryInterval)
	assert.Equal(t, 2147483647, rmConfig.LockConfig.RetryTimes)
	assert.True(t, rmConfig.LockConfig.RetryPolicyBranchRollbackOnConflict)
	assert.Equal(t, "very-long-application-id-that-exceeds-normal-length", rmConfig.ApplicationID)
	assert.Equal(t, "very-long-tx-service-group-that-exceeds-normal-length", rmConfig.TxServiceGroup)
}

func TestGlobalRmConfigVariable(t *testing.T) {
	assert.NotNil(t, &rmConfig, "Global rmConfig variable should exist")

	var zeroConfig RmConfig
	if rmConfig == zeroConfig {
		assert.True(t, true, "Global rmConfig starts as zero value")
	}
}
