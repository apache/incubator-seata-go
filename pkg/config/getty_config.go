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
	"time"
)

// GettyConfig
// Config holds supported types by the multiconfig package
type GettyConfig struct {
	ReconnectInterval int `default:"0" yaml:"reconnect_interval" json:"reconnect_interval,omitempty"`
	// getty_session pool
	ConnectionNum int `default:"16" yaml:"connection_number" json:"connection_number,omitempty"`

	// heartbeat
	HeartbeatPeriod time.Duration `default:"15s" yaml:"heartbeat_period" json:"heartbeat_period,omitempty"`

	// getty_session tcp parameters
	GettySessionParam GettySessionParam `required:"true" yaml:"getty_session_param" json:"getty_session_param,omitempty"`
}

// GetDefaultGettyConfig ...
func GetDefaultGettyConfig() GettyConfig {
	return GettyConfig{
		ReconnectInterval: 0,
		ConnectionNum:     1,
		HeartbeatPeriod:   10 * time.Second,
		GettySessionParam: GettySessionParam{
			CompressEncoding: false,
			TCPNoDelay:       true,
			TCPKeepAlive:     true,
			KeepAlivePeriod:  180 * time.Second,
			TCPRBufSize:      2144,
			TCPWBufSize:      65536,
			TCPReadTimeout:   time.Second,
			TCPWriteTimeout:  5 * time.Second,
			WaitTimeout:      time.Second,
			CronPeriod:       time.Second,
			MaxMsgLen:        4096,
			SessionName:      "rpc_client",
		},
	}
}

type Shutdown struct {
	Wait time.Duration `yaml:"wait" json:"wait,omitempty" property:"wait"`
}

type Transport struct {
	Shutdown                       Shutdown      `yaml:"shutdown" json:"shutdown,omitempty" property:"shutdown"`
	Type                           string        `yaml:"type" json:"type,omitempty" property:"type"`
	Server                         string        `yaml:"server" json:"server,omitempty" property:"server"`
	Heartbeat                      bool          `yaml:"heartbeat" json:"heartbeat,omitempty" property:"heartbeat"`
	Serialization                  string        `yaml:"serialization" json:"serialization,omitempty" property:"serialization"`
	Compressor                     string        `yaml:"compressor" json:"compressor,omitempty" property:"compressor"`
	EnableTmClientBatchSendRequest bool          `yaml:"enable-tm-client-batch-send-request" json:"enable-tm-client-batch-send-request,omitempty" property:"enable-tm-client-batch-send-request"`
	EnableRmClientBatchSendRequest bool          `yaml:"enable-rm-client-batch-send-request" json:"enable-rm-client-batch-send-request,omitempty" property:"enable-rm-client-batch-send-request"`
	RPCRmRequestTimeout            time.Duration `yaml:"rpc-rm-request-timeout" json:"rpc-rm-request-timeout,omitempty" property:"rpc-rm-request-timeout"`
	RPCTmRequestTimeout            time.Duration `yaml:"rpc-tm-request-timeout" json:"rpc-tm-request-timeout,omitempty" property:"rpc-tm-request-timeout"`
}

type GettySessionParam struct {
	CompressEncoding bool          `yaml:"compress-encoding" json:"compress-encoding,omitempty" property:"compress-encoding"`
	TCPNoDelay       bool          `yaml:"tcp-no-delay" json:"tcp-no-delay,omitempty" property:"tcp-no-delay"`
	TCPKeepAlive     bool          `yaml:"tcp-keep-alive" json:"tcp-keep-alive,omitempty" property:"tcp-keep-alive"`
	KeepAlivePeriod  time.Duration `yaml:"keep-alive-period" json:"keep-alive-period,omitempty" property:"keep-alive-period"`
	TCPRBufSize      int           `yaml:"tcp-r-buf-size" json:"tcp-r-buf-size,omitempty" property:"tcp-r-buf-size"`
	TCPWBufSize      int           `yaml:"tcp-w-buf-size" json:"tcp-w-buf-size,omitempty" property:"tcp-w-buf-size"`
	TCPReadTimeout   time.Duration `yaml:"tcp-read-timeout" json:"tcp-read-timeout,omitempty" property:"tcp-read-timeout"`
	TCPWriteTimeout  time.Duration `yaml:"tcp-write-timeout" json:"tcp-write-timeout,omitempty" property:"tcp-write-timeout"`
	WaitTimeout      time.Duration `yaml:"wait-timeout" json:"wait-timeout,omitempty" property:"wait-timeout"`
	MaxMsgLen        int           `yaml:"max-msg-len" json:"max-msg-len,omitempty" property:"max-msg-len"`
	SessionName      string        `yaml:"session-name" json:"session-name,omitempty" property:"session-name"`
	CronPeriod       time.Duration `default:"1" yaml:"cron_period" json:"cron_period,omitempty"`
}
