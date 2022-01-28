package storage

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
)

// SessionManager stored the globalTransactions and branchTransactions.
type SessionManager interface {
	// Add global session.
	AddGlobalSession(session *apis.GlobalSession) error

	// Find global session.
	FindGlobalSession(xid string) *apis.GlobalSession

	// Find global sessions list.
	FindGlobalSessions(statuses []apis.GlobalSession_GlobalStatus) []*apis.GlobalSession

	// Find global sessions list with addressing identities
	FindGlobalSessionsWithAddressingIdentities(statuses []apis.GlobalSession_GlobalStatus, addressingIdentities []string) []*apis.GlobalSession

	// All sessions collection.
	AllSessions() []*apis.GlobalSession

	// Update global session status.
	UpdateGlobalSessionStatus(session *apis.GlobalSession, status apis.GlobalSession_GlobalStatus) error

	// Inactive global session.
	InactiveGlobalSession(session *apis.GlobalSession) error

	// Remove global session.
	RemoveGlobalSession(session *apis.GlobalSession) error

	// Add branch session.
	AddBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error

	// Find branch session.
	FindBranchSessions(xid string) []*apis.BranchSession

	// Find branch session.
	FindBatchBranchSessions(xids []string) []*apis.BranchSession

	// Update branch session status.
	UpdateBranchSessionStatus(session *apis.BranchSession, status apis.BranchSession_BranchStatus) error

	// Remove branch session.
	RemoveBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error
}

type LockManager interface {
	// AcquireLock Acquire lock boolean.
	AcquireLock(rowLocks []*apis.RowLock, skipCheckLock bool) bool

	// ReleaseLock Unlock boolean.
	ReleaseLock(rowLocks []*apis.RowLock) bool

	// IsLockable Is lockable boolean.
	IsLockable(xid string, resourceID string, lockKey string) bool
}

type Driver interface {
	SessionManager
	LockManager
}
