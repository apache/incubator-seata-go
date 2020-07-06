package holder

import (
	"github.com/pkg/errors"
)

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/model"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type DataBaseSessionManager struct {
	TaskName                string
	conf                    config.DBStoreConfig
	TransactionStoreManager TransactionStoreManager
}

func NewDataBaseSessionManager(taskName string, conf config.DBStoreConfig) SessionManager {
	logStore := &LogStoreDataBaseDAO{engine: conf.Engine}
	transactionStoreManager := &DBTransactionStoreManager{
		logQueryLimit: conf.LogQueryLimit,
		LogStore:      logStore,
	}
	sessionManager := &DataBaseSessionManager{
		TaskName:                taskName,
		conf:                    conf,
		TransactionStoreManager: transactionStoreManager,
	}
	return sessionManager
}

func (sessionManager *DataBaseSessionManager) AddGlobalSession(session *session.GlobalSession) error {
	if sessionManager.TaskName == "" {
		ret := sessionManager.TransactionStoreManager.WriteSession(LogOperationGlobalAdd, session)
		if !ret {
			return errors.New("addGlobalSession failed.")
		}
	} else {
		ret := sessionManager.TransactionStoreManager.WriteSession(LogOperationGlobalUpdate, session)
		if !ret {
			return errors.New("addGlobalSession failed.")
		}
	}
	return nil
}

func (sessionManager *DataBaseSessionManager) FindGlobalSession(xid string) *session.GlobalSession {
	return sessionManager.FindGlobalSessionWithBranchSessions(xid, true)
}

func (sessionManager *DataBaseSessionManager) FindGlobalSessionWithBranchSessions(xid string, withBranchSessions bool) *session.GlobalSession {
	return sessionManager.TransactionStoreManager.ReadSessionWithBranchSessions(xid, withBranchSessions)
}

func (sessionManager *DataBaseSessionManager) UpdateGlobalSessionStatus(session *session.GlobalSession, status meta.GlobalStatus) error {
	if sessionManager.TaskName != "" {
		return nil
	}
	session.Status = status
	ret := sessionManager.TransactionStoreManager.WriteSession(LogOperationGlobalUpdate, session)
	if !ret {
		return errors.New("updateGlobalSessionStatus failed.")
	}
	return nil
}

func (sessionManager *DataBaseSessionManager) RemoveGlobalSession(session *session.GlobalSession) error {
	ret := sessionManager.TransactionStoreManager.WriteSession(LogOperationGlobalRemove, session)
	if !ret {
		return errors.New("removeGlobalSession failed.")
	}
	return nil
}

func (sessionManager *DataBaseSessionManager) AddBranchSession(globalSession *session.GlobalSession, session *session.BranchSession) error {
	if sessionManager.TaskName != "" {
		return nil
	}
	ret := sessionManager.TransactionStoreManager.WriteSession(LogOperationBranchAdd, session)
	if !ret {
		return errors.New("addBranchSession failed.")
	}
	return nil
}

func (sessionManager *DataBaseSessionManager) UpdateBranchSessionStatus(session *session.BranchSession, status meta.BranchStatus) error {
	if sessionManager.TaskName != "" {
		return nil
	}
	ret := sessionManager.TransactionStoreManager.WriteSession(LogOperationBranchUpdate, session)
	if !ret {
		return errors.New("addBranchSession failed.")
	}
	return nil
}

func (sessionManager *DataBaseSessionManager) RemoveBranchSession(globalSession *session.GlobalSession, session *session.BranchSession) error {
	if sessionManager.TaskName != "" {
		return nil
	}
	ret := sessionManager.TransactionStoreManager.WriteSession(LogOperationBranchRemove, session)
	if !ret {
		return errors.New("addBranchSession failed.")
	}
	return nil
}

func (sessionManager *DataBaseSessionManager) AllSessions() []*session.GlobalSession {
	if sessionManager.TaskName == ASYNC_COMMITTING_SESSION_MANAGER_NAME {
		return sessionManager.FindGlobalSessions(model.SessionCondition{
			Statuses: []meta.GlobalStatus{meta.GlobalStatusAsyncCommitting},
		})
	} else if sessionManager.TaskName == RETRY_COMMITTING_SESSION_MANAGER_NAME {
		return sessionManager.FindGlobalSessions(model.SessionCondition{
			Statuses: []meta.GlobalStatus{meta.GlobalStatusCommitRetrying},
		})
	} else if sessionManager.TaskName == RETRY_ROLLBACKING_SESSION_MANAGER_NAME {
		ss := sessionManager.FindGlobalSessions(model.SessionCondition{
			Statuses: []meta.GlobalStatus{meta.GlobalStatusRollbackRetrying,
				meta.GlobalStatusRollbacking,
				meta.GlobalStatusTimeoutRollbacking,
				meta.GlobalStatusTimeoutRollbackRetrying,
			},
		})
		return ss
	} else {
		return sessionManager.FindGlobalSessions(model.SessionCondition{
			Statuses: []meta.GlobalStatus{meta.GlobalStatusUnknown, meta.GlobalStatusBegin,
				meta.GlobalStatusCommitting, meta.GlobalStatusCommitRetrying, meta.GlobalStatusRollbacking,
				meta.GlobalStatusRollbackRetrying, meta.GlobalStatusTimeoutRollbacking, meta.GlobalStatusTimeoutRollbackRetrying,
				meta.GlobalStatusAsyncCommitting,
			},
		})
	}
}

func (sessionManager *DataBaseSessionManager) FindGlobalSessions(condition model.SessionCondition) []*session.GlobalSession {
	return sessionManager.TransactionStoreManager.ReadSessionWithSessionCondition(condition)
}

func (sessionManager *DataBaseSessionManager) Reload() {
	maxSessionId := sessionManager.TransactionStoreManager.GetCurrentMaxSessionId()
	if maxSessionId > uuid.UUID {
		uuid.SetUUID(uuid.UUID, maxSessionId)
	}
}
