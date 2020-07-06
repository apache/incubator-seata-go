package lock

import (
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dk-lockdown/seata-golang/base/common"
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

func TestLockManager_AcquireLock(t *testing.T) {
	bs := branchSessionProvider()
	ok := GetLockManager().AcquireLock(bs)
	assert.Equal(t, ok, true)
}

func TestLockManager_IsLockable(t *testing.T) {
	transId := uuid.GeneratorUUID()
	ok := GetLockManager().IsLockable(common.XID.GenerateXID(transId), "tb_1", "tb_1:13")
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
		logging.Logger.Infof("1: %v", result)
	}(sessions[0])

	go func(session *session.BranchSession) {
		defer wg.Done()
		result := GetLockManager().AcquireLock(session)
		logging.Logger.Infof("2: %v", result)
	}(sessions[1])
	wg.Wait()
	assert.True(t, true)
}

func TestLockManager_IsLockable2(t *testing.T) {
	bs := branchSessionProvider()
	bs.LockKey = "t:4"
	result1 := GetLockManager().IsLockable(bs.Xid, bs.ResourceId, bs.LockKey)
	assert.True(t, result1)
	GetLockManager().AcquireLock(bs)
	bs.TransactionId = uuid.GeneratorUUID()
	result2 := GetLockManager().IsLockable(bs.Xid, bs.ResourceId, bs.LockKey)
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

func baseBranchSessionsProvider(resourceId string, lockKey1 string, lockKey2 string) []*session.BranchSession {
	var branchSessions = make([]*session.BranchSession, 0)
	transId := uuid.GeneratorUUID()
	transId2 := uuid.GeneratorUUID()
	bs := session.NewBranchSession(
		session.WithBsXid(common.XID.GenerateXID(transId)),
		session.WithBsTransactionId(transId),
		session.WithBsBranchId(1),
		session.WithBsResourceGroupId("my_test_tx_group"),
		session.WithBsResourceId(resourceId),
		session.WithBsLockKey(lockKey1),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsStatus(meta.BranchStatusUnknown),
		session.WithBsClientId("c1"),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	bs1 := session.NewBranchSession(
		session.WithBsXid(common.XID.GenerateXID(transId2)),
		session.WithBsTransactionId(transId2),
		session.WithBsBranchId(1),
		session.WithBsResourceGroupId("my_test_tx_group"),
		session.WithBsResourceId(resourceId),
		session.WithBsLockKey(lockKey2),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsStatus(meta.BranchStatusUnknown),
		session.WithBsClientId("c1"),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	branchSessions = append(branchSessions, bs)
	branchSessions = append(branchSessions, bs1)
	return branchSessions
}

func branchSessionProvider() *session.BranchSession {
	common.XID.IpAddress = "127.0.0.1"
	common.XID.Port = 9876

	transId := uuid.GeneratorUUID()
	bs := session.NewBranchSession(
		session.WithBsXid(common.XID.GenerateXID(transId)),
		session.WithBsTransactionId(transId),
		session.WithBsBranchId(1),
		session.WithBsResourceGroupId("my_test_tx_group"),
		session.WithBsResourceId("tb_1"),
		session.WithBsLockKey("tb_1:13"),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsStatus(meta.BranchStatusUnknown),
		session.WithBsClientId("c1"),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	return bs
}
