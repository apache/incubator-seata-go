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
	"github.com/seata/seata-go/pkg/common/constant"
)

type TransportConfig struct {
	Type                            string               `default:"TCP" yaml:"type"  json:"type,omitempty" property:"type"`
	Server                          string               `default:"NIO" yaml:"server"  json:"server,omitempty" property:"server"`
	Heartbeat                       bool                 `default:"true" yaml:"heartbeat"  json:"heartbeat,omitempty" property:"heartbeat"`
	EnableTmClientBatchSendRequest  bool                 `default:"false" yaml:"enable-tm-client-batch-send-request"  json:"enableTmClientBatchSendRequest,omitempty" property:"enableTmClientBatchSendRequest"`
	EnableRmClientBatchSendRequest  bool                 `default:"true" yaml:"enable-rm-client-batch-send-request"  json:"enableRmClientBatchSendRequest,omitempty" property:"enableRmClientBatchSendRequest"`
	EnableTcServerBatchSendResponse bool                 `default:"false" yaml:"enable-tc-server-batch-send-response"  json:"enableTcServerBatchSendResponse,omitempty" property:"enableTcServerBatchSendResponse"`
	RpcRmRequestTimeout             int                  `default:"30000" yaml:"rpc-rm-request-timeout"  json:"rpcRmRequestTimeout,omitempty" property:"rpcRmRequestTimeout"`
	RpcTmRequestTimeout             int                  `default:"30000" yaml:"rpc-tm-request-timeout"  json:"rpcTmRequestTimeout,omitempty" property:"rpcTmRequestTimeout"`
	RpcTcRequestTimeout             int                  `default:"30000" yaml:"rpc-tc-request-timeout"  json:"rpcTcRequestTimeout,omitempty" property:"rpcTcRequestTimeout"`
	Serialization                   string               `default:"seata" yaml:"serialization"  json:"serialization,omitempty" property:"serialization"`
	Compressor                      string               `default:"none" yaml:"compressor"  json:"compressor,omitempty" property:"compressor"`
	ThreadFactory                   *ThreadFactoryConfig `yaml:"thread-factory"  json:"threadFactory,omitempty" property:"threadFactory"`
	Shutdown                        *ShutdownConfig      `yaml:"shutdown"  json:"shutdown,omitempty" property:"shutdown"`
}

type ThreadFactoryConfig struct {
	BossThreadPrefix           string `default:"NettyBoss" yaml:"boss-thread-prefix"  json:"bossThreadPrefix,omitempty" property:"bossThreadPrefix"`
	WorkerThreadPrefix         string `default:"NettyServerNIOWorker" yaml:"worker-thread-prefix"  json:"workerThreadPrefix,omitempty" property:"workerThreadPrefix"`
	ServerExecutorThreadPrefix string `default:"NettyServerBizHandler" yaml:"server-executor-thread-prefix"  json:"serverExecutorThreadPrefix,omitempty" property:"serverExecutorThreadPrefix"`
	ShareBossWorker            bool   `default:"false" yaml:"shareBossWorker"  json:"shareBossWorker,omitempty" property:"shareBossWorker"`
	ClientSelectorThreadPrefix string `default:"NettyClientSelector" yaml:"clientSelectorThreadPrefix"  json:"clientSelectorThreadPrefix,omitempty" property:"clientSelectorThreadPrefix"`
	ClientSelectorThreadSize   int    `default:"1" yaml:"client-selector-thread-size"  json:"clientSelectorThreadSize,omitempty" property:"clientSelectorThreadSize"`
	ClientWorkerThreadPrefix   string `default:"NettyClientWorkerThread" yaml:"clientWorkerThreadPrefix"  json:"clientWorkerThreadPrefix,omitempty" property:"clientWorkerThreadPrefix"`
	BossThreadSize             int    `default:"3" yaml:"boss-thread-size"  json:"bossThreadSize,omitempty" property:"bossThreadSize"`
	WorkerThreadSize           string `default:"default" yaml:"workerThreadSize"  json:"workerThreadSize,omitempty" property:"workerThreadSize"`
}

type ShutdownConfig struct {
	Wait int `default:"3" yaml:"wait"  json:"wait,omitempty" property:"wait"`
}

// Prefix seata.transport
func (TransportConfig) Prefix() string {
	return constant.TransportConfigPrefix
}

// Prefix seata.transport.threadFactory
func (ThreadFactoryConfig) Prefix() string {
	return constant.TransportThreadFactoryConfigPrefix
}

// Prefix seata.transport.shutdown
func (ShutdownConfig) Prefix() string {
	return constant.TransportShutdownConfigPrefix
}
