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
	Type                    string                   `yaml:"type"  json:"type,omitempty" property:"type" hcl:"type"`
	PreferredNetworks       string                   `yaml:"preferred-networks"  json:"preferred-networks,omitempty" property:"preferred-networks" hcl:"preferred-networks"`
	RegistryZooKeeperConfig *RegistryZooKeeperConfig `yaml:"zk" json:"zk" property:"zk" hcl:"zk"`
	RegistryNacosConfig     *RegistryNacosConfig     `yaml:"nacos" json:"nacos" property:"nacos" hcl:"nacos"`
	RegistryRedisConfig     *RegistryRedisConfig     `yaml:"redis" json:"redis" property:"redis" hcl:"redis"`
	RegistryEurekaConfig    *RegistryEurekaConfig    `yaml:"eureka" json:"eureka" property:"eureka" hcl:"eureka"`
	RegistryConsulConfig    *RegistryConsulConfig    `yaml:"consul" json:"consul" property:"consul" hcl:"consul"`
	RegistryEtcd3Config     *RegistryEtcd3Config     `yaml:"etcd3" json:"etcd3" property:"etcd3" hcl:"etcd3"`
	RegistrySofaPConfig     *RegistrySofaPConfig     `yaml:"sofa" json:"sofa" property:"sofa" hcl:"sofa"`
}

type RegistryZooKeeperConfig struct {
	Cluster        string `default:"default" yaml:"cluster" json:"cluster,omitempty" property:"cluster" hcl:"cluster"`
	ServerAddr     string `default:"127.0.0.1:2181" yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr" hcl:"serverAddr"`
	Username       string `yaml:"username" json:"username,omitempty" property:"username" hcl:"username"`
	Password       string `yaml:"password" json:"password,omitempty"  property:"password" hcl:"password"`
	SessionTimeout int64  `default:"6000" yaml:"session-timeout" json:"session-timeout,omitempty" property:"session-timeout" hcl:"sessionTimeout"`
	ConnectTimeout int64  `default:"2000" yaml:"connect-timeout" json:"connect-timeout,omitempty" property:"connect-timeout" hcl:"connectTimeout"`
}

type RegistryNacosConfig struct {
	ServerAddr  string `default:"localhost:8848" yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr" hcl:"serverAddr"`
	Group       string `default:"SEATA_GROUP" yaml:"group" json:"group,omitempty" property:"group" hcl:"group"`
	Namespace   string `yaml:"namespace" json:"namespace,omitempty" property:"namespace" hcl:"namespace"`
	Cluster     string `default:"default" yaml:"cluster" json:"cluster,omitempty" property:"cluster" hcl:"cluster"`
	Username    string `yaml:"username" json:"username,omitempty" property:"username" hcl:"username"`
	Password    string `yaml:"password" json:"password,omitempty"  property:"password" hcl:"password"`
	AccessKey   string `yaml:"access-key" json:"access-key,omitempty"  property:"access-key" hcl:"accessKey"`
	SecretKey   string `yaml:"secret-key" json:"secret-key,omitempty" property:"secret-key" hcl:"secretKey"`
	Application string `dafault:"seata-server" yaml:"application" json:"application,omitempty" property:"application" hcl:"application"`
	SlbPattern  string `yaml:"slb-pattern" json:"slb-pattern,omitempty" property:"slb-pattern" hcl:"slb-pattern"`
}

type RegistryRedisConfig struct {
	ServerAddr string `default:"localhost:6379" yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr" hcl:"serverAddr"`
	Password   string `yaml:"password" json:"password,omitempty"  property:"password" hcl:"password"`
	Db         int32  `default:"0" yaml:"db" json:"db,omitempty" property:"db" hcl:"db"`
	Cluster    string `default:"default" yaml:"cluster" json:"cluster,omitempty" property:"cluster" hcl:"cluster"`
	Timeout    int64  `default:"0" yaml:"timeout" json:"timeout,omitempty" property:"timeout" hcl:"timeout"`
}

type RegistryEurekaConfig struct {
	ServiceUrl  string `default:"http://localhost:8761/eureka" yaml:"service-url" json:"service-url,omitempty" property:"service-url" hcl:"serviceUrl"`
	Application string `dafault:"default" yaml:"application" json:"application,omitempty" property:"application" hcl:"application"`
	Weight      string `yaml:"weight" json:"weight,omitempty" property:"weight" hcl:"weight"`
}

type RegistryConsulConfig struct {
	Cluster    string `default:"default" yaml:"cluster" json:"cluster,omitempty" property:"cluster" hcl:"cluster"`
	ServerAddr string `default:"127.0.0.1:8500" yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr" hcl:"serverAddr"`
	AclToken   string `yaml:"aclToken" json:"aclToken,omitempty" property:"aclToken" hcl:"aclToken"`
}

type RegistryEtcd3Config struct {
	Cluster    string `default:"default" yaml:"cluster" json:"cluster,omitempty" property:"cluster" hcl:"cluster"`
	ServerAddr string `default:"http://localhost:2379" yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr" hcl:"serverAddr"`
}

type RegistrySofaPConfig struct {
	ServerAddr      string `default:"127.0.0.1:9603" yaml:"server-addr" json:"server-addr,omitempty" property:"server-addr" hcl:"serverAddr"`
	Application     string `dafault:"default" yaml:"application" json:"application,omitempty" property:"application" hcl:"application"`
	Region          string `dafault:"DEFAULT_ZONE" yaml:"region" json:"region,omitempty" property:"region" hcl:"region"`
	Datacenter      string `default:"DefaultDataCenter" yaml:"datacenter" json:"datacenter,omitempty" property:"datacenter" hcl:"datacenter"`
	Cluster         string `default:"default" yaml:"cluster" json:"cluster,omitempty" property:"cluster" hcl:"cluster"`
	Group           string `default:"SEATA_GROUP" yaml:"group" json:"group,omitempty" property:"group" hcl:"group"`
	AddressWaitTime int32  `default:"3000"  yaml:"address-wait-time" json:"addressWaitTime,omitempty"  property:"addressWaitTime" hcl:"addressWaitTime"`
}

// Prefix seata.registries
func (RegistryConfig) Prefix() string {
	return constant.RegistryConfigPrefix
}

func (c *RegistryConfig) Init() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return c.startRegistryConfig()
}

func (c *RegistryConfig) startRegistryConfig() error {
	return verify(c)
}
