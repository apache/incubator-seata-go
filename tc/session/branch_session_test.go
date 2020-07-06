package session

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
)

func TestBranchSession_Encode_Decode(t *testing.T) {
	bs := branchSessionProvider()
	result, _ := bs.Encode()
	newBs := &BranchSession{}
	newBs.Decode(result)

	assert.Equal(t, bs.TransactionId, newBs.TransactionId)
	assert.Equal(t, bs.BranchId, newBs.BranchId)
	assert.Equal(t, bs.ResourceId, newBs.ResourceId)
	assert.Equal(t, bs.LockKey, newBs.LockKey)
	assert.Equal(t, bs.ClientId, newBs.ClientId)
	assert.Equal(t, bs.ApplicationData, newBs.ApplicationData)
}

func branchSessionProvider() *BranchSession {
	bs := NewBranchSession(
		WithBsTransactionId(uuid.GeneratorUUID()),
		WithBsBranchId(1),
		WithBsResourceGroupId("my_test_tx_group"),
		WithBsResourceId("tb_1"),
		WithBsLockKey("t_1"),
		WithBsBranchType(meta.BranchTypeAT),
		WithBsStatus(meta.BranchStatusUnknown),
		WithBsClientId("c1"),
		WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	return bs
}
