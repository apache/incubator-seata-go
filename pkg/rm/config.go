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
	"time"
)

type Config struct {
	AsyncCommitBufferLimit          int        `yaml:"async-commit-buffer-limit" json:"async-commit-buffer-limit,omitempty" koanf:"async-commit-buffer-limit"`
	ReportRetryCount                int        `yaml:"report-retry-count" json:"report-retry-count,omitempty" koanf:"report-retry-count"`
	TableMetaCheckEnable            bool       `yaml:"table-meta-check-enable" json:"table-meta-check-enable" koanf:"table-meta-check-enable"`
	ReportSuccessEnable             bool       `yaml:"report-success-enable" json:"report-success-enable,omitempty" koanf:"report-success-enable"`
	SagaBranchRegisterEnable        bool       `yaml:"saga-branch-register-enable" json:"saga-branch-register-enable,omitempty" koanf:"saga-branch-register-enable"`
	SagaJsonParser                  string     `yaml:"saga-json-parser" json:"saga-json-parser,omitempty" koanf:"saga-json-parser"`
	SagaRetryPersistModeUpdate      bool       `yaml:"saga-retry-persist-mode-update" json:"saga-retry-persist-mode-update,omitempty" koanf:"saga-retry-persist-mode-update"`
	SagaCompensatePersistModeUpdate bool       `yaml:"saga-compensate-persist-mode-update" json:"saga-compensate-persist-mode-update,omitempty" koanf:"saga-compensate-persist-mode-update"`
	TccActionInterceptorOrder       int        `yaml:"tcc-action-interceptor-order" json:"tcc-action-interceptor-order,omitempty" koanf:"tcc-action-interceptor-order"`
	SqlParserType                   string     `yaml:"sql-parser-type" json:"sql-parser-type,omitempty" koanf:"sql-parser-type"`
	LockConfig                      LockConfig `yaml:"lock" json:"lock,omitempty" koanf:"lock"`
}

type LockConfig struct {
	RetryInterval                       time.Duration `yaml:"retry-interval" json:"retry-interval,omitempty" koanf:"retry-interval"`
	RetryTimes                          int           `yaml:"retry-times" json:"retry-times,omitempty" koanf:"retry-times"`
	RetryPolicyBranchRollbackOnConflict bool          `yaml:"retry-policy-branch-rollback-on-conflict" json:"retry-policy-branch-rollback-on-conflict,omitempty" koanf:"retry-policy-branch-rollback-on-conflict"`
}

func (cfg *Config) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.AsyncCommitBufferLimit, prefix+".async-commit-buffer-limit", 10000, "The Maximum cache length of asynchronous queue.")
	f.IntVar(&cfg.ReportRetryCount, prefix+".report-retry-count", 5, "The maximum number of retries when report reports the status.")
	f.BoolVar(&cfg.TableMetaCheckEnable, prefix+".table-meta-check-enable", false, "Whether to check the metadata of the db（AT）.")
	f.BoolVar(&cfg.ReportSuccessEnable, prefix+".report-success-enable", false, "Whether to report the status if the transaction is successfully executed（AT）")
	f.BoolVar(&cfg.SagaBranchRegisterEnable, prefix+".saga-branch-register-enable", false, "Whether to allow regular check of db metadata（AT）")
	f.StringVar(&cfg.SagaJsonParser, prefix+".saga-json-parser", "fastjson", "The saga JsonParser.")
	f.BoolVar(&cfg.SagaRetryPersistModeUpdate, prefix+".saga-retry-persist-mode-update", false, "Whether to retry SagaRetryPersistModeUpdate")
	f.BoolVar(&cfg.SagaCompensatePersistModeUpdate, prefix+".saga-compensate-persist-mode-update", false, "")
	f.IntVar(&cfg.TccActionInterceptorOrder, prefix+".tcc-action-interceptor-order", -2147482648, "The order of tccActionInterceptor.")
	f.StringVar(&cfg.SqlParserType, prefix+".sql-parser-type", "druid", "The type of sql parser.")
	cfg.LockConfig.RegisterFlagsWithPrefix(prefix, f)
}

func (cfg *LockConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.RetryInterval, prefix+".retry-interval", 30*time.Second, "The maximum number of retries when lock fail.")
	f.IntVar(&cfg.RetryTimes, prefix+".retry-times", 10, "The duration allowed for lock retrying.")
	f.BoolVar(&cfg.RetryPolicyBranchRollbackOnConflict, prefix+".retry-policy-branch-rollback-on-conflict", true, "The switch for lock conflict.")
}
