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
	"github.com/creasty/defaults"
)
import (
	"github.com/seata/seata-go/pkg/common/constant"
)

// RegistryConfig is the configuration of the registry center
type RegistryConfig struct {
	Type              string                         `yaml:"type"  json:"type,omitempty" property:"type" hcl:"type"`
	PreferredNetworks string                         `yaml:"preferred-networks"  json:"preferred-networks,omitempty" property:"preferred-networks" hcl:"preferred-networks"`
	Registry          map[string]*RegistryInfoConfig `yaml:"registry" json:"registry" property:"registry" hcl:"registry"`
}

type RegistryInfoConfig struct {
	Address        string `yaml:"address" json:"address,omitempty" property:"address" hcl:"address"`
	ServerAddr     string `yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr" hcl:"server-addr"`
	Application    string `yaml:"application" json:"application,omitempty" property:"application" hcl:"application"`
	Group          string `yaml:"group" json:"group,omitempty" property:"group" hcl:"group"`
	Namespace      string `yaml:"namespace" json:"namespace,omitempty" property:"namespace" hcl:"namespace"`
	Username       string `yaml:"username" json:"username,omitempty" property:"username" hcl:"username"`
	Password       string `yaml:"password" json:"password,omitempty"  property:"password" hcl:"password"`
	AccessKey      string `yaml:"access-key" json:"access-key,omitempty"  property:"access-key" hcl:"access-key"`
	SecretKey      string `yaml:"secret-key" json:"secret-key,omitempty" property:"secret-key" hcl:"secret-key"`
	SlbPattern     string `yaml:"slb-pattern" json:"slb-pattern,omitempty" property:"slb-pattern" hcl:"slb-pattern"`
	Db             string `yaml:"db" json:"db,omitempty" property:"db" hcl:"db"`
	Timeout        int64  `yaml:"timeout" json:"timeout,omitempty" property:"timeout" hcl:"timeout"`
	SessionTimeout int64  `default:"6000" yaml:"session-timeout" json:"session-timeout,omitempty" property:"session-timeout" hcl:"session-timeout"`
	ConnectTimeout int64  `default:"2000" yaml:"connect-timeout" json:"connect-timeout,omitempty" property:"connect-timeout" hcl:"connect-timeout"`
}

// Prefix seata.registries
func (RegistryConfig) Prefix() string {
	return constant.RegistryConfigPrefix
}

// Prefix seata.registries
func (RegistryInfoConfig) Prefix() string {
	return constant.RegistryConfigPrefix
}

func (c *RegistryConfig) Init() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return c.startRegistryConfig()
}

func (c *RegistryInfoConfig) Init() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return c.startRegistryConfig()
}

func (c *RegistryConfig) startRegistryConfig() error {
	return verify(c)
}

func (c *RegistryInfoConfig) startRegistryConfig() error {
	return verify(c)
}
