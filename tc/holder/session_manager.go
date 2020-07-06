package holder

import (
	"errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/model"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type SessionManager interface {
	// Add global session.
	AddGlobalSession(session *session.GlobalSession) error

	// Find global session global session.
	FindGlobalSession(xid string) *session.GlobalSession

	// Find global session global session.
	FindGlobalSessionWithBranchSessions(xid string, withBranchSessions bool) *session.GlobalSession

	// Update global session status.
	UpdateGlobalSessionStatus(session *session.GlobalSession, status meta.GlobalStatus) error

	// Remove global session.
	RemoveGlobalSession(session *session.GlobalSession) error

	// Add branch session.
	AddBranchSession(globalSession *session.GlobalSession, session *session.BranchSession) error

	// Update branch session status.
	UpdateBranchSessionStatus(session *session.BranchSession, status meta.BranchStatus) error

	// Remove branch session.
	RemoveBranchSession(globalSession *session.GlobalSession, session *session.BranchSession) error

	// All sessions collection.
	AllSessions() []*session.GlobalSession

	// Find global sessions list.
	FindGlobalSessions(condition model.SessionCondition) []*session.GlobalSession
}

type AbstractSessionManager struct {
	TransactionStoreManager TransactionStoreManager
	Name                    string
}

func (sessionManager *AbstractSessionManager) AddGlobalSession(session *session.GlobalSession) error {
	logging.Logger.Debugf("MANAGER[%s] SESSION[%v] %s", sessionManager.Name, session, LogOperationGlobalAdd.String())
	sessionManager.writeSession(LogOperationGlobalAdd, session)
	return nil
}

func (sessionManager *AbstractSessionManager) UpdateGlobalSessionStatus(session *session.GlobalSession, status meta.GlobalStatus) error {
	logging.Logger.Debugf("MANAGER[%s] SESSION[%v] %s", sessionManager.Name, session, LogOperationGlobalUpdate.String())
	sessionManager.writeSession(LogOperationGlobalUpdate, session)
	return nil
}

func (sessionManager *AbstractSessionManager) RemoveGlobalSession(session *session.GlobalSession) error {
	logging.Logger.Debugf("MANAGER[%s] SESSION[%v] %s", sessionManager.Name, session, LogOperationGlobalRemove.String())
	sessionManager.writeSession(LogOperationGlobalRemove, session)
	return nil
}

func (sessionManager *AbstractSessionManager) AddBranchSession(globalSession *session.GlobalSession, session *session.BranchSession) error {
	logging.Logger.Debugf("MANAGER[%s] SESSION[%v] %s", sessionManager.Name, session, LogOperationBranchAdd.String())
	sessionManager.writeSession(LogOperationBranchAdd, session)
	return nil
}

func (sessionManager *AbstractSessionManager) UpdateBranchSessionStatus(session *session.BranchSession, status meta.BranchStatus) error {
	logging.Logger.Debugf("MANAGER[%s] SESSION[%v] %s", sessionManager.Name, session, LogOperationBranchUpdate.String())
	sessionManager.writeSession(LogOperationBranchUpdate, session)
	return nil
}

func (sessionManager *AbstractSessionManager) RemoveBranchSession(globalSession *session.GlobalSession, session *session.BranchSession) error {
	logging.Logger.Debugf("MANAGER[%s] SESSION[%v] %s", sessionManager.Name, session, LogOperationBranchRemove.String())
	sessionManager.writeSession(LogOperationBranchRemove, session)
	return nil
}

func (sessionManager *AbstractSessionManager) writeSession(logOperation LogOperation, sessionStorable session.SessionStorable) error {
	result := sessionManager.TransactionStoreManager.WriteSession(logOperation, sessionStorable)
	if !result {
		if logOperation == LogOperationGlobalAdd {
			return &meta.TransactionException{
				Code:    meta.TransactionExceptionCodeFailedWriteSession,
				Message: "Fail to holder global session",
			}
		}
		if logOperation == LogOperationGlobalUpdate {
			return &meta.TransactionException{
				Code:    meta.TransactionExceptionCodeFailedWriteSession,
				Message: "Fail to update global session",
			}
		}
		if logOperation == LogOperationGlobalRemove {
			return &meta.TransactionException{
				Code:    meta.TransactionExceptionCodeFailedWriteSession,
				Message: "Fail to remove global session",
			}
		}
		if logOperation == LogOperationBranchAdd {
			return &meta.TransactionException{
				Code:    meta.TransactionExceptionCodeFailedWriteSession,
				Message: "Fail to holder branch session",
			}
		}
		if logOperation == LogOperationBranchUpdate {
			return &meta.TransactionException{
				Code:    meta.TransactionExceptionCodeFailedWriteSession,
				Message: "Fail to update branch session",
			}
		}
		if logOperation == LogOperationBranchRemove {
			return &meta.TransactionException{
				Code:    meta.TransactionExceptionCodeFailedWriteSession,
				Message: "Fail to remove branch session",
			}
		}
		return errors.New("Unknown LogOperation:" + logOperation.String())
	}
	return nil
}
