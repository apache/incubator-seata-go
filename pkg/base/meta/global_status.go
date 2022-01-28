package meta

import (
	"fmt"
)

type GlobalStatus int32

const (
	// Un known global status.
	// BranchStatus_Unknown
	GlobalStatusUnknown GlobalStatus = iota

	// The GlobalStatus_Begin.
	// PHASE 1: can accept new branch registering.
	GlobalStatusBegin

	// PHASE 2: Running Status: may be changed any time.
	// Committing.
	GlobalStatusCommitting

	// The Commit retrying.
	// Retrying commit after a recoverable failure.
	GlobalStatusCommitRetrying

	// Rolling back global status.
	GlobalStatusRollingBack

	// The Rollback retrying.
	// Retrying rollback after a recoverable failure.
	GlobalStatusRollbackRetrying

	// The Timeout rollingBack.
	// RollingBack since timeout
	GlobalStatusTimeoutRollingBack

	// The Timeout rollback retrying.
	// Retrying rollback (since timeout) after a recoverable failure.
	GlobalStatusTimeoutRollbackRetrying

	// All branches can be async committed. The committing is NOT done yet, but it can be seen as committed for TM/RM
	// rpc_client.
	GlobalStatusAsyncCommitting

	// PHASE 2: Final Status: will NOT change any more.
	// Finally: global transaction is successfully committed.
	GlobalStatusCommitted

	// The Commit failed.
	// Finally: failed to commit
	GlobalStatusCommitFailed

	// The Rolled back.
	// Finally: global transaction is successfully rolled back.
	GlobalStatusRolledBack

	// The Rollback failed.
	// Finally: failed to rollback
	GlobalStatusRollbackFailed

	// The Timeout rolled back.
	// Finally: global transaction is successfully rolled back since timeout.
	GlobalStatusTimeoutRolledBack

	// The Timeout rollback failed.
	// Finally: failed to rollback since timeout
	GlobalStatusTimeoutRollbackFailed

	// The Finished.
	// Not managed in getty_session MAP any more
	GlobalStatusFinished
)

// String string of global status
func (s GlobalStatus) String() string {
	switch s {
	case GlobalStatusUnknown:
		return "Unknown"
	case GlobalStatusBegin:
		return "Begin"
	case GlobalStatusCommitting:
		return "Committing"
	case GlobalStatusCommitRetrying:
		return "CommitRetrying"
	case GlobalStatusRollingBack:
		return "RollingBack"
	case GlobalStatusRollbackRetrying:
		return "RollbackRetrying"
	case GlobalStatusTimeoutRollingBack:
		return "TimeoutRollingBack"
	case GlobalStatusTimeoutRollbackRetrying:
		return "TimeoutRollbackRetrying"
	case GlobalStatusAsyncCommitting:
		return "AsyncCommitting"
	case GlobalStatusCommitted:
		return "Committed"
	case GlobalStatusCommitFailed:
		return "CommitFailed"
	case GlobalStatusRolledBack:
		return "RolledBack"
	case GlobalStatusRollbackFailed:
		return "RollbackFailed"
	case GlobalStatusTimeoutRolledBack:
		return "TimeoutRolledBack"
	case GlobalStatusTimeoutRollbackFailed:
		return "TimeoutRollbackFailed"
	case GlobalStatusFinished:
		return "Finished"
	default:
		return fmt.Sprintf("%d", s)
	}
}
