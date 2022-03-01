package lock

import (
	"encoding/json"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
)

type LockManagerInterface interface {
	AcquireLock(branchSession *apis.BranchSession) bool
	ReleaseLock(branchSession *apis.BranchSession) bool
	ReleaseGlobalSessionLock(globalTransaction *model.GlobalTransaction) bool
	IsLockable(xid string, resourceID string, lockKey string) bool
}

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

	applicationData := branchSession.ApplicationData
	if applicationData != nil && branchSession.Type == apis.AT {
		applicationDataMap := make(map[string]bool)
		err := json.Unmarshal(applicationData, &applicationDataMap)
		if err != nil {
			return locker.manager.AcquireLock(locks, false)
		}
		skipCheckLock := applicationDataMap["skipCheckLock"]
		return locker.manager.AcquireLock(locks, skipCheckLock)
	}
	return locker.manager.AcquireLock(locks, false)
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
