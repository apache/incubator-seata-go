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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"seata.apache.org/seata-go/pkg/util/flagext"
)

func TestConfig_RegisterFlagsWithPrefix_Defaults(t *testing.T) {
	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("remoting", fs)

	// Test default values
	assert.Equal(t, 0, cfg.ReconnectInterval, "Default ReconnectInterval should be 0")
	assert.Equal(t, 1, cfg.ConnectionNum, "Default ConnectionNum should be 1")
	assert.Equal(t, "XID", cfg.LoadBalanceType, "Default LoadBalanceType should be XID")
}

func TestConfig_RegisterFlagsWithPrefix_CustomValues(t *testing.T) {
	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("remoting", fs)

	// Parse custom values
	args := []string{
		"-remoting.reconnect-interval=5000",
		"-remoting.connection-num=10",
		"-remoting.load-balance-type=ROUND_ROBIN",
	}
	err := fs.Parse(args)
	assert.NoError(t, err, "Flag parsing should not return an error")

	// Verify custom values
	assert.Equal(t, 5000, cfg.ReconnectInterval, "ReconnectInterval should be 5000")
	assert.Equal(t, 10, cfg.ConnectionNum, "ConnectionNum should be 10")
	assert.Equal(t, "ROUND_ROBIN", cfg.LoadBalanceType, "LoadBalanceType should be ROUND_ROBIN")
}

func TestConfig_RegisterFlagsWithPrefix_EmptyPrefix(t *testing.T) {
	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("", fs)

	// Parse with empty prefix
	args := []string{
		"-.reconnect-interval=100",
		"-.connection-num=5",
	}
	err := fs.Parse(args)
	assert.NoError(t, err, "Flag parsing should not return an error")

	assert.Equal(t, 100, cfg.ReconnectInterval)
	assert.Equal(t, 5, cfg.ConnectionNum)
}

func TestConfig_RegisterFlagsWithPrefix_NestedSession(t *testing.T) {
	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("remoting", fs)

	// Parse nested session config values
	args := []string{
		"-remoting.session.tcp-no-delay=false",
		"-remoting.session.max-msg-len=204800",
	}
	err := fs.Parse(args)
	assert.NoError(t, err, "Flag parsing should not return an error")

	assert.False(t, cfg.SessionConfig.TCPNoDelay, "TCPNoDelay should be false")
	assert.Equal(t, 204800, cfg.SessionConfig.MaxMsgLen, "MaxMsgLen should be 204800")
}

func TestShutdownConfig_RegisterFlagsWithPrefix_Defaults(t *testing.T) {
	cfg := &ShutdownConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("shutdown", fs)

	// Test default value
	assert.Equal(t, 3*time.Second, cfg.Wait, "Default Wait should be 3 seconds")
}

func TestShutdownConfig_RegisterFlagsWithPrefix_CustomValue(t *testing.T) {
	cfg := &ShutdownConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("shutdown", fs)

	args := []string{"-shutdown.wait=10s"}
	err := fs.Parse(args)
	assert.NoError(t, err, "Flag parsing should not return an error")

	assert.Equal(t, 10*time.Second, cfg.Wait, "Wait should be 10 seconds")
}

func TestTransportConfig_RegisterFlagsWithPrefix_Defaults(t *testing.T) {
	cfg := &TransportConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("transport", fs)

	// Test default values
	assert.Equal(t, "TCP", cfg.Type, "Default Type should be TCP")
	assert.Equal(t, "NIO", cfg.Server, "Default Server should be NIO")
	assert.True(t, cfg.Heartbeat, "Default Heartbeat should be true")
	assert.Equal(t, "seata", cfg.Serialization, "Default Serialization should be seata")
	assert.Equal(t, "none", cfg.Compressor, "Default Compressor should be none")
	assert.False(t, cfg.EnableTmClientBatchSendRequest, "Default EnableTmClientBatchSendRequest should be false")
	assert.True(t, cfg.EnableRmClientBatchSendRequest, "Default EnableRmClientBatchSendRequest should be true")
	assert.Equal(t, 30*time.Second, cfg.RPCRmRequestTimeout, "Default RPCRmRequestTimeout should be 30 seconds")
	assert.Equal(t, 30*time.Second, cfg.RPCTmRequestTimeout, "Default RPCTmRequestTimeout should be 30 seconds")
	assert.Equal(t, 3*time.Second, cfg.ShutdownConfig.Wait, "Default ShutdownConfig.Wait should be 3 seconds")
}

