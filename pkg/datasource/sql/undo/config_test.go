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

package undo

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/compressor"
)

func TestInitUndoConfig(t *testing.T) {
	config := Config{
		DataValidation:        true,
		LogSerialization:      "json",
		LogTable:              "test_undo_log",
		OnlyCareUpdateColumns: false,
		CompressConfig: CompressConfig{
			Enable:    true,
			Type:      "gzip",
			Threshold: "32k",
		},
	}

	InitUndoConfig(config)

	assert.Equal(t, config.DataValidation, UndoConfig.DataValidation)
	assert.Equal(t, config.LogSerialization, UndoConfig.LogSerialization)
	assert.Equal(t, config.LogTable, UndoConfig.LogTable)
	assert.Equal(t, config.OnlyCareUpdateColumns, UndoConfig.OnlyCareUpdateColumns)
	assert.Equal(t, config.CompressConfig.Enable, UndoConfig.CompressConfig.Enable)
	assert.Equal(t, config.CompressConfig.Type, UndoConfig.CompressConfig.Type)
	assert.Equal(t, config.CompressConfig.Threshold, UndoConfig.CompressConfig.Threshold)
}

func TestConfig_RegisterFlagsWithPrefix(t *testing.T) {
	tests := []struct {
		name         string
		prefix       string
		expectPrefix string
	}{
		{
			name:         "empty prefix",
			prefix:       "",
			expectPrefix: "",
		},
		{
			name:         "with prefix",
			prefix:       "undo",
			expectPrefix: "undo",
		},
		{
			name:         "nested prefix",
			prefix:       "seata.undo",
			expectPrefix: "seata.undo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

			config.RegisterFlagsWithPrefix(tt.prefix, flagSet)

			expectedFlags := []string{
				tt.expectPrefix + ".data-validation",
				tt.expectPrefix + ".log-serialization",
				tt.expectPrefix + ".log-table",
				tt.expectPrefix + ".only-care-update-columns",
				tt.expectPrefix + ".compress.type",
				tt.expectPrefix + ".compress.threshold",
			}

			flagMap := make(map[string]*flag.Flag)
			flagSet.VisitAll(func(f *flag.Flag) {
				flagMap[f.Name] = f
			})

			for _, expectedFlag := range expectedFlags {
				assert.Contains(t, flagMap, expectedFlag, "Flag %s should be registered", expectedFlag)
			}

			dataValidationFlag := flagMap[tt.expectPrefix+".data-validation"]
			assert.Equal(t, "true", dataValidationFlag.DefValue)

			logSerializationFlag := flagMap[tt.expectPrefix+".log-serialization"]
			assert.Equal(t, "json", logSerializationFlag.DefValue)

			logTableFlag := flagMap[tt.expectPrefix+".log-table"]
			assert.Equal(t, "undo_log", logTableFlag.DefValue)

			onlyCareUpdateFlag := flagMap[tt.expectPrefix+".only-care-update-columns"]
			assert.Equal(t, "true", onlyCareUpdateFlag.DefValue)
		})
	}
}

func TestCompressConfig_RegisterFlagsWithPrefix(t *testing.T) {
	tests := []struct {
		name         string
		prefix       string
		expectPrefix string
	}{
		{
			name:         "empty prefix",
			prefix:       "",
			expectPrefix: "",
		},
		{
			name:         "with prefix",
			prefix:       "compress",
			expectPrefix: "compress",
		},
		{
			name:         "nested prefix",
			prefix:       "undo.compress",
			expectPrefix: "undo.compress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &CompressConfig{}
			flagSet := flag.NewFlagSet("test", flag.ContinueOnError)

			config.RegisterFlagsWithPrefix(tt.prefix, flagSet)

			expectedFlags := []string{
				tt.expectPrefix + ".type",
				tt.expectPrefix + ".threshold",
			}

			flagMap := make(map[string]*flag.Flag)
			flagSet.VisitAll(func(f *flag.Flag) {
				flagMap[f.Name] = f
			})

			for _, expectedFlag := range expectedFlags {
				assert.Contains(t, flagMap, expectedFlag, "Flag %s should be registered", expectedFlag)
			}

			typeFlag := flagMap[tt.expectPrefix+".type"]
			assert.Equal(t, string(compressor.CompressorNone), typeFlag.DefValue)

			thresholdFlag := flagMap[tt.expectPrefix+".threshold"]
			assert.Equal(t, "64k", thresholdFlag.DefValue)
		})
	}
}

func TestConfigStructFields(t *testing.T) {
	config := Config{
		DataValidation:        true,
		LogSerialization:      "protobuf",
		LogTable:              "custom_undo_log",
		OnlyCareUpdateColumns: false,
		CompressConfig: CompressConfig{
			Enable:    true,
			Type:      "lz4",
			Threshold: "128k",
		},
	}

	assert.True(t, config.DataValidation)
	assert.Equal(t, "protobuf", config.LogSerialization)
	assert.Equal(t, "custom_undo_log", config.LogTable)
	assert.False(t, config.OnlyCareUpdateColumns)
	assert.True(t, config.CompressConfig.Enable)
	assert.Equal(t, "lz4", config.CompressConfig.Type)
	assert.Equal(t, "128k", config.CompressConfig.Threshold)
}

func TestCompressConfigStructFields(t *testing.T) {
	config := CompressConfig{
		Enable:    false,
		Type:      "zstd",
		Threshold: "256k",
	}

	assert.False(t, config.Enable)
	assert.Equal(t, "zstd", config.Type)
	assert.Equal(t, "256k", config.Threshold)
}
