package server

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/rm"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
	"github.com/transaction-wg/seata-golang/pkg/tm"
)

type TransactionCoordinatorInbound interface {
	tm.TransactionManager
	rm.ResourceManagerOutbound
}

type TransactionCoordinatorOutbound interface {
	// Commit a branch transaction.
	branchCommit(globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error)

	// Rollback a branch transaction.
	branchRollback(globalSession *session.GlobalSession, branchSession *session.BranchSession) (meta.BranchStatus, error)
}

type TransactionCoordinator interface {
	TransactionCoordinatorInbound
	TransactionCoordinatorOutbound

	// Do global commit.
	doGlobalCommit(globalSession *session.GlobalSession, retrying bool) (bool, error)

	// Do global rollback.
	doGlobalRollback(globalSession *session.GlobalSession, retrying bool) (bool, error)

	// Do global report.
	doGlobalReport(globalSession *session.GlobalSession, xid string, param meta.GlobalStatus) error
}
