package model

type TransactionManager interface {
	// Begin a new global transaction.
	Begin(applicationId, transactionServiceGroup, name string, timeout int64) (string, error)

	// Global commit.
	Commit(xid string) (GlobalStatus, error)

	//Global rollback.
	Rollback(xid string) (GlobalStatus, error)

	// Get current status of the give transaction.
	GetStatus(xid string) (GlobalStatus, error)

	// Global report.
	GlobalReport(xid string, globalStatus GlobalStatus) (GlobalStatus, error)
}
