package lock

import (
	"fmt"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/model"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type DataBaseLocker struct {
	LockStore LockStore
}

func (locker *DataBaseLocker) AcquireLock(branchSession *session.BranchSession) bool {
	if branchSession == nil {
		logging.Logger.Errorf("branchSession can't be null for memory/file locker.")
		panic(errors.New("branchSession can't be null for memory/file locker."))
	}

	lockKey := branchSession.LockKey
	if lockKey == "" {
		return true
	}

	locks := collectRowLocksByBranchSession(branchSession)
	if locks == nil {
		return true
	}

	return locker.LockStore.AcquireLock(convertToLockDO(locks))
}

func (locker *DataBaseLocker) ReleaseLock(branchSession *session.BranchSession) bool {
	if branchSession == nil {
		logging.Logger.Info("branchSession can't be null for memory/file locker.")
		panic(errors.New("branchSession can't be null for memory/file locker"))
	}

	return locker.releaseLockByXidBranchId(branchSession.Xid, branchSession.BranchId)
}

func (locker *DataBaseLocker) releaseLockByXidBranchId(xid string, branchId int64) bool {
	return locker.LockStore.UnLockByXidAndBranchId(xid, branchId)
}

func (locker *DataBaseLocker) releaseLockByXidBranchIds(xid string, branchIds []int64) bool {
	return locker.LockStore.UnLockByXidAndBranchIds(xid, branchIds)
}

func (locker *DataBaseLocker) ReleaseGlobalSessionLock(globalSession *session.GlobalSession) bool {
	var branchIds = make([]int64, 0)
	branchSessions := globalSession.GetSortedBranches()
	for _, branchSession := range branchSessions {
		branchIds = append(branchIds, branchSession.BranchId)
	}
	return locker.releaseLockByXidBranchIds(globalSession.Xid, branchIds)
}

func (locker *DataBaseLocker) IsLockable(xid string, resourceId string, lockKey string) bool {
	locks := collectRowLocksByLockKeyResourceIdXid(lockKey, resourceId, xid)
	return locker.LockStore.IsLockable(convertToLockDO(locks))
}

func (locker *DataBaseLocker) CleanAllLocks() {

}

func (locker *DataBaseLocker) GetLockKeyCount() int64 {
	return locker.LockStore.GetLockCount()
}

func convertToLockDO(locks []*RowLock) []*model.LockDO {
	lockDOs := make([]*model.LockDO, 0)
	if locks == nil || len(locks) == 0 {
		return lockDOs
	}
	for _, lock := range locks {
		lockDO := &model.LockDO{
			Xid:           lock.Xid,
			TransactionId: lock.TransactionId,
			BranchId:      lock.BranchId,
			ResourceId:    lock.ResourceId,
			TableName:     lock.TableName,
			Pk:            lock.Pk,
			RowKey:        getRowKey(lock.ResourceId, lock.TableName, lock.Pk),
		}
		lockDOs = append(lockDOs, lockDO)
	}
	return lockDOs
}

func getRowKey(resourceId string, tableName string, pk string) string {
	return fmt.Sprintf("%s^^^%s^^^%s", resourceId, tableName, pk)
}
