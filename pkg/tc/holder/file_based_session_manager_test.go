package holder

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/tc/model"
)

func TestFileBasedSessionManager_AddGlobalSession(t *testing.T) {
	gs := globalSessionProvider(t)

	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())
	sessionManager.AddGlobalSession(gs)
	sessionManager.RemoveGlobalSession(gs)
}

func TestFileBasedSessionManager_FindGlobalSession(t *testing.T) {
	gs := globalSessionProvider(t)
	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())
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

func TestFileBasedSessionManager_UpdateGlobalSessionStatus(t *testing.T) {
	gs := globalSessionProvider(t)
	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())
	sessionManager.AddGlobalSession(gs)
	gs.Status = meta.GlobalStatusFinished
	sessionManager.UpdateGlobalSessionStatus(gs, meta.GlobalStatusFinished)

	expected := sessionManager.FindGlobalSession(gs.XID)
	assert.NotNil(t, gs)
	assert.Equal(t, meta.GlobalStatusFinished, expected.Status)

	sessionManager.RemoveGlobalSession(gs)
}

func TestFileBasedSessionManager_RemoveGlobalSession(t *testing.T) {
	gs := globalSessionProvider(t)

	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())
	sessionManager.AddGlobalSession(gs)
	sessionManager.RemoveGlobalSession(gs)

	expected := sessionManager.FindGlobalSession(gs.XID)
	assert.Nil(t, expected)
}

func TestFileBasedSessionManager_AddBranchSession(t *testing.T) {
	gs := globalSessionProvider(t)
	bs := branchSessionProvider(gs)

	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())
	sessionManager.AddGlobalSession(gs)
	sessionManager.AddBranchSession(gs, bs)
	sessionManager.RemoveBranchSession(gs, bs)
	sessionManager.RemoveGlobalSession(gs)
}

func TestFileBasedSessionManager_UpdateBranchSessionStatus(t *testing.T) {
	gs := globalSessionProvider(t)
	bs := branchSessionProvider(gs)

	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())
	sessionManager.AddGlobalSession(gs)
	sessionManager.AddBranchSession(gs, bs)
	sessionManager.UpdateBranchSessionStatus(bs, meta.BranchStatusPhasetwoCommitted)
	sessionManager.RemoveBranchSession(gs, bs)
	sessionManager.RemoveGlobalSession(gs)
}

func TestFileBasedSessionManager_RemoveBranchSession(t *testing.T) {
	gs := globalSessionProvider(t)
	bs := branchSessionProvider(gs)

	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())
	sessionManager.AddGlobalSession(gs)
	sessionManager.AddBranchSession(gs, bs)
	sessionManager.RemoveBranchSession(gs, bs)
	sessionManager.RemoveGlobalSession(gs)
}

func TestFileBasedSessionManager_AllSessions(t *testing.T) {
	gss := globalSessionsProvider()
	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())

	for _, gs := range gss {
		sessionManager.AddGlobalSession(gs)
	}
	allGs := sessionManager.AllSessions()
	assert.NotNil(t, allGs)
	assert.Equal(t, 2, len(allGs))

	for _, gs := range gss {
		sessionManager.RemoveGlobalSession(gs)
	}

	allGs2 := sessionManager.AllSessions()
	assert.Equal(t, 0, len(allGs2))
}

func TestFileBasedSessionManager_FindGlobalSessionTest(t *testing.T) {
	gss := globalSessionsProvider()
	sessionManager := NewFileBasedSessionManager(config.GetDefaultFileStoreConfig())

	for _, gs := range gss {
		sessionManager.AddGlobalSession(gs)
	}
	sessionCondition := model.SessionCondition{
		OverTimeAliveMills: 30 * 24 * 3600,
	}

	expectedGlobalSessions := sessionManager.FindGlobalSessions(sessionCondition)

	assert.NotNil(t, expectedGlobalSessions)
	assert.Equal(t, 2, len(expectedGlobalSessions))

	for _, gs := range gss {
		sessionManager.RemoveGlobalSession(gs)
	}
}
