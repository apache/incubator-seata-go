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
	"seata.apache.org/seata-go/pkg/protocol"
	remoteConfig "seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/remoting/grpc"
	"seata.apache.org/seata-go/pkg/remoting/loadbalance"
	"seata.apache.org/seata-go/pkg/remoting/processor/client"
	"seata.apache.org/seata-go/pkg/rm"
	gettyRM "seata.apache.org/seata-go/pkg/rm/remoting/getty"
	grpcRM "seata.apache.org/seata-go/pkg/rm/remoting/grpc"
	"seata.apache.org/seata-go/pkg/rm/tcc"
	"seata.apache.org/seata-go/pkg/tm"
	gettyTM "seata.apache.org/seata-go/pkg/tm/transaction/getty"
	grpcTM "seata.apache.org/seata-go/pkg/tm/transaction/grpc"
	"seata.apache.org/seata-go/pkg/util/log"
)

// Init seata client
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
		switch protocol.Protocol(remoteConfig.GetTransportConfig().Protocol) {
		case protocol.ProtocolGRPC:
			tm.SetGlobalTransactionManager(&grpcTM.GrpcGlobalTransactionManager{})
		default:
			tm.SetGlobalTransactionManager(&gettyTM.GettyGlobalTransactionManager{})
		}
	})
}

// initRemoting init remoting
func initRemoting(cfg *Config) {
	seataConfig := remoteConfig.SeataConfig{
		ApplicationID:        cfg.ApplicationID,
		TxServiceGroup:       cfg.TxServiceGroup,
		ServiceVgroupMapping: cfg.ServiceConfig.VgroupMapping,
		ServiceGrouplist:     cfg.ServiceConfig.Grouplist,
	}

	remoteConfig.InitTransportConfig(&cfg.TransportConfig)
	remoteConfig.InitSeataConfig(&seataConfig)
	switch protocol.Protocol(remoteConfig.GetTransportConfig().Protocol) {
	case protocol.ProtocolGRPC:
		grpc.InitGrpc(&cfg.RemotingConfig)
	default:
		getty.InitGetty(&cfg.RemotingConfig)
	}
}

// InitRmClient init client rm client
func initRmClient(cfg *Config) {
	onceInitRmClient.Do(func() {
		log.Init()
		loadbalance.InitLoadBalanceConfig(cfg.ClientConfig.LoadBalanceConfig)
		initRemoting(cfg)
		rm.InitRm(rm.RmConfig{
			Config:         cfg.ClientConfig.RmConfig,
			ApplicationID:  cfg.ApplicationID,
			TxServiceGroup: cfg.TxServiceGroup,
		})
		config.Init(cfg.ClientConfig.RmConfig.LockConfig)
		client.RegisterProcessor()
		integration.Init()
		tcc.InitTCC(cfg.TCCConfig.FenceConfig)
		at.InitAT(cfg.ClientConfig.UndoConfig, cfg.AsyncWorkerConfig)
		at.InitXA(cfg.ClientConfig.XaConfig)
		switch protocol.Protocol(remoteConfig.GetTransportConfig().Protocol) {
		case protocol.ProtocolGRPC:
			rm.SetRMRemotingInstance(&grpcRM.GrpcRMRemoting{})
		default:
			rm.SetRMRemotingInstance(&gettyRM.GettyRMRemoting{})
		}
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
