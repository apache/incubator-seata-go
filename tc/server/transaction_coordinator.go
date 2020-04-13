package server

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/client/rm"
	"github.com/dk-lockdown/seata-golang/client/tm"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type ITransactionCoordinatorInbound interface {
	tm.ITransactionManager
	rm.IResourceManagerOutbound
}

type ITransactionCoordinatorOutbound interface {
	/**
	 * Commit a branch transaction.
	 *
	 * @param globalSession the global session
	 * @param branchSession the branch session
	 * @return Status of the branch after committing.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 *                              out.
	 */
	branchCommit(globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error)

	/**
	 * Rollback a branch transaction.
	 *
	 * @param globalSession the global session
	 * @param branchSession the branch session
	 * @return Status of the branch after rollbacking.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 *                              out.
	 */
	branchRollback(globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error)

}

type ITransactionCoordinator interface {
	ITransactionCoordinatorInbound
	ITransactionCoordinatorOutbound

	/**
	 * Do global commit.
	 *
	 * @param globalSession the global session
	 * @param retrying      the retrying
	 * @return is global commit.
	 * @throws TransactionException the transaction exception
	 */
	doGlobalCommit(globalSession *session.GlobalSession, retrying bool) (bool, error)

	/**
	 * Do global rollback.
	 *
	 * @param globalSession the global session
	 * @param retrying      the retrying
	 * @return is global rollback.
	 * @throws TransactionException the transaction exception
	 */
	doGlobalRollback(globalSession *session.GlobalSession, retrying bool) (bool, error)

	/**
	 * Do global report.
	 *
	 * @param globalSession the global session
	 * @param xid           Transaction id.
	 * @param param         the global status
	 * @throws TransactionException the transaction exception
	 */
	doGlobalReport(globalSession *session.GlobalSession, xid string, param meta.GlobalStatus) error
}
