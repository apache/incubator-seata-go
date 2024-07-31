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

type SessionConfig struct {
	CompressEncoding bool          `yaml:"compress-encoding" json:"compress-encoding" koanf:"compress-encoding"`
	TCPNoDelay       bool          `yaml:"tcp-no-delay" json:"tcp-no-delay" koanf:"tcp-no-delay"`
	TCPKeepAlive     bool          `yaml:"tcp-keep-alive" json:"tcp-keep-alive" koanf:"tcp-keep-alive"`
	KeepAlivePeriod  time.Duration `yaml:"keep-alive-period" json:"keep-alive-period" koanf:"keep-alive-period"`
	TCPRBufSize      int           `yaml:"tcp-r-buf-size" json:"tcp-r-buf-size" koanf:"tcp-r-buf-size"`
	TCPWBufSize      int           `yaml:"tcp-w-buf-size" json:"tcp-w-buf-size" koanf:"tcp-w-buf-size"`
	TCPReadTimeout   time.Duration `yaml:"tcp-read-timeout" json:"tcp-read-timeout" koanf:"tcp-read-timeout"`
	TCPWriteTimeout  time.Duration `yaml:"tcp-write-timeout" json:"tcp-write-timeout" koanf:"tcp-write-timeout"`
	WaitTimeout      time.Duration `yaml:"wait-timeout" json:"wait-timeout" koanf:"wait-timeout"`
	MaxMsgLen        int           `yaml:"max-msg-len" json:"max-msg-len" koanf:"max-msg-len"`
	SessionName      string        `yaml:"session-name" json:"session-name" koanf:"session-name"`
	CronPeriod       time.Duration `yaml:"cron-period" json:"cron-period" koanf:"cron-period"`
}

// RegisterFlagsWithPrefix for Config.
func (cfg *SessionConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.CompressEncoding, prefix+".compress-encoding", false, "Enable compress encoding")
	f.BoolVar(&cfg.TCPNoDelay, prefix+".tcp-no-delay", true, "Disable the nagle algorithm.")
	f.BoolVar(&cfg.TCPKeepAlive, prefix+".tcp-keep-alive", true, "Keep connection alive.")
	f.DurationVar(&cfg.KeepAlivePeriod, prefix+".keep-alive-period", 3*time.Minute, "Period between keep-alives.")
	f.IntVar(&cfg.TCPRBufSize, prefix+".tcp-r-buf-size", 262144, "The size of the socket receive buffer.")
	f.IntVar(&cfg.TCPWBufSize, prefix+".tcp-w-buf-size", 65536, "The size of the socket send buffer.")
	f.DurationVar(&cfg.TCPReadTimeout, prefix+".tcp-read-timeout", time.Second, "The read timeout of the channel.")
	f.DurationVar(&cfg.TCPWriteTimeout, prefix+".tcp-write-timeout", 5*time.Second, "The write timeout of the channel.")
	f.DurationVar(&cfg.WaitTimeout, prefix+".wait-timeout", time.Second, "Maximum wait time when session got error or got exit signal.")
	f.IntVar(&cfg.MaxMsgLen, prefix+".max-msg-len", 102400, "maximum package length of every package in (EventListener)OnMessage(@pkgs).")
	f.StringVar(&cfg.SessionName, prefix+".session-name", "client", "The session name.")
	f.DurationVar(&cfg.CronPeriod, prefix+".cron-period", time.Second, "The session heartbeat period.")
}
