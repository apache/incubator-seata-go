package holder

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/tc/model"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/uuid"
)

type DBTransactionStoreManager struct {
	logQueryLimit int
	LogStore      LogStore
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
	log.Errorf("Unknown LogOperation:%v", logOperation)
	return false
}

func (storeManager *DBTransactionStoreManager) ReadSession(xid string) *session.GlobalSession {
	return storeManager.ReadSessionWithBranchSessions(xid, true)
}

func (storeManager *DBTransactionStoreManager) ReadSessionWithBranchSessions(xid string, withBranchSessions bool) *session.GlobalSession {
	globalTransactionDO := storeManager.LogStore.QueryGlobalTransactionDOByXID(xid)
	if globalTransactionDO == nil {
		return nil
	}

	var branchTransactionDOs []*model.BranchTransactionDO
	if withBranchSessions {
		branchTransactionDOs = storeManager.LogStore.QueryBranchTransactionDOByXID(globalTransactionDO.XID)
	}

	return getGlobalSession(globalTransactionDO, branchTransactionDOs)
}

func (storeManager *DBTransactionStoreManager) ReadSessionByTransactionID(transactionID int64) *session.GlobalSession {
	globalTransactionDO := storeManager.LogStore.QueryGlobalTransactionDOByTransactionID(transactionID)
	if globalTransactionDO == nil {
		return nil
	}

	branchTransactionDOs := storeManager.LogStore.QueryBranchTransactionDOByXID(globalTransactionDO.XID)

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
		xids = append(xids, globalTransactionDO.XID)
	}
	branchTransactionDOs := storeManager.LogStore.QueryBranchTransactionDOByXIDs(xids)
	branchTransactionMap := make(map[string][]*model.BranchTransactionDO)
	for _, branchTransactionDO := range branchTransactionDOs {
		branchTransactions, ok := branchTransactionMap[branchTransactionDO.XID]
		if ok {
			branchTransactions = append(branchTransactions, branchTransactionDO)
			branchTransactionMap[branchTransactionDO.XID] = branchTransactions
		} else {
			branchTransactions = make([]*model.BranchTransactionDO, 0)
			branchTransactions = append(branchTransactions, branchTransactionDO)
			branchTransactionMap[branchTransactionDO.XID] = branchTransactions
		}
	}
	globalSessions := make([]*session.GlobalSession, 0)
	for _, globalTransaction := range globalTransactionDOs {
		globalSession := getGlobalSession(globalTransaction, branchTransactionMap[globalTransaction.XID])
		globalSessions = append(globalSessions, globalSession)
	}
	return globalSessions
}

func (storeManager *DBTransactionStoreManager) ReadSessionWithSessionCondition(sessionCondition model.SessionCondition) []*session.GlobalSession {
	if sessionCondition.XID != "" {
		globalSession := storeManager.ReadSession(sessionCondition.XID)
		if globalSession != nil {
			globalSessions := make([]*session.GlobalSession, 0)
			globalSessions = append(globalSessions, globalSession)
			return globalSessions
		}
	} else if sessionCondition.TransactionID != 0 {
		globalSession := storeManager.ReadSessionByTransactionID(sessionCondition.TransactionID)
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

func (storeManager *DBTransactionStoreManager) GetCurrentMaxSessionID() int64 {
	return storeManager.LogStore.GetCurrentMaxSessionID(uuid.GetMaxUUID(), uuid.GetInitUUID())
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
		session.WithGsXID(globalTransactionDO.XID),
		session.WithGsApplicationID(globalTransactionDO.ApplicationID),
		session.WithGsTransactionID(globalTransactionDO.TransactionID),
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
		session.WithBsXid(branchTransactionDO.XID),
		session.WithBsTransactionID(branchTransactionDO.TransactionID),
		session.WithBsApplicationData(branchTransactionDO.ApplicationData),
		session.WithBsBranchID(branchTransactionDO.BranchID),
		session.WithBsBranchType(meta.ValueOfBranchType(branchTransactionDO.BranchType)),
		session.WithBsResourceID(branchTransactionDO.ResourceID),
		session.WithBsClientID(branchTransactionDO.ClientID),
		session.WithBsResourceGroupID(branchTransactionDO.ResourceGroupID),
		session.WithBsStatus(meta.BranchStatus(branchTransactionDO.Status)),
	)
	return branchSession
}

func convertGlobalTransactionDO(sessionStorable session.SessionStorable) *model.GlobalTransactionDO {
	globalSession, ok := sessionStorable.(*session.GlobalSession)
	if sessionStorable == nil || !ok {
		log.Errorf("the parameter of SessionStorable is not available, SessionStorable:%v", sessionStorable)
		return nil
	}
	globalTransactionDO := &model.GlobalTransactionDO{
		XID:                     globalSession.XID,
		TransactionID:           globalSession.TransactionID,
		Status:                  int32(globalSession.Status),
		ApplicationID:           globalSession.ApplicationID,
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
		log.Errorf("the parameter of SessionStorable is not available, SessionStorable:%v", sessionStorable)
		return nil
	}

	branchSessionDO := &model.BranchTransactionDO{
		XID:             branchSession.XID,
		TransactionID:   branchSession.TransactionID,
		BranchID:        branchSession.BranchID,
		ResourceGroupID: branchSession.ResourceGroupID,
		ResourceID:      branchSession.ResourceID,
		BranchType:      branchSession.BranchType.String(),
		Status:          int32(branchSession.Status),
		ClientID:        branchSession.ClientID,
		ApplicationData: branchSession.ApplicationData,
	}
	return branchSessionDO
}
