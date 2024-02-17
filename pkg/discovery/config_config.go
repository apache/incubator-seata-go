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

// Configuration center for (micro) services.
// Config for configuration center self. Using this config, we can connect to configuration center. In most cases, configuration
// center is using the registry type's configuration, such as nacos configuration center, kubernetes configmap, etc.
// For example, with registry type being `nacos`, seata server's configuration is usaually saved in nacos configuration `data-id=seataServer.Properties`.
// Client could push custom configs using go-SDK provided by configuration center (as mentioned above, in most cases, it's the registry center),
// so services registered within registry center can make use of these custom configs.
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
