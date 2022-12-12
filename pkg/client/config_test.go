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

package client

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadPath(t *testing.T) {
	cfg := LoadPath("../../testdata/conf/seatago.yml")
	assert.NotNil(t, cfg)

	assert.NotNil(t, cfg.TCCConfig)
	assert.NotNil(t, cfg.TCCConfig.FenceConfig)
	assert.Equal(t, "tcc_fence_log_test", cfg.TCCConfig.FenceConfig.LogTableName)
	assert.Equal(t, time.Second*60, cfg.TCCConfig.FenceConfig.CleanPeriod)

	assert.NotNil(t, cfg.ClientConfig)
	assert.NotNil(t, cfg.ClientConfig.TmConfig)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.CommitRetryCount)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.RollbackRetryCount)
	assert.Equal(t, time.Second*60, cfg.ClientConfig.TmConfig.DefaultGlobalTransactionTimeout)
	assert.Equal(t, false, cfg.ClientConfig.TmConfig.DegradeCheck)
	assert.Equal(t, 2000, cfg.ClientConfig.TmConfig.DegradeCheckPeriod)
	assert.Equal(t, time.Second*10, cfg.ClientConfig.TmConfig.DegradeCheckAllowTimes)
	assert.Equal(t, -2147482648, cfg.ClientConfig.TmConfig.InterceptorOrder)

	assert.NotNil(t, cfg.GettyConfig)
	assert.Equal(t, true, cfg.GettyConfig.CompressEncoding)
	assert.Equal(t, false, cfg.GettyConfig.TCPNoDelay)
	assert.Equal(t, false, cfg.GettyConfig.TCPKeepAlive)
	assert.Equal(t, time.Minute*2, cfg.GettyConfig.KeepAlivePeriod)
	assert.Equal(t, 261120, cfg.GettyConfig.TCPRBufSize)
	assert.Equal(t, 32768, cfg.GettyConfig.TCPWBufSize)
	assert.Equal(t, time.Second*2, cfg.GettyConfig.TCPReadTimeout)
	assert.Equal(t, time.Second*8, cfg.GettyConfig.TCPWriteTimeout)
	assert.Equal(t, time.Second*2, cfg.GettyConfig.WaitTimeout)
	assert.Equal(t, 261120, cfg.GettyConfig.MaxMsgLen)
	assert.Equal(t, "client_test", cfg.GettyConfig.SessionName)

	// reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestLoadJson(t *testing.T) {
	confJson := `{"client":{"tm":{"commit-retry-count":5,"rollback-retry-count":5,"default-global-transaction-timeout":"60s","degrade-check":false,"degrade-check-period":2000,"degrade-check-allow-times":"10s","interceptor-order":-2147482648}},"tcc":{"fence":{"log-table-name":"tcc_fence_log_test2","clean-period":80000000000}}
				"getty-session-param":{"compress-encoding":true,"tcp-no-delay":false,"tcp-keep-alive":false,"keep-alive-period":"120s","tcp-r-buf-size":261120,"tcp-w-buf-size":32768,"tcp-read-timeout":"2s","tcp-write-timeout":"8s","wait-timeout":"2s","max-msg-len":261120,"session-name":"client_test"}}`
	cfg := LoadJson([]byte(confJson))
	assert.NotNil(t, cfg)

	assert.NotNil(t, cfg.TCCConfig)
	assert.NotNil(t, cfg.TCCConfig.FenceConfig)
	assert.Equal(t, "tcc_fence_log_test2", cfg.TCCConfig.FenceConfig.LogTableName)
	assert.Equal(t, time.Second*80, cfg.TCCConfig.FenceConfig.CleanPeriod)

	assert.NotNil(t, cfg.ClientConfig)
	assert.NotNil(t, cfg.ClientConfig.TmConfig)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.CommitRetryCount)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.RollbackRetryCount)
	assert.Equal(t, time.Second*60, cfg.ClientConfig.TmConfig.DefaultGlobalTransactionTimeout)
	assert.Equal(t, false, cfg.ClientConfig.TmConfig.DegradeCheck)
	assert.Equal(t, 2000, cfg.ClientConfig.TmConfig.DegradeCheckPeriod)
	assert.Equal(t, time.Second*10, cfg.ClientConfig.TmConfig.DegradeCheckAllowTimes)
	assert.Equal(t, -2147482648, cfg.ClientConfig.TmConfig.InterceptorOrder)

	assert.NotNil(t, cfg.GettyConfig)
	assert.Equal(t, true, cfg.GettyConfig.CompressEncoding)
	assert.Equal(t, false, cfg.GettyConfig.TCPNoDelay)
	assert.Equal(t, false, cfg.GettyConfig.TCPKeepAlive)
	assert.Equal(t, time.Minute*2, cfg.GettyConfig.KeepAlivePeriod)
	assert.Equal(t, 261120, cfg.GettyConfig.TCPRBufSize)
	assert.Equal(t, 32768, cfg.GettyConfig.TCPWBufSize)
	assert.Equal(t, time.Second*2, cfg.GettyConfig.TCPReadTimeout)
	assert.Equal(t, time.Second*8, cfg.GettyConfig.TCPWriteTimeout)
	assert.Equal(t, time.Second*2, cfg.GettyConfig.WaitTimeout)
	assert.Equal(t, 261120, cfg.GettyConfig.MaxMsgLen)
	assert.Equal(t, "client_test", cfg.GettyConfig.SessionName)

	// reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}
