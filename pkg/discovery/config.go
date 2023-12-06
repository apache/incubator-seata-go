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

	"github.com/seata/seata-go/pkg/util/flagext"
)

type ServiceConfig struct {
	VgroupMapping            flagext.StringMap `yaml:"vgroup-mapping" json:"vgroup-mapping" koanf:"vgroup-mapping"`
	Grouplist                flagext.StringMap `yaml:"grouplist" json:"grouplist" koanf:"grouplist"`
	EnableDegrade            bool              `yaml:"enable-degrade" json:"enable-degrade" koanf:"enable-degrade"`
	DisableGlobalTransaction bool              `yaml:"disable-global-transaction" json:"disable-global-transaction" koanf:"disable-global-transaction"`
}

func (c *ServiceConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.BoolVar(&c.EnableDegrade, prefix+".enable-degrade", false, "degrade current not support.")
	f.BoolVar(&c.DisableGlobalTransaction, prefix+".disable-global-transaction", false, "disable globalTransaction.")
	f.Var(&c.VgroupMapping, prefix+".vgroup-mapping", "The vgroup mapping.")
	f.Var(&c.Grouplist, prefix+".grouplist", "The group list.")
}

type RegistryConfig struct {
	Type   string        `yaml:"type" json:"type" koanf:"type"`
	File   FileConfig    `yaml:"file" json:"file" koanf:"file"`
	Nacos  NacosConfig   `yaml:"nacos" json:"nacos" koanf:"nacos"`
	Etcd3  Etcd3Config   `yaml:"etcd3" json:"etcd3" koanf:"etcd3"`
	Consul *ConsulConfig `yaml:"consul" json:"consul" koanf:"consul"`
}

func (c *RegistryConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.Type, prefix+".type", "file", "The registry type.")
	c.File.RegisterFlagsWithPrefix(prefix+".file", f)
	c.Nacos.RegisterFlagsWithPrefix(prefix+".nacos", f)
	c.Etcd3.RegisterFlagsWithPrefix(prefix+".etcd3", f)
}

type FileConfig struct {
	Name string `yaml:"name" json:"name" koanf:"name"`
}

func (c *FileConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.Name, prefix+".name", "registry.conf", "The file name of registry.")
}

type NacosConfig struct {
	Application string `yaml:"application" json:"application" koanf:"application"`
	ServerAddr  string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
	Group       string `yaml:"group" json:"group" koanf:"group"`
	Namespace   string `yaml:"namespace" json:"namespace" koanf:"namespace"`
	Username    string `yaml:"username" json:"username" koanf:"username"`
	Password    string `yaml:"password" json:"password" koanf:"password"`
	AccessKey   string `yaml:"access-key" json:"access-key" koanf:"access-key"`
	SecretKey   string `yaml:"secret-key" json:"secret-key" koanf:"secret-key"`
}

func (c *NacosConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.Application, prefix+".application", "seata", "The application name of registry.")
	f.StringVar(&c.ServerAddr, prefix+".server-addr", "", "The server address of registry.")
	f.StringVar(&c.Group, prefix+".group", "SEATA_GROUP", "The group of registry.")
	f.StringVar(&c.Namespace, prefix+".namespace", "", "The namespace of registry.")
	f.StringVar(&c.Username, prefix+".username", "", "The username of registry.")
	f.StringVar(&c.Password, prefix+".password", "", "The password of registry.")
	f.StringVar(&c.AccessKey, prefix+".access-key", "", "The access key of registry.")
	f.StringVar(&c.SecretKey, prefix+".secret-key", "", "The secret key of registry.")
}

type Etcd3Config struct {
	Cluster    string `yaml:"cluster" json:"cluster" koanf:"cluster"`
	ServerAddr string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
}

func (c *Etcd3Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.Cluster, prefix+".cluster", "default", "The server address of registry.")
	f.StringVar(&c.ServerAddr, prefix+".server-addr", "http://localhost:2379", "The server address of registry.")
}

type ConsulConfig struct {
	Cluster    string `yaml:"cluster" json:"cluster" koanf:"cluster"`
	ServerAddr string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
}

func (c *ConsulConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&c.Cluster, prefix+".cluster", "default", "The server address name of registry.")
	f.StringVar(&c.ServerAddr, prefix+".server-addr", "http://localhost:8500", "The server address list of registry.")
}
