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
		LockRetryInterval   time.Duration `default:"10ms" yaml:"lock_retry_interval" json:"lock_retry_interval,omitempty"`
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
