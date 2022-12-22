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

package rm

import (
	"flag"
)

type Config struct {
	AsyncCommitBufferLimit          int        `yaml:"async_commit_buffer_limit" json:"async_commit_buffer_limit,omitempty" koanf:"async_commit_buffer_limit"`
	ReportRetryCount                int        `yaml:"report_retry_count" json:"report_retry_count,omitempty" koanf:"report_retry_count"`
	TableMetaCheckEnable            bool       `yaml:"table_meta_check_enable" json:"table_meta_check_enable" koanf:"table_meta_check_enable"`
	ReportSuccessEnable             bool       `yaml:"report_success_enable" json:"report_success_enable,omitempty" koanf:"report_success_enable"`
	SagaBranchRegisterEnable        bool       `yaml:"saga_branch_register_enable" json:"saga_branch_register_enable,omitempty" koanf:"saga_branch_register_enable"`
	SagaJsonParser                  string     `yaml:"saga_json_parser" json:"saga_json_parser,omitempty" koanf:"saga_json_parser"`
	SagaRetryPersistModeUpdate      bool       `yaml:"saga_retry_persist_mode_update" json:"saga_retry_persist_mode_update,omitempty" koanf:"saga_retry_persist_mode_update"`
	SagaCompensatePersistModeUpdate bool       `yaml:"saga_compensate_persist_mode_update" json:"saga_compensate_persist_mode_update,omitempty" koanf:"saga_compensate_persist_mode_update"`
	TccActionInterceptorOrder       int        `yaml:"tcc_action_interceptor_order" json:"tcc_action_interceptor_order,omitempty" koanf:"tcc_action_interceptor_order"`
	SqlParserType                   string     `yaml:"sql_parser_type" json:"sql_parser_type,omitempty" koanf:"sql_parser_type"`
	LockConfig                      LockConfig `yaml:"lock" json:"lock,omitempty" koanf:"lock"`
}

func (cfg *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.AsyncCommitBufferLimit, prefix+".async_commit_buffer_limit", 10000, "The Maximum cache length of asynchronous queue.")
	f.IntVar(&cfg.ReportRetryCount, prefix+".report_retry_count", 5, "The maximum number of retries when report reports the status.")
	f.BoolVar(&cfg.TableMetaCheckEnable, prefix+".table_meta_check_enable", false, "Whether to check the metadata of the db（AT）.")
	f.BoolVar(&cfg.ReportSuccessEnable, prefix+".report_success_enable", false, "Whether to report the status if the transaction is successfully executed（AT）")
	f.BoolVar(&cfg.SagaBranchRegisterEnable, prefix+".saga_branch_register_enable", false, "Whether to allow regular check of db metadata（AT）")
	f.StringVar(&cfg.SagaJsonParser, prefix+".saga_json_parser", "fastjson", "The saga JsonParser.")
	f.BoolVar(&cfg.SagaRetryPersistModeUpdate, prefix+".saga_retry_persist_mode_update", false, "Whether to retry SagaRetryPersistModeUpdate")
	f.BoolVar(&cfg.SagaCompensatePersistModeUpdate, prefix+".saga_compensate_persist_mode_update", false, "")
	f.IntVar(&cfg.TccActionInterceptorOrder, prefix+".tcc_action_interceptor_order", -2147482648, "The order of tccActionInterceptor.")
	f.StringVar(&cfg.SqlParserType, prefix+".sql_parser_type", "druid", "The type of sql parser.")
	cfg.LockConfig.RegisterFlagsWithPrefix(prefix+".lock", f)
}
