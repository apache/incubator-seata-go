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
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_DefaultValues(t *testing.T) {
	cfg := &Config{}

	assert.Equal(t, 0, cfg.AsyncCommitBufferLimit)
	assert.Equal(t, 0, cfg.ReportRetryCount)
	assert.False(t, cfg.TableMetaCheckEnable)
	assert.False(t, cfg.ReportSuccessEnable)
	assert.False(t, cfg.SagaBranchRegisterEnable)
	assert.Equal(t, "", cfg.SagaJsonParser)
	assert.False(t, cfg.SagaRetryPersistModeUpdate)
	assert.False(t, cfg.SagaCompensatePersistModeUpdate)
	assert.Equal(t, 0, cfg.TccActionInterceptorOrder)
	assert.Equal(t, "", cfg.SqlParserType)
	assert.Equal(t, time.Duration(0), cfg.LockConfig.RetryInterval)
	assert.Equal(t, 0, cfg.LockConfig.RetryTimes)
	assert.False(t, cfg.LockConfig.RetryPolicyBranchRollbackOnConflict)
}

func TestConfig_SetValues(t *testing.T) {
	cfg := &Config{
		AsyncCommitBufferLimit:          10000,
		ReportRetryCount:                5,
		TableMetaCheckEnable:            true,
		ReportSuccessEnable:             true,
		SagaBranchRegisterEnable:        true,
		SagaJsonParser:                  "fastjson",
		SagaRetryPersistModeUpdate:      true,
		SagaCompensatePersistModeUpdate: true,
		TccActionInterceptorOrder:       -2147482648,
		SqlParserType:                   "druid",
		LockConfig: LockConfig{
			RetryInterval:                       30 * time.Second,
			RetryTimes:                          10,
			RetryPolicyBranchRollbackOnConflict: true,
		},
	}

	assert.Equal(t, 10000, cfg.AsyncCommitBufferLimit)
	assert.Equal(t, 5, cfg.ReportRetryCount)
	assert.True(t, cfg.TableMetaCheckEnable)
	assert.True(t, cfg.ReportSuccessEnable)
	assert.True(t, cfg.SagaBranchRegisterEnable)
	assert.Equal(t, "fastjson", cfg.SagaJsonParser)
	assert.True(t, cfg.SagaRetryPersistModeUpdate)
	assert.True(t, cfg.SagaCompensatePersistModeUpdate)
	assert.Equal(t, -2147482648, cfg.TccActionInterceptorOrder)
	assert.Equal(t, "druid", cfg.SqlParserType)
	assert.Equal(t, 30*time.Second, cfg.LockConfig.RetryInterval)
	assert.Equal(t, 10, cfg.LockConfig.RetryTimes)
	assert.True(t, cfg.LockConfig.RetryPolicyBranchRollbackOnConflict)
}

