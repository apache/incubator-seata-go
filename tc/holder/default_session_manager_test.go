package holder

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dk-lockdown/seata-golang/base/common"
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

func TestDefaultSessionManager_AddGlobalSession_RemoveGlobalSession(t *testing.T) {
	gs := globalSessionProvider()

	sessionManager := NewDefaultSessionManager("default")
	sessionManager.AddGlobalSession(gs)
	sessionManager.RemoveGlobalSession(gs)
}

func TestDefaultSessionManager_FindGlobalSession(t *testing.T) {
	gs := globalSessionProvider()
	sessionManager := NewDefaultSessionManager("default")
	sessionManager.AddGlobalSession(gs)
	expected := sessionManager.FindGlobalSession(gs.Xid)

	assert.NotNil(t, expected)
	assert.Equal(t, gs.TransactionId, expected.TransactionId)
	assert.Equal(t, gs.ApplicationId, expected.ApplicationId)
	assert.Equal(t, gs.TransactionServiceGroup, expected.TransactionServiceGroup)
	assert.Equal(t, gs.TransactionName, expected.TransactionName)
	assert.Equal(t, gs.Status, expected.Status)

	sessionManager.RemoveGlobalSession(gs)
}

func globalSessionsProvider() []*session.GlobalSession {
	common.XID.IpAddress = "127.0.0.1"
	common.XID.Port = 9876

	result := make([]*session.GlobalSession, 0)
	gs1 := session.NewGlobalSession(
		session.WithGsApplicationId("demo-app"),
		session.WithGsTransactionServiceGroup("my_test_tx_group"),
		session.WithGsTransactionName("test"),
		session.WithGsTimeout(6000),
	)

	gs2 := session.NewGlobalSession(
		session.WithGsApplicationId("demo-app"),
		session.WithGsTransactionServiceGroup("my_test_tx_group"),
		session.WithGsTransactionName("test"),
		session.WithGsTimeout(6000),
	)

	result = append(result, gs1)
	result = append(result, gs2)
	return result
}

func globalSessionProvider() *session.GlobalSession {
	common.XID.IpAddress = "127.0.0.1"
	common.XID.Port = 9876

	gs := session.NewGlobalSession(
		session.WithGsApplicationId("demo-app"),
		session.WithGsTransactionServiceGroup("my_test_tx_group"),
		session.WithGsTransactionName("test"),
		session.WithGsTimeout(6000),
	)
	return gs
}

func branchSessionProvider(globalSession *session.GlobalSession) *session.BranchSession {
	bs := session.NewBranchSession(
		session.WithBsTransactionId(globalSession.TransactionId),
		session.WithBsBranchId(1),
		session.WithBsResourceGroupId("my_test_tx_group"),
		session.WithBsResourceId("tb_1"),
		session.WithBsLockKey("t_1"),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	return bs
}
