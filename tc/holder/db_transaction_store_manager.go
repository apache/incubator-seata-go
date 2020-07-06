package holder

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/dk-lockdown/seata-golang/tc/model"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type DBTransactionStoreManager struct {
	logQueryLimit int
	LogStore      ILogStore
}

func (storeManager *DBTransactionStoreManager) WriteSession(logOperation LogOperation, session session.SessionStorable) bool {
	if LogOperationGlobalAdd == logOperation {
		globalSession := convertGlobalTransactionDO(session)
		if globalSession != nil {
			storeManager.LogStore.InsertGlobalTransactionDO(*globalSession)
		}
		return true
	} else if LogOperationGlobalUpdate == logOperation {
		globalSession := convertGlobalTransactionDO(session)
		if globalSession != nil {
			storeManager.LogStore.UpdateGlobalTransactionDO(*globalSession)
		}
		return true
	} else if LogOperationGlobalRemove == logOperation {
		globalSession := convertGlobalTransactionDO(session)
		if globalSession != nil {
			storeManager.LogStore.DeleteGlobalTransactionDO(*globalSession)
		}
		return true
	} else if LogOperationBranchAdd == logOperation {
		branchSession := convertBranchTransactionDO(session)
		if branchSession != nil {
			storeManager.LogStore.InsertBranchTransactionDO(*branchSession)
		}
		return true
	} else if LogOperationBranchUpdate == logOperation {
		branchSession := convertBranchTransactionDO(session)
		if branchSession != nil {
			storeManager.LogStore.UpdateBranchTransactionDO(*branchSession)
		}
		return true
	} else if LogOperationBranchRemove == logOperation {
		branchSession := convertBranchTransactionDO(session)
		if branchSession != nil {
			storeManager.LogStore.DeleteBranchTransactionDO(*branchSession)
		}
		return true
	}
	logging.Logger.Errorf("Unknown LogOperation:%v", logOperation)
	return false
}

func (storeManager *DBTransactionStoreManager) ReadSession(xid string) *session.GlobalSession {
	return storeManager.ReadSessionWithBranchSessions(xid, true)
}

func (storeManager *DBTransactionStoreManager) ReadSessionWithBranchSessions(xid string, withBranchSessions bool) *session.GlobalSession {
	globalTransactionDO := storeManager.LogStore.QueryGlobalTransactionDOByXid(xid)
	if globalTransactionDO == nil {
		return nil
	}

	var branchTransactionDOs []*model.BranchTransactionDO
	if withBranchSessions {
		branchTransactionDOs = storeManager.LogStore.QueryBranchTransactionDOByXid(globalTransactionDO.Xid)
	}

	return getGlobalSession(globalTransactionDO, branchTransactionDOs)
}

func (storeManager *DBTransactionStoreManager) ReadSessionByTransactionId(transactionId int64) *session.GlobalSession {
	globalTransactionDO := storeManager.LogStore.QueryGlobalTransactionDOByTransactionId(transactionId)
	if globalTransactionDO == nil {
		return nil
	}

	branchTransactionDOs := storeManager.LogStore.QueryBranchTransactionDOByXid(globalTransactionDO.Xid)

	return getGlobalSession(globalTransactionDO, branchTransactionDOs)
}

func (storeManager *DBTransactionStoreManager) readSessionByStatuses(statuses []meta.GlobalStatus) []*session.GlobalSession {
	states := make([]int, 0)
	for _, status := range statuses {
		states = append(states, int(status))
	}
	globalTransactionDOs := storeManager.LogStore.QueryGlobalTransactionDOByStatuses(states, storeManager.logQueryLimit)
	if globalTransactionDOs == nil || len(globalTransactionDOs) == 0 {
		return nil
	}
	xids := make([]string, 0)
	for _, globalTransactionDO := range globalTransactionDOs {
		xids = append(xids, globalTransactionDO.Xid)
	}
	branchTransactionDOs := storeManager.LogStore.QueryBranchTransactionDOByXids(xids)
	branchTransactionMap := make(map[string][]*model.BranchTransactionDO)
	for _, branchTransactionDO := range branchTransactionDOs {
		branchTransactions, ok := branchTransactionMap[branchTransactionDO.Xid]
		if ok {
			branchTransactions = append(branchTransactions, branchTransactionDO)
			branchTransactionMap[branchTransactionDO.Xid] = branchTransactions
		} else {
			branchTransactions = make([]*model.BranchTransactionDO, 0)
			branchTransactions = append(branchTransactions, branchTransactionDO)
			branchTransactionMap[branchTransactionDO.Xid] = branchTransactions
		}
	}
	globalSessions := make([]*session.GlobalSession, 0)
	for _, globalTransaction := range globalTransactionDOs {
		globalSession := getGlobalSession(globalTransaction, branchTransactionMap[globalTransaction.Xid])
		globalSessions = append(globalSessions, globalSession)
	}
	return globalSessions
}

