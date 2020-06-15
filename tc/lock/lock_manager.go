package lock

import (
	"sync"

	"github.com/xiaobudongzhang/seata-golang/tc/config"
	"github.com/xiaobudongzhang/seata-golang/tc/session"
)

var lockManager LockManager

type LockManager interface {
	// Acquire lock boolean.
	AcquireLock(branchSession *session.BranchSession) bool

	// Un lock boolean.
	ReleaseLock(branchSession *session.BranchSession) bool

	// GlobalSession 是没有锁的，所有的锁都在 BranchSession 上，因为 BranchSession 才
	// 持有资源，释放 GlobalSession 锁是指释放它所有的 BranchSession 上的锁
	// Un lock boolean.
	ReleaseGlobalSessionLock(globalSession *session.GlobalSession) bool

	// Is lockable boolean.
	IsLockable(xid string, resourceId string, lockKey string) bool

	// Clean all locks.
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
