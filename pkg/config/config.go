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
	"flag"

	"github.com/seata/seata-go/pkg/remoting/getty"
)

type Config struct {
	Target                       string       `json:"target" yaml:"target"`
	ApplicationID                string       `yaml:"application_id" json:"application_id,omitempty"`
	TransactionServiceGroup      string       `yaml:"transaction_service_group" json:"transaction_service_group,omitempty"`
	EnableClientBatchSendRequest bool         `yaml:"enable_rpc_client-batch-send-request" json:"enable-rpc_client-batch-send-request,omitempty"`
	SeataVersion                 string       `yaml:"seata_version" json:"seata_version,omitempty"`
	BranchType                   int          `yaml:"branch_type" json:"branch_type"`
	GettyConfig                  getty.Config `yaml:"getty" json:"getty"`

	//ATConfig struct {
	//	DSN                 string        `yaml:"dsn" json:"dsn,omitempty"`
	//	ReportRetryCount    int           `default:"5" yaml:"report_retry_count" json:"report_retry_count,omitempty"`
	//	ReportSuccessEnable bool          `default:"false" yaml:"report_success_enable" json:"report_success_enable,omitempty"`
	//	LockRetryInterval   time.Duration `default:"10" yaml:"lock_retry_interval" json:"lock_retry_interval,omitempty"`
	//	LockRetryTimes      int           `default:"30" yaml:"lock_retry_times" json:"lock_retry_times,omitempty"`
	//} `yaml:"at" json:"at,omitempty"`
}

// RegisterFlags registers flag.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	f.StringVar(&cfg.Target, "target", "all", "the target config")
	f.StringVar(&cfg.ApplicationID, "application_id", "seata-go", "the application id")
	f.StringVar(&cfg.TransactionServiceGroup, "transaction_service_group", "127.0.0.1:8091", "the transaction service group")
	f.BoolVar(&cfg.EnableClientBatchSendRequest, "report_success_enable", false, "enable client batch send request")
	f.StringVar(&cfg.SeataVersion, "seata_version", "1.1.0", "seata version")

	cfg.GettyConfig.RegisterFlags(f)
}
