package model

import (
	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/util/time"
)

type GlobalTransaction struct {
	*apis.GlobalSession

	BranchSessions map[*apis.BranchSession]bool
}

func (gt *GlobalTransaction) Add(branchSession *apis.BranchSession) {
	branchSession.Status = apis.Registered
	gt.BranchSessions[branchSession] = true
}

func (gt *GlobalTransaction) Remove(branchSession *apis.BranchSession) {
	delete(gt.BranchSessions, branchSession)
}

func (gt *GlobalTransaction) GetBranch(branchID int64) *apis.BranchSession {
	for branchSession := range gt.BranchSessions {
		if branchSession.BranchID == branchID {
			return branchSession
		}
	}
	return nil
}

func (gt *GlobalTransaction) CanBeCommittedAsync() bool {
	for branchSession := range gt.BranchSessions {
		if branchSession.Type == apis.TCC {
			return false
		}
	}
	return true
}

func (gt *GlobalTransaction) IsSaga() bool {
	for branchSession := range gt.BranchSessions {
		if branchSession.Type == apis.SAGA {
			return true
		}
	}
	return false
}

func (gt *GlobalTransaction) IsTimeout() bool {
	return (time.CurrentTimeMillis() - uint64(gt.BeginTime)) > uint64(gt.Timeout)
}

func (gt *GlobalTransaction) IsTimeoutGlobalStatus() bool {
	return gt.Status == apis.TimeoutRolledBack ||
		gt.Status == apis.TimeoutRollbackFailed ||
		gt.Status == apis.TimeoutRollingBack ||
		gt.Status == apis.TimeoutRollbackRetrying
}

func (gt *GlobalTransaction) HasBranch() bool {
	return len(gt.BranchSessions) > 0
}

func (gt *GlobalTransaction) Begin() {
	gt.Status = apis.Begin
	gt.BeginTime = int64(time.CurrentTimeMillis())
	gt.Active = true
}
