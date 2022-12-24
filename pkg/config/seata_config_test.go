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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	Init()
	assert.Equal(t, DefaultSeataConf.Seata.Enabled, true)
	assert.Equal(t, DefaultSeataConf.Seata.ApplicationID, "applicationName")
	assert.Equal(t, DefaultSeataConf.Seata.TxServiceGroup, "default_tx_group")
	assert.Equal(t, DefaultSeataConf.Seata.AccessKey, "aliyunAccessKey")
	assert.Equal(t, DefaultSeataConf.Seata.SecretKey, "aliyunSecretKey")
	assert.Equal(t, DefaultSeataConf.Seata.EnableAutoDataSourceProxy, true)
	assert.Equal(t, DefaultSeataConf.Seata.DataSourceProxyMode, "AT")

	// client
	// rm
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.AsyncCommitBufferLimit, 10000)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.ReportRetryCount, 5)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.TableMetaCheckEnable, false)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.ReportSuccessEnable, false)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.SagaBranchRegisterEnable, false)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.SagaJSONParser, "fastjson")
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.SagaRetryPersistModeUpdate, false)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.SagaCompensatePersistModeUpdate, false)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.TccActionInterceptorOrder, -2147482648)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.SQLParserType, "druid")
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.Lock.RetryInterval, 10)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.Lock.RetryTimes, time.Duration(30_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Rmconf.Lock.RetryPolicyBranchRollbackOnConflict, true)
	// tm
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Tmconf.CommitRetryCount, 5)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Tmconf.RollbackRetryCount, 5)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Tmconf.DefaultGlobalTransactionTimeout, time.Duration(10_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Tmconf.DegradeCheck, false)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Tmconf.DegradeCheckPeriod, 2000)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Tmconf.DegradeCheckAllowTimes, time.Duration(10_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Tmconf.InterceptorOrder, -2147482648)
	// undo
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Undo.DataValidation, true)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Undo.LogSerialization, "jackson")
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Undo.LogTable, "undo_log")
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Undo.OnlyCareUpdateColumns, true)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Undo.Compress.Enable, true)
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Undo.Compress.Type, "zip")
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.Undo.Compress.Threshold, 64)
	// load-balance
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.LoadBalance.Type, "RandomLoadBalance")
	assert.Equal(t, DefaultSeataConf.Seata.ClientConf.LoadBalance.VirtualNodes, 10)

	// service
	assert.Equal(t, DefaultSeataConf.Seata.Service.VgroupMapping.DefaultTxGroup, "default")
	assert.Equal(t, DefaultSeataConf.Seata.Service.Grouplist.Default, "127.0.0.1:8091")
	assert.Equal(t, DefaultSeataConf.Seata.Service.EnableDegrade, false)
	assert.Equal(t, DefaultSeataConf.Seata.Service.DisableGlobalTransaction, false)

	// transport
	assert.Equal(t, DefaultSeataConf.Seata.Transport.Shutdown.Wait, time.Duration(3_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.Transport.Type, "TCP")
	assert.Equal(t, DefaultSeataConf.Seata.Transport.Server, "NIO")
	assert.Equal(t, DefaultSeataConf.Seata.Transport.Heartbeat, true)
	assert.Equal(t, DefaultSeataConf.Seata.Transport.Serialization, "seata")
	assert.Equal(t, DefaultSeataConf.Seata.Transport.Compressor, "none")
	assert.Equal(t, DefaultSeataConf.Seata.Transport.EnableTmClientBatchSendRequest, false)
	assert.Equal(t, DefaultSeataConf.Seata.Transport.EnableRmClientBatchSendRequest, true)
	assert.Equal(t, DefaultSeataConf.Seata.Transport.RPCRmRequestTimeout, time.Duration(30_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.Transport.RPCTmRequestTimeout, time.Duration(30_000_000_000))

	// config
	assert.Equal(t, DefaultSeataConf.Seata.Config.Type, "file")
	assert.Equal(t, DefaultSeataConf.Seata.Config.File.Name, "config.conf")
	assert.Equal(t, DefaultSeataConf.Seata.Config.Nacos.Namespace, "")
	assert.Equal(t, DefaultSeataConf.Seata.Config.Nacos.ServerAddr, "127.0.0.1:8848")
	assert.Equal(t, DefaultSeataConf.Seata.Config.Nacos.Group, "SEATA_GROUP")
	assert.Equal(t, DefaultSeataConf.Seata.Config.Nacos.Username, "")
	assert.Equal(t, DefaultSeataConf.Seata.Config.Nacos.Password, "")
	assert.Equal(t, DefaultSeataConf.Seata.Config.Nacos.DataID, "seata.properties")

	// registry
	assert.Equal(t, DefaultSeataConf.Seata.Registry.Type, "file")
	assert.Equal(t, DefaultSeataConf.Seata.Registry.File.Name, "registry.conf")
	assert.Equal(t, DefaultSeataConf.Seata.Registry.Nacos.Application, "seata-server")
	assert.Equal(t, DefaultSeataConf.Seata.Registry.Nacos.ServerAddr, "127.0.0.1:8848")
	assert.Equal(t, DefaultSeataConf.Seata.Registry.Nacos.Group, "SEATA_GROUP")
	assert.Equal(t, DefaultSeataConf.Seata.Registry.Nacos.Namespace, "")
	assert.Equal(t, DefaultSeataConf.Seata.Registry.Nacos.Username, "")
	assert.Equal(t, DefaultSeataConf.Seata.Registry.Nacos.Password, "")

	// log
	assert.Equal(t, DefaultSeataConf.Seata.LogConf.ExceptionRate, 100)

	// tcc
	assert.Equal(t, DefaultSeataConf.Seata.TccConf.Fence.LogTableName, "tcc_fence_log")
	assert.Equal(t, DefaultSeataConf.Seata.TccConf.Fence.CleanPeriod, time.Duration(60_000_000_000))

	// getty-session-param
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.CompressEncoding, false)
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.TCPNoDelay, true)
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.TCPKeepAlive, true)
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.KeepAlivePeriod, time.Duration(120_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.TCPRBufSize, 262144)
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.TCPWBufSize, 65536)
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.TCPReadTimeout, time.Duration(1_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.TCPWriteTimeout, time.Duration(5_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.WaitTimeout, time.Duration(1_000_000_000))
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.MaxMsgLen, 16498688)
	assert.Equal(t, DefaultSeataConf.Seata.GettySessionParam.SessionName, "client")
}
