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
	"time"
)

var clientConfig *ClientConfig

type ClientConfig struct {
	ApplicationID                string      `yaml:"application_id" json:"application_id,omitempty"`
	TransactionServiceGroup      string      `yaml:"transaction_service_group" json:"transaction_service_group,omitempty"`
	EnableClientBatchSendRequest bool        `yaml:"enable-rpc_client-batch-send-request" json:"enable-rpc_client-batch-send-request,omitempty"`
	SeataVersion                 string      `yaml:"seata_version" json:"seata_version,omitempty"`
	GettyConfig                  GettyConfig `yaml:"getty" json:"getty,omitempty"`

	ATConfig struct {
		DSN                 string        `yaml:"dsn" json:"dsn,omitempty"`
		ReportRetryCount    int           `default:"5" yaml:"report_retry_count" json:"report_retry_count,omitempty"`
		ReportSuccessEnable bool          `default:"false" yaml:"report_success_enable" json:"report_success_enable,omitempty"`
		LockRetryInterval   time.Duration `default:"10" yaml:"lock_retry_interval" json:"lock_retry_interval,omitempty"`
		LockRetryTimes      int           `default:"30" yaml:"lock_retry_times" json:"lock_retry_times,omitempty"`
	} `yaml:"at" json:"at,omitempty"`
}

func GetClientConfig() *ClientConfig {
	return &ClientConfig{
		GettyConfig: GetDefaultGettyConfig(),
	}
}

func GetDefaultClientConfig(applicationID string) *ClientConfig {
	return &ClientConfig{
		ApplicationID:                applicationID,
		TransactionServiceGroup:      "127.0.0.1:8091",
		EnableClientBatchSendRequest: false,
		SeataVersion:                 "1.1.0",
		GettyConfig:                  GetDefaultGettyConfig(),
	}
}

type Compress struct {
	Enable    bool   `yaml:"enable" json:"enable,omitempty" property:"enable"`
	Type      string `yaml:"type" json:"type,omitempty" property:"type"`
	Threshold int    `yaml:"threshold" json:"threshold,omitempty" property:"threshold"`
}

type Lock struct {
	RetryInterval                       int           `yaml:"retry-interval" json:"retry-interval,omitempty" property:"retry-interval"`
	RetryTimes                          time.Duration `yaml:"retry-times" json:"retry-times,omitempty" property:"retry-times"`
	RetryPolicyBranchRollbackOnConflict bool          `yaml:"retry-policy-branch-rollback-on-conflict" json:"retry-policy-branch-rollback-on-conflict,omitempty" property:"retry-policy-branch-rollback-on-conflict"`
}

type RmConf struct {
	AsyncCommitBufferLimit          int    `yaml:"async-commit-buffer-limit" json:"async-commit-buffer-limit,omitempty" property:"async-commit-buffer-limit"`
	ReportRetryCount                int    `yaml:"report-retry-count" json:"report-retry-count,omitempty" property:"report-retry-count"`
	TableMetaCheckEnable            bool   `yaml:"table-meta-check-enable" json:"table-meta-check-enable,omitempty" property:"table-meta-check-enable"`
	ReportSuccessEnable             bool   `yaml:"report-success-enable" json:"report-success-enable,omitempty" property:"report-success-enable"`
	SagaBranchRegisterEnable        bool   `yaml:"saga-branch-register-enable" json:"saga-branch-register-enable,omitempty" property:"saga-branch-register-enable"`
	SagaJSONParser                  string `yaml:"saga-json-parser" json:"saga-json-parser,omitempty" property:"saga-json-parser"`
	SagaRetryPersistModeUpdate      bool   `yaml:"saga-retry-persist-mode-update" json:"saga-retry-persist-mode-update,omitempty" property:"saga-retry-persist-mode-update"`
	SagaCompensatePersistModeUpdate bool   `yaml:"saga-compensate-persist-mode-update" json:"saga-compensate-persist-mode-update,omitempty" property:"saga-compensate-persist-mode-update"`
	TccActionInterceptorOrder       int    `yaml:"tcc-action-interceptor-order" json:"tcc-action-interceptor-order,omitempty" property:"tcc-action-interceptor-order"`
	SQLParserType                   string `yaml:"sql-parser-type" json:"sql-parser-type,omitempty" property:"sql-parser-type"`
	Lock                            Lock   `yaml:"lock" json:"lock,omitempty" property:"lock"`
}

type TmConf struct {
	CommitRetryCount                int           `yaml:"commit-retry-count" json:"commit-retry-count,omitempty" property:"commit-retry-count"`
	RollbackRetryCount              int           `yaml:"rollback-retry-count" json:"rollback-retry-count,omitempty" property:"rollback-retry-count"`
	DefaultGlobalTransactionTimeout time.Duration `yaml:"default-global-transaction-timeout" json:"default-global-transaction-timeout,omitempty" property:"default-global-transaction-timeout"`
	DegradeCheck                    bool          `yaml:"degrade-check" json:"degrade-check,omitempty" property:"degrade-check"`
	DegradeCheckPeriod              int           `yaml:"degrade-check-period" json:"degrade-check-period,omitempty" property:"degrade-check-period"`
	DegradeCheckAllowTimes          time.Duration `yaml:"degrade-check-allow-times" json:"degrade-check-allow-times,omitempty" property:"degrade-check-allow-times"`
	InterceptorOrder                int           `yaml:"interceptor-order" json:"interceptor-order,omitempty" property:"interceptor-order"`
}

type Undo struct {
	DataValidation        bool     `yaml:"data-validation" json:"data-validation,omitempty" property:"data-validation"`
	LogSerialization      string   `yaml:"log-serialization" json:"log-serialization,omitempty" property:"log-serialization"`
	LogTable              string   `yaml:"log-table" json:"log-table,omitempty" property:"log-table"`
	OnlyCareUpdateColumns bool     `yaml:"only-care-update-columns" json:"only-care-update-columns,omitempty" property:"only-care-update-columns"`
	Compress              Compress `yaml:"compress" json:"compress,omitempty" property:"compress"`
}

type LoadBalance struct {
	Type         string `yaml:"type" json:"type,omitempty" property:"type"`
	VirtualNodes int    `yaml:"virtual-nodes" json:"virtual-nodes,omitempty" property:"virtual-nodes"`
}

type ClientConf struct {
	Rmconf      RmConf      `yaml:"rm" json:"rm,omitempty" property:"rm"`
	Tmconf      TmConf      `yaml:"tm" json:"tm,omitempty" property:"tm"`
	Undo        Undo        `yaml:"undo" json:"undo,omitempty" property:"undo"`
	LoadBalance LoadBalance `yaml:"load-balance" json:"load-balance,omitempty" property:"load-balance"`
}
