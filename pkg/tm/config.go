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

package tm

import (
	"flag"
	"time"
)

type TmConfig struct {
	CommitRetryCount                int           `yaml:"commit-retry-count" json:"commit-retry-count" koanf:"commit-retry-count"`
	RollbackRetryCount              int           `yaml:"rollback-retry-count" json:"rollback-retry-count" koanf:"rollback-retry-count"`
	DefaultGlobalTransactionTimeout time.Duration `yaml:"default-global-transaction-timeout" json:"default-global-transaction-timeout,omitempty" koanf:"default-global-transaction-timeout"`
	DegradeCheck                    bool          `yaml:"degrade-check" json:"degrade-check" koanf:"degrade-check"`
	DegradeCheckPeriod              int           `yaml:"degrade-check-period" json:"degrade-check-period" koanf:"degrade-check-period"`
	DegradeCheckAllowTimes          time.Duration `yaml:"degrade-check-allow-times" json:"degrade-check-allow-times" koanf:"degrade-check-allow-times"`
	InterceptorOrder                int           `yaml:"interceptor-order" json:"interceptor-order" koanf:"interceptor-order"`
}

func (cfg *TmConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.IntVar(&cfg.CommitRetryCount, prefix+".commit-retry-count", 5, "The maximum number of retries when commit global transaction.")
	f.IntVar(&cfg.RollbackRetryCount, prefix+".rollback-retry-count", 5, "The maximum number of retries when rollback global transaction.")
	f.DurationVar(&cfg.DefaultGlobalTransactionTimeout, prefix+".default-global-transaction-timeout", 60*time.Second, "The timeout for a global transaction.")
	f.BoolVar(&cfg.DegradeCheck, prefix+".degrade-check", false, "The switch for degrade check.")
	f.IntVar(&cfg.DegradeCheckPeriod, prefix+".degrade-check-period", 2000, "The period for degrade checking.")
	f.DurationVar(&cfg.DegradeCheckAllowTimes, prefix+".degrade-check-allow-times", 10*time.Second, "The duration allowed for degrade checking.")
	f.IntVar(&cfg.InterceptorOrder, prefix+".interceptor-order", -2147482648, "The order of interceptor.")
}
