package holder

import (
	"github.com/transaction-wg/seata-golang/pkg/base/meta"
	"github.com/transaction-wg/seata-golang/pkg/tc/config"
	"github.com/transaction-wg/seata-golang/pkg/tc/session"
	"github.com/transaction-wg/seata-golang/pkg/util/log"
	"github.com/transaction-wg/seata-golang/pkg/util/uuid"
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
			found := sessionManager.SessionMap[branchSession.XID]
			if found == nil {
				log.Warnf("GlobalSession Does Not Exists For BranchSession [%d/%s]", branchSession.BranchID, branchSession.XID)
			} else {
				existingBranch := found.GetBranch(branchSession.BranchID)
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
				delete(sessionManager.SessionMap, globalSession.XID)
				break
			default:
				break
			}
		}
	}
}

func (sessionManager *FileBasedSessionManager) restore(stores []*TransactionWriteStore, unhandledBranchSessions map[int64]*session.BranchSession) {
	maxRecoverID := uuid.UUID
	for _, store := range stores {
		logOperation := store.LogOperation
		sessionStorable := store.SessionRequest
		maxRecoverID = getMaxID(maxRecoverID, sessionStorable)
		switch logOperation {
		case LogOperationGlobalAdd, LogOperationGlobalUpdate:
			{
				globalSession := sessionStorable.(*session.GlobalSession)
				if globalSession.TransactionID == int64(0) {
					log.Errorf("Restore globalSession from file failed, the transactionID is zero , xid:%s", globalSession.XID)
					break
				}
				foundGlobalSession := sessionManager.SessionMap[globalSession.XID]
				if foundGlobalSession == nil {
					sessionManager.SessionMap[globalSession.XID] = globalSession
				} else {
					foundGlobalSession.Status = globalSession.Status
				}
				break
			}
		case LogOperationGlobalRemove:
			{
				globalSession := sessionStorable.(*session.GlobalSession)
				if globalSession.TransactionID == int64(0) {
					log.Errorf("Restore globalSession from file failed, the transactionID is zero , xid:%s", globalSession.XID)
					break
				}
				delete(sessionManager.SessionMap, globalSession.XID)
				break
			}
		case LogOperationBranchAdd, LogOperationBranchUpdate:
			{
				branchSession := sessionStorable.(*session.BranchSession)
				if branchSession.TransactionID == int64(0) {
					log.Errorf("Restore branchSession from file failed, the transactionID is zero , xid:%s", branchSession.XID)
					break
				}
				foundGlobalSession := sessionManager.SessionMap[branchSession.XID]
				if foundGlobalSession == nil {
					unhandledBranchSessions[branchSession.BranchID] = branchSession
				} else {
					existingBranch := foundGlobalSession.GetBranch(branchSession.BranchID)
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
				if branchSession.TransactionID == int64(0) {
					log.Errorf("Restore branchSession from file failed, the transactionID is zero , xid:%s", branchSession.XID)
					break
				}
				foundGlobalSession := sessionManager.SessionMap[branchSession.XID]
				if foundGlobalSession == nil {
					log.Infof("GlobalSession To Be Updated (Remove Branch) Does Not Exists [%d/%s]", branchSession.BranchID, branchSession.XID)
				} else {
					existingBranch := foundGlobalSession.GetBranch(branchSession.BranchID)
					if existingBranch == nil {
						log.Infof("BranchSession To Be Updated Does Not Exists [%d/%s]", branchSession.BranchID, branchSession.XID)
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
	setMaxID(maxRecoverID)
}

func getMaxID(maxRecoverID int64, sessionStorable session.SessionStorable) int64 {
	var currentID int64 = 0
	var gs, ok1 = sessionStorable.(*session.GlobalSession)
	if ok1 {
		currentID = gs.TransactionID
	}

	var bs, ok2 = sessionStorable.(*session.BranchSession)
	if ok2 {
		currentID = bs.BranchID
	}

	if maxRecoverID > currentID {
		return maxRecoverID
	} else {
		return currentID
	}
}

func setMaxID(maxRecoverID int64) {
	var currentID int64
	// will be recover multi-thread later
	for {
		currentID = uuid.UUID
		if currentID < maxRecoverID {
			if uuid.SetUUID(currentID, maxRecoverID) {
				break
			}
		}
		break
	}
}
