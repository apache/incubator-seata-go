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
	"path/filepath"

	"github.com/seata/seata-go/pkg/config/parser"
	"github.com/seata/seata-go/pkg/util/log"
)

var (
	DefaultSeataConf SeataConf
)

// SeataConf is seata config object
type SeataConf struct {
	Seata Seata `default:"" yaml:"" json:"seata"`
}

// Init seata configurator
func Init() error {
	configPath, err := filepath.Abs("../../conf/seata_config.yml")
	if err != nil {
		log.Errorf("Init seata config error [get config file path error]: {%#v}", err.Error())
		return err
	}
	configByte, err := parser.LoadYMLConfig(configPath)
	if err != nil {
		log.Errorf("Init seata config error [load yml file error]: {%#v}", err.Error())
		return err
	}
	err = parser.UnmarshalYML(configByte, &DefaultSeataConf)
	if err != nil {
		log.Errorf("Init seata config error [unmarshal yml to seataconf error]:{%#v}", err.Error())
		return err
	}
	return nil
}

type Seata struct {
	Enabled                   bool              `yaml:"enabled" json:"enabled,omitempty" property:"enabled"`
	ApplicationID             string            `yaml:"application-id" json:"application-id,omitempty" property:"application-id"`
	TxServiceGroup            string            `yaml:"tx-service-group" json:"tx-service-group,omitempty" property:"tx-service-group"`
	AccessKey                 string            `yaml:"access-key" json:"access-key,omitempty" property:"access-key"`
	SecretKey                 string            `yaml:"secret-key" json:"secret-key,omitempty" property:"secret-key"`
	EnableAutoDataSourceProxy bool              `yaml:"enable-auto-data-source-proxy" json:"enable-auto-data-source-proxy,omitempty" property:"enable-auto-data-source-proxy"`
	DataSourceProxyMode       string            `yaml:"data-source-proxy-mode" json:"data-source-proxy-mode,omitempty" property:"data-source-proxy-mode"`
	ClientConf                ClientConf        `yaml:"client" json:"client,omitempty" property:"client"`
	Service                   Service           `yaml:"service" json:"service,omitempty" property:"service"`
	Transport                 Transport         `yaml:"transport" json:"transport,omitempty" property:"transport"`
	Config                    Config            `yaml:"config" json:"config,omitempty" property:"config"`
	Registry                  Registry          `yaml:"registry" json:"registry,omitempty" property:"registry"`
	LogConf                   LogConf           `yaml:"log" json:"log,omitempty" property:"log"`
	TccConf                   TccConf           `yaml:"tcc" json:"tcc,omitempty" property:"tcc"`
	GettySessionParam         GettySessionParam `yaml:"getty-session-param" json:"getty-session-param,omitempty" property:"getty-session-param"`
}
