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
	"flag"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/util/flagext"
)

func TestConfig_RegisterFlagsWithPrefix(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected Config
	}{
		{
			name: "Defaults",
			args: []string{},
			expected: Config{
				ReconnectInterval: 0,
				ConnectionNum:     1,
			},
		},
		{
			name: "Custom Values",
			args: []string{
				"-remoting.reconnect-interval=5000",
				"-remoting.connection-num=10",
			},
			expected: Config{
				ReconnectInterval: 5000,
				ConnectionNum:     10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("remoting", fs)
			_ = fs.Parse(tt.args)
			assert.Equal(t, tt.expected.ReconnectInterval, cfg.ReconnectInterval)
			assert.Equal(t, tt.expected.ConnectionNum, cfg.ConnectionNum)
		})
	}
}

func TestShutdownConfig_RegisterFlagsWithPrefix(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected time.Duration
	}{
		{"Default", []string{}, 3 * time.Second},
		{"Custom", []string{"-shutdown.wait=10s"}, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ShutdownConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("shutdown", fs)
			_ = fs.Parse(tt.args)
			assert.Equal(t, tt.expected, cfg.Wait)
		})
	}
}

func TestShutdownConfig_KoanfTag(t *testing.T) {
	field, ok := reflect.TypeOf(ShutdownConfig{}).FieldByName("Wait")
	assert.True(t, ok)
	assert.Equal(t, "wait", field.Tag.Get("koanf"))
}

func TestTransportConfig_RegisterFlagsWithPrefix(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected TransportConfig
	}{
		{
			name: "Defaults",
			args: []string{},
			expected: TransportConfig{
				Type:                           "TCP",
				Server:                         "NIO",
				Heartbeat:                      true,
				Serialization:                  "seata",
				Compressor:                     "none",
				EnableTmClientBatchSendRequest: false,
				EnableRmClientBatchSendRequest: true,
				RPCRmRequestTimeout:            30 * time.Second,
				RPCTmRequestTimeout:            30 * time.Second,
				ShutdownConfig:                 ShutdownConfig{Wait: 3 * time.Second},
			},
		},
		{
			name: "Custom Values",
			args: []string{
				"-transport.type=UDP",
				"-transport.server=NETTY",
				"-transport.heartbeat=false",
				"-transport.serialization=protobuf",
				"-transport.compressor=gzip",
				"-transport.enable-tm-client-batch-send-request=true",
				"-transport.enable-rm-client-batch-send-request=false",
				"-transport.rpc-rm-request-timeout=60s",
				"-transport.rpc-tm-request-timeout=45s",
				"-transport.shutdown.wait=5s",
			},
			expected: TransportConfig{
				Type:                           "UDP",
				Server:                         "NETTY",
				Heartbeat:                      false,
				Serialization:                  "protobuf",
				Compressor:                     "gzip",
				EnableTmClientBatchSendRequest: true,
				EnableRmClientBatchSendRequest: false,
				RPCRmRequestTimeout:            60 * time.Second,
				RPCTmRequestTimeout:            45 * time.Second,
				ShutdownConfig:                 ShutdownConfig{Wait: 5 * time.Second},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &TransportConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("transport", fs)
			_ = fs.Parse(tt.args)
			assert.Equal(t, tt.expected.Type, cfg.Type)
			assert.Equal(t, tt.expected.Server, cfg.Server)
			assert.Equal(t, tt.expected.Heartbeat, cfg.Heartbeat)
			assert.Equal(t, tt.expected.Serialization, cfg.Serialization)
			assert.Equal(t, tt.expected.Compressor, cfg.Compressor)
			assert.Equal(t, tt.expected.EnableTmClientBatchSendRequest, cfg.EnableTmClientBatchSendRequest)
			assert.Equal(t, tt.expected.EnableRmClientBatchSendRequest, cfg.EnableRmClientBatchSendRequest)
			assert.Equal(t, tt.expected.RPCRmRequestTimeout, cfg.RPCRmRequestTimeout)
			assert.Equal(t, tt.expected.RPCTmRequestTimeout, cfg.RPCTmRequestTimeout)
			assert.Equal(t, tt.expected.ShutdownConfig.Wait, cfg.ShutdownConfig.Wait)
		})
	}
}

func TestSeataConfig_InitAndGet(t *testing.T) {
	tests := []struct {
		name     string
		initConf *SeataConfig
		expected *SeataConfig
	}{
		{
			name:     "Nil Config",
			initConf: nil,
			expected: nil,
		},
		{
			name: "Basic Config",
			initConf: &SeataConfig{
				ApplicationID:  "test-app",
				TxServiceGroup: "test-group",
			},
			expected: &SeataConfig{
				ApplicationID:  "test-app",
				TxServiceGroup: "test-group",
			},
		},
		{
			name: "Full Config",
			initConf: &SeataConfig{
				ApplicationID:        "app",
				TxServiceGroup:       "group",
				ServiceVgroupMapping: flagext.StringMap{"a": "b"},
				ServiceGrouplist:     flagext.StringMap{"x": "y"},
			},
			expected: &SeataConfig{
				ApplicationID:        "app",
				TxServiceGroup:       "group",
				ServiceVgroupMapping: flagext.StringMap{"a": "b"},
				ServiceGrouplist:     flagext.StringMap{"x": "y"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seataConfig = nil
			if tt.initConf != nil {
				InitSeataConfig(tt.initConf)
			}
			got := GetSeataConfig()
			if tt.expected == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.expected.ApplicationID, got.ApplicationID)
			assert.Equal(t, tt.expected.TxServiceGroup, got.TxServiceGroup)
			assert.Equal(t, tt.expected.ServiceVgroupMapping, got.ServiceVgroupMapping)
			assert.Equal(t, tt.expected.ServiceGrouplist, got.ServiceGrouplist)
		})
	}
}
