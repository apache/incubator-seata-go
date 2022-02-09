package lock

import (
	"fmt"
)

import (
	"github.com/pkg/errors"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/tc/model"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
)

type DataBaseLocker struct {
	LockStore LockStore
}

func (locker *DataBaseLocker) AcquireLock(branchSession *session.BranchSession) bool {
	if branchSession == nil {
		log.Errorf("branchSession can't be null for memory/file locker.")
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
		log.Info("branchSession can't be null for memory/file locker.")
		panic(errors.New("branchSession can't be null for memory/file locker"))
	}

	return locker.releaseLockByXidBranchID(branchSession.XID, branchSession.BranchID)
}

func (locker *DataBaseLocker) releaseLockByXidBranchID(xid string, branchID int64) bool {
	return locker.LockStore.UnLockByXIDAndBranchID(xid, branchID)
}

func (locker *DataBaseLocker) releaseLockByXidBranchIDs(xid string, branchIDs []int64) bool {
	return locker.LockStore.UnLockByXIDAndBranchIDs(xid, branchIDs)
}

func (locker *DataBaseLocker) ReleaseGlobalSessionLock(globalSession *session.GlobalSession) bool {
	var branchIDs = make([]int64, 0)
	branchSessions := globalSession.GetSortedBranches()
	for _, branchSession := range branchSessions {
		branchIDs = append(branchIDs, branchSession.BranchID)
	}
	return locker.releaseLockByXidBranchIDs(globalSession.XID, branchIDs)
}

func (locker *DataBaseLocker) IsLockable(xid string, resourceID string, lockKey string) bool {
	locks := collectRowLocksByLockKeyResourceIDAndXID(lockKey, resourceID, xid)
	return locker.LockStore.IsLockable(convertToLockDO(locks))
}

func (locker *DataBaseLocker) CleanAllLocks() {

}

func (locker *DataBaseLocker) GetLockKeyCount() int64 {
	return locker.LockStore.GetLockCount()
}

func convertToLockDO(locks []*RowLock) []*model.LockDO {
	lockDOs := make([]*model.LockDO, 0)
	if len(locks) == 0 {
		return lockDOs
	}
	for _, lock := range locks {
		lockDO := &model.LockDO{
			Xid:           lock.XID,
			TransactionID: lock.TransactionID,
			BranchID:      lock.BranchID,
			ResourceID:    lock.ResourceID,
			TableName:     lock.TableName,
			Pk:            lock.Pk,
			RowKey:        getRowKey(lock.ResourceID, lock.TableName, lock.Pk),
		}
		lockDOs = append(lockDOs, lockDO)
	}
	return lockDOs
}

func getRowKey(resourceID string, tableName string, pk string) string {
	return fmt.Sprintf("%s^^^%s^^^%s", resourceID, tableName, pk)
}
