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

package discovery

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestRaftConfigYAMLServerAddr(t *testing.T) {
	type testConfig struct {
		Raft RaftConfig `yaml:"raft"`
	}

	input := []byte(`raft:
  metadata-max-age-ms: 30000
  server-addr: 127.0.0.1:7091
  token-validity-in-milliseconds: 1740000
`)

	var cfg testConfig
	err := yaml.Unmarshal(input, &cfg)
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1:7091", cfg.Raft.ServerAddr)
	assert.Equal(t, int64(30000), cfg.Raft.MetadataMaxAgeMs)
	assert.Equal(t, int64(1740000), cfg.Raft.TokenValidityInMilliseconds)
}

func TestRegistryConfigRegisterFlagsWithPrefix(t *testing.T) {
	cfg := &RegistryConfig{}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg.RegisterFlagsWithPrefix("registry", fs)

	err := fs.Parse([]string{
		"-registry.namingserver-addr=127.0.0.1:8081",
		"-registry.raft.server-addr=127.0.0.1:7091",
	})
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1:8081", cfg.NamingserverAddr)
	assert.Equal(t, "127.0.0.1:7091", cfg.Raft.ServerAddr)
}
