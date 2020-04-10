package tm

import "github.com/dk-lockdown/seata-golang/meta"

type ITransactionManager interface {
	/**
	 * GlobalStatus_Begin a new global transaction.
	 *
	 * @param applicationId           ID of the application who begins this transaction.
	 * @param transactionServiceGroup ID of the transaction service group.
	 * @param name                    Give a name to the global transaction.
	 * @param timeout                 Timeout of the global transaction.
	 * @return XID of the global transaction
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 * out.
	 */
	Begin(applicationId string, transactionServiceGroup string, name string, timeout int32) (string, error)

	/**
	 * Global commit.
	 *
	 * @param xid XID of the global transaction.
	 * @return Status of the global transaction after committing.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 * out.
	 */
	Commit(xid string) (meta.GlobalStatus, error)

	/**
	 * Global rollback.
	 *
	 * @param xid XID of the global transaction
	 * @return Status of the global transaction after rollbacking.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 * out.
	 */
	Rollback(xid string) (meta.GlobalStatus, error)

	/**
	 * Get current status of the give transaction.
	 *
	 * @param xid XID of the global transaction.
	 * @return Current status of the global transaction.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 * out.
	 */
	GetStatus(xid string) (meta.GlobalStatus, error)

	/**
	 * Global report.
	 *
	 * @param xid XID of the global transaction.
	 * @param globalStatus Status of the global transaction.
	 * @return Status of the global transaction.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 * out.
	 */
	GlobalReport(xid string, globalStatus meta.GlobalStatus) (meta.GlobalStatus, error)
}
