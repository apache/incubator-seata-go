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

	"seata.apache.org/seata-go/pkg/util/flagext"
)

type ServiceConfig struct {
	VgroupMapping            flagext.StringMap `yaml:"vgroup-mapping" json:"vgroup-mapping" koanf:"vgroup-mapping"`
	Grouplist                flagext.StringMap `yaml:"grouplist" json:"grouplist" koanf:"grouplist"`
	EnableDegrade            bool              `yaml:"enable-degrade" json:"enable-degrade" koanf:"enable-degrade"`
	DisableGlobalTransaction bool              `yaml:"disable-global-transaction" json:"disable-global-transaction" koanf:"disable-global-transaction"`
}

func (cfg *ServiceConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.BoolVar(&cfg.EnableDegrade, prefix+".enable-degrade", false, "degrade current not support.")
	f.BoolVar(&cfg.DisableGlobalTransaction, prefix+".disable-global-transaction", false, "disable globalTransaction.")
	f.Var(&cfg.VgroupMapping, prefix+".vgroup-mapping", "The vgroup mapping.")
	f.Var(&cfg.Grouplist, prefix+".grouplist", "The group list.")
}

type RegistryConfig struct {
	Type             string      `yaml:"type" json:"type" koanf:"type"`
	NamingserverAddr string      `yaml:"namingserver-addr" json:"namingserver-addr" koanf:"namingserver-addr"`
	Username         string      `yaml:"username" json:"username" koanf:"username"`
	Password         string      `yaml:"password" json:"password" koanf:"password"`
	File             FileConfig  `yaml:"file" json:"file" koanf:"file"`
	Nacos            NacosConfig `yaml:"nacos" json:"nacos" koanf:"nacos"`
	Etcd3            Etcd3Config `yaml:"etcd3" json:"etcd3" koanf:"etcd3"`
	Raft             RaftConfig  `yaml:"raft" json:"raft" koanf:"raft"`
}

func (cfg *RegistryConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Type, prefix+".type", "file", "The registry type.")
	f.StringVar(&cfg.Username, prefix+".username", "seata", "Username for authentication")
	f.StringVar(&cfg.Password, prefix+".password", "seata", "Password for authentication")
	cfg.File.RegisterFlagsWithPrefix(prefix+".file", f)
	cfg.Nacos.RegisterFlagsWithPrefix(prefix+".nacos", f)
	cfg.Etcd3.RegisterFlagsWithPrefix(prefix+".etcd3", f)
	cfg.Raft.RegisterFlagsWithPrefix(prefix+".raft", f)
}

type FileConfig struct {
	Name string `yaml:"name" json:"name" koanf:"name"`
}

func (cfg *FileConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Name, prefix+".name", "registry.conf", "The file name of registry.")
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

func (cfg *NacosConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Application, prefix+".application", "seata", "The application name of registry.")
	f.StringVar(&cfg.ServerAddr, prefix+".server-addr", "", "The server address of registry.")
	f.StringVar(&cfg.Group, prefix+".group", "SEATA_GROUP", "The group of registry.")
	f.StringVar(&cfg.Namespace, prefix+".namespace", "", "The namespace of registry.")
	f.StringVar(&cfg.Username, prefix+".username", "", "The username of registry.")
	f.StringVar(&cfg.Password, prefix+".password", "", "The password of registry.")
	f.StringVar(&cfg.AccessKey, prefix+".access-key", "", "The access key of registry.")
	f.StringVar(&cfg.SecretKey, prefix+".secret-key", "", "The secret key of registry.")
}

type Etcd3Config struct {
	Cluster    string `yaml:"cluster" json:"cluster" koanf:"cluster"`
	ServerAddr string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
}

func (cfg *Etcd3Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Cluster, prefix+".cluster", "default", "The server address of registry.")
	f.StringVar(&cfg.ServerAddr, prefix+".server-addr", "http://localhost:2379", "The server address of registry.")
}

type RaftConfig struct {
	MetadataMaxAgeMs            int64  `yaml:"metadata-max-age-ms" json:"metadata-max-age-ms" koanf:"metadata-max-age-ms"`
	ServerAddr                  string `yaml:"serverAddr" json:"server-addr" koanf:"server-addr"`
	TokenValidityInMilliseconds int64  `yaml:"token-validity-in-milliseconds" json:"token-validity-in-milliseconds" koanf:"token-validity-in-milliseconds"`
}

func (cfg *RaftConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.Int64Var(&cfg.MetadataMaxAgeMs, prefix+".metadata-max-age-ms", 30000, "Maximum age of metadata in milliseconds before refresh")
	f.StringVar(&cfg.ServerAddr, prefix+".server-addr", "127.0.0.1:7091", "The server address of raft registry")
	f.Int64Var(&cfg.TokenValidityInMilliseconds, prefix+".token-validity-in-milliseconds", 1740000, "Token validity duration in milliseconds")
}