func (storeManager *DBTransactionStoreManager) ReadSessionWithSessionCondition(sessionCondition model.SessionCondition) []*session.GlobalSession {
	if sessionCondition.Xid != "" {
		globalSession := storeManager.ReadSession(sessionCondition.Xid)
		if globalSession != nil {
			globalSessions := make([]*session.GlobalSession, 0)
			globalSessions = append(globalSessions, globalSession)
			return globalSessions
		}
	} else if sessionCondition.TransactionId != 0 {
		globalSession := storeManager.ReadSessionByTransactionId(sessionCondition.TransactionId)
		if globalSession != nil {
			globalSessions := make([]*session.GlobalSession, 0)
			globalSessions = append(globalSessions, globalSession)
			return globalSessions
		}
	} else if sessionCondition.Statuses != nil && len(sessionCondition.Statuses) > 0 {
		return storeManager.readSessionByStatuses(sessionCondition.Statuses)
	}

	return nil
}

func (storeManager *DBTransactionStoreManager) Shutdown() {

}

func (storeManager *DBTransactionStoreManager) GetCurrentMaxSessionId() int64 {
	return storeManager.LogStore.GetCurrentMaxSessionId(uuid.GetMaxUUID(), uuid.GetInitUUID())
}

func getGlobalSession(globalTransactionDO *model.GlobalTransactionDO, branchTransactionDOs []*model.BranchTransactionDO) *session.GlobalSession {
	globalSession := convertGlobalTransaction(globalTransactionDO)
	if branchTransactionDOs != nil && len(branchTransactionDOs) > 0 {
		for _, branchTransactionDO := range branchTransactionDOs {
			globalSession.Add(convertBranchSession(branchTransactionDO))
		}
	}
	return globalSession
}

func convertGlobalTransaction(globalTransactionDO *model.GlobalTransactionDO) *session.GlobalSession {
	globalSession := session.NewGlobalSession(
		session.WithGsXid(globalTransactionDO.Xid),
		session.WithGsApplicationId(globalTransactionDO.ApplicationId),
		session.WithGsTransactionId(globalTransactionDO.TransactionId),
		session.WithGsTransactionName(globalTransactionDO.TransactionName),
		session.WithGsTransactionServiceGroup(globalTransactionDO.TransactionServiceGroup),
		session.WithGsStatus(meta.GlobalStatus(globalTransactionDO.Status)),
		session.WithGsTimeout(globalTransactionDO.Timeout),
		session.WithGsBeginTime(globalTransactionDO.BeginTime),
		session.WithGsApplicationData(globalTransactionDO.ApplicationData),
	)
	return globalSession
}

func convertBranchSession(branchTransactionDO *model.BranchTransactionDO) *session.BranchSession {
	branchSession := session.NewBranchSession(
		session.WithBsXid(branchTransactionDO.Xid),
		session.WithBsTransactionId(branchTransactionDO.TransactionId),
		session.WithBsApplicationData(branchTransactionDO.ApplicationData),
		session.WithBsBranchId(branchTransactionDO.BranchId),
		session.WithBsBranchType(meta.ValueOfBranchType(branchTransactionDO.BranchType)),
		session.WithBsResourceId(branchTransactionDO.ResourceId),
		session.WithBsClientId(branchTransactionDO.ClientId),
		session.WithBsResourceGroupId(branchTransactionDO.ResourceGroupId),
		session.WithBsStatus(meta.BranchStatus(branchTransactionDO.Status)),
	)
	return branchSession
}

func convertGlobalTransactionDO(sessionStorable session.SessionStorable) *model.GlobalTransactionDO {
	globalSession, ok := sessionStorable.(*session.GlobalSession)
	if sessionStorable == nil || !ok {
		logging.Logger.Errorf("the parameter of SessionStorable is not available, SessionStorable:%v", sessionStorable)
		return nil
	}
	globalTransactionDO := &model.GlobalTransactionDO{
		Xid:                     globalSession.Xid,
		TransactionId:           globalSession.TransactionId,
		Status:                  int32(globalSession.Status),
		ApplicationId:           globalSession.ApplicationId,
		TransactionServiceGroup: globalSession.TransactionServiceGroup,
		TransactionName:         globalSession.TransactionName,
		Timeout:                 globalSession.Timeout,
		BeginTime:               globalSession.BeginTime,
		ApplicationData:         globalSession.ApplicationData,
	}
	return globalTransactionDO
}

func convertBranchTransactionDO(sessionStorable session.SessionStorable) *model.BranchTransactionDO {
	branchSession, ok := sessionStorable.(*session.BranchSession)
	if sessionStorable == nil || !ok {
		logging.Logger.Errorf("the parameter of SessionStorable is not available, SessionStorable:%v", sessionStorable)
		return nil
	}

	branchSessionDO := &model.BranchTransactionDO{
		Xid:             branchSession.Xid,
		TransactionId:   branchSession.TransactionId,
		BranchId:        branchSession.BranchId,
		ResourceGroupId: branchSession.ResourceGroupId,
		ResourceId:      branchSession.ResourceId,
		BranchType:      branchSession.BranchType.String(),
		Status:          int32(branchSession.Status),
		ClientId:        branchSession.ClientId,
		ApplicationData: branchSession.ApplicationData,
	}
	return branchSessionDO
}
