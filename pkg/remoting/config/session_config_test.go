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
)

func TestSessionConfig_RegisterFlagsWithPrefix(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected SessionConfig
	}{
		{
			name: "Defaults",
			args: []string{},
			expected: SessionConfig{
				CompressEncoding: false,
				TCPNoDelay:       true,
				TCPKeepAlive:     true,
				KeepAlivePeriod:  3 * time.Minute,
				TCPRBufSize:      262144,
				TCPWBufSize:      65536,
				TCPReadTimeout:   time.Second,
				TCPWriteTimeout:  5 * time.Second,
				WaitTimeout:      time.Second,
				MaxMsgLen:        102400,
				SessionName:      "client",
				CronPeriod:       time.Second,
			},
		},
		{
			name: "Custom Values",
			args: []string{
				"-session.compress-encoding=true",
				"-session.tcp-no-delay=false",
				"-session.tcp-keep-alive=false",
				"-session.keep-alive-period=5m",
				"-session.tcp-r-buf-size=524288",
				"-session.tcp-w-buf-size=131072",
				"-session.tcp-read-timeout=2s",
				"-session.tcp-write-timeout=10s",
				"-session.wait-timeout=3s",
				"-session.max-msg-len=204800",
				"-session.session-name=test-session",
				"-session.cron-period=2s",
			},
			expected: SessionConfig{
				CompressEncoding: true,
				TCPNoDelay:       false,
				TCPKeepAlive:     false,
				KeepAlivePeriod:  5 * time.Minute,
				TCPRBufSize:      524288,
				TCPWBufSize:      131072,
				TCPReadTimeout:   2 * time.Second,
				TCPWriteTimeout:  10 * time.Second,
				WaitTimeout:      3 * time.Second,
				MaxMsgLen:        204800,
				SessionName:      "test-session",
				CronPeriod:       2 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SessionConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("session", fs)
			_ = fs.Parse(tt.args)

			assert.Equal(t, tt.expected.CompressEncoding, cfg.CompressEncoding)
			assert.Equal(t, tt.expected.TCPNoDelay, cfg.TCPNoDelay)
			assert.Equal(t, tt.expected.TCPKeepAlive, cfg.TCPKeepAlive)
			assert.Equal(t, tt.expected.KeepAlivePeriod, cfg.KeepAlivePeriod)
			assert.Equal(t, tt.expected.TCPRBufSize, cfg.TCPRBufSize)
			assert.Equal(t, tt.expected.TCPWBufSize, cfg.TCPWBufSize)
			assert.Equal(t, tt.expected.TCPReadTimeout, cfg.TCPReadTimeout)
			assert.Equal(t, tt.expected.TCPWriteTimeout, cfg.TCPWriteTimeout)
			assert.Equal(t, tt.expected.WaitTimeout, cfg.WaitTimeout)
			assert.Equal(t, tt.expected.MaxMsgLen, cfg.MaxMsgLen)
			assert.Equal(t, tt.expected.SessionName, cfg.SessionName)
			assert.Equal(t, tt.expected.CronPeriod, cfg.CronPeriod)
		})
	}
}

func TestSessionConfig_BooleanCases(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected SessionConfig
	}{
		{
			name: "All True",
			args: []string{
				"-session.compress-encoding=true",
				"-session.tcp-no-delay=true",
				"-session.tcp-keep-alive=true",
			},
			expected: SessionConfig{CompressEncoding: true, TCPNoDelay: true, TCPKeepAlive: true},
		},
		{
			name: "All False",
			args: []string{
				"-session.compress-encoding=false",
				"-session.tcp-no-delay=false",
				"-session.tcp-keep-alive=false",
			},
			expected: SessionConfig{CompressEncoding: false, TCPNoDelay: false, TCPKeepAlive: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SessionConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("session", fs)
			_ = fs.Parse(tt.args)

			assert.Equal(t, tt.expected.CompressEncoding, cfg.CompressEncoding)
			assert.Equal(t, tt.expected.TCPNoDelay, cfg.TCPNoDelay)
			assert.Equal(t, tt.expected.TCPKeepAlive, cfg.TCPKeepAlive)
		})
	}
}

func TestSessionConfig_DurationCases(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected SessionConfig
	}{
		{
			name: "Milliseconds",
			args: []string{
				"-session.keep-alive-period=500ms",
				"-session.tcp-read-timeout=100ms",
				"-session.tcp-write-timeout=200ms",
				"-session.wait-timeout=50ms",
				"-session.cron-period=300ms",
			},
			expected: SessionConfig{
				KeepAlivePeriod: 500 * time.Millisecond,
				TCPReadTimeout:  100 * time.Millisecond,
				TCPWriteTimeout: 200 * time.Millisecond,
				WaitTimeout:     50 * time.Millisecond,
				CronPeriod:      300 * time.Millisecond,
			},
		},
		{
			name: "Minutes",
			args: []string{
				"-session.keep-alive-period=10m",
				"-session.tcp-read-timeout=1m",
				"-session.tcp-write-timeout=2m",
				"-session.wait-timeout=30s",
				"-session.cron-period=45s",
			},
			expected: SessionConfig{
				KeepAlivePeriod: 10 * time.Minute,
				TCPReadTimeout:  time.Minute,
				TCPWriteTimeout: 2 * time.Minute,
				WaitTimeout:     30 * time.Second,
				CronPeriod:      45 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SessionConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("session", fs)
			_ = fs.Parse(tt.args)

			assert.Equal(t, tt.expected.KeepAlivePeriod, cfg.KeepAlivePeriod)
			assert.Equal(t, tt.expected.TCPReadTimeout, cfg.TCPReadTimeout)
			assert.Equal(t, tt.expected.TCPWriteTimeout, cfg.TCPWriteTimeout)
			assert.Equal(t, tt.expected.WaitTimeout, cfg.WaitTimeout)
			assert.Equal(t, tt.expected.CronPeriod, cfg.CronPeriod)
		})
	}
}

func TestSessionConfig_IntegerCases(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected SessionConfig
	}{
		{
			name: "Small Buffers",
			args: []string{
				"-session.tcp-r-buf-size=1024",
				"-session.tcp-w-buf-size=512",
				"-session.max-msg-len=2048",
			},
			expected: SessionConfig{TCPRBufSize: 1024, TCPWBufSize: 512, MaxMsgLen: 2048},
		},
		{
			name: "Large Buffers",
			args: []string{
				"-session.tcp-r-buf-size=1048576",
				"-session.tcp-w-buf-size=524288",
				"-session.max-msg-len=1024000",
			},
			expected: SessionConfig{TCPRBufSize: 1048576, TCPWBufSize: 524288, MaxMsgLen: 1024000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SessionConfig{}
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg.RegisterFlagsWithPrefix("session", fs)
			_ = fs.Parse(tt.args)

			assert.Equal(t, tt.expected.TCPRBufSize, cfg.TCPRBufSize)
			assert.Equal(t, tt.expected.TCPWBufSize, cfg.TCPWBufSize)
			assert.Equal(t, tt.expected.MaxMsgLen, cfg.MaxMsgLen)
		})
	}
}
