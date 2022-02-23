package holder

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage"
)

type SessionHolderInterface interface {
	AddGlobalSession(session *apis.GlobalSession) error
	FindGlobalSession(xid string) *apis.GlobalSession
	FindGlobalTransaction(xid string) *model.GlobalTransaction
	FindAsyncCommittingGlobalTransactions(addressingIdentities []string) []*model.GlobalTransaction
	FindRetryCommittingGlobalTransactions(addressingIdentities []string) []*model.GlobalTransaction
	FindRetryRollbackGlobalTransactions(addressingIdentities []string) []*model.GlobalTransaction
	FindGlobalSessions(statuses []apis.GlobalSession_GlobalStatus) []*apis.GlobalSession
	AllSessions() []*apis.GlobalSession
	UpdateGlobalSessionStatus(session *apis.GlobalSession, status apis.GlobalSession_GlobalStatus) error
	InactiveGlobalSession(session *apis.GlobalSession) error
	RemoveGlobalSession(session *apis.GlobalSession) error
	RemoveGlobalTransaction(globalTransaction *model.GlobalTransaction) error
	AddBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error
	FindBranchSession(xid string) []*apis.BranchSession
	UpdateBranchSessionStatus(session *apis.BranchSession, status apis.BranchSession_BranchStatus) error
	RemoveBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error
}

type SessionHolder struct {
	manager storage.SessionManager
}

func NewSessionHolder(manager storage.SessionManager) *SessionHolder {
	return &SessionHolder{manager: manager}
}

func (holder *SessionHolder) AddGlobalSession(session *apis.GlobalSession) error {
	return holder.manager.AddGlobalSession(session)
}

func (holder *SessionHolder) FindGlobalSession(xid string) *apis.GlobalSession {
	return holder.manager.FindGlobalSession(xid)
}

func (holder *SessionHolder) FindGlobalTransaction(xid string) *model.GlobalTransaction {
	globalSession := holder.manager.FindGlobalSession(xid)
	if globalSession != nil {
		gt := &model.GlobalTransaction{GlobalSession: globalSession}
		branchSessions := holder.manager.FindBranchSessions(xid)
		if len(branchSessions) != 0 {
			gt.BranchSessions = make(map[*apis.BranchSession]bool, len(branchSessions))
			for i := 0; i < len(branchSessions); i++ {
				gt.BranchSessions[branchSessions[i]] = true
			}
		}
		return gt
	}
	return nil
}

func (holder *SessionHolder) FindAsyncCommittingGlobalTransactions(addressingIdentities []string) []*model.GlobalTransaction {
	return holder.findGlobalTransactionsWithAddressingIdentities([]apis.GlobalSession_GlobalStatus{
		apis.AsyncCommitting,
	}, addressingIdentities)
}

func (holder *SessionHolder) FindRetryCommittingGlobalTransactions(addressingIdentities []string) []*model.GlobalTransaction {
	return holder.findGlobalTransactionsWithAddressingIdentities([]apis.GlobalSession_GlobalStatus{
		apis.CommitRetrying,
	}, addressingIdentities)
}

func (holder *SessionHolder) FindRetryRollbackGlobalTransactions(addressingIdentities []string) []*model.GlobalTransaction {
	return holder.findGlobalTransactionsWithAddressingIdentities([]apis.GlobalSession_GlobalStatus{
		apis.RollingBack, apis.RollbackRetrying, apis.TimeoutRollingBack, apis.TimeoutRollbackRetrying,
	}, addressingIdentities)
}

func (holder *SessionHolder) findGlobalTransactions(statuses []apis.GlobalSession_GlobalStatus) []*model.GlobalTransaction {
	gts := holder.manager.FindGlobalSessions(statuses)
	return holder.findGlobalTransactionsByGlobalSessions(gts)
}

