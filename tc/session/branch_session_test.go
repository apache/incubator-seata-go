package session

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBranchSession_Encode_Decode(t *testing.T) {
	bs := branchSessionProvider()
	result,_ := bs.Encode()
	newBs := &BranchSession{}
	newBs.Decode(result)

	assert.Equal(t,bs.TransactionId,newBs.TransactionId)
	assert.Equal(t,bs.BranchId,newBs.BranchId)
	assert.Equal(t,bs.ResourceId,newBs.ResourceId)
	assert.Equal(t,bs.LockKey,newBs.LockKey)
	assert.Equal(t,bs.ClientId,newBs.ClientId)
	assert.Equal(t,bs.ApplicationData,newBs.ApplicationData)
}

func branchSessionProvider() *BranchSession {
	bs := NewBranchSession().
		SetTransactionId(uuid.GeneratorUUID()).
		SetBranchId(1).
		SetResourceGroupId("my_test_tx_group").
		SetResourceId("tb_1").
		SetLockKey("t_1").
		SetBranchType(meta.BranchTypeAT).
		SetStatus(meta.BranchStatusUnknown).
		SetClientId("c1").
		SetApplicationData([]byte("{\"data\":\"test\"}"))

	return bs
}