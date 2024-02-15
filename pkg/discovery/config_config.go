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

// Configuration center for (micro) service
// 配置中心的自身配置，使用该配置，可以连接到配置中心(配置中心一般直接使用的是注册中心的配置服务，比如nacos的Configuration, k8s的configmap).
// 例如：注册中心是nacos，一般seata server的配置保存在nacos的一个`data-id=seataServer.Properties`的配置中.
// 客服端可以使用配置中心（一般也就是注册中心）提供的Go SDK，自己推送特定的配置到配置中心.
type ConfigConfig struct {
	Type  string      `yaml:"type" json:"type" koanf:"type"`
	File  FileConfig  `yaml:"file" json:"file" koanf:"file"`
	Nacos NacosConfig `yaml:"nacos" json:"nacos" koanf:"nacos"`
	// @todo add other configs, such as etcd.
}

func (cfg *ConfigConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Type, prefix+".type", "file", "The config type.")
	cfg.Nacos.RegisterFlagsWithPrefix(prefix+".nacos", f)
}

type FileConfig struct {
	Name string `yaml:"name" json:"name" koanf:"name"`
}

func (cfg *FileConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.Name, prefix+".name", "config.conf", "The file name of config.")
}

type NacosConfig struct {
	// nacos server address
	ServerAddr  string `yaml:"server-addr" json:"server-addr" koanf:"server-addr"`
	Group       string `yaml:"group" json:"group" koanf:"group"`
	Namespace   string `yaml:"namespace" json:"namespace" koanf:"namespace"`
	Username    string `yaml:"username" json:"username" koanf:"username"`
	Password    string `yaml:"password" json:"password" koanf:"password"`
	AccessKey   string `yaml:"access-key" json:"access-key" koanf:"access-key"`
	SecretKey   string `yaml:"secret-key" json:"secret-key" koanf:"secret-key"`
	ContextPath string `yaml:"context-path" json:"context-path" koanf:"context-path"`
}

func (cfg *NacosConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.StringVar(&cfg.ServerAddr, prefix+".server-addr", "", "The server address of registry.")
	f.StringVar(&cfg.Group, prefix+".group", "SEATA_GROUP", "The group of registry.")
	f.StringVar(&cfg.Namespace, prefix+".namespace", "", "The namespace of registry.")
	f.StringVar(&cfg.Username, prefix+".username", "", "The username of registry.")
	f.StringVar(&cfg.Password, prefix+".password", "", "The password of registry.")
	f.StringVar(&cfg.AccessKey, prefix+".access-key", "", "The access key of registry.")
	f.StringVar(&cfg.SecretKey, prefix+".secret-key", "", "The secret key of registry.")
	f.StringVar(&cfg.ContextPath, prefix+".context-path", "", "The context path of registry.")
}
