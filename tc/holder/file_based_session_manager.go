package holder

import (
	"github.com/dk-lockdown/seata-golang/base/meta"
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/pkg/uuid"
	"github.com/dk-lockdown/seata-golang/tc/config"
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type FileBasedSessionManager struct {
	conf config.FileStoreConfig
	DefaultSessionManager
}

func NewFileBasedSessionManager(conf config.FileStoreConfig) SessionManager {
	transactionStoreManager := &FileTransactionStoreManager{}
	transactionStoreManager.InitFile(conf.FileDir)
	sessionManager := DefaultSessionManager{
		AbstractSessionManager: AbstractSessionManager{
			TransactionStoreManager: transactionStoreManager,
			Name:                    conf.FileDir,
		},
		SessionMap: make(map[string]*session.GlobalSession),
	}
	transactionStoreManager.SessionManager = &sessionManager
	return &FileBasedSessionManager{
		conf:                  conf,
		DefaultSessionManager: sessionManager,
	}
}

func (sessionManager *FileBasedSessionManager) Reload() {
	sessionManager.restoreSessions()
	sessionManager.washSessions()
}

func (sessionManager *FileBasedSessionManager) restoreSessions() {
	unhandledBranchBuffer := make(map[int64]*session.BranchSession)
	sessionManager.restoreSessionsToUnhandledBranchBuffer(true, unhandledBranchBuffer)
	sessionManager.restoreSessionsToUnhandledBranchBuffer(false, unhandledBranchBuffer)
	if len(unhandledBranchBuffer) > 0 {
		for _, branchSession := range unhandledBranchBuffer {
			found := sessionManager.SessionMap[branchSession.Xid]
			if found == nil {
				logging.Logger.Infof("GlobalSession Does Not Exists For BranchSession [%d/%s]", branchSession.BranchId, branchSession.Xid)
			} else {
				existingBranch := found.GetBranch(branchSession.BranchId)
				if existingBranch == nil {
					found.Add(branchSession)
				} else {
					existingBranch.Status = branchSession.Status
				}
			}
		}
	}
}

func (sessionManager *FileBasedSessionManager) restoreSessionsToUnhandledBranchBuffer(isHistory bool, unhandledBranchSessions map[int64]*session.BranchSession) {
	transactionStoreManager, ok := sessionManager.TransactionStoreManager.(ReloadableStore)
	if !ok {
		return
	}
	for {
		if transactionStoreManager.HasRemaining(isHistory) {
			stores := transactionStoreManager.ReadWriteStore(sessionManager.conf.SessionReloadReadSize, isHistory)
			sessionManager.restore(stores, unhandledBranchSessions)
		} else {
			break
		}
	}
}

func (sessionManager *FileBasedSessionManager) washSessions() {
	if len(sessionManager.SessionMap) > 0 {
		for _, globalSession := range sessionManager.SessionMap {
			switch globalSession.Status {
			case meta.GlobalStatusUnknown, meta.GlobalStatusCommitted, meta.GlobalStatusCommitFailed, meta.GlobalStatusRollbacked,
				meta.GlobalStatusRollbackFailed, meta.GlobalStatusTimeoutRollbacked, meta.GlobalStatusTimeoutRollbackFailed,
				meta.GlobalStatusFinished:
				// Remove all sessions finished
				delete(sessionManager.SessionMap, globalSession.Xid)
				break
			default:
				break
			}
		}
	}
}

func (sessionManager *FileBasedSessionManager) restore(stores []*TransactionWriteStore, unhandledBranchSessions map[int64]*session.BranchSession) {
	maxRecoverId := uuid.UUID
	for _, store := range stores {
		logOperation := store.LogOperation
		sessionStorable := store.SessionRequest
		maxRecoverId = getMaxId(maxRecoverId, sessionStorable)
		switch logOperation {
		case LogOperationGlobalAdd, LogOperationGlobalUpdate:
			{
				globalSession := sessionStorable.(*session.GlobalSession)
				if globalSession.TransactionId == int64(0) {
					logging.Logger.Errorf("Restore globalSession from file failed, the transactionId is zero , xid:%s", globalSession.Xid)
					break
				}
				foundGlobalSession := sessionManager.SessionMap[globalSession.Xid]
				if foundGlobalSession == nil {
					sessionManager.SessionMap[globalSession.Xid] = globalSession
				} else {
					foundGlobalSession.Status = globalSession.Status
				}
				break
			}
		case LogOperationGlobalRemove:
			{
				globalSession := sessionStorable.(*session.GlobalSession)
				if globalSession.TransactionId == int64(0) {
					logging.Logger.Errorf("Restore globalSession from file failed, the transactionId is zero , xid:%s", globalSession.Xid)
					break
				}
				delete(sessionManager.SessionMap, globalSession.Xid)
				break
			}
		case LogOperationBranchAdd, LogOperationBranchUpdate:
			{
				branchSession := sessionStorable.(*session.BranchSession)
				if branchSession.TransactionId == int64(0) {
					logging.Logger.Errorf("Restore branchSession from file failed, the transactionId is zero , xid:%s", branchSession.Xid)
					break
				}
				foundGlobalSession := sessionManager.SessionMap[branchSession.Xid]
				if foundGlobalSession == nil {
					unhandledBranchSessions[branchSession.BranchId] = branchSession
				} else {
					existingBranch := foundGlobalSession.GetBranch(branchSession.BranchId)
					if existingBranch == nil {
						foundGlobalSession.Add(branchSession)
					} else {
						existingBranch.Status = branchSession.Status
					}
				}
				break
			}
		case LogOperationBranchRemove:
			{
				branchSession := sessionStorable.(*session.BranchSession)
				if branchSession.TransactionId == int64(0) {
					logging.Logger.Errorf("Restore branchSession from file failed, the transactionId is zero , xid:%s", branchSession.Xid)
					break
				}
				foundGlobalSession := sessionManager.SessionMap[branchSession.Xid]
				if foundGlobalSession == nil {
					logging.Logger.Infof("GlobalSession To Be Updated (Remove Branch) Does Not Exists [%d/%s]", branchSession.BranchId, branchSession.Xid)
				} else {
					existingBranch := foundGlobalSession.GetBranch(branchSession.BranchId)
					if existingBranch == nil {
						logging.Logger.Infof("BranchSession To Be Updated Does Not Exists [%d/%s]", branchSession.BranchId, branchSession.Xid)
					} else {
						foundGlobalSession.Remove(existingBranch)
					}
				}
				break
			}
		default:
			break
		}
	}
	setMaxId(maxRecoverId)
}

func getMaxId(maxRecoverId int64, sessionStorable session.SessionStorable) int64 {
	var currentId int64 = 0
	var gs, ok1 = sessionStorable.(*session.GlobalSession)
	if ok1 {
		currentId = gs.TransactionId
	}

	var bs, ok2 = sessionStorable.(*session.BranchSession)
	if ok2 {
		currentId = bs.BranchId
	}

	if maxRecoverId > currentId {
		return maxRecoverId
	} else {
		return currentId
	}
}

func setMaxId(maxRecoverId int64) {
	var currentId int64
	// will be recover multi-thread later
	for {
		currentId = uuid.UUID
		if currentId < maxRecoverId {
			if uuid.SetUUID(currentId, maxRecoverId) {
				break
			}
		}
		break
	}
}
