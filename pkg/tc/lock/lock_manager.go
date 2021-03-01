package lock

import (
	"sync"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
)

var lockManager LockManager

type LockManager interface {
	// AcquireLock Acquire lock boolean.
	AcquireLock(branchSession *session.BranchSession) bool

	// ReleaseLock Unlock boolean.
	ReleaseLock(branchSession *session.BranchSession) bool

	// GlobalSession 是没有锁的，所有的锁都在 BranchSession 上，因为
	// BranchSession 才持有资源，释放 GlobalSession 锁是指释放它所有
	// 的 BranchSession 上的锁.
	// ReleaseGlobalSessionLock Unlock boolean.
	ReleaseGlobalSessionLock(globalSession *session.GlobalSession) bool

	// IsLockable Is lockable boolean.
	IsLockable(xid string, resourceID string, lockKey string) bool

	// CleanAllLocks Clean all locks.
	CleanAllLocks()

	GetLockKeyCount() int64
}

func Init() {
	if config.GetStoreConfig().StoreMode == "db" {
		lockStore := &LockStoreDataBaseDao{engine: config.GetStoreConfig().DBStoreConfig.Engine}
		lockManager = &DataBaseLocker{LockStore: lockStore}
	} else {
		lockManager = &MemoryLocker{
			LockMap:      &sync.Map{},
			BucketHolder: &sync.Map{},
		}
	}
}

func GetLockManager() LockManager {
	return lockManager
}
