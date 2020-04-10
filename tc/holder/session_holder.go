package holder

import (
	"github.com/dk-lockdown/seata-golang/logging"
	"github.com/dk-lockdown/seata-golang/meta"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/lock"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type SessionHolder struct {
	RootSessionManager             ISessionManager
	AsyncCommittingSessionManager  ISessionManager
	RetryCommittingSessionManager  ISessionManager
	RetryRollbackingSessionManager ISessionManager
}

var sessionHolder SessionHolder

func init() {
	sessionHolder = SessionHolder{
		RootSessionManager:             NewFileBasedSessionManager(config.GetDefaultFileStoreConfig()),
		AsyncCommittingSessionManager:  NewDefaultSessionManager("default"),
		RetryCommittingSessionManager:  NewDefaultSessionManager("default"),
		RetryRollbackingSessionManager: NewDefaultSessionManager("default"),
	}

	sessionHolder.reload()
}

func GetSessionHolder() SessionHolder {
	return sessionHolder
}

func (sessionHolder SessionHolder) FindGlobalSession(xid string) *session.GlobalSession {
	return sessionHolder.FindGlobalSessionWithBranchSessions(xid, true)
}

func (sessionHolder SessionHolder) FindGlobalSessionWithBranchSessions(xid string, withBranchSessions bool) *session.GlobalSession {
	return sessionHolder.RootSessionManager.FindGlobalSessionWithBranchSessions(xid, withBranchSessions)
}

func (sessionHolder SessionHolder) reload() {
	sessionManager, reloadable := sessionHolder.RootSessionManager.(Reloadable)
	if reloadable {
		sessionManager.Reload()

		reloadedSessions := sessionHolder.RootSessionManager.AllSessions()
		if reloadedSessions != nil && len(reloadedSessions) > 0 {
			for _,globalSession := range reloadedSessions {
				switch globalSession.Status {
				case meta.GlobalStatusUnknown:
				case meta.GlobalStatusCommitted:
				case meta.GlobalStatusCommitFailed:
				case meta.GlobalStatusRollbacked:
				case meta.GlobalStatusRollbackFailed:
				case meta.GlobalStatusTimeoutRollbacked:
				case meta.GlobalStatusTimeoutRollbackFailed:
				case meta.GlobalStatusFinished:
					logging.Logger.Errorf("Reloaded Session should NOT be %s",globalSession.Status.String())
					break
				case meta.GlobalStatusAsyncCommitting:
					sessionHolder.AsyncCommittingSessionManager.AddGlobalSession(globalSession)
					break
				default:
					branchSessions := globalSession.GetSortedBranches()
					for _,branchSession := range branchSessions {
						lock.GetLockManager().AcquireLock(branchSession)
					}
					switch globalSession.Status {
					case meta.GlobalStatusCommitting:
					case meta.GlobalStatusCommitRetrying:
						sessionHolder.RetryCommittingSessionManager.AddGlobalSession(globalSession)
						break
					case meta.GlobalStatusRollbacking:
					case meta.GlobalStatusRollbackRetrying:
					case meta.GlobalStatusTimeoutRollbacking:
					case meta.GlobalStatusTimeoutRollbackRetrying:
						sessionHolder.RetryRollbackingSessionManager.AddGlobalSession(globalSession)
						break
					case meta.GlobalStatusBegin:
						globalSession.SetActive(true)
						break
					default:
						logging.Logger.Errorf("NOT properly handled %s",globalSession.Status)
						break
					}
					break
				}
			}
		}
	}
}