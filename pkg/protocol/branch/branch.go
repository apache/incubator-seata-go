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
	/**
	 * The BranchStatus_Unknown.
	 * description:BranchStatus_Unknown branch status.
	 */
	BranchStatusUnknown = BranchStatus(0)

	/**
	 * The BranchStatus_Registered.
	 * description:BranchStatus_Registered to TC.
	 */
	BranchStatusRegistered = BranchStatus(1)

	/**
	 * The Phase one done.
	 * description:Branch logic is successfully done at phase one.
	 */
	BranchStatusPhaseoneDone = BranchStatus(2)

	/**
	 * The Phase one failed.
	 * description:Branch logic is failed at phase one.
	 */
	BranchStatusPhaseoneFailed = BranchStatus(3)

	/**
	 * The Phase one timeout.
	 * description:Branch logic is NOT reported for a timeout.
	 */
	BranchStatusPhaseoneTimeout = BranchStatus(4)

	/**
	 * The Phase two committed.
	 * description:Commit logic is successfully done at phase two.
	 */
	BranchStatusPhasetwoCommitted = BranchStatus(5)

	/**
	 * The Phase two commit failed retryable.
	 * description:Commit logic is failed but retryable.
	 */
	BranchStatusPhasetwoCommitFailedRetryable = BranchStatus(6)

	/**
	 * The Phase two commit failed unretryable.
	 * description:Commit logic is failed and NOT retryable.
	 */
	BranchStatusPhasetwoCommitFailedUnretryable = BranchStatus(7)

	/**
	 * The Phase two rollbacked.
	 * description:Rollback logic is successfully done at phase two.
	 */
	BranchStatusPhasetwoRollbacked = BranchStatus(8)

	/**
	 * The Phase two rollback failed retryable.
	 * description:Rollback logic is failed but retryable.
	 */
	BranchStatusPhasetwoRollbackFailedRetryable = BranchStatus(9)

	/**
	 * The Phase two rollback failed unretryable.
	 * description:Rollback logic is failed but NOT retryable.
	 */
	BranchStatusPhasetwoRollbackFailedUnretryable = BranchStatus(10)
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
