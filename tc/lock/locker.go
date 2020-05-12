package lock

import (
	"github.com/dk-lockdown/seata-golang/tc/config"
	"sync"
)

var lockManager ILockManager

func Init() {
	if config.GetStoreConfig().StoreMode == "db" {
		lockStore := &LockStoreDataBaseDao{engine:config.GetStoreConfig().DBStoreConfig.Engine}
		lockManager = &DataBaseLocker{LockStore:lockStore}
	} else {
		lockManager = &MemoryLocker{
			LockMap:      &sync.Map{},
			BucketHolder: &sync.Map{},
		}
	}
}

func GetLockManager() ILockManager {
	return lockManager
}