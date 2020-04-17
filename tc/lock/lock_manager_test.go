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
	ok,err := GetLockManager().AcquireLock(bs)
	assert.Equal(t,ok,true)
	assert.Equal(t,err,nil)
}

func TestLockManager_IsLockable(t *testing.T) {
	transId := uuid.GeneratorUUID()
	ok := GetLockManager().IsLockable(common.XID.GenerateXID(transId),"tb_1","tb_1:13")
	assert.Equal(t,ok,true)
}

func TestLockManager_AcquireLock_Fail(t *testing.T) {
	sessions := branchSessionsProvider()
	result1,err1 := GetLockManager().AcquireLock(sessions[0])
	result2,err2 := GetLockManager().AcquireLock(sessions[1])
	assert.True(t,result1)
	assert.Equal(t,err1,nil)
	assert.False(t,result2)
	assert.Equal(t,err2,nil)
}

func  TestLockManager_AcquireLock_DeadLock(t *testing.T) {
	sessions := deadlockBranchSessionsProvider()
	defer func() {
		GetLockManager().ReleaseLock(sessions[0])
		GetLockManager().ReleaseLock(sessions[1])
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func(session *session.BranchSession) {
		defer wg.Done()
		result, err := GetLockManager().AcquireLock(session)
		logging.Logger.Infof("1: %v %v",result,err)
	}(sessions[0])

	go func(session *session.BranchSession) {
		defer wg.Done()
		result, err := GetLockManager().AcquireLock(session)
		logging.Logger.Infof("2: %v %v",result,err)
	}(sessions[1])
	wg.Wait()
	assert.True(t,true)
}

func TestLockManager_IsLockable2(t *testing.T) {
	bs := branchSessionProvider()
	bs.SetLockKey("t:4")
	result1 := GetLockManager().IsLockable(bs.Xid,bs.ResourceId,bs.LockKey)
	assert.True(t,result1)
	GetLockManager().AcquireLock(bs)
	bs.SetTransactionId(uuid.GeneratorUUID())
	result2 := GetLockManager().IsLockable(bs.Xid,bs.ResourceId,bs.LockKey)
	assert.False(t,result2)
}

func  TestLockManager_AcquireLock_SessionHolder(t *testing.T) {
	sessions := duplicatePkBranchSessionsProvider()
	result1, _ := GetLockManager().AcquireLock(sessions[0])
	assert.True(t,result1)
	assert.Equal(t,int64(4),GetLockManager().GetLockKeyCount())
	result2, _ := GetLockManager().ReleaseLock(sessions[0])
	assert.True(t,result2)
	assert.Equal(t,int64(0),GetLockManager().GetLockKeyCount())

	result3, _ := GetLockManager().AcquireLock(sessions[1])
	assert.True(t,result3)
	assert.Equal(t,int64(4),GetLockManager().GetLockKeyCount())
	result4, _ := GetLockManager().ReleaseLock(sessions[1])
	assert.True(t,result4)
	assert.Equal(t,int64(0),GetLockManager().GetLockKeyCount())
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
	var branchSessions = make([]*session.BranchSession,0)
	transId := uuid.GeneratorUUID()
	transId2 := uuid.GeneratorUUID()
	bs := session.NewBranchSession().
		SetXid(common.XID.GenerateXID(transId)).
		SetTransactionId(transId).
		SetBranchId(1).
		SetResourceGroupId("my_test_tx_group").
		SetResourceId(resourceId).
		SetLockKey(lockKey1).
		SetBranchType(meta.BranchTypeAT).
		SetStatus(meta.BranchStatusUnknown).
		SetClientId("c1").
		SetApplicationData([]byte("{\"data\":\"test\"}"))

	bs1 := session.NewBranchSession().
		SetXid(common.XID.GenerateXID(transId2)).
		SetTransactionId(transId2).
		SetBranchId(1).
		SetResourceGroupId("my_test_tx_group").
		SetResourceId(resourceId).
		SetLockKey(lockKey2).
		SetBranchType(meta.BranchTypeAT).
		SetStatus(meta.BranchStatusUnknown).
		SetClientId("c1").
		SetApplicationData([]byte("{\"data\":\"test\"}"))

	branchSessions = append(branchSessions,bs)
	branchSessions = append(branchSessions,bs1)
	return branchSessions
}

func branchSessionProvider() *session.BranchSession {
	common.XID.IpAddress="127.0.0.1"
	common.XID.Port=9876

	transId := uuid.GeneratorUUID()
	bs := session.NewBranchSession().
		SetXid(common.XID.GenerateXID(transId)).
		SetTransactionId(transId).
		SetBranchId(1).
		SetResourceGroupId("my_test_tx_group").
		SetResourceId("tb_1").
		SetLockKey("tb_1:13").
		SetBranchType(meta.BranchTypeAT).
		SetStatus(meta.BranchStatusUnknown).
		SetClientId("c1").
		SetApplicationData([]byte("{\"data\":\"test\"}"))

	return bs
}