package lock

import (
	"github.com/dk-lockdown/seata-golang/logging"
	"github.com/dk-lockdown/seata-golang/model"
	"github.com/dk-lockdown/seata-golang/tc/session"
	"github.com/dk-lockdown/seata-golang/util"
	"github.com/pkg/errors"
	"strconv"
	"sync"
	"sync/atomic"
)

const BucketPerTable = 128

type MemoryLocker struct {
	LockMap *sync.Map
	// 高流量下，锁资源越多，BucketHolder 的性能越下降
	BucketHolder *sync.Map

	LockKeyCount int64
}


func (ml *MemoryLocker) AcquireLock(branchSession *session.BranchSession) (bool, error) {
	if branchSession == nil {
		logging.Logger.Errorf("branchSession can't be null for memory/file locker.")
		return false, errors.New("branchSession can't be null for memory/file locker.")
	}

	lockKey := branchSession.LockKey
	if lockKey == "" {
		return true,nil
	}

	locks := collectRowLocksByBranchSession(branchSession)
	if locks == nil { return true,nil }
	return ml.acquireLockByRowLocks(branchSession,locks)
}


func (ml *MemoryLocker) ReleaseLock(branchSession *session.BranchSession) (bool, error) {
	if branchSession == nil {
		logging.Logger.Info("branchSession can't be null for memory/file locker.")
		return false,errors.New("branchSession can't be null for memory/file locker")
	}

	locks := collectRowLocksByBranchSession(branchSession)
	return ml.releaseLockByRowLocks(branchSession,locks)
}

func (ml *MemoryLocker) ReleaseGlobalSessionLock(globalSession *session.GlobalSession) (bool, error) {
	branchSessions := globalSession.GetSortedBranches()
	releaseLockResult := true
	for _,branchSession := range branchSessions {
		ok, err := ml.ReleaseLock(branchSession)
		if err != nil {
			return ok,err
		}
		if !ok { releaseLockResult = false }

	}
	return releaseLockResult, nil
}

func (ml *MemoryLocker) IsLockable(xid string, resourceId string, lockKey string) bool {
	locks := collectRowLocksByLockKeyResourceIdXid(lockKey, resourceId, xid)
	return ml.isLockableByRowLocks(locks)
}

func  (ml *MemoryLocker) CleanAllLocks() {
	ml.LockMap = &sync.Map{}
	ml.BucketHolder = &sync.Map{}
	ml.LockKeyCount = 0
}

func  (ml *MemoryLocker) GetLockKeyCount() int64 {
	return ml.LockKeyCount
}

// AcquireLock 申请锁资源，resourceId -> tableName -> bucketId -> pk -> transactionId
func (ml *MemoryLocker) acquireLockByRowLocks(branchSession *session.BranchSession,rowLocks []*RowLock) (bool, error) {
	if rowLocks == nil { return true, nil }

	resourceId := branchSession.ResourceId
	transactionId := branchSession.TransactionId

	dbLockMap,_ := ml.LockMap.LoadOrStore(resourceId,&sync.Map{})

	cDbLockMap := dbLockMap.(*sync.Map)
	for _, rowLock := range rowLocks {
		tableLockMap,_ := cDbLockMap.LoadOrStore(rowLock.TableName,&sync.Map{})

		cTableLockMap := tableLockMap.(*sync.Map)

		bucketId := util.String(rowLock.Pk) % BucketPerTable
		bucketKey := strconv.Itoa(bucketId)
		bucketLockMap,_ := cTableLockMap.LoadOrStore(bucketKey,&sync.Map{})

		cBucketLockMap := bucketLockMap.(*sync.Map)

		previousLockTransactionId,loaded := cBucketLockMap.LoadOrStore(rowLock.Pk, transactionId)
		if !loaded {

			//No existing rowLock, and now locked by myself
			keysInHolder,_ := ml.BucketHolder.LoadOrStore(cBucketLockMap, model.NewSet())

			sKeysInHolder := keysInHolder.(*model.Set)
			sKeysInHolder.Add(rowLock.Pk)

			atomic.AddInt64(&ml.LockKeyCount,1)
		} else if previousLockTransactionId == transactionId {
			// Locked by me before
			continue
		} else {
			logging.Logger.Infof("Global rowLock on [%s:%s] is holding by %d", rowLock.TableName, rowLock.Pk,previousLockTransactionId)
			// branchSession unlock
			_,err := ml.ReleaseLock(branchSession)
			return false,err
		}
	}

	return true, nil
}

func (ml *MemoryLocker) releaseLockByRowLocks(branchSession *session.BranchSession,rowLocks []*RowLock) (bool,error) {
	if rowLocks == nil { return false, nil }

	releaseLock := func (key, value interface{}) bool {
		cBucketLockMap := key.(*sync.Map)
		keys := value.(*model.Set)

		for _, key := range keys.List() {
			transId, ok := cBucketLockMap.Load(key)
			if ok && transId == branchSession.TransactionId {
				cBucketLockMap.Delete(key)
				// keys.List() 是一个新的 slice，移除 key 并不会导致错误发生
				keys.Remove(key)
				atomic.AddInt64(&ml.LockKeyCount,-1)
			}
		}
		return true
	}

	ml.BucketHolder.Range(releaseLock)

	return true, nil
}

func  (ml *MemoryLocker) isLockableByRowLocks(rowLocks []*RowLock) bool {
	if rowLocks == nil { return true }

	resourceId := rowLocks[0].ResourceId
	transactionId := rowLocks[0].TransactionId

	dbLockMap, ok := ml.LockMap.Load(resourceId)
	if  !ok {
		return true
	}

	cDbLockMap := dbLockMap.(*sync.Map)
	for _, rowLock := range rowLocks {
		tableLockMap,ok := cDbLockMap.Load(rowLock.TableName)
		if !ok {
			continue
		}
		cTableLockMap := tableLockMap.(*sync.Map)

		bucketId := util.String(rowLock.Pk) % BucketPerTable
		bucketKey := strconv.Itoa(bucketId)
		bucketLockMap,ok := cTableLockMap.Load(bucketKey)
		if !ok {
			continue
		}
		cBucketLockMap := bucketLockMap.(*sync.Map)

		previousLockTransactionId,ok := cBucketLockMap.Load(rowLock.Pk)
		if !ok || previousLockTransactionId == transactionId {
			// Locked by me before
			continue
		} else {
			logging.Logger.Infof("Global rowLock on [%s:%s] is holding by %d", rowLock.TableName, rowLock.Pk,previousLockTransactionId)
			return false
		}
	}

	return true
}