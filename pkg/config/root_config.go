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
	"github.com/seata/seata-go/pkg/common/constant"
	"github.com/seata/seata-go/pkg/common/log"
	"sync"
)

var (
	startOnce sync.Once
)

// RootConfig is the root config
type RootConfig struct {
	ServerConf    *ServerConfig    `yaml:"server" json:"server" property:"server" hcl:"server"`
	StoreConf     *StoreConfig     `yaml:"store" json:"store" property:"store" hcl:"store"`
	RegistryConf  *RegistryConfig  `yaml:"registry" json:"registry" property:"registry" hcl:"registry"`
	MetricsConf   *MetricsConfig   `yaml:"metrics" json:"metrics" property:"metrics" hcl:"metrics"`
	TransportConf *TransportConfig `yaml:"transport" json:"transport" property:"transport" hcl:"transport"`
}

// Prefix seata
func (rc *RootConfig) Prefix() string {
	return constant.Seata
}

func (rc *RootConfig) Init() error {
	//init serverConfig
	if err := rc.ServerConf.Init(); err != nil {
		return err
	}

	// init registry
	//registry := rc.RegistryConf
	//if registry != nil {
	//	for _, reg := range registry.Registry {
	//		if err := reg.Init(); err != nil {
	//			return err
	//		}
	//	}
	//}
	SetRootConfig(*rc)
	rc.Start()
	return nil
}

func (rc RootConfig) Start() {
	startOnce.Do(func() {
		log.Info("stary rootConfig")
		//gracefulShutdownInit()
	})
}

func SetRootConfig(r RootConfig) {
	rootConfig = &r
}
