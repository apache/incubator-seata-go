package holder

import (
	"fmt"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage"
)

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

func (holder *SessionHolder) FindAsyncCommittingGlobalTransactions() []*model.GlobalTransaction {
	return holder.findGlobalTransactions([]apis.GlobalSession_GlobalStatus{
		apis.AsyncCommitting,
	})
}

func (holder *SessionHolder) FindRetryCommittingGlobalTransactions() []*model.GlobalTransaction {
	return holder.findGlobalTransactions([]apis.GlobalSession_GlobalStatus{
		apis.CommitRetrying,
	})
}

func (holder *SessionHolder) FindRetryRollbackGlobalTransactions() []*model.GlobalTransaction {
	return holder.findGlobalTransactions([]apis.GlobalSession_GlobalStatus{
		apis.RollingBack, apis.RollbackRetrying, apis.TimeoutRollingBack, apis.TimeoutRollbackRetrying,
	})
}

func (holder *SessionHolder) findGlobalTransactions(statuses []apis.GlobalSession_GlobalStatus) []*model.GlobalTransaction {
	gts := holder.manager.FindGlobalSessions(statuses)
	if len(gts) == 0 {
		return nil
	}

	xids := make([]string, 0, len(gts))
	for _, gt := range gts {
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

	globalTransactions := make([]*model.GlobalTransaction, 0, len(gts))
	for j := 0; j < len(gts); j++ {
		globalTransaction := &model.GlobalTransaction{
			GlobalSession:  gts[j],
			BranchSessions: make(map[*apis.BranchSession]bool),
		}

		branchSessionSlice := branchSessionMap[gts[j].XID]
		if len(branchSessionSlice) > 0 {
			for x := 0; x < len(branchSessionSlice); x++ {
				globalTransaction.BranchSessions[branchSessionSlice[x]] = true
			}
		}
		globalTransactions = append(globalTransactions, globalTransaction)
	}

	return globalTransactions
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
		return fmt.Errorf("RemoveGlobalSession Err: %+v", err)
	}

	for bs := range globalTransaction.BranchSessions {
		err = holder.manager.RemoveBranchSession(globalTransaction.GlobalSession, bs)
		if err != nil {
			return fmt.Errorf("RemoveBranchSession Err: %+v", err)
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
