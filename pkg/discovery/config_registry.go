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
)

// Registry center config.
// Client can connect to registry centers (such as nacos, zookeep, etcd) using these configs, and find seater
// server instances there (seata server should register into the registry center before client starts connecting to registry center).
// Client may also register itself into the registry center to build a micro service system, something like Java Spring Cloud.
type RegistryConfig struct {
	// registry center type, such as file，nacos
	Type  string        `yaml:"type" json:"type" koanf:"type"`
	File  FileRegistry  `yaml:"file" json:"file" koanf:"file"`
	Nacos NacosRegistry `yaml:"nacos" json:"nacos" koanf:"nacos"`
	Etcd3 Etcd3Registry `yaml:"etcd3" json:"etcd3" koanf:"etcd3"`
}

func (cfg *RegistryConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Type, prefix+".type", "file", "The registry type.")
	cfg.File.RegisterFlagsWithPrefix(prefix+".file", f)
	cfg.Nacos.RegisterFlagsWithPrefix(prefix+".nacos", f)
	cfg.Etcd3.RegisterFlagsWithPrefix(prefix+".etcd3", f)
}

type FileRegistry struct {
	Name string `yaml:"name" json:"name" koanf:"name"`
}

func (cfg *FileRegistry) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Name, prefix+".name", "registry.conf", "The file name of registry.")
}

type NacosRegistry struct {
	// This application refers to seata server's application name registered at nacos.
	Application string `yaml:"application" json:"application" koanf:"application"`
	// This is the nacos server address, including port. For example, "127.0.0.1:8848" at local host.
	ServerAddr  string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
	Group       string `yaml:"group" json:"group" koanf:"group"`
	Namespace   string `yaml:"namespace" json:"namespace" koanf:"namespace"`
	Username    string `yaml:"username" json:"username" koanf:"username"`
	Password    string `yaml:"password" json:"password" koanf:"password"`
	AccessKey   string `yaml:"access-key" json:"access-key" koanf:"access-key"`
	SecretKey   string `yaml:"secret-key" json:"secret-key" koanf:"secret-key"`
	ContextPath string `yaml:"context-path" json:"context-path" koanf:"context-path"`
}

func (cfg *NacosRegistry) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Application, prefix+".application", "seata-server", "The application name of registry.")
	f.StringVar(&cfg.ServerAddr, prefix+".server-addr", "", "The server address of registry.")
	f.StringVar(&cfg.Group, prefix+".group", "SEATA_GROUP", "The group of registry.")
	f.StringVar(&cfg.Namespace, prefix+".namespace", "", "The namespace of registry.")
	f.StringVar(&cfg.Username, prefix+".username", "", "The username of registry.")
	f.StringVar(&cfg.Password, prefix+".password", "", "The password of registry.")
	f.StringVar(&cfg.AccessKey, prefix+".access-key", "", "The access key of registry.")
	f.StringVar(&cfg.SecretKey, prefix+".secret-key", "", "The secret key of registry.")
	f.StringVar(&cfg.ContextPath, prefix+".context-path", "", "The context path of registry.")
}

type Etcd3Registry struct {
	Cluster    string `yaml:"cluster" json:"cluster" koanf:"cluster"`
	ServerAddr string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
}

func (cfg *Etcd3Registry) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Cluster, prefix+".cluster", "default", "The server address of registry.")
	f.StringVar(&cfg.ServerAddr, prefix+".server-addr", "http://localhost:2379", "The server address of registry.")
}
