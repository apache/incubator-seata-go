package lock

import (
	"strings"
)

import (
	"github.com/dk-lockdown/seata-golang/base/common"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

const LOCK_SPLIT = "^^^"

type RowLock struct {
	Xid string

	TransactionId int64

	BranchId int64

	ResourceId string

	TableName string

	Pk string

	RowKey string

	Feature string
}

func collectRowLocksByBranchSession(branchSession *session.BranchSession) []*RowLock {
	if branchSession == nil || branchSession.LockKey == "" {
		return nil
	}
	return collectRowLocks(branchSession.LockKey, branchSession.ResourceId, branchSession.Xid, branchSession.TransactionId, branchSession.BranchId)
}

func collectRowLocksByLockKeyResourceIdXid(lockKey string,
	resourceId string,
	xid string) []*RowLock {

	return collectRowLocks(lockKey, resourceId, xid, common.XID.GetTransactionId(xid), 0)
}

func collectRowLocks(lockKey string,
	resourceId string,
	xid string,
	transactionId int64,
	branchId int64) []*RowLock {
	var locks = make([]*RowLock, 0)
	tableGroupedLockKeys := strings.Split(lockKey, ";")
	for _, tableGroupedLockKey := range tableGroupedLockKeys {
		if tableGroupedLockKey != "" {
			idx := strings.Index(tableGroupedLockKey, ":")
			if idx < 0 {
				return nil
			}

			tableName := tableGroupedLockKey[0:idx]
			mergedPKs := tableGroupedLockKey[idx+1:]

			if mergedPKs == "" {
				return nil
			}

			pks := strings.Split(mergedPKs, ",")
			if len(pks) == 0 {
				return nil
			}

			for _, pk := range pks {
				if pk != "" {
					rowLock := &RowLock{
						Xid:           xid,
						TransactionId: transactionId,
						BranchId:      branchId,
						ResourceId:    resourceId,
						TableName:     tableName,
						Pk:            pk,
					}
					locks = append(locks, rowLock)
				}
			}
		}
	}
	return locks
}