func TestConfig_RegisterFlagsWithPrefix(t *testing.T) {
	cfg := &Config{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	assert.NotPanics(t, func() {
		cfg.RegisterFlagsWithPrefix("", flagSet)
	})

	assert.NotPanics(t, func() {
		cfg.RegisterFlagsWithPrefix("rm", flagSet)
	})
}

func TestConfig_RegisterFlagsWithPrefix_FunctionExists(t *testing.T) {
	cfg := &Config{}

	assert.NotNil(t, cfg.RegisterFlagsWithPrefix)

	assert.IsType(t, func(string, *flag.FlagSet) {}, cfg.RegisterFlagsWithPrefix)
}

func TestConfig_FieldTypes(t *testing.T) {
	cfg := &Config{}

	assert.IsType(t, int(0), cfg.AsyncCommitBufferLimit)
	assert.IsType(t, int(0), cfg.ReportRetryCount)
	assert.IsType(t, false, cfg.TableMetaCheckEnable)
	assert.IsType(t, false, cfg.ReportSuccessEnable)
	assert.IsType(t, false, cfg.SagaBranchRegisterEnable)
	assert.IsType(t, "", cfg.SagaJsonParser)
	assert.IsType(t, false, cfg.SagaRetryPersistModeUpdate)
	assert.IsType(t, false, cfg.SagaCompensatePersistModeUpdate)
	assert.IsType(t, int(0), cfg.TccActionInterceptorOrder)
	assert.IsType(t, "", cfg.SqlParserType)
}

func TestConfig_StructInstantiation(t *testing.T) {
	cfg1 := Config{}
	cfg2 := &Config{}
	cfg3 := new(Config)

	assert.NotNil(t, &cfg1)
	assert.NotNil(t, cfg2)
	assert.NotNil(t, cfg3)

	assert.Equal(t, 0, cfg1.AsyncCommitBufferLimit)
	assert.Equal(t, 0, cfg2.AsyncCommitBufferLimit)
	assert.Equal(t, 0, cfg3.AsyncCommitBufferLimit)
}

func TestConfig_FieldAssignment(t *testing.T) {
	cfg := &Config{}

	cfg.AsyncCommitBufferLimit = 5000
	cfg.ReportRetryCount = 3
	cfg.TableMetaCheckEnable = true
	cfg.ReportSuccessEnable = true
	cfg.SagaBranchRegisterEnable = true
	cfg.SagaJsonParser = "custom-json"
	cfg.SagaRetryPersistModeUpdate = true
	cfg.SagaCompensatePersistModeUpdate = true
	cfg.TccActionInterceptorOrder = -1000
	cfg.SqlParserType = "custom-parser"
	cfg.LockConfig.RetryInterval = 60 * time.Second
	cfg.LockConfig.RetryTimes = 15
	cfg.LockConfig.RetryPolicyBranchRollbackOnConflict = false

	assert.Equal(t, 5000, cfg.AsyncCommitBufferLimit)
	assert.Equal(t, 3, cfg.ReportRetryCount)
	assert.True(t, cfg.TableMetaCheckEnable)
	assert.True(t, cfg.ReportSuccessEnable)
	assert.True(t, cfg.SagaBranchRegisterEnable)
	assert.Equal(t, "custom-json", cfg.SagaJsonParser)
	assert.True(t, cfg.SagaRetryPersistModeUpdate)
	assert.True(t, cfg.SagaCompensatePersistModeUpdate)
	assert.Equal(t, -1000, cfg.TccActionInterceptorOrder)
	assert.Equal(t, "custom-parser", cfg.SqlParserType)
	assert.Equal(t, 60*time.Second, cfg.LockConfig.RetryInterval)
	assert.Equal(t, 15, cfg.LockConfig.RetryTimes)
	assert.False(t, cfg.LockConfig.RetryPolicyBranchRollbackOnConflict)
}

func TestConfig_NegativeValues(t *testing.T) {
	cfg := &Config{
		AsyncCommitBufferLimit:          -1,
		ReportRetryCount:                -5,
		TableMetaCheckEnable:            false,
		ReportSuccessEnable:             false,
		SagaBranchRegisterEnable:        false,
		SagaJsonParser:                  "",
		SagaRetryPersistModeUpdate:      false,
		SagaCompensatePersistModeUpdate: false,
		TccActionInterceptorOrder:       -2147483648,
		SqlParserType:                   "",
		LockConfig: LockConfig{
			RetryInterval:                       -1 * time.Second,
			RetryTimes:                          -10,
			RetryPolicyBranchRollbackOnConflict: false,
		},
	}

	assert.Equal(t, -1, cfg.AsyncCommitBufferLimit)
	assert.Equal(t, -5, cfg.ReportRetryCount)
	assert.False(t, cfg.TableMetaCheckEnable)
	assert.False(t, cfg.ReportSuccessEnable)
	assert.False(t, cfg.SagaBranchRegisterEnable)
	assert.Equal(t, "", cfg.SagaJsonParser)
	assert.False(t, cfg.SagaRetryPersistModeUpdate)
	assert.False(t, cfg.SagaCompensatePersistModeUpdate)
	assert.Equal(t, -2147483648, cfg.TccActionInterceptorOrder)
	assert.Equal(t, "", cfg.SqlParserType)
	assert.Equal(t, -1*time.Second, cfg.LockConfig.RetryInterval)
	assert.Equal(t, -10, cfg.LockConfig.RetryTimes)
	assert.False(t, cfg.LockConfig.RetryPolicyBranchRollbackOnConflict)
}

func TestConfig_ExtremeValues(t *testing.T) {
	cfg := &Config{
		AsyncCommitBufferLimit:          2147483647,
		ReportRetryCount:                2147483647,
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
	}

	assert.Equal(t, 2147483647, cfg.AsyncCommitBufferLimit)
	assert.Equal(t, 2147483647, cfg.ReportRetryCount)
	assert.True(t, cfg.TableMetaCheckEnable)
	assert.True(t, cfg.ReportSuccessEnable)
	assert.True(t, cfg.SagaBranchRegisterEnable)
	assert.Equal(t, "very-long-json-parser-name-that-exceeds-normal-length", cfg.SagaJsonParser)
	assert.True(t, cfg.SagaRetryPersistModeUpdate)
	assert.True(t, cfg.SagaCompensatePersistModeUpdate)
	assert.Equal(t, 2147483647, cfg.TccActionInterceptorOrder)
	assert.Equal(t, "very-long-sql-parser-type-name", cfg.SqlParserType)
	assert.Equal(t, 24*time.Hour, cfg.LockConfig.RetryInterval)
	assert.Equal(t, 2147483647, cfg.LockConfig.RetryTimes)
	assert.True(t, cfg.LockConfig.RetryPolicyBranchRollbackOnConflict)
}

func TestLockConfig_DefaultValues(t *testing.T) {
	lockCfg := &LockConfig{}

	assert.Equal(t, time.Duration(0), lockCfg.RetryInterval)
	assert.Equal(t, 0, lockCfg.RetryTimes)
	assert.False(t, lockCfg.RetryPolicyBranchRollbackOnConflict)
}

func TestLockConfig_SetValues(t *testing.T) {
	lockCfg := &LockConfig{
		RetryInterval:                       30 * time.Second,
		RetryTimes:                          10,
		RetryPolicyBranchRollbackOnConflict: true,
	}

	assert.Equal(t, 30*time.Second, lockCfg.RetryInterval)
	assert.Equal(t, 10, lockCfg.RetryTimes)
	assert.True(t, lockCfg.RetryPolicyBranchRollbackOnConflict)
}

func TestLockConfig_RegisterFlagsWithPrefix(t *testing.T) {
	lockCfg := &LockConfig{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	assert.NotPanics(t, func() {
		lockCfg.RegisterFlagsWithPrefix("", flagSet)
	})

	assert.NotPanics(t, func() {
		lockCfg.RegisterFlagsWithPrefix("lock", flagSet)
	})
}

func TestLockConfig_RegisterFlagsWithPrefix_FunctionExists(t *testing.T) {
	lockCfg := &LockConfig{}

	assert.NotNil(t, lockCfg.RegisterFlagsWithPrefix)

	assert.IsType(t, func(string, *flag.FlagSet) {}, lockCfg.RegisterFlagsWithPrefix)
}
