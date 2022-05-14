package protocol

type MessageType byte

const (
	/**
	 * The constant TYPE_GLOBAL_BEGIN.
	 */
	MessageTypeGlobalBegin MessageType = 1
	/**
	 * The constant TYPE_GLOBAL_BEGIN_RESULT.
	 */
	MessageTypeGlobalBeginResult MessageType = 2
	/**
	 * The constant TYPE_GLOBAL_COMMIT.
	 */
	MessageTypeGlobalCommit MessageType = 7
	/**
	 * The constant TYPE_GLOBAL_COMMIT_RESULT.
	 */
	MessageTypeGlobalCommitResult MessageType = 8
	/**
	 * The constant TYPE_GLOBAL_ROLLBACK.
	 */
	MessageTypeGlobalRollback MessageType = 9
	/**
	 * The constant TYPE_GLOBAL_ROLLBACK_RESULT.
	 */
	MessageTypeGlobalRollbackResult MessageType = 10
	/**
	 * The constant TYPE_GLOBAL_STATUS.
	 */
	MessageTypeGlobalStatus MessageType = 15
	/**
	 * The constant TYPE_GLOBAL_STATUS_RESULT.
	 */
	MessageTypeGlobalStatusResult MessageType = 16
	/**
	 * The constant TYPE_GLOBAL_REPORT.
	 */
	MessageTypeGlobalReport MessageType = 17
	/**
	 * The constant TYPE_GLOBAL_REPORT_RESULT.
	 */
	MessageTypeGlobalReportResult MessageType = 18
	/**
	 * The constant TYPE_GLOBAL_LOCK_QUERY.
	 */
	MessageTypeGlobalLockQuery MessageType = 21
	/**
	 * The constant TYPE_GLOBAL_LOCK_QUERY_RESULT.
	 */
	MessageTypeGlobalLockQueryResult MessageType = 22

	/**
	 * The constant TYPE_BRANCH_COMMIT.
	 */
	MessageTypeBranchCommit MessageType = 3
	/**
	 * The constant TYPE_BRANCH_COMMIT_RESULT.
	 */
	MessageTypeBranchCommitResult MessageType = 4
	/**
	 * The constant TYPE_BRANCH_ROLLBACK.
	 */
	MessageTypeBranchRollback MessageType = 5
	/**
	 * The constant TYPE_BRANCH_ROLLBACK_RESULT.
	 */
	MessageTypeBranchRollbackResult MessageType = 6
	/**
	 * The constant TYPE_BRANCH_REGISTER.
	 */
	MessageTypeBranchRegister MessageType = 11
	/**
	 * The constant TYPE_BRANCH_REGISTER_RESULT.
	 */
	MessageTypeBranchRegisterResult MessageType = 12
	/**
	 * The constant TYPE_BRANCH_STATUS_REPORT.
	 */
	MessageTypeBranchStatusReport MessageType = 13
	/**
	 * The constant TYPE_BRANCH_STATUS_REPORT_RESULT.
	 */
	MessageTypeBranchStatusReportResult MessageType = 14

	/**
	 * The constant TYPE_SEATA_MERGE.
	 */
	MessageTypeSeataMerge MessageType = 59
	/**
	 * The constant TYPE_SEATA_MERGE_RESULT.
	 */
	MessageTypeSeataMergeResult MessageType = 60

	/**
	 * The constant TYPE_REG_CLT.
	 */
	MessageTypeRegClt MessageType = 101
	/**
	 * The constant TYPE_REG_CLT_RESULT.
	 */
	MessageTypeRegCltResult MessageType = 102
	/**
	 * The constant TYPE_REG_RM.
	 */
	MessageTypeRegRm MessageType = 103
	/**
	 * The constant TYPE_REG_RM_RESULT.
	 */
	MessageTypeRegRmResult MessageType = 104
	/**
	 * The constant TYPE_RM_DELETE_UNDOLOG.
	 */
	MessageTypeRmDeleteUndolog MessageType = 111
	/**
	 * the constant TYPE_HEARTBEAT_MSG
	 */
	MessageTypeHeartbeatMsg MessageType = 120
)
