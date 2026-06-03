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

package loadbalance

import "flag"

var (
	loadBalanceConfig Config
)

func InitLoadBalanceConfig(cfg Config) {
	loadBalanceConfig = cfg
	virtualNodeNumber = cfg.VirtualNodes
}

type Config struct {
	Type         string `yaml:"type" json:"type" koanf:"type"`
	VirtualNodes int    `yaml:"virtual-nodes" json:"virtual-nodes" koanf:"virtual-nodes"`
}

func (cfg *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Type, prefix+".type", "RandomLoadBalance", "The default load balance type")
	f.IntVar(&cfg.VirtualNodes, prefix+".virtual-nodes", 10, "Used to Consistent Hashing load balance")
}

func GetLoadBalanceConfig() Config {
	return loadBalanceConfig
}
