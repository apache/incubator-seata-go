package storage

import (
	"fmt"
	"strings"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/util/common"
)

const LockSplit = "^^^"

func CollectBranchSessionRowLocks(branchSession *apis.BranchSession) []*apis.RowLock {
	if branchSession == nil || branchSession.LockKey == "" {
		return nil
	}
	return collectRowLocks(branchSession.LockKey, branchSession.ResourceID, branchSession.XID, branchSession.TransactionID, branchSession.BranchID)
}

func CollectRowLocks(lockKey string, resourceID string, xid string) []*apis.RowLock {
	return collectRowLocks(lockKey, resourceID, xid, common.GetTransactionID(xid), 0)
}

func collectRowLocks(lockKey string,
	resourceID string,
	xid string,
	transactionID int64,
	branchID int64) []*apis.RowLock {
	var locks = make([]*apis.RowLock, 0)
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
					rowLock := &apis.RowLock{
						XID:           xid,
						TransactionID: transactionID,
						BranchID:      branchID,
						ResourceID:    resourceID,
						TableName:     tableName,
						PK:            pk,
						RowKey:        getRowKey(resourceID, tableName, pk),
					}
					locks = append(locks, rowLock)
				}
			}
		}
	}
	return locks
}

func getRowKey(resourceID string, tableName string, pk string) string {
	return fmt.Sprintf("%s^^^%s^^^%s", resourceID, tableName, pk)
}
