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
	"time"
)

type StreamConfig struct {
	MaxRecvMsgSize      int           `yaml:"max-recv-msg-size" json:"max-recv-msg-size" koanf:"max-recv-msg-size"`
	MaxSendMsgSize      int           `yaml:"max-send-msg-size" json:"max-send-msg-size" koanf:"max-send-msg-size"`
	KeepAliveTime       time.Duration `yaml:"keep-alive-time" json:"keep-alive-time" koanf:"keep-alive-time"`
	KeepAliveTimeout    time.Duration `yaml:"keep-alive-timeout" json:"keep-alive-timeout" koanf:"keep-alive-timeout"`
	PermitWithoutStream bool          `yaml:"permit-without-stream" json:"permit-without-stream" koanf:"permit-without-stream"`
	HeartbeatInterval   time.Duration `yaml:"heartbeat-interval" json:"heartbeat-interval" koanf:"heartbeat-interval"`
	DialTimeout         time.Duration `yaml:"dial-timeout" json:"dial-timeout" koanf:"dial-timeout"`
}

// RegisterFlagsWithPrefix for Config.
func (cfg *StreamConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.MaxRecvMsgSize, prefix+".max-recv-msg-size", 4*1024*1024, "Maximum size of received gRPC message in bytes.")
	f.IntVar(&cfg.MaxSendMsgSize, prefix+".max-send-msg-size", 4*1024*1024, "Maximum size of sent gRPC message in bytes.")
	f.DurationVar(&cfg.KeepAliveTime, prefix+".keep-alive-time", 10*time.Second, "The interval of sending keepalive pings.")
	f.DurationVar(&cfg.KeepAliveTimeout, prefix+".keep-alive-timeout", 3*time.Second, "The timeout for keepalive ping ack before closing the connection.")
	f.BoolVar(&cfg.PermitWithoutStream, prefix+".permit-without-stream", false, "Allow keepalive pings when there are no active streams.")
	f.DurationVar(&cfg.HeartbeatInterval, prefix+".heartbeat-interval", 15*time.Second, "Interval for sending heartbeat messages.")
	f.DurationVar(&cfg.DialTimeout, prefix+".dial-timeout", 5*time.Second, "Timeout for dialing a new gRPC connection.")
}
