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

	"seata.apache.org/seata-go/pkg/datasource/sql/exec/at"
	"seata.apache.org/seata-go/pkg/rm"
)

// TestInit tests the Init function with various configurations
func TestInit(t *testing.T) {
	tests := []struct {
		name   string
		config rm.LockConfig
		want   rm.LockConfig
	}{
		{
			name: "typical configuration",
			config: rm.LockConfig{
				RetryInterval:                       10 * time.Millisecond,
				RetryTimes:                          30,
				RetryPolicyBranchRollbackOnConflict: true,
			},
			want: rm.LockConfig{
				RetryInterval:                       10 * time.Millisecond,
				RetryTimes:                          30,
				RetryPolicyBranchRollbackOnConflict: true,
			},
		},
		{
			name: "zero values",
			config: rm.LockConfig{
				RetryInterval:                       0,
				RetryTimes:                          0,
				RetryPolicyBranchRollbackOnConflict: false,
			},
			want: rm.LockConfig{
				RetryInterval:                       0,
				RetryTimes:                          0,
				RetryPolicyBranchRollbackOnConflict: false,
			},
		},
		{
			name: "edge case values",
			config: rm.LockConfig{
				RetryInterval:                       5 * time.Second,
				RetryTimes:                          1000,
				RetryPolicyBranchRollbackOnConflict: true,
			},
			want: rm.LockConfig{
				RetryInterval:                       5 * time.Second,
				RetryTimes:                          1000,
				RetryPolicyBranchRollbackOnConflict: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			at.LockConfig = rm.LockConfig{}

			Init(tt.config)

			if at.LockConfig != tt.want {
				t.Errorf("Init() = %+v, want %+v", at.LockConfig, tt.want)
			}
		})
	}
}

// TestInitMultipleCalls tests that calling Init multiple times overwrites the previous configuration
func TestInitMultipleCalls(t *testing.T) {
	firstConfig := rm.LockConfig{
		RetryInterval:                       10 * time.Millisecond,
		RetryTimes:                          10,
		RetryPolicyBranchRollbackOnConflict: true,
	}
	Init(firstConfig)

	if at.LockConfig != firstConfig {
		t.Errorf("After first Init(), got %+v, want %+v", at.LockConfig, firstConfig)
	}

	secondConfig := rm.LockConfig{
		RetryInterval:                       20 * time.Millisecond,
		RetryTimes:                          20,
		RetryPolicyBranchRollbackOnConflict: false,
	}
	Init(secondConfig)

	if at.LockConfig != secondConfig {
		t.Errorf("After second Init(), got %+v, want %+v", at.LockConfig, secondConfig)
	}
}