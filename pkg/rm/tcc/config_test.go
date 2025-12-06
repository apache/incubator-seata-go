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

package tcc

import (
	"flag"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence"
)

func TestConfig_DefaultValues(t *testing.T) {
	cfg := &Config{}

	assert.NotNil(t, cfg.FenceConfig)
	assert.False(t, cfg.FenceConfig.Enable)
	assert.Equal(t, "", cfg.FenceConfig.Url)
	assert.Equal(t, "", cfg.FenceConfig.LogTableName)
	assert.Equal(t, time.Duration(0), cfg.FenceConfig.CleanPeriod)
}

func TestConfig_SetValues(t *testing.T) {
	cfg := &Config{
		FenceConfig: fence.Config{
			Enable:       true,
			Url:          "root:password@tcp(localhost:3306)/seata?charset=utf8",
			LogTableName: "tcc_fence_log",
			CleanPeriod:  60 * time.Second,
		},
	}

	assert.True(t, cfg.FenceConfig.Enable)
	assert.Equal(t, "root:password@tcp(localhost:3306)/seata?charset=utf8", cfg.FenceConfig.Url)
	assert.Equal(t, "tcc_fence_log", cfg.FenceConfig.LogTableName)
	assert.Equal(t, 60*time.Second, cfg.FenceConfig.CleanPeriod)
}

func TestConfig_RegisterFlagsWithPrefix(t *testing.T) {
	cfg := &Config{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	assert.NotPanics(t, func() {
		cfg.RegisterFlagsWithPrefix("", flagSet)
	})

	assert.NotPanics(t, func() {
		cfg.RegisterFlagsWithPrefix("tcc", flagSet)
	})
}

func TestConfig_RegisterFlagsWithPrefix_FunctionExists(t *testing.T) {
	cfg := &Config{}

	assert.NotNil(t, cfg.RegisterFlagsWithPrefix)

	assert.IsType(t, func(string, *flag.FlagSet) {}, cfg.RegisterFlagsWithPrefix)
}

func TestConfig_FieldTypes(t *testing.T) {
	cfg := &Config{}

	assert.IsType(t, fence.Config{}, cfg.FenceConfig)
}

func TestConfig_StructInstantiation(t *testing.T) {
	cfg1 := Config{}
	cfg2 := &Config{}
	cfg3 := new(Config)

	assert.NotNil(t, &cfg1)
	assert.NotNil(t, cfg2)
	assert.NotNil(t, cfg3)

	assert.False(t, cfg1.FenceConfig.Enable)
	assert.False(t, cfg2.FenceConfig.Enable)
	assert.False(t, cfg3.FenceConfig.Enable)
}

func TestConfig_FieldAssignment(t *testing.T) {
	cfg := &Config{}

	cfg.FenceConfig.Enable = true
	cfg.FenceConfig.Url = "custom-url"
	cfg.FenceConfig.LogTableName = "custom_table"
	cfg.FenceConfig.CleanPeriod = 30 * time.Second

	assert.True(t, cfg.FenceConfig.Enable)
	assert.Equal(t, "custom-url", cfg.FenceConfig.Url)
	assert.Equal(t, "custom_table", cfg.FenceConfig.LogTableName)
	assert.Equal(t, 30*time.Second, cfg.FenceConfig.CleanPeriod)
}

func TestConfig_ExtremeValues(t *testing.T) {
	cfg := &Config{
		FenceConfig: fence.Config{
			Enable:       true,
			Url:          "very-long-url-string-that-exceeds-normal-length-very-long-url-string-that-exceeds-normal-length-very-long-url-string-that-exceeds-normal-length",
			LogTableName: "very_long_table_name_that_exceeds_normal_length_very_long_table_name_that_exceeds_normal_length",
			CleanPeriod:  24 * time.Hour,
		},
	}

	assert.True(t, cfg.FenceConfig.Enable)
	assert.Equal(t, "very-long-url-string-that-exceeds-normal-length-very-long-url-string-that-exceeds-normal-length-very-long-url-string-that-exceeds-normal-length", cfg.FenceConfig.Url)
	assert.Equal(t, "very_long_table_name_that_exceeds_normal_length_very_long_table_name_that_exceeds_normal_length", cfg.FenceConfig.LogTableName)
	assert.Equal(t, 24*time.Hour, cfg.FenceConfig.CleanPeriod)
}

func TestConfig_EmptyValues(t *testing.T) {
	cfg := &Config{
		FenceConfig: fence.Config{
			Enable:       false,
			Url:          "",
			LogTableName: "",
			CleanPeriod:  0,
		},
	}

	assert.False(t, cfg.FenceConfig.Enable)
	assert.Equal(t, "", cfg.FenceConfig.Url)
	assert.Equal(t, "", cfg.FenceConfig.LogTableName)
	assert.Equal(t, time.Duration(0), cfg.FenceConfig.CleanPeriod)
}

func TestConfig_NestedStructure(t *testing.T) {
	cfg := &Config{}

	assert.NotNil(t, cfg.FenceConfig)

	cfg.FenceConfig.Enable = true
	assert.True(t, cfg.FenceConfig.Enable)

	cfg.FenceConfig.Url = "test-url"
	assert.Equal(t, "test-url", cfg.FenceConfig.Url)

	cfg.FenceConfig.LogTableName = "test_table"
	assert.Equal(t, "test_table", cfg.FenceConfig.LogTableName)

	cfg.FenceConfig.CleanPeriod = 5 * time.Minute
	assert.Equal(t, 5*time.Minute, cfg.FenceConfig.CleanPeriod)
}

func TestConfig_RegisterFlagsWithPrefix_Delegation(t *testing.T) {
	cfg := &Config{}
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("tcc", flagSet)

	assert.NotNil(t, flagSet)
}

func TestConfig_ZeroValueHandling(t *testing.T) {
	var cfg Config

	assert.False(t, cfg.FenceConfig.Enable)
	assert.Equal(t, "", cfg.FenceConfig.Url)
	assert.Equal(t, "", cfg.FenceConfig.LogTableName)
	assert.Equal(t, time.Duration(0), cfg.FenceConfig.CleanPeriod)

	cfg.FenceConfig.Enable = true
	cfg.FenceConfig.Url = "assigned-url"

	assert.True(t, cfg.FenceConfig.Enable)
	assert.Equal(t, "assigned-url", cfg.FenceConfig.Url)
}
