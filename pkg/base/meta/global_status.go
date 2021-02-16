package meta

import "fmt"

type GlobalStatus int32

const (
	/**
	 * Un known global status.
	 */
	// BranchStatus_Unknown
	GlobalStatusUnknown GlobalStatus = iota

	/**
	 * The GlobalStatus_Begin.
	 */
	// PHASE 1: can accept new branch registering.
	GlobalStatusBegin

	/**
	 * PHASE 2: Running Status: may be changed any time.
	 */
	// Committing.
	GlobalStatusCommitting

	/**
	 * The Commit retrying.
	 */
	// Retrying commit after a recoverable failure.
	GlobalStatusCommitRetrying

	/**
	 * Rollbacking global status.
	 */
	// Rollbacking
	GlobalStatusRollbacking

	/**
	 * The Rollback retrying.
	 */
	// Retrying rollback after a recoverable failure.
	GlobalStatusRollbackRetrying

	/**
	 * The Timeout rollbacking.
	 */
	// Rollbacking since timeout
	GlobalStatusTimeoutRollbacking

	/**
	 * The Timeout rollback retrying.
	 */
	// Retrying rollback (since timeout) after a recoverable failure.
	GlobalStatusTimeoutRollbackRetrying

	/**
	 * All branches can be async committed. The committing is NOT done yet, but it can be seen as committed for TM/RM
	 * rpc_client.
	 */
	GlobalStatusAsyncCommitting

	/**
	 * PHASE 2: Final Status: will NOT change any more.
	 */
	// Finally: global transaction is successfully committed.
	GlobalStatusCommitted

	/**
	 * The Commit failed.
	 */
	// Finally: failed to commit
	GlobalStatusCommitFailed

	/**
	 * The Rollbacked.
	 */
	// Finally: global transaction is successfully rollbacked.
	GlobalStatusRollbacked

	/**
	 * The Rollback failed.
	 */
	// Finally: failed to rollback
	GlobalStatusRollbackFailed

	/**
	 * The Timeout rollbacked.
	 */
	// Finally: global transaction is successfully rollbacked since timeout.
	GlobalStatusTimeoutRollbacked

	/**
	 * The Timeout rollback failed.
	 */
	// Finally: failed to rollback since timeout
	GlobalStatusTimeoutRollbackFailed

	/**
	 * The Finished.
	 */
	// Not managed in getty_session MAP any more
	GlobalStatusFinished
)

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
	case GlobalStatusRollbacking:
		return "Rollbacking"
	case GlobalStatusRollbackRetrying:
		return "RollbackRetrying"
	case GlobalStatusTimeoutRollbacking:
		return "TimeoutRollbacking"
	case GlobalStatusTimeoutRollbackRetrying:
		return "TimeoutRollbackRetrying"
	case GlobalStatusAsyncCommitting:
		return "AsyncCommitting"
	case GlobalStatusCommitted:
		return "Committed"
	case GlobalStatusCommitFailed:
		return "CommitFailed"
	case GlobalStatusRollbacked:
		return "Rollbacked"
	case GlobalStatusRollbackFailed:
		return "RollbackFailed"
	case GlobalStatusTimeoutRollbacked:
		return "TimeoutRollbacked"
	case GlobalStatusTimeoutRollbackFailed:
		return "TimeoutRollbackFailed"
	case GlobalStatusFinished:
		return "Finished"
	default:
		return fmt.Sprintf("%d", s)
	}
}
