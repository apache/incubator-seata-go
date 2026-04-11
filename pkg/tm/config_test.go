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
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTmConfig_DefaultValues(t *testing.T) {
	cfg := &TmConfig{}

	// Test that zero values are correct
	assert.Equal(t, 0, cfg.CommitRetryCount)
	assert.Equal(t, 0, cfg.RollbackRetryCount)
	assert.Equal(t, time.Duration(0), cfg.DefaultGlobalTransactionTimeout)
	assert.False(t, cfg.DegradeCheck)
	assert.Equal(t, 0, cfg.DegradeCheckPeriod)
	assert.Equal(t, time.Duration(0), cfg.DegradeCheckAllowTimes)
	assert.Equal(t, 0, cfg.InterceptorOrder)
}

func TestTmConfig_SetValues(t *testing.T) {
	cfg := &TmConfig{
		CommitRetryCount:                10,
		RollbackRetryCount:              5,
		DefaultGlobalTransactionTimeout: 30 * time.Second,
		DegradeCheck:                    true,
		DegradeCheckPeriod:              1000,
		DegradeCheckAllowTimes:          5 * time.Second,
		InterceptorOrder:                100,
	}

	assert.Equal(t, 10, cfg.CommitRetryCount)
	assert.Equal(t, 5, cfg.RollbackRetryCount)
	assert.Equal(t, 30*time.Second, cfg.DefaultGlobalTransactionTimeout)
	assert.True(t, cfg.DegradeCheck)
	assert.Equal(t, 1000, cfg.DegradeCheckPeriod)
	assert.Equal(t, 5*time.Second, cfg.DegradeCheckAllowTimes)
	assert.Equal(t, 100, cfg.InterceptorOrder)
}

func TestTmConfig_RegisterFlagsWithPrefix(t *testing.T) {
	cfg := &TmConfig{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	// Test that the function doesn't panic
	assert.NotPanics(t, func() {
		cfg.RegisterFlagsWithPrefix("", flagSet)
	})

	// Test that the function doesn't panic with a prefix
	assert.NotPanics(t, func() {
		cfg.RegisterFlagsWithPrefix("tm", flagSet)
	})
}

func TestTmConfig_RegisterFlagsWithPrefix_FunctionExists(t *testing.T) {
	cfg := &TmConfig{}

	// Test that the method exists
	assert.NotNil(t, cfg.RegisterFlagsWithPrefix)

	// Test that it has the correct signature
	assert.IsType(t, func(string, *flag.FlagSet) {}, cfg.RegisterFlagsWithPrefix)
}

func TestTmConfig_FieldTypes(t *testing.T) {
	cfg := &TmConfig{}

	// Test that fields have correct types
	assert.IsType(t, int(0), cfg.CommitRetryCount)
	assert.IsType(t, int(0), cfg.RollbackRetryCount)
	assert.IsType(t, time.Duration(0), cfg.DefaultGlobalTransactionTimeout)
	assert.IsType(t, false, cfg.DegradeCheck)
	assert.IsType(t, int(0), cfg.DegradeCheckPeriod)
	assert.IsType(t, time.Duration(0), cfg.DegradeCheckAllowTimes)
	assert.IsType(t, int(0), cfg.InterceptorOrder)
}

func TestTmConfig_StructInstantiation(t *testing.T) {
	// Test struct instantiation with different methods
	cfg1 := TmConfig{}
	cfg2 := &TmConfig{}
	cfg3 := new(TmConfig)

	assert.NotNil(t, &cfg1)
	assert.NotNil(t, cfg2)
	assert.NotNil(t, cfg3)

	// All should have zero values
	assert.Equal(t, 0, cfg1.CommitRetryCount)
	assert.Equal(t, 0, cfg2.CommitRetryCount)
	assert.Equal(t, 0, cfg3.CommitRetryCount)
}

func TestTmConfig_FieldAssignment(t *testing.T) {
	cfg := &TmConfig{}

	// Test field assignment
	cfg.CommitRetryCount = 5
	cfg.RollbackRetryCount = 3
	cfg.DefaultGlobalTransactionTimeout = 60 * time.Second
	cfg.DegradeCheck = true
	cfg.DegradeCheckPeriod = 2000
	cfg.DegradeCheckAllowTimes = 10 * time.Second
	cfg.InterceptorOrder = -1000

	assert.Equal(t, 5, cfg.CommitRetryCount)
	assert.Equal(t, 3, cfg.RollbackRetryCount)
	assert.Equal(t, 60*time.Second, cfg.DefaultGlobalTransactionTimeout)
	assert.True(t, cfg.DegradeCheck)
	assert.Equal(t, 2000, cfg.DegradeCheckPeriod)
	assert.Equal(t, 10*time.Second, cfg.DegradeCheckAllowTimes)
	assert.Equal(t, -1000, cfg.InterceptorOrder)
}

func TestTmConfig_NegativeValues(t *testing.T) {
	cfg := &TmConfig{
		CommitRetryCount:                -1,
		RollbackRetryCount:              -5,
		DefaultGlobalTransactionTimeout: -1 * time.Second,
		DegradeCheck:                    false,
		DegradeCheckPeriod:              -100,
		DegradeCheckAllowTimes:          -5 * time.Second,
		InterceptorOrder:                -2147483648,
	}

	// Negative values should be preserved
	assert.Equal(t, -1, cfg.CommitRetryCount)
	assert.Equal(t, -5, cfg.RollbackRetryCount)
	assert.Equal(t, -1*time.Second, cfg.DefaultGlobalTransactionTimeout)
	assert.False(t, cfg.DegradeCheck)
	assert.Equal(t, -100, cfg.DegradeCheckPeriod)
	assert.Equal(t, -5*time.Second, cfg.DegradeCheckAllowTimes)
	assert.Equal(t, -2147483648, cfg.InterceptorOrder)
}

func TestTmConfig_ExtremeValues(t *testing.T) {
	cfg := &TmConfig{
		CommitRetryCount:                2147483647, // Max int32
		RollbackRetryCount:              2147483647, // Max int32
		DefaultGlobalTransactionTimeout: 24 * time.Hour,
		DegradeCheck:                    true,
		DegradeCheckPeriod:              2147483647, // Max int32
		DegradeCheckAllowTimes:          24 * time.Hour,
		InterceptorOrder:                2147483647, // Max int32
	}

	assert.Equal(t, 2147483647, cfg.CommitRetryCount)
	assert.Equal(t, 2147483647, cfg.RollbackRetryCount)
	assert.Equal(t, 24*time.Hour, cfg.DefaultGlobalTransactionTimeout)
	assert.True(t, cfg.DegradeCheck)
	assert.Equal(t, 2147483647, cfg.DegradeCheckPeriod)
	assert.Equal(t, 24*time.Hour, cfg.DegradeCheckAllowTimes)
	assert.Equal(t, 2147483647, cfg.InterceptorOrder)
}
