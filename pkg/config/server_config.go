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
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common/constant"
)

// ServerConfig holds supported types by the multiconfig package
type ServerConfig struct {
	// undo log
	UndoLog     UndoLogConfig `yaml:"undo" json:"undo,omitempty" property:"undo"`
	ServicePort int           `default:"8091" yaml:"service-port" json:"servicePort,omitempty" property:"servicePort"`
	// Two-phase commit retry timeout duration
	MaxCommitRetryTimeout int `default:"-1" yaml:"max-commit-retry-timeout" json:"maxCommitRetryTimeout,omitempty" property:"maxCommitRetryTimeout"`
	// Two-phase rollback retry timeout duration
	MaxRollbackRetryTimeout          int            `default:"-1" yaml:"max-rollback-retry-timeout" json:"maxRollbackRetryTimeout,omitempty" property:"maxRollbackRetryTimeout"`
	RollbackRetryTimeoutUnlockEnable bool           `default:"false" yaml:"rollback-retry-timeout-unlock-enable" json:"rollbackRetryTimeoutUnlockEnable,omitempty" property:"rollbackRetryTimeoutUnlockEnable"`
	DistributedLockExpireTime        bool           `default:"false" yaml:"distributed-lock-expire-time" json:"distributedLockExpireTime,omitempty" property:"distributedLockExpireTime"`
	EnableCheckAuth                  bool           `default:"true" yaml:"enable-check-auth" json:"enableCheckAuth,omitempty" property:"enableCheckAuth"`
	EnableParallelRequestHandle      bool           `default:"false" yaml:"enable-parallel-request-handle" json:"enableParallelRequestHandle,omitempty" property:"enableParallelRequestHandle"`
	RetryDeadThreshold               int            `default:"130000" yaml:"retry-dead-threshold" json:"RetryDeadThreshold,omitempty" property:"RetryDeadThreshold"`
	XaerNotaRetryTimeout             int            `default:"60000" yaml:"xaer-nota-retry-timeout" json:"xaerNotaRetryTimeout,omitempty" property:"xaerNotaRetryTimeout"`
	Recovery                         RecoveryConfig `yaml:"recovery" json:"recovery,omitempty" property:"recovery"`
	Session                          SessionConfig  `yaml:"session" json:"session,omitempty" property:"session"`
}
type UndoLogConfig struct {
	// undo log
	LogSaveDays     int `default:"7" yaml:"log-save-days" json:"logSaveDays,omitempty" property:"logSaveDays"`
	LogDeletePeriod int `default:"86400000" yaml:"log-delete-period" json:"logDeletePeriod,omitempty" property:"logDeletePeriod"`
}

type RecoveryConfig struct {
	// Two-phase commit incomplete state global transaction retry commit thread interval time
	CommittingRetryPeriod     int `default:"1000" yaml:"committing-retry-period" json:"committingRetryPeriod,omitempty" property:"committingRetryPeriod"`
	AsynCommittingRetryPeriod int `default:"1000" yaml:"asyn-committing-retry-period" json:"asynCommittingRetryPeriod,omitempty" property:"asynCommittingRetryPeriod"`
	// Two-phase rollback state retry rollback thread interval
	RollbackingRetryPeriod int `default:"1000" yaml:"rollbacking-retry-period" json:"rollbackingRetryPeriod,omitempty" property:"rollbackingRetryPeriod"`
	// Timeout state detection retry thread interval time
	TimeoutRetryPeriod     int `default:"1000" yaml:"timeout-retry-period" json:"timeoutRetryPeriod,omitempty" property:"timeoutRetryPeriod"`
	HandleAllSessionPeriod int `default:"1000" yaml:"handle-all-session-period" json:"handleAllSessionPeriod,omitempty" property:"handleAllSessionPeriod"`
}
type SessionConfig struct {
	// Two-phase commit incomplete state global transaction retry commit thread interval time
	BranchAsyncQueueSize    int  `default:"5000" yaml:"branch-async-queue-size" json:"branchAsyncQueueSize,omitempty" property:"branchAsyncQueueSize"`
	EnableBranchAsyncRemove bool `default:"false" yaml:"enable-branch-async-remove" json:"enableBranchAsyncRemove" property:"enableBranchAsyncRemove"`
}

// Prefix seata
func (ServerConfig) Prefix() string {
	return constant.ServerConfigPrefix
}

// Prefix seata
func (UndoLogConfig) Prefix() string {
	return constant.ServerUndoConfigPrefix
}

// Prefix seata.undo
func (RecoveryConfig) Prefix() string {
	return constant.ServerRecoveryConfigPrefix
}

// Prefix seata.session
func (SessionConfig) Prefix() string {
	return constant.ServerSessionConfigPrefix
}

func (c *ServerConfig) Init() error {
	if c == nil {
		return errors.New("application is null")
	}
	if err := c.check(); err != nil {
		return err
	}
	return nil
}

func (ac *ServerConfig) check() error {
	if err := defaults.Set(ac); err != nil {
		return err
	}
	return verify(ac)
}
