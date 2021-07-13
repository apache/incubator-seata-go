package lock

import (
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/uuid"
)

func TestLockManager_AcquireLock(t *testing.T) {
	bs := branchSessionProvider()
	ok := GetLockManager().AcquireLock(bs)
	assert.Equal(t, ok, true)
}

func TestLockManager_IsLockable(t *testing.T) {
	transID := uuid.NextID()
	common.Init("127.0.0.1", 9876)
	ok := GetLockManager().IsLockable(common.GenerateXID(transID), "tb_1", "tb_1:13")
	assert.Equal(t, ok, true)
}

func TestLockManager_AcquireLock_Fail(t *testing.T) {
	sessions := branchSessionsProvider()
	result1 := GetLockManager().AcquireLock(sessions[0])
	result2 := GetLockManager().AcquireLock(sessions[1])
	assert.True(t, result1)
	assert.False(t, result2)
}

func TestLockManager_AcquireLock_DeadLock(t *testing.T) {
	sessions := deadlockBranchSessionsProvider()
	defer func() {
		GetLockManager().ReleaseLock(sessions[0])
		GetLockManager().ReleaseLock(sessions[1])
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func(session *session.BranchSession) {
		defer wg.Done()
		result := GetLockManager().AcquireLock(session)
		log.Infof("1: %v", result)
	}(sessions[0])

	go func(session *session.BranchSession) {
		defer wg.Done()
		result := GetLockManager().AcquireLock(session)
		log.Infof("2: %v", result)
	}(sessions[1])
	wg.Wait()
	assert.True(t, true)
}

func TestLockManager_IsLockable2(t *testing.T) {
	bs := branchSessionProvider()
	bs.LockKey = "t:4"
	result1 := GetLockManager().IsLockable(bs.XID, bs.ResourceID, bs.LockKey)
	assert.True(t, result1)
	GetLockManager().AcquireLock(bs)
	bs.TransactionID = uuid.NextID()
	result2 := GetLockManager().IsLockable(bs.XID, bs.ResourceID, bs.LockKey)
	assert.False(t, result2)
}

func TestLockManager_AcquireLock_SessionHolder(t *testing.T) {
	sessions := duplicatePkBranchSessionsProvider()
	result1 := GetLockManager().AcquireLock(sessions[0])
	assert.True(t, result1)
	assert.Equal(t, int64(4), GetLockManager().GetLockKeyCount())
	result2 := GetLockManager().ReleaseLock(sessions[0])
	assert.True(t, result2)
	assert.Equal(t, int64(0), GetLockManager().GetLockKeyCount())

	result3 := GetLockManager().AcquireLock(sessions[1])
	assert.True(t, result3)
	assert.Equal(t, int64(4), GetLockManager().GetLockKeyCount())
	result4 := GetLockManager().ReleaseLock(sessions[1])
	assert.True(t, result4)
	assert.Equal(t, int64(0), GetLockManager().GetLockKeyCount())
}

func deadlockBranchSessionsProvider() []*session.BranchSession {
	return baseBranchSessionsProvider("tb_2", "t:1,2,3,4,5", "t:5,4,3,2,1")
}

func duplicatePkBranchSessionsProvider() []*session.BranchSession {
	return baseBranchSessionsProvider("tb_2", "t:1,2;t1:1;t2:2", "t:1,2;t1:1;t2:2")
}

func branchSessionsProvider() []*session.BranchSession {
	return baseBranchSessionsProvider("tb_1", "t:1,2", "t:1,2")
}

func baseBranchSessionsProvider(resourceID string, lockKey1 string, lockKey2 string) []*session.BranchSession {
	var branchSessions = make([]*session.BranchSession, 0)
	transID := uuid.NextID()
	transID2 := uuid.NextID()
	common.Init("127.0.0.1", 9876)
	bs := session.NewBranchSession(
		session.WithBsXid(common.GenerateXID(transID)),
		session.WithBsTransactionID(transID),
		session.WithBsBranchID(1),
		session.WithBsResourceGroupID("my_test_tx_group"),
		session.WithBsResourceID(resourceID),
		session.WithBsLockKey(lockKey1),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsStatus(meta.BranchStatusUnknown),
		session.WithBsClientID("c1"),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	bs1 := session.NewBranchSession(
		session.WithBsXid(common.GenerateXID(transID2)),
		session.WithBsTransactionID(transID2),
		session.WithBsBranchID(1),
		session.WithBsResourceGroupID("my_test_tx_group"),
		session.WithBsResourceID(resourceID),
		session.WithBsLockKey(lockKey2),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsStatus(meta.BranchStatusUnknown),
		session.WithBsClientID("c1"),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	branchSessions = append(branchSessions, bs)
	branchSessions = append(branchSessions, bs1)
	return branchSessions
}

func branchSessionProvider() *session.BranchSession {
	common.Init("127.0.0.1", 9876)

	transID := uuid.NextID()
	bs := session.NewBranchSession(
		session.WithBsXid(common.GenerateXID(transID)),
		session.WithBsTransactionID(transID),
		session.WithBsBranchID(1),
		session.WithBsResourceGroupID("my_test_tx_group"),
		session.WithBsResourceID("tb_1"),
		session.WithBsLockKey("tb_1:13"),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsStatus(meta.BranchStatusUnknown),
		session.WithBsClientID("c1"),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	return bs
}
