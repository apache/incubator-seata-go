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
	// reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestLoadJson(t *testing.T) {
	confJson := `{"tcc":{"fence":{"log-table-name":"tcc_fence_log_test2","clean-period":80000000000}}}`
	cfg := LoadJson([]byte(confJson))
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.TCCConfig)
	assert.NotNil(t, cfg.TCCConfig.FenceConfig)

	assert.Equal(t, "tcc_fence_log_test2", cfg.TCCConfig.FenceConfig.LogTableName)
	assert.Equal(t, time.Second*80, cfg.TCCConfig.FenceConfig.CleanPeriod)
	// reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}
