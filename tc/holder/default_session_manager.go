package holder

import (
	"github.com/dk-lockdown/seata-golang/pkg/time"
	"github.com/dk-lockdown/seata-golang/tc/model"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type DefaultSessionManager struct {
	AbstractSessionManager
	SessionMap map[string]*session.GlobalSession
}

func NewDefaultSessionManager(name string) SessionManager {
	return &DefaultSessionManager{
		AbstractSessionManager: AbstractSessionManager{
			TransactionStoreManager: &AbstractTransactionStoreManager{},
			Name:                    name,
		},
		SessionMap: make(map[string]*session.GlobalSession),
	}
}

func (sessionManager *DefaultSessionManager) AddGlobalSession(session *session.GlobalSession) error {
	sessionManager.AbstractSessionManager.AddGlobalSession(session)
	sessionManager.SessionMap[session.Xid] = session
	return nil
}

func (sessionManager *DefaultSessionManager) FindGlobalSession(xid string) *session.GlobalSession {
	return sessionManager.SessionMap[xid]
}

func (sessionManager *DefaultSessionManager) FindGlobalSessionWithBranchSessions(xid string, withBranchSessions bool) *session.GlobalSession {
	return sessionManager.SessionMap[xid]
}

func (sessionManager *DefaultSessionManager) RemoveGlobalSession(session *session.GlobalSession) error {
	sessionManager.AbstractSessionManager.RemoveGlobalSession(session)
	delete(sessionManager.SessionMap, session.Xid)
	return nil
}

func (sessionManager *DefaultSessionManager) AllSessions() []*session.GlobalSession {
	var sessions = make([]*session.GlobalSession, 0)
	for _, session := range sessionManager.SessionMap {
		sessions = append(sessions, session)
	}
	return sessions
}

func (sessionManager *DefaultSessionManager) FindGlobalSessions(condition model.SessionCondition) []*session.GlobalSession {
	var sessions = make([]*session.GlobalSession, 0)
	for _, session := range sessionManager.SessionMap {
		if int64(time.CurrentTimeMillis())-session.BeginTime > condition.OverTimeAliveMills {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func (sessionManager *DefaultSessionManager) SetTransactionStoreManager(transactionStoreManager TransactionStoreManager) {
	sessionManager.TransactionStoreManager = transactionStoreManager
}
