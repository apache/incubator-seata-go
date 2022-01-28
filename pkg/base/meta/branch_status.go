package meta

import (
	"fmt"
)

// BranchStatus
type BranchStatus byte

const (
	// The BranchStatus_Unknown.
	// description:BranchStatus_Unknown branch status.
	BranchStatusUnknown BranchStatus = iota

	// The BranchStatus_Registered.
	// description:BranchStatus_Registered to TC.
	BranchStatusRegistered

	// The Phase one done.
	// description:Branch logic is successfully done at phase one.
	BranchStatusPhaseOneDone

	// The Phase one failed.
	// description:Branch logic is failed at phase one.
	BranchStatusPhaseOneFailed

	// The Phase one timeout.
	// description:Branch logic is NOT reported for a timeout.
	BranchStatusPhaseOneTimeout

	// The Phase two committed.
	// description:Commit logic is successfully done at phase two.
	BranchStatusPhaseTwoCommitted

	// The Phase two commit failed retryable.
	// description:Commit logic is failed but retryable.
	BranchStatusPhaseTwoCommitFailedRetryable

	// The Phase two commit failed can not retry.
	// description:Commit logic is failed and NOT retryable.
	BranchStatusPhaseTwoCommitFailedCanNotRetry

	// The Phase two rolled back.
	// description:Rollback logic is successfully done at phase two.
	BranchStatusPhaseTwoRolledBack

	// The Phase two rollback failed retryable.
	// description:Rollback logic is failed but retryable.
	BranchStatusPhaseTwoRollbackFailedRetryable

	// The Phase two rollback failed can not retry.
	// description:Rollback logic is failed but NOT retryable.
	BranchStatusPhaseTwoRollbackFailedCanNotRetry
)

func (s BranchStatus) String() string {
	switch s {
	case BranchStatusUnknown:
		return "Unknown"
	case BranchStatusRegistered:
		return "Registered"
	case BranchStatusPhaseOneDone:
		return "PhaseOneDone"
	case BranchStatusPhaseOneFailed:
		return "PhaseOneFailed"
	case BranchStatusPhaseOneTimeout:
		return "PhaseOneTimeout"
	case BranchStatusPhaseTwoCommitted:
		return "PhaseTwoCommitted"
	case BranchStatusPhaseTwoCommitFailedRetryable:
		return "PhaseTwoCommitFailedRetryable"
	case BranchStatusPhaseTwoCommitFailedCanNotRetry:
		return "CommitFailedCanNotRetry"
	case BranchStatusPhaseTwoRolledBack:
		return "PhaseTwoRolledBack"
	case BranchStatusPhaseTwoRollbackFailedRetryable:
		return "RollbackFailedRetryable"
	case BranchStatusPhaseTwoRollbackFailedCanNotRetry:
		return "RollbackFailedCanNotRetry"
	default:
		return fmt.Sprintf("%d", s)
	}
}
