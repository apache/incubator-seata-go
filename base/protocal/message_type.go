package protocal

type MessageType int16

const (
	/**
	 * The constant TYPE_GLOBAL_BEGIN.
	 */
	TypeGlobalBegin = 1
	/**
	 * The constant TYPE_GLOBAL_BEGIN_RESULT.
	 */
	TypeGlobalBeginResult = 2
	/**
	 * The constant TYPE_GLOBAL_COMMIT.
	 */
	TypeGlobalCommit = 7
	/**
	 * The constant TYPE_GLOBAL_COMMIT_RESULT.
	 */
	TypeGlobalCommitResult = 8
	/**
	 * The constant TYPE_GLOBAL_ROLLBACK.
	 */
	TypeGlobalRollback = 9
	/**
	 * The constant TYPE_GLOBAL_ROLLBACK_RESULT.
	 */
	TypeGlobalRollbackResult = 10
	/**
	 * The constant TYPE_GLOBAL_STATUS.
	 */
	TypeGlobalStatus = 15
	/**
	 * The constant TYPE_GLOBAL_STATUS_RESULT.
	 */
	TypeGlobalStatusResult = 16
	/**
	 * The constant TYPE_GLOBAL_REPORT.
	 */
	TypeGlobalReport = 17
	/**
	 * The constant TYPE_GLOBAL_REPORT_RESULT.
	 */
	TypeGlobalReportResult = 18
	/**
	 * The constant TYPE_GLOBAL_LOCK_QUERY.
	 */
	TypeGlobalLockQuery = 21
	/**
	 * The constant TYPE_GLOBAL_LOCK_QUERY_RESULT.
	 */
	TypeGlobalLockQueryResult = 22

	/**
	 * The constant TYPE_BRANCH_COMMIT.
	 */
	TypeBranchCommit = 3
	/**
	 * The constant TYPE_BRANCH_COMMIT_RESULT.
	 */
	TypeBranchCommitResult = 4
	/**
	 * The constant TYPE_BRANCH_ROLLBACK.
	 */
	TypeBranchRollback = 5
	/**
	 * The constant TYPE_BRANCH_ROLLBACK_RESULT.
	 */
	TypeBranchRollbackResult = 6
	/**
	 * The constant TYPE_BRANCH_REGISTER.
	 */
	TypeBranchRegister = 11
	/**
	 * The constant TYPE_BRANCH_REGISTER_RESULT.
	 */
	TypeBranchRegisterResult = 12
	/**
	 * The constant TYPE_BRANCH_STATUS_REPORT.
	 */
	TypeBranchStatusReport = 13
	/**
	 * The constant TYPE_BRANCH_STATUS_REPORT_RESULT.
	 */
	TypeBranchStatusReportResult = 14

	/**
	 * The constant TYPE_SEATA_MERGE.
	 */
	TypeSeataMerge = 59
	/**
	 * The constant TYPE_SEATA_MERGE_RESULT.
	 */
	TypeSeataMergeResult = 60

	/**
	 * The constant TYPE_REG_CLT.
	 */
	TypeRegClt = 101
	/**
	 * The constant TYPE_REG_CLT_RESULT.
	 */
	TypeRegCltResult = 102
	/**
	 * The constant TYPE_REG_RM.
	 */
	TypeRegRm = 103
	/**
	 * The constant TYPE_REG_RM_RESULT.
	 */
	TypeRegRmResult = 104
	/**
	 * The constant TYPE_RM_DELETE_UNDOLOG.
	 */
	TypeRmDeleteUndolog = 111
)
