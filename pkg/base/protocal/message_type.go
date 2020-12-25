package protocal

const (
	/**
	 * The constant TYPE_GLOBAL_BEGIN.
	 */
	TypeGlobalBegin int16 = 1
	/**
	 * The constant TYPE_GLOBAL_BEGIN_RESULT.
	 */
	TypeGlobalBeginResult int16 = 2
	/**
	 * The constant TYPE_GLOBAL_COMMIT.
	 */
	TypeGlobalCommit int16 = 7
	/**
	 * The constant TYPE_GLOBAL_COMMIT_RESULT.
	 */
	TypeGlobalCommitResult int16 = 8
	/**
	 * The constant TYPE_GLOBAL_ROLLBACK.
	 */
	TypeGlobalRollback int16 = 9
	/**
	 * The constant TYPE_GLOBAL_ROLLBACK_RESULT.
	 */
	TypeGlobalRollbackResult int16 = 10
	/**
	 * The constant TYPE_GLOBAL_STATUS.
	 */
	TypeGlobalStatus int16 = 15
	/**
	 * The constant TYPE_GLOBAL_STATUS_RESULT.
	 */
	TypeGlobalStatusResult int16 = 16
	/**
	 * The constant TYPE_GLOBAL_REPORT.
	 */
	TypeGlobalReport int16 = 17
	/**
	 * The constant TYPE_GLOBAL_REPORT_RESULT.
	 */
	TypeGlobalReportResult int16 = 18
	/**
	 * The constant TYPE_GLOBAL_LOCK_QUERY.
	 */
	TypeGlobalLockQuery int16 = 21
	/**
	 * The constant TYPE_GLOBAL_LOCK_QUERY_RESULT.
	 */
	TypeGlobalLockQueryResult int16 = 22

	/**
	 * The constant TYPE_BRANCH_COMMIT.
	 */
	TypeBranchCommit int16 = 3
	/**
	 * The constant TYPE_BRANCH_COMMIT_RESULT.
	 */
	TypeBranchCommitResult int16 = 4
	/**
	 * The constant TYPE_BRANCH_ROLLBACK.
	 */
	TypeBranchRollback int16 = 5
	/**
	 * The constant TYPE_BRANCH_ROLLBACK_RESULT.
	 */
	TypeBranchRollbackResult int16 = 6
	/**
	 * The constant TYPE_BRANCH_REGISTER.
	 */
	TypeBranchRegister int16 = 11
	/**
	 * The constant TYPE_BRANCH_REGISTER_RESULT.
	 */
	TypeBranchRegisterResult int16 = 12
	/**
	 * The constant TYPE_BRANCH_STATUS_REPORT.
	 */
	TypeBranchStatusReport int16 = 13
	/**
	 * The constant TYPE_BRANCH_STATUS_REPORT_RESULT.
	 */
	TypeBranchStatusReportResult int16 = 14

	/**
	 * The constant TYPE_SEATA_MERGE.
	 */
	TypeSeataMerge int16 = 59
	/**
	 * The constant TYPE_SEATA_MERGE_RESULT.
	 */
	TypeSeataMergeResult int16 = 60

	/**
	 * The constant TYPE_REG_CLT.
	 */
	TypeRegClt int16 = 101
	/**
	 * The constant TYPE_REG_CLT_RESULT.
	 */
	TypeRegCltResult int16 = 102
	/**
	 * The constant TYPE_REG_RM.
	 */
	TypeRegRm int16 = 103
	/**
	 * The constant TYPE_REG_RM_RESULT.
	 */
	TypeRegRmResult int16 = 104
	/**
	 * The constant TYPE_RM_DELETE_UNDOLOG.
	 */
	TypeRmDeleteUndolog int16 = 111
)
