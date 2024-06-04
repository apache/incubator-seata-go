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

package client

import (
	"sync"

	"seata.apache.org/seata-go/pkg/datasource"
	at "seata.apache.org/seata-go/pkg/datasource/sql"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec/config"
	"seata.apache.org/seata-go/pkg/discovery"
	"seata.apache.org/seata-go/pkg/integration"
	remoteConfig "seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/remoting/processor/client"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/rm/tcc"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

// Init seata client client
func Init() {
	InitPath("")
}

// InitPath init client with config path
func InitPath(configFilePath string) {
	cfg := LoadPath(configFilePath)
	initRegistry(cfg)
	initRmClient(cfg)
	initTmClient(cfg)
	initDatasource()
}

var (
	onceInitTmClient   sync.Once
	onceInitRmClient   sync.Once
	onceInitDatasource sync.Once
	onceInitRegistry   sync.Once
)

// InitTmClient init client tm client
func initTmClient(cfg *Config) {
	onceInitTmClient.Do(func() {
		tm.InitTm(cfg.ClientConfig.TmConfig)
	})
}

// initRemoting init rpc client
func initRemoting(cfg *Config) {
	getty.InitRpcClient(&cfg.GettyConfig, &remoteConfig.SeataConfig{
		ApplicationID:        cfg.ApplicationID,
		TxServiceGroup:       cfg.TxServiceGroup,
		ServiceVgroupMapping: cfg.ServiceConfig.VgroupMapping,
		ServiceGrouplist:     cfg.ServiceConfig.Grouplist,
		LoadBalanceType:      cfg.GettyConfig.LoadBalanceType,
	})
}

// InitRmClient init client rm client
func initRmClient(cfg *Config) {
	onceInitRmClient.Do(func() {
		log.Init()
		initRemoting(cfg)
		rm.InitRm(rm.RmConfig{
			Config:         cfg.ClientConfig.RmConfig,
			ApplicationID:  cfg.ApplicationID,
			TxServiceGroup: cfg.TxServiceGroup,
		})
		config.Init(cfg.ClientConfig.RmConfig.LockConfig)
		client.RegisterProcessor()
		integration.Init()
		tcc.InitTCC()
		at.InitAT(cfg.ClientConfig.UndoConfig, cfg.AsyncWorkerConfig)
		at.InitXA(cfg.ClientConfig.XaConfig)
	})
}

func initDatasource() {
	onceInitDatasource.Do(func() {
		datasource.Init()
	})
}

func initRegistry(cfg *Config) {
	onceInitRegistry.Do(func() {
		discovery.InitRegistry(&cfg.ServiceConfig, &cfg.RegistryConfig)
	})
}
