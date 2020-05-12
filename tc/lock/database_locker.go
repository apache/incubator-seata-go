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
	LockStore ILockStore
}

func (locker *DataBaseLocker) AcquireLock(branchSession *session.BranchSession) (bool, error) {
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

	return locker.LockStore.AcquireLock(convertToLockDO(locks)),nil
}

func (locker *DataBaseLocker) ReleaseLock(branchSession *session.BranchSession) (bool, error) {
	if branchSession == nil {
		logging.Logger.Info("branchSession can't be null for memory/file locker.")
		return false,errors.New("branchSession can't be null for memory/file locker")
	}

	locks := collectRowLocksByBranchSession(branchSession)

	return locker.LockStore.UnLock(convertToLockDO(locks)),nil
}

func (locker *DataBaseLocker) ReleaseGlobalSessionLock(globalSession *session.GlobalSession) (bool, error) {
	branchSessions := globalSession.GetSortedBranches()
	releaseLockResult := true
	for _,branchSession := range branchSessions {
		ok, err := locker.ReleaseLock(branchSession)
		if err != nil {
			return ok,err
		}
		if !ok { releaseLockResult = false }

	}
	return releaseLockResult, nil
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
	lockDOs := make([]*model.LockDO,0)
	if locks == nil || len(locks) == 0 {
		return lockDOs
	}
	for _,lock := range locks {
		lockDO := &model.LockDO{
			Xid:           lock.Xid,
			TransactionId: lock.TransactionId,
			BranchId:      lock.BranchId,
			ResourceId:    lock.ResourceId,
			TableName:     lock.TableName,
			Pk:            lock.Pk,
			RowKey:        getRowKey(lock.ResourceId,lock.TableName,lock.Pk),
		}
		lockDOs = append(lockDOs, lockDO)
	}
	return lockDOs
}

func getRowKey(resourceId string,tableName string,pk string) string {
	return fmt.Sprintf("%s^^^%s^^^%s",resourceId,tableName,pk)
}