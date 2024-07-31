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
)

type (
	BranchType   int8
	BranchStatus int8
)

const (
	BranchTypeUnknow BranchType = -1
	BranchTypeAT     BranchType = 0
	BranchTypeTCC    BranchType = 1
	BranchTypeSAGA   BranchType = 2
	BranchTypeXA     BranchType = 3
)

const (
	// BranchStatusUnknown the BranchStatus_Unknown. description:BranchStatus_Unknown branch status.
	BranchStatusUnknown = iota

	// BranchStatusRegistered the BranchStatus_Registered. description:BranchStatus_Registered to TC.
	BranchStatusRegistered

	// BranchStatusPhaseoneDone the Phase one done. description:Branch logic is successfully done at phase one.
	BranchStatusPhaseoneDone

	// BranchStatusPhaseoneFailed the Phase one failed. description:Branch logic is failed at phase one.
	BranchStatusPhaseoneFailed

	// BranchStatusPhaseoneTimeout the Phase one timeout. description:Branch logic is NOT reported for a timeout.
	BranchStatusPhaseoneTimeout

	// BranchStatusPhasetwoCommitted the Phase two committed. description:Commit logic is successfully done at phase two.
	BranchStatusPhasetwoCommitted

	// BranchStatusPhasetwoCommitFailedRetryable the Phase two commit failed retryable. description:Commit logic is failed but retryable.
	BranchStatusPhasetwoCommitFailedRetryable

	// BranchStatusPhasetwoCommitFailedUnretryable the Phase two commit failed unretryable.
	// description:Commit logic is failed and NOT retryable.
	BranchStatusPhasetwoCommitFailedUnretryable

	// BranchStatusPhasetwoRollbacked The Phase two rollbacked.
	// description:Rollback logic is successfully done at phase two.
	BranchStatusPhasetwoRollbacked

	// BranchStatusPhasetwoRollbackFailedRetryable the Phase two rollback failed retryable.
	// description:Rollback logic is failed but retryable.
	BranchStatusPhasetwoRollbackFailedRetryable

	// BranchStatusPhasetwoRollbackFailedUnretryable the Phase two rollback failed unretryable.
	// description:Rollback logic is failed but NOT retryable.
	BranchStatusPhasetwoRollbackFailedUnretryable
)

func (s BranchStatus) String() string {
	switch s {
	case BranchStatusUnknown:
		return "Unknown"
	case BranchStatusRegistered:
		return "Registered"
	case BranchStatusPhaseoneDone:
		return "PhaseoneDone"
	case BranchStatusPhaseoneFailed:
		return "PhaseoneFailed"
	case BranchStatusPhaseoneTimeout:
		return "PhaseoneTimeout"
	case BranchStatusPhasetwoCommitted:
		return "PhasetwoCommitted"
	case BranchStatusPhasetwoCommitFailedRetryable:
		return "PhasetwoCommitFailedRetryable"
	case BranchStatusPhasetwoCommitFailedUnretryable:
		return "CommitFailedUnretryable"
	case BranchStatusPhasetwoRollbacked:
		return "PhasetwoRollbacked"
	case BranchStatusPhasetwoRollbackFailedRetryable:
		return "RollbackFailedRetryable"
	case BranchStatusPhasetwoRollbackFailedUnretryable:
		return "RollbackFailedUnretryable"
	default:
		return fmt.Sprintf("%d", s)
	}
}
