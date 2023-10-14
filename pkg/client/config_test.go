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
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadPath(t *testing.T) {
	cfg := LoadPath("../../testdata/conf/seatago.yml")
	assert.NotNil(t, cfg)
	assert.Equal(t, true, cfg.Enabled)
	assert.Equal(t, "applicationName", cfg.ApplicationID)
	assert.Equal(t, "default_tx_group", cfg.TxServiceGroup)
	assert.Equal(t, "aliyunAccessKey", cfg.AccessKey)
	assert.Equal(t, "aliyunSecretKey", cfg.SecretKey)
	assert.Equal(t, true, cfg.EnableAutoDataSourceProxy)
	assert.Equal(t, "AT", cfg.DataSourceProxyMode)

	assert.NotNil(t, cfg.TCCConfig)
	assert.NotNil(t, cfg.TCCConfig.FenceConfig)
	assert.Equal(t, "tcc_fence_log_test", cfg.TCCConfig.FenceConfig.LogTableName)
	assert.Equal(t, time.Second*60, cfg.TCCConfig.FenceConfig.CleanPeriod)

	assert.NotNil(t, cfg.ClientConfig)
	assert.NotNil(t, cfg.ClientConfig.TmConfig)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.CommitRetryCount)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.RollbackRetryCount)
	assert.Equal(t, time.Second*60, cfg.ClientConfig.TmConfig.DefaultGlobalTransactionTimeout)
	assert.Equal(t, false, cfg.ClientConfig.TmConfig.DegradeCheck)
	assert.Equal(t, 2000, cfg.ClientConfig.TmConfig.DegradeCheckPeriod)
	assert.Equal(t, time.Second*10, cfg.ClientConfig.TmConfig.DegradeCheckAllowTimes)
	assert.Equal(t, -2147482648, cfg.ClientConfig.TmConfig.InterceptorOrder)

	assert.Equal(t, 10000, cfg.ClientConfig.RmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, 5, cfg.ClientConfig.RmConfig.ReportRetryCount)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.TableMetaCheckEnable)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.ReportSuccessEnable)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.SagaBranchRegisterEnable)
	assert.Equal(t, "fastjson", cfg.ClientConfig.RmConfig.SagaJsonParser)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.SagaCompensatePersistModeUpdate)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.SagaRetryPersistModeUpdate)
	assert.Equal(t, -2147482648, cfg.ClientConfig.RmConfig.TccActionInterceptorOrder)
	assert.Equal(t, "druid", cfg.ClientConfig.RmConfig.SqlParserType)
	assert.Equal(t, 30*time.Second, cfg.ClientConfig.RmConfig.LockConfig.RetryInterval)
	assert.Equal(t, 10, cfg.ClientConfig.RmConfig.LockConfig.RetryTimes)
	assert.Equal(t, true, cfg.ClientConfig.RmConfig.LockConfig.RetryPolicyBranchRollbackOnConflict)

	assert.NotNil(t, cfg.ClientConfig.UndoConfig)
	assert.Equal(t, true, cfg.ClientConfig.UndoConfig.DataValidation)
	assert.Equal(t, "json", cfg.ClientConfig.UndoConfig.LogSerialization)
	assert.Equal(t, "undo_log", cfg.ClientConfig.UndoConfig.LogTable)
	assert.Equal(t, true, cfg.ClientConfig.UndoConfig.OnlyCareUpdateColumns)
	assert.NotNil(t, cfg.ClientConfig.UndoConfig.CompressConfig)
	assert.Equal(t, true, cfg.ClientConfig.UndoConfig.CompressConfig.Enable)
	assert.Equal(t, "zip", cfg.ClientConfig.UndoConfig.CompressConfig.Type)
	assert.Equal(t, "64k", cfg.ClientConfig.UndoConfig.CompressConfig.Threshold)

	assert.NotNil(t, cfg.GettyConfig)
	assert.NotNil(t, cfg.GettyConfig.SessionConfig)
	assert.Equal(t, 0, cfg.GettyConfig.ReconnectInterval)
	assert.Equal(t, 1, cfg.GettyConfig.ConnectionNum)
	assert.Equal(t, false, cfg.GettyConfig.SessionConfig.CompressEncoding)
	assert.Equal(t, true, cfg.GettyConfig.SessionConfig.TCPNoDelay)
	assert.Equal(t, true, cfg.GettyConfig.SessionConfig.TCPKeepAlive)
	assert.Equal(t, time.Minute*2, cfg.GettyConfig.SessionConfig.KeepAlivePeriod)
	assert.Equal(t, 262144, cfg.GettyConfig.SessionConfig.TCPRBufSize)
	assert.Equal(t, 65536, cfg.GettyConfig.SessionConfig.TCPWBufSize)
	assert.Equal(t, time.Second, cfg.GettyConfig.SessionConfig.TCPReadTimeout)
	assert.Equal(t, time.Second*5, cfg.GettyConfig.SessionConfig.TCPWriteTimeout)
	assert.Equal(t, time.Second, cfg.GettyConfig.SessionConfig.WaitTimeout)
	assert.Equal(t, 16498688, cfg.GettyConfig.SessionConfig.MaxMsgLen)
	assert.Equal(t, "client_test", cfg.GettyConfig.SessionConfig.SessionName)
	assert.Equal(t, time.Second, cfg.GettyConfig.SessionConfig.CronPeriod)

	assert.NotNil(t, cfg.TransportConfig)
	assert.NotNil(t, cfg.TransportConfig.ShutdownConfig)
	assert.Equal(t, time.Second*3, cfg.TransportConfig.ShutdownConfig.Wait)
	assert.Equal(t, "TCP", cfg.TransportConfig.Type)
	assert.Equal(t, "NIO", cfg.TransportConfig.Server)
	assert.Equal(t, true, cfg.TransportConfig.Heartbeat)
	assert.Equal(t, "seata", cfg.TransportConfig.Serialization)
	assert.Equal(t, "none", cfg.TransportConfig.Compressor)
	assert.Equal(t, false, cfg.TransportConfig.EnableTmClientBatchSendRequest)
	assert.Equal(t, true, cfg.TransportConfig.EnableRmClientBatchSendRequest)
	assert.Equal(t, time.Second*30, cfg.TransportConfig.RPCRmRequestTimeout)
	assert.Equal(t, time.Second*30, cfg.TransportConfig.RPCTmRequestTimeout)

	assert.NotNil(t, cfg.ServiceConfig)
	assert.Equal(t, false, cfg.ServiceConfig.EnableDegrade)
	assert.Equal(t, false, cfg.ServiceConfig.DisableGlobalTransaction)
	assert.Equal(t, "default", cfg.ServiceConfig.VgroupMapping["default_tx_group"])
	assert.Equal(t, "127.0.0.1:8091", cfg.ServiceConfig.Grouplist["default"])

	assert.NotNil(t, cfg.RegistryConfig)
	assert.Equal(t, "file", cfg.RegistryConfig.Type)
	assert.Equal(t, "seatago.yml", cfg.RegistryConfig.File.Name)
	assert.Equal(t, "seata-server", cfg.RegistryConfig.Nacos.Application)
	assert.Equal(t, "127.0.0.1:8848", cfg.RegistryConfig.Nacos.ServerAddr)
	assert.Equal(t, "SEATA_GROUP", cfg.RegistryConfig.Nacos.Group)
	assert.Equal(t, "test-namespace", cfg.RegistryConfig.Nacos.Namespace)
	assert.Equal(t, "test-username", cfg.RegistryConfig.Nacos.Username)
	assert.Equal(t, "test-password", cfg.RegistryConfig.Nacos.Password)
	assert.Equal(t, "test-access-key", cfg.RegistryConfig.Nacos.AccessKey)
	assert.Equal(t, "test-secret-key", cfg.RegistryConfig.Nacos.SecretKey)
	assert.Equal(t, "default", cfg.RegistryConfig.Etcd3.Cluster)
	assert.Equal(t, "http://localhost:2379", cfg.RegistryConfig.Etcd3.ServerAddr)

	// reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestLoadJson(t *testing.T) {
	confJson := `{"enabled":false,"application-id":"application_test","tx-service-group":"default_tx_group","access-key":"test","secret-key":"test","enable-auto-data-source-proxy":false,"data-source-proxy-mode":"AT","client":{"rm":{"async-commit-buffer-limit":10000,"report-retry-count":5,"table-meta-check-enable":false,"report-success-enable":false,"saga-branch-register-enable":false,"saga-json-parser":"fastjson","saga-retry-persist-mode-update":false,"saga-compensate-persist-mode-update":false,"tcc-action-interceptor-order":-2147482648,"sql-parser-type":"druid","lock":{"retry-interval":"30s","retry-times":10,"retry-policy-branch-rollback-on-conflict":true}},"tm":{"commit-retry-count":5,"rollback-retry-count":5,"default-global-transaction-timeout":"60s","degrade-check":false,"degrade-check-period":2000,"degrade-check-allow-times":"10s","interceptor-order":-2147482648},"undo":{"data-validation":false,"log-serialization":"jackson222","only-care-update-columns":false,"log-table":"undo_log333","compress":{"enable":false,"type":"zip111","threshold":"128k"}}},"tcc":{"fence":{"log-table-name":"tcc_fence_log_test2","clean-period":80000000000}},"getty":{"reconnect-interval":1,"connection-num":10,"session":{"compress-encoding":true,"tcp-no-delay":false,"tcp-keep-alive":false,"keep-alive-period":"120s","tcp-r-buf-size":261120,"tcp-w-buf-size":32768,"tcp-read-timeout":"2s","tcp-write-timeout":"8s","wait-timeout":"2s","max-msg-len":261120,"session-name":"client_test","cron-period":"2s"}},"transport":{"shutdown":{"wait":"3s"},"type":"TCP","server":"NIO","heartbeat":true,"serialization":"seata","compressor":"none"," enable-tm-client-batch-send-request":false,"enable-rm-client-batch-send-request":true,"rpc-rm-request-timeout":"30s","rpc-tm-request-timeout":"30s"},"service":{"enable-degrade":true,"disable-global-transaction":true,"vgroup-mapping":{"default_tx_group":"default_test"},"grouplist":{"default":"127.0.0.1:8092"}}}`
	cfg := LoadJson([]byte(confJson))
	assert.NotNil(t, cfg)
	assert.Equal(t, false, cfg.Enabled)
	assert.Equal(t, "application_test", cfg.ApplicationID)
	assert.Equal(t, "default_tx_group", cfg.TxServiceGroup)
	assert.Equal(t, "test", cfg.AccessKey)
	assert.Equal(t, "test", cfg.SecretKey)
	assert.Equal(t, false, cfg.EnableAutoDataSourceProxy)
	assert.Equal(t, "AT", cfg.DataSourceProxyMode)

	assert.Equal(t, 10000, cfg.ClientConfig.RmConfig.AsyncCommitBufferLimit)
	assert.Equal(t, 5, cfg.ClientConfig.RmConfig.ReportRetryCount)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.TableMetaCheckEnable)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.ReportSuccessEnable)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.SagaBranchRegisterEnable)
	assert.Equal(t, "fastjson", cfg.ClientConfig.RmConfig.SagaJsonParser)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.SagaCompensatePersistModeUpdate)
	assert.Equal(t, false, cfg.ClientConfig.RmConfig.SagaRetryPersistModeUpdate)
	assert.Equal(t, -2147482648, cfg.ClientConfig.RmConfig.TccActionInterceptorOrder)
	assert.Equal(t, "druid", cfg.ClientConfig.RmConfig.SqlParserType)
	assert.Equal(t, 30*time.Second, cfg.ClientConfig.RmConfig.LockConfig.RetryInterval)
	assert.Equal(t, 10, cfg.ClientConfig.RmConfig.LockConfig.RetryTimes)
	assert.Equal(t, true, cfg.ClientConfig.RmConfig.LockConfig.RetryPolicyBranchRollbackOnConflict)

	assert.NotNil(t, cfg.ClientConfig.UndoConfig)
	assert.Equal(t, false, cfg.ClientConfig.UndoConfig.DataValidation)
	assert.Equal(t, "jackson222", cfg.ClientConfig.UndoConfig.LogSerialization)
	assert.Equal(t, "undo_log333", cfg.ClientConfig.UndoConfig.LogTable)
	assert.Equal(t, false, cfg.ClientConfig.UndoConfig.OnlyCareUpdateColumns)
	assert.NotNil(t, cfg.ClientConfig.UndoConfig.CompressConfig)
	assert.Equal(t, false, cfg.ClientConfig.UndoConfig.CompressConfig.Enable)
	assert.Equal(t, "zip111", cfg.ClientConfig.UndoConfig.CompressConfig.Type)
	assert.Equal(t, "128k", cfg.ClientConfig.UndoConfig.CompressConfig.Threshold)

	assert.NotNil(t, cfg.TCCConfig)
	assert.NotNil(t, cfg.TCCConfig.FenceConfig)
	assert.Equal(t, "tcc_fence_log_test2", cfg.TCCConfig.FenceConfig.LogTableName)
	assert.Equal(t, time.Second*80, cfg.TCCConfig.FenceConfig.CleanPeriod)

	assert.NotNil(t, cfg.ClientConfig)
	assert.NotNil(t, cfg.ClientConfig.TmConfig)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.CommitRetryCount)
	assert.Equal(t, 5, cfg.ClientConfig.TmConfig.RollbackRetryCount)
	assert.Equal(t, time.Second*60, cfg.ClientConfig.TmConfig.DefaultGlobalTransactionTimeout)
	assert.Equal(t, false, cfg.ClientConfig.TmConfig.DegradeCheck)
	assert.Equal(t, 2000, cfg.ClientConfig.TmConfig.DegradeCheckPeriod)
	assert.Equal(t, time.Second*10, cfg.ClientConfig.TmConfig.DegradeCheckAllowTimes)
	assert.Equal(t, -2147482648, cfg.ClientConfig.TmConfig.InterceptorOrder)

	assert.NotNil(t, cfg.GettyConfig)
	assert.NotNil(t, cfg.GettyConfig.SessionConfig)
	assert.Equal(t, 1, cfg.GettyConfig.ReconnectInterval)
	assert.Equal(t, 10, cfg.GettyConfig.ConnectionNum)
	assert.Equal(t, true, cfg.GettyConfig.SessionConfig.CompressEncoding)
	assert.Equal(t, false, cfg.GettyConfig.SessionConfig.TCPNoDelay)
	assert.Equal(t, false, cfg.GettyConfig.SessionConfig.TCPKeepAlive)
	assert.Equal(t, time.Minute*2, cfg.GettyConfig.SessionConfig.KeepAlivePeriod)
	assert.Equal(t, 261120, cfg.GettyConfig.SessionConfig.TCPRBufSize)
	assert.Equal(t, 32768, cfg.GettyConfig.SessionConfig.TCPWBufSize)
	assert.Equal(t, time.Second*2, cfg.GettyConfig.SessionConfig.TCPReadTimeout)
	assert.Equal(t, time.Second*8, cfg.GettyConfig.SessionConfig.TCPWriteTimeout)
	assert.Equal(t, time.Second*2, cfg.GettyConfig.SessionConfig.WaitTimeout)
	assert.Equal(t, 261120, cfg.GettyConfig.SessionConfig.MaxMsgLen)
	assert.Equal(t, "client_test", cfg.GettyConfig.SessionConfig.SessionName)
	assert.Equal(t, time.Second*2, cfg.GettyConfig.SessionConfig.CronPeriod)

	assert.NotNil(t, cfg.TransportConfig)
	assert.NotNil(t, cfg.TransportConfig.ShutdownConfig)
	assert.Equal(t, time.Second*3, cfg.TransportConfig.ShutdownConfig.Wait)
	assert.Equal(t, "TCP", cfg.TransportConfig.Type)
	assert.Equal(t, "NIO", cfg.TransportConfig.Server)
	assert.Equal(t, true, cfg.TransportConfig.Heartbeat)
	assert.Equal(t, "seata", cfg.TransportConfig.Serialization)
	assert.Equal(t, "none", cfg.TransportConfig.Compressor)
	assert.Equal(t, false, cfg.TransportConfig.EnableTmClientBatchSendRequest)
	assert.Equal(t, true, cfg.TransportConfig.EnableRmClientBatchSendRequest)
	assert.Equal(t, time.Second*30, cfg.TransportConfig.RPCRmRequestTimeout)
	assert.Equal(t, time.Second*30, cfg.TransportConfig.RPCTmRequestTimeout)

	assert.NotNil(t, cfg.ServiceConfig)
	assert.Equal(t, true, cfg.ServiceConfig.EnableDegrade)
	assert.Equal(t, true, cfg.ServiceConfig.DisableGlobalTransaction)
	assert.Equal(t, "default_test", cfg.ServiceConfig.VgroupMapping["default_tx_group"])
	assert.Equal(t, "127.0.0.1:8092", cfg.ServiceConfig.Grouplist["default"])

	// reset flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}
