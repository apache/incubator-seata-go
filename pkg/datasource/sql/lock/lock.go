/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lock

import (
	"strings"
	"sync"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

var (
	lockManager     *LockManager
	onceLockManager = &sync.Once{}
)

type LockManager struct {
	mutex     sync.Mutex
	lockerMap map[types.LockType]Locker
}

func GetLockManager() *LockManager {
	if lockManager == nil {
		onceLockManager.Do(func() {
			lockManager = &LockManager{
				lockerMap: make(map[types.LockType]Locker, 0),
			}
		})
	}
	return lockManager
}

func (c *LockManager) RegisterLocker(lockType types.LockType, locker Locker) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.lockerMap[lockType] = locker
}

func (c *LockManager) GetLocker(lockType types.LockType) Locker {
	return c.lockerMap[lockType]
}

type RowLock struct {
	xid           string
	branchId      int64
	resourceId    string
	transactionId int64
	tableName     string
	pk            string
	rowKey        string
	feature       string
}

type Locker interface {
	AcquireLock(rowLock []*RowLock) (bool, error)
	ReleaseLock(rowLock []*RowLock) (bool, error)
	IsLockable(rowLock []*RowLock) (bool, error)
	CleanAllLocks()
	UpdateLockStatus(xid string, lockStatus types.LockStatus)
}

// Collect row locks list.
func CollectRowLocks(lockKey string, resourceId string, xid string, transactionId int64, branchId int64) []*RowLock {
	rowlocks := make([]*RowLock, 0)
	tableGroupedLockKeys := strings.Split(lockKey, ";")
	for k := range tableGroupedLockKeys {
		tableGroupedLockKey := tableGroupedLockKeys[k]
		idx := strings.Index(tableGroupedLockKey, ":")
		if idx < 0 {
			return rowlocks
		}

		tableName := tableGroupedLockKey[0:idx]
		mergedPKs := tableGroupedLockKey[idx+1:]
		if mergedPKs == "" {
			return rowlocks
		}
		pks := strings.Split(mergedPKs, ",")
		if pks == nil || len(pks) == 0 {
			return rowlocks
		}

		rowlocks = make([]*RowLock, len(pks))
		for i := range pks {
			pk := pks[i]
			rowlocks[i] = &RowLock{
				xid:           xid,
				transactionId: transactionId,
				branchId:      branchId,
				tableName:     tableName,
				pk:            pk,
				resourceId:    resourceId,
			}
		}
	}
	return rowlocks
}