func (holder *SessionHolder) findGlobalTransactionsWithAddressingIdentities(statuses []apis.GlobalSession_GlobalStatus,
	addressingIdentities []string) []*model.GlobalTransaction {
	gts := holder.manager.FindGlobalSessionsWithAddressingIdentities(statuses, addressingIdentities)
	return holder.findGlobalTransactionsByGlobalSessions(gts)
}

func (holder *SessionHolder) findGlobalTransactionsByGlobalSessions(sessions []*apis.GlobalSession) []*model.GlobalTransaction {
	if len(sessions) == 0 {
		return nil
	}

	xids := make([]string, 0, len(sessions))
	for _, gt := range sessions {
		xids = append(xids, gt.XID)
	}
	branchSessions := holder.manager.FindBatchBranchSessions(xids)
	branchSessionMap := make(map[string][]*apis.BranchSession)
	for i := 0; i < len(branchSessions); i++ {
		branchSessionSlice, ok := branchSessionMap[branchSessions[i].XID]
		if ok {
			branchSessionSlice = append(branchSessionSlice, branchSessions[i])
			branchSessionMap[branchSessions[i].XID] = branchSessionSlice
		} else {
			branchSessionSlice = make([]*apis.BranchSession, 0)
			branchSessionSlice = append(branchSessionSlice, branchSessions[i])
			branchSessionMap[branchSessions[i].XID] = branchSessionSlice
		}
	}

	globalTransactions := make([]*model.GlobalTransaction, 0, len(sessions))
	for j := 0; j < len(sessions); j++ {
		globalTransaction := &model.GlobalTransaction{
			GlobalSession:  sessions[j],
			BranchSessions: map[*apis.BranchSession]bool{},
		}

		branchSessionSlice := branchSessionMap[sessions[j].XID]
		if len(branchSessionSlice) > 0 {
			for x := 0; x < len(branchSessionSlice); x++ {
				globalTransaction.BranchSessions[branchSessionSlice[x]] = true
			}
		}
		globalTransactions = append(globalTransactions, globalTransaction)
	}

	return globalTransactions
}

func (holder *SessionHolder) FindGlobalSessions(statuses []apis.GlobalSession_GlobalStatus) []*apis.GlobalSession {
	return holder.manager.FindGlobalSessions(statuses)
}

func (holder *SessionHolder) AllSessions() []*apis.GlobalSession {
	return holder.manager.AllSessions()
}

func (holder *SessionHolder) UpdateGlobalSessionStatus(session *apis.GlobalSession, status apis.GlobalSession_GlobalStatus) error {
	session.Status = status
	return holder.manager.UpdateGlobalSessionStatus(session, status)
}

func (holder *SessionHolder) InactiveGlobalSession(session *apis.GlobalSession) error {
	session.Active = false
	return holder.manager.InactiveGlobalSession(session)
}

func (holder *SessionHolder) RemoveGlobalSession(session *apis.GlobalSession) error {
	return holder.manager.RemoveGlobalSession(session)
}

func (holder *SessionHolder) RemoveGlobalTransaction(globalTransaction *model.GlobalTransaction) error {
	err := holder.manager.RemoveGlobalSession(globalTransaction.GlobalSession)
	if err != nil {
		return err
	}
	for bs := range globalTransaction.BranchSessions {
		err = holder.manager.RemoveBranchSession(globalTransaction.GlobalSession, bs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (holder *SessionHolder) AddBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error {
	return holder.manager.AddBranchSession(globalSession, session)
}

func (holder *SessionHolder) FindBranchSession(xid string) []*apis.BranchSession {
	return holder.manager.FindBranchSessions(xid)
}

func (holder *SessionHolder) UpdateBranchSessionStatus(session *apis.BranchSession, status apis.BranchSession_BranchStatus) error {
	return holder.manager.UpdateBranchSessionStatus(session, status)
}

func (holder *SessionHolder) RemoveBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error {
	return holder.manager.RemoveBranchSession(globalSession, session)
}
