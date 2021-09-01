package lock

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
)

type LockManager struct {
	manager storage.LockManager
}

func NewLockManager(manager storage.LockManager) *LockManager {
	return &LockManager{manager: manager}
}

func (locker *LockManager) AcquireLock(branchSession *apis.BranchSession) bool {
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

func (locker *LockManager) ReleaseLock(branchSession *apis.BranchSession) bool {
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

func (locker *LockManager) ReleaseGlobalSessionLock(globalTransaction *model.GlobalTransaction) bool {
	locks := make([]*apis.RowLock, 0)
	for branchSession := range globalTransaction.BranchSessions {
		rowLocks := storage.CollectBranchSessionRowLocks(branchSession)
		locks = append(locks, rowLocks...)
	}
	return locker.manager.ReleaseLock(locks)
}

func (locker *LockManager) IsLockable(xid string, resourceID string, lockKey string) bool {
	return locker.manager.IsLockable(xid, resourceID, lockKey)
}
