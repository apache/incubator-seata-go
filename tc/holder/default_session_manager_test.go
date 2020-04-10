package holder

import (
	"github.com/stretchr/testify/assert"
	"github.com/dk-lockdown/seata-golang/common"
	"github.com/dk-lockdown/seata-golang/meta"
	"github.com/dk-lockdown/seata-golang/tc/session"
	"github.com/dk-lockdown/seata-golang/util"
	"testing"
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

	assert.NotNil(t,expected)
	assert.Equal(t,gs.TransactionId,expected.TransactionId)
	assert.Equal(t,gs.ApplicationId,expected.ApplicationId)
	assert.Equal(t,gs.TransactionServiceGroup,expected.TransactionServiceGroup)
	assert.Equal(t,gs.TransactionName,expected.TransactionName)
	assert.Equal(t,gs.Status,expected.Status)

	sessionManager.RemoveGlobalSession(gs)
}

func globalSessionsProvider() []*session.GlobalSession {
	common.XID.IpAddress="127.0.0.1"
	common.XID.Port=9876

	result := make([]*session.GlobalSession,0)
	gs1 := session.NewGlobalSession().
		SetApplicationId("demo-app").
		SetTransactionId(util.GeneratorUUID()).
		SetTransactionServiceGroup("my_test_tx_group").
		SetTransactionName("test").
		SetTimeout(6000)
	gs1.SetXid(common.XID.GenerateXID(gs1.TransactionId))

	gs2 := session.NewGlobalSession().
		SetApplicationId("demo-app").
		SetTransactionId(util.GeneratorUUID()).
		SetTransactionServiceGroup("my_test_tx_group").
		SetTransactionName("test").
		SetTimeout(6000)
	gs2.SetXid(common.XID.GenerateXID(gs2.TransactionId))

	result = append(result,gs1)
	result = append(result,gs2)
	return result
}

func globalSessionProvider() *session.GlobalSession {
	common.XID.IpAddress="127.0.0.1"
	common.XID.Port=9876

	gs := session.NewGlobalSession().
		SetApplicationId("demo-app").
		SetTransactionId(util.GeneratorUUID()).
		SetTransactionServiceGroup("my_test_tx_group").
		SetTransactionName("test").
		SetTimeout(6000)
	gs.SetXid(common.XID.GenerateXID(gs.TransactionId))
	return gs
}

func branchSessionProvider(globalSession *session.GlobalSession) *session.BranchSession {
	bs := session.NewBranchSession().
		SetTransactionId(globalSession.TransactionId).
		SetBranchId(1).
		SetResourceGroupId("my_test_tx_group").
		SetResourceId("tb_1").
		SetLockKey("t_1").
		SetBranchType(meta.BranchTypeAT).
		SetApplicationData([]byte("{\"data\":\"test\"}"))

	return bs
}