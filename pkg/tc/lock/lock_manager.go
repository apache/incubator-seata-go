package lock

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
)

type Manager struct {
	manager storage.LockManager
}

func NewLockManager(manager storage.LockManager) *Manager {
	return &Manager{manager: manager}
}

func (locker *Manager) AcquireLock(branchSession *apis.BranchSession) bool {
	if branchSession == nil {
		log.Debug("branchSession can't be null for memory/file locker.")
		return true
	}

	if branchSession.LockKey == "" {
		return true
	}

	locks := storage.CollectBranchSessionRowLocks(branchSession)
	if len(locks) == 0 {
		return true
	}

	return locker.manager.AcquireLock(locks)
}

func (locker *Manager) ReleaseLock(branchSession *apis.BranchSession) bool {
	if branchSession == nil {
		log.Debug("branchSession can't be null for memory/file locker.")
		return true
	}

	if branchSession.LockKey == "" {
		return true
	}

	locks := storage.CollectBranchSessionRowLocks(branchSession)
	if len(locks) == 0 {
		return true
	}

	return locker.manager.ReleaseLock(locks)
}

func (locker *Manager) ReleaseGlobalSessionLock(globalTransaction *model.GlobalTransaction) bool {
	locks := make([]*apis.RowLock, 0)
	for branchSession := range globalTransaction.BranchSessions {
		rowLocks := storage.CollectBranchSessionRowLocks(branchSession)
		locks = append(locks, rowLocks...)
	}
	return locker.manager.ReleaseLock(locks)
}

func (locker *Manager) IsLockable(xid string, resourceID string, lockKey string) bool {
	return locker.manager.IsLockable(xid, resourceID, lockKey)
}