func TestTransportConfig_RegisterFlagsWithPrefix_CustomValues(t *testing.T) {
	cfg := &TransportConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("transport", fs)

	args := []string{
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
	}
	err := fs.Parse(args)
	assert.NoError(t, err, "Flag parsing should not return an error")

	// Verify custom values
	assert.Equal(t, "UDP", cfg.Type, "Type should be UDP")
	assert.Equal(t, "NETTY", cfg.Server, "Server should be NETTY")
	assert.False(t, cfg.Heartbeat, "Heartbeat should be false")
	assert.Equal(t, "protobuf", cfg.Serialization, "Serialization should be protobuf")
	assert.Equal(t, "gzip", cfg.Compressor, "Compressor should be gzip")
	assert.True(t, cfg.EnableTmClientBatchSendRequest, "EnableTmClientBatchSendRequest should be true")
	assert.False(t, cfg.EnableRmClientBatchSendRequest, "EnableRmClientBatchSendRequest should be false")
	assert.Equal(t, 60*time.Second, cfg.RPCRmRequestTimeout, "RPCRmRequestTimeout should be 60 seconds")
	assert.Equal(t, 45*time.Second, cfg.RPCTmRequestTimeout, "RPCTmRequestTimeout should be 45 seconds")
	assert.Equal(t, 5*time.Second, cfg.ShutdownConfig.Wait, "ShutdownConfig.Wait should be 5 seconds")
}

func TestTransportConfig_RegisterFlagsWithPrefix_BooleanFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected TransportConfig
	}{
		{
			name: "All boolean flags true",
			args: []string{
				"-transport.heartbeat=true",
				"-transport.enable-tm-client-batch-send-request=true",
				"-transport.enable-rm-client-batch-send-request=true",
			},
			expected: TransportConfig{
				Heartbeat:                      true,
				EnableTmClientBatchSendRequest: true,
				EnableRmClientBatchSendRequest: true,
			},
		},
		{
			name: "All boolean flags false",
			args: []string{
				"-transport.heartbeat=false",
				"-transport.enable-tm-client-batch-send-request=false",
				"-transport.enable-rm-client-batch-send-request=false",
			},
			expected: TransportConfig{
				Heartbeat:                      false,
				EnableTmClientBatchSendRequest: false,
				EnableRmClientBatchSendRequest: false,
			},
		},
		{
			name: "Mixed boolean flags",
			args: []string{
				"-transport.heartbeat=false",
				"-transport.enable-tm-client-batch-send-request=true",
				"-transport.enable-rm-client-batch-send-request=false",
			},
			expected: TransportConfig{
				Heartbeat:                      false,
				EnableTmClientBatchSendRequest: true,
				EnableRmClientBatchSendRequest: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &TransportConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("transport", fs)

			err := fs.Parse(tt.args)
			assert.NoError(t, err)

			assert.Equal(t, tt.expected.Heartbeat, cfg.Heartbeat)
			assert.Equal(t, tt.expected.EnableTmClientBatchSendRequest, cfg.EnableTmClientBatchSendRequest)
			assert.Equal(t, tt.expected.EnableRmClientBatchSendRequest, cfg.EnableRmClientBatchSendRequest)
		})
	}
}

func TestSeataConfig_InitAndGet(t *testing.T) {
	// Reset global config before test
	seataConfig = nil

	testConfig := &SeataConfig{
		ApplicationID:  "test-app",
		TxServiceGroup: "test-group",
	}

	InitConfig(testConfig)

	retrievedConfig := GetSeataConfig()
	assert.NotNil(t, retrievedConfig, "GetSeataConfig should return non-nil config")
	assert.Equal(t, "test-app", retrievedConfig.ApplicationID, "ApplicationID should match")
	assert.Equal(t, "test-group", retrievedConfig.TxServiceGroup, "TxServiceGroup should match")
}

func TestSeataConfig_InitAndGet_Nil(t *testing.T) {
	// Reset global config
	seataConfig = nil

	retrievedConfig := GetSeataConfig()
	assert.Nil(t, retrievedConfig, "GetSeataConfig should return nil when not initialized")
}

func TestSeataConfig_InitAndGet_Overwrite(t *testing.T) {
	// Initialize first config
	firstConfig := &SeataConfig{
		ApplicationID:  "first-app",
		TxServiceGroup: "first-group",
	}
	InitConfig(firstConfig)

	// Overwrite with second config
	secondConfig := &SeataConfig{
		ApplicationID:  "second-app",
		TxServiceGroup: "second-group",
	}
	InitConfig(secondConfig)

	retrievedConfig := GetSeataConfig()
	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, "second-app", retrievedConfig.ApplicationID, "ApplicationID should be from second config")
	assert.Equal(t, "second-group", retrievedConfig.TxServiceGroup, "TxServiceGroup should be from second config")
}

func TestSeataConfig_AllFields(t *testing.T) {
	seataConfig = nil

	testConfig := &SeataConfig{
		ApplicationID:        "test-application",
		TxServiceGroup:       "test-tx-group",
		ServiceVgroupMapping: flagext.StringMap{"key1": "value1"},
		ServiceGrouplist:     flagext.StringMap{"key2": "value2"},
		LoadBalanceType:      "RANDOM",
	}

	InitConfig(testConfig)

	retrievedConfig := GetSeataConfig()
	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, "test-application", retrievedConfig.ApplicationID)
	assert.Equal(t, "test-tx-group", retrievedConfig.TxServiceGroup)
	assert.NotNil(t, retrievedConfig.ServiceVgroupMapping)
	assert.NotNil(t, retrievedConfig.ServiceGrouplist)
	assert.Equal(t, "RANDOM", retrievedConfig.LoadBalanceType)
}

