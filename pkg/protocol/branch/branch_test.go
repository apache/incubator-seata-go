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

package branch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchStatus_String_AllCases(t *testing.T) {
	tests := []struct {
		status   BranchStatus
		expected string
	}{
		{BranchStatusUnknown, "Unknown"},
		{BranchStatusRegistered, "Registered"},
		{BranchStatusPhaseoneDone, "PhaseoneDone"},
		{BranchStatusPhaseoneFailed, "PhaseoneFailed"},
		{BranchStatusPhaseoneTimeout, "PhaseoneTimeout"},
		{BranchStatusPhasetwoCommitted, "PhasetwoCommitted"},
		{BranchStatusPhasetwoCommitFailedRetryable, "PhasetwoCommitFailedRetryable"},
		{BranchStatusPhasetwoCommitFailedUnretryable, "CommitFailedUnretryable"},
		{BranchStatusPhasetwoRollbacked, "PhasetwoRollbacked"},
		{BranchStatusPhasetwoRollbackFailedRetryable, "RollbackFailedRetryable"},
		{BranchStatusPhasetwoRollbackFailedUnretryable, "RollbackFailedUnretryable"},
		{BranchStatus(99), "99"}, // default case
	}

	for i, tt := range tests {
		actual := tt.status.String()
		assert.Equal(t, tt.expected, actual, "test case %d failed", i)

	}
}

func TestBranchType_Constants(t *testing.T) {
	assert.Equal(t, int8(-1), int8(BranchTypeUnknow))
	assert.Equal(t, int8(0), int8(BranchTypeAT))
	assert.Equal(t, int8(1), int8(BranchTypeTCC))
	assert.Equal(t, int8(2), int8(BranchTypeSAGA))
	assert.Equal(t, int8(3), int8(BranchTypeXA))
}

func TestBranchStatus_EnumOrder(t *testing.T) {
	for i := 0; i <= int(BranchStatusPhasetwoRollbackFailedUnretryable); i++ {
		s := BranchStatus(i)
		assert.Contains(t, fmt.Sprintf("%T", s), "branch.BranchStatus")
	}
}
