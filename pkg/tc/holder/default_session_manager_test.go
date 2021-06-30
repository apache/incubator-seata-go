package holder

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/common"
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
)

func TestDefaultSessionManager_AddGlobalSession_RemoveGlobalSession(t *testing.T) {
	gs := globalSessionProvider(t)

	sessionManager := NewDefaultSessionManager("default")
	sessionManager.AddGlobalSession(gs)
	sessionManager.RemoveGlobalSession(gs)
}

func TestDefaultSessionManager_FindGlobalSession(t *testing.T) {
	gs := globalSessionProvider(t)
	sessionManager := NewDefaultSessionManager("default")
	sessionManager.AddGlobalSession(gs)
	expected := sessionManager.FindGlobalSession(gs.XID)

	assert.NotNil(t, expected)
	assert.Equal(t, gs.TransactionID, expected.TransactionID)
	assert.Equal(t, gs.ApplicationID, expected.ApplicationID)
	assert.Equal(t, gs.TransactionServiceGroup, expected.TransactionServiceGroup)
	assert.Equal(t, gs.TransactionName, expected.TransactionName)
	assert.Equal(t, gs.Status, expected.Status)

	sessionManager.RemoveGlobalSession(gs)
}

func globalSessionsProvider() []*session.GlobalSession {
	common.GetXID().Init("127.0.0.1", 9876)

	result := make([]*session.GlobalSession, 0)
	gs1 := session.NewGlobalSession(
		session.WithGsApplicationID("demo-cmd"),
		session.WithGsTransactionServiceGroup("my_test_tx_group"),
		session.WithGsTransactionName("test"),
		session.WithGsTimeout(6000),
	)

	gs2 := session.NewGlobalSession(
		session.WithGsApplicationID("demo-cmd"),
		session.WithGsTransactionServiceGroup("my_test_tx_group"),
		session.WithGsTransactionName("test"),
		session.WithGsTimeout(6000),
	)

	result = append(result, gs1)
	result = append(result, gs2)
	return result
}

func globalSessionProvider(t *testing.T) *session.GlobalSession {
	testutil.InitServerConfig(t)

	common.GetXID().Init("127.0.0.1", 9876)

	gs := session.NewGlobalSession(
		session.WithGsApplicationID("demo-cmd"),
		session.WithGsTransactionServiceGroup("my_test_tx_group"),
		session.WithGsTransactionName("test"),
		session.WithGsTimeout(6000),
	)
	return gs
}

func branchSessionProvider(globalSession *session.GlobalSession) *session.BranchSession {
	bs := session.NewBranchSession(
		session.WithBsTransactionID(globalSession.TransactionID),
		session.WithBsBranchID(1),
		session.WithBsResourceGroupID("my_test_tx_group"),
		session.WithBsResourceID("tb_1"),
		session.WithBsLockKey("t_1"),
		session.WithBsBranchType(meta.BranchTypeAT),
		session.WithBsApplicationData([]byte("{\"data\":\"test\"}")),
	)

	return bs
}