func TestConfig_CompleteIntegration(t *testing.T) {
	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("remoting", fs)

	// Parse a complete set of flags
	args := []string{
		"-remoting.reconnect-interval=1000",
		"-remoting.connection-num=5",
		"-remoting.load-balance-type=CONSISTENT_HASH",
		"-remoting.session.compress-encoding=true",
		"-remoting.session.tcp-no-delay=false",
		"-remoting.session.tcp-keep-alive=true",
		"-remoting.session.keep-alive-period=2m",
		"-remoting.session.tcp-r-buf-size=131072",
		"-remoting.session.tcp-w-buf-size=65536",
		"-remoting.session.tcp-read-timeout=3s",
		"-remoting.session.tcp-write-timeout=7s",
		"-remoting.session.wait-timeout=2s",
		"-remoting.session.max-msg-len=204800",
		"-remoting.session.session-name=integration-test",
		"-remoting.session.cron-period=5s",
	}

	err := fs.Parse(args)
	assert.NoError(t, err)

	// Verify main config
	assert.Equal(t, 1000, cfg.ReconnectInterval)
	assert.Equal(t, 5, cfg.ConnectionNum)
	assert.Equal(t, "CONSISTENT_HASH", cfg.LoadBalanceType)

	// Verify nested session config
	assert.True(t, cfg.SessionConfig.CompressEncoding)
	assert.False(t, cfg.SessionConfig.TCPNoDelay)
	assert.True(t, cfg.SessionConfig.TCPKeepAlive)
	assert.Equal(t, 2*time.Minute, cfg.SessionConfig.KeepAlivePeriod)
	assert.Equal(t, 131072, cfg.SessionConfig.TCPRBufSize)
	assert.Equal(t, 65536, cfg.SessionConfig.TCPWBufSize)
	assert.Equal(t, 3*time.Second, cfg.SessionConfig.TCPReadTimeout)
	assert.Equal(t, 7*time.Second, cfg.SessionConfig.TCPWriteTimeout)
	assert.Equal(t, 2*time.Second, cfg.SessionConfig.WaitTimeout)
	assert.Equal(t, 204800, cfg.SessionConfig.MaxMsgLen)
	assert.Equal(t, "integration-test", cfg.SessionConfig.SessionName)
	assert.Equal(t, 5*time.Second, cfg.SessionConfig.CronPeriod)
}

func TestTransportConfig_CompleteIntegration(t *testing.T) {
	cfg := &TransportConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("transport", fs)

	args := []string{
		"-transport.type=WEBSOCKET",
		"-transport.server=UNDERTOW",
		"-transport.heartbeat=false",
		"-transport.serialization=json",
		"-transport.compressor=snappy",
		"-transport.enable-tm-client-batch-send-request=true",
		"-transport.enable-rm-client-batch-send-request=true",
		"-transport.rpc-rm-request-timeout=120s",
		"-transport.rpc-tm-request-timeout=90s",
		"-transport.shutdown.wait=15s",
	}

	err := fs.Parse(args)
	assert.NoError(t, err)

	assert.Equal(t, "WEBSOCKET", cfg.Type)
	assert.Equal(t, "UNDERTOW", cfg.Server)
	assert.False(t, cfg.Heartbeat)
	assert.Equal(t, "json", cfg.Serialization)
	assert.Equal(t, "snappy", cfg.Compressor)
	assert.True(t, cfg.EnableTmClientBatchSendRequest)
	assert.True(t, cfg.EnableRmClientBatchSendRequest)
	assert.Equal(t, 120*time.Second, cfg.RPCRmRequestTimeout)
	assert.Equal(t, 90*time.Second, cfg.RPCTmRequestTimeout)
	assert.Equal(t, 15*time.Second, cfg.ShutdownConfig.Wait)
}

func TestConfig_ZeroValues(t *testing.T) {
	cfg := &Config{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg.RegisterFlagsWithPrefix("remoting", fs)

	args := []string{
		"-remoting.reconnect-interval=0",
		"-remoting.connection-num=0",
		"-remoting.session.tcp-r-buf-size=0",
		"-remoting.session.tcp-w-buf-size=0",
		"-remoting.session.max-msg-len=0",
	}

	err := fs.Parse(args)
	assert.NoError(t, err)

	assert.Equal(t, 0, cfg.ReconnectInterval)
	assert.Equal(t, 0, cfg.ConnectionNum)
	assert.Equal(t, 0, cfg.SessionConfig.TCPRBufSize)
	assert.Equal(t, 0, cfg.SessionConfig.TCPWBufSize)
	assert.Equal(t, 0, cfg.SessionConfig.MaxMsgLen)
}
