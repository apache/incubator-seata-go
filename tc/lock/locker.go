package lock

import "sync"

var lockManager ILockManager

func init() {
	lockManager = &MemoryLocker{
		LockMap:      &sync.Map{},
		BucketHolder: &sync.Map{},
	}
}

func GetLockManager() ILockManager {
	return lockManager
}