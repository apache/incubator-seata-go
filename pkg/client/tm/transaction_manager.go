package tm

import "github.com/transaction-wg/seata-golang/pkg/base/meta"

type TransactionManager interface {
	// GlobalStatus_Begin a new global transaction.
	Begin(applicationID string, transactionServiceGroup string, name string, timeout int32) (string, error)

	// Global commit.
	Commit(xid string) (meta.GlobalStatus, error)

	// Global rollback.
	Rollback(xid string) (meta.GlobalStatus, error)

	// Get current status of the give transaction.
	GetStatus(xid string) (meta.GlobalStatus, error)

	// Global report.
	GlobalReport(xid string, globalStatus meta.GlobalStatus) (meta.GlobalStatus, error)
}
