package holder

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/lock"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

var (
	ASYNC_COMMITTING_SESSION_MANAGER_NAME  = "async.commit.data"
	RETRY_COMMITTING_SESSION_MANAGER_NAME  = "retry.commit.data"
	RETRY_ROLLBACKING_SESSION_MANAGER_NAME = "retry.rollback.data"
)

type SessionHolder struct {
	RootSessionManager             SessionManager
	AsyncCommittingSessionManager  SessionManager
	RetryCommittingSessionManager  SessionManager
	RetryRollbackingSessionManager SessionManager
}

var sessionHolder SessionHolder

func Init() {
	if config.GetStoreConfig().StoreMode == "file" {
		sessionHolder = SessionHolder{
			RootSessionManager:             NewFileBasedSessionManager(config.GetStoreConfig().FileStoreConfig),
			AsyncCommittingSessionManager:  NewDefaultSessionManager(ASYNC_COMMITTING_SESSION_MANAGER_NAME),
			RetryCommittingSessionManager:  NewDefaultSessionManager(RETRY_COMMITTING_SESSION_MANAGER_NAME),
			RetryRollbackingSessionManager: NewDefaultSessionManager(RETRY_ROLLBACKING_SESSION_MANAGER_NAME),
		}
		sessionHolder.reload()
	}
	if config.GetStoreConfig().StoreMode == "db" {
		sessionHolder = SessionHolder{
			RootSessionManager:             NewDataBaseSessionManager("", config.GetStoreConfig().DBStoreConfig),
			AsyncCommittingSessionManager:  NewDataBaseSessionManager(ASYNC_COMMITTING_SESSION_MANAGER_NAME, config.GetStoreConfig().DBStoreConfig),
			RetryCommittingSessionManager:  NewDataBaseSessionManager(RETRY_COMMITTING_SESSION_MANAGER_NAME, config.GetStoreConfig().DBStoreConfig),
			RetryRollbackingSessionManager: NewDataBaseSessionManager(RETRY_ROLLBACKING_SESSION_MANAGER_NAME, config.GetStoreConfig().DBStoreConfig),
		}
		sessionHolder.reload()
	}
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
			for _, globalSession := range reloadedSessions {
				switch globalSession.Status {
				case meta.GlobalStatusUnknown, meta.GlobalStatusCommitted, meta.GlobalStatusCommitFailed, meta.GlobalStatusRollbacked,
					meta.GlobalStatusRollbackFailed, meta.GlobalStatusTimeoutRollbacked, meta.GlobalStatusTimeoutRollbackFailed,
					meta.GlobalStatusFinished:
					logging.Logger.Errorf("Reloaded Session should NOT be %s", globalSession.Status.String())
					break
				case meta.GlobalStatusAsyncCommitting:
					sessionHolder.AsyncCommittingSessionManager.AddGlobalSession(globalSession)
					break
				default:
					branchSessions := globalSession.GetSortedBranches()
					for _, branchSession := range branchSessions {
						lock.GetLockManager().AcquireLock(branchSession)
					}
					switch globalSession.Status {
					case meta.GlobalStatusCommitting, meta.GlobalStatusCommitRetrying:
						sessionHolder.RetryCommittingSessionManager.AddGlobalSession(globalSession)
						break
					case meta.GlobalStatusRollbacking, meta.GlobalStatusRollbackRetrying, meta.GlobalStatusTimeoutRollbacking,
						meta.GlobalStatusTimeoutRollbackRetrying:
						sessionHolder.RetryRollbackingSessionManager.AddGlobalSession(globalSession)
						break
					case meta.GlobalStatusBegin:
						globalSession.Active = true
						break
					default:
						logging.Logger.Errorf("NOT properly handled %s", globalSession.Status)
						break
					}
					break
				}
			}
		}
	}
}
