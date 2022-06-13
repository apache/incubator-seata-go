package transaction

type GlobalStatus int64

const (

	/**
	 * Un known global status.
	 */
	// Unknown
	GlobalStatusUnKnown GlobalStatus = 0

	/**
	 * The GlobalStatusBegin.
	 */
	// PHASE 1: can accept new branch registering.
	GlobalStatusBegin GlobalStatus = 1

	/**
	 * PHASE 2: Running Status: may be changed any time.
	 */
	// Committing.
	GlobalStatusCommitting GlobalStatus = 2

	/**
	 * The Commit retrying.
	 */
	// Retrying commit after a recoverable failure.
	GlobalStatusCommitRetrying GlobalStatus = 3

	/**
	 * Rollbacking global status.
	 */
	// Rollbacking
	GlobalStatusRollbacking GlobalStatus = 4

	/**
	 * The Rollback retrying.
	 */
	// Retrying rollback after a recoverable failure.
	GlobalStatusRollbackRetrying GlobalStatus = 5

	/**
	 * The Timeout rollbacking.
	 */
	// Rollbacking since timeout
	GlobalStatusTimeoutRollbacking GlobalStatus = 6

	/**
	 * The Timeout rollback retrying.
	 */
	// Retrying rollback  GlobalStatus = since timeout) after a recoverable failure.
	GlobalStatusTimeoutRollbackRetrying GlobalStatus = 7

	/**
	 * All branches can be async committed. The committing is NOT done yet, but it can be seen as committed for TM/RM
	 * client.
	 */
	GlobalStatusAsyncCommitting GlobalStatus = 8

	/**
	 * PHASE 2: Final Status: will NOT change any more.
	 */
	// Finally: global transaction is successfully committed.
	GlobalStatusCommitted GlobalStatus = 9

	/**
	 * The Commit failed.
	 */
	// Finally: failed to commit
	GlobalStatusCommitFailed GlobalStatus = 10

	/**
	 * The Rollbacked.
	 */
	// Finally: global transaction is successfully rollbacked.
	GlobalStatusRollbacked GlobalStatus = 11

	/**
	 * The Rollback failed.
	 */
	// Finally: failed to rollback
	GlobalStatusRollbackFailed GlobalStatus = 12

	/**
	 * The Timeout rollbacked.
	 */
	// Finally: global transaction is successfully rollbacked since timeout.
	GlobalStatusTimeoutRollbacked GlobalStatus = 13

	/**
	 * The Timeout rollback failed.
	 */
	// Finally: failed to rollback since timeout
	GlobalStatusTimeoutRollbackFailed GlobalStatus = 14

	/**
	 * The Finished.
	 */
	// Not managed in session MAP any more
	GlobalStatusFinished GlobalStatus = 15
)
