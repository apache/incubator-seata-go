package inmemory

import (
	"fmt"
	"sync"

	"github.com/opentrx/seata-golang/v2/pkg/apis"
	"github.com/opentrx/seata-golang/v2/pkg/tc/model"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage"
	"github.com/opentrx/seata-golang/v2/pkg/tc/storage/driver/factory"
	"github.com/opentrx/seata-golang/v2/pkg/util/log"
)

func init() {
	factory.Register("inmemory", &inMemoryFactory{})
}

// inMemoryFactory implements the factory.StorageDriverFactory interface
type inMemoryFactory struct{}

func (factory *inMemoryFactory) Create(parameters map[string]interface{}) (storage.Driver, error) {
	return &driver{
		SessionMap: &sync.Map{},
		LockMap:    &sync.Map{},
	}, nil
}

// inMemory driver only for testing
type driver struct {
	SessionMap *sync.Map

	LockMap *sync.Map
}

// Add global session.
func (driver *driver) AddGlobalSession(session *apis.GlobalSession) error {
	driver.SessionMap.Store(session.XID, &model.GlobalTransaction{
		GlobalSession:  session,
		BranchSessions: make(map[*apis.BranchSession]bool),
	})
	return nil
}

// Find global session.
func (driver *driver) FindGlobalSession(xid string) *apis.GlobalSession {
	globalTransaction, ok := driver.SessionMap.Load(xid)
	if ok {
		gt := globalTransaction.(*model.GlobalTransaction)
		return gt.GlobalSession
	}
	return nil
}

// Find global sessions list.
func (driver *driver) FindGlobalSessions(statuses []apis.GlobalSession_GlobalStatus) []*apis.GlobalSession {
	contains := func(statuses []apis.GlobalSession_GlobalStatus, status apis.GlobalSession_GlobalStatus) bool {
		for _, s := range statuses {
			if s == status {
				return true
			}
		}
		return false
	}
	var sessions = make([]*apis.GlobalSession, 0)
	driver.SessionMap.Range(func(key, value interface{}) bool {
		session := value.(*model.GlobalTransaction)
		if contains(statuses, session.Status) {
			sessions = append(sessions, session.GlobalSession)
		}
		return true
	})
	return sessions
}

// Find global sessions list with addressing identities
func (driver *driver) FindGlobalSessionsWithAddressingIdentities(statuses []apis.GlobalSession_GlobalStatus,
	addressingIdentities []string) []*apis.GlobalSession {
	contain := func(statuses []apis.GlobalSession_GlobalStatus, status apis.GlobalSession_GlobalStatus) bool {
		for _, s := range statuses {
			if s == status {
				return true
			}
		}
		return false
	}
	containAddressing := func(addressingIdentities []string, addressing string) bool {
		for _, s := range addressingIdentities {
			if s == addressing {
				return true
			}
		}
		return false
	}
	var sessions = make([]*apis.GlobalSession, 0)
	driver.SessionMap.Range(func(key, value interface{}) bool {
		session := value.(*model.GlobalTransaction)
		if contain(statuses, session.Status) && containAddressing(addressingIdentities, session.Addressing) {
			sessions = append(sessions, session.GlobalSession)
		}
		return true
	})
	return sessions
}

// All sessions collection.
func (driver *driver) AllSessions() []*apis.GlobalSession {
	var sessions = make([]*apis.GlobalSession, 0)
	driver.SessionMap.Range(func(key, value interface{}) bool {
		session := value.(*model.GlobalTransaction)
		sessions = append(sessions, session.GlobalSession)
		return true
	})
	return sessions
}

// Update global session status.
func (driver *driver) UpdateGlobalSessionStatus(session *apis.GlobalSession, status apis.GlobalSession_GlobalStatus) error {
	globalTransaction, ok := driver.SessionMap.Load(session.XID)
	if ok {
		gt := globalTransaction.(*model.GlobalTransaction)
		gt.Status = status
		return nil
	}
	return fmt.Errorf("could not find global transaction xid = %s", session.XID)
}

// Inactive global session.
func (driver *driver) InactiveGlobalSession(session *apis.GlobalSession) error {
	globalTransaction, ok := driver.SessionMap.Load(session.XID)
	if ok {
		gt := globalTransaction.(*model.GlobalTransaction)
		gt.Active = false
		return nil
	}
	return fmt.Errorf("could not find global transaction xid = %s", session.XID)
}

// Remove global session.
func (driver *driver) RemoveGlobalSession(session *apis.GlobalSession) error {
	driver.SessionMap.Delete(session.XID)
	return nil
}

// Add branch session.
func (driver *driver) AddBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error {
	globalTransaction, ok := driver.SessionMap.Load(globalSession.XID)
	if ok {
		gt := globalTransaction.(*model.GlobalTransaction)
		gt.BranchSessions[session] = true
		return nil
	}
	return fmt.Errorf("could not find global transaction xid = %s", session.XID)
}

// Find branch session.
func (driver *driver) FindBranchSessions(xid string) []*apis.BranchSession {
	globalTransaction, ok := driver.SessionMap.Load(xid)
	if ok {
		gt := globalTransaction.(*model.GlobalTransaction)
		branchSessions := make([]*apis.BranchSession, 0)
		for bs := range gt.BranchSessions {
			branchSessions = append(branchSessions, bs)
		}
		return branchSessions
	}
	return nil
}

// Find branch session.
func (driver *driver) FindBatchBranchSessions(xids []string) []*apis.BranchSession {
	branchSessions := make([]*apis.BranchSession, 0)
	for i := 0; i < len(xids); i++ {
		globalTransaction, ok := driver.SessionMap.Load(xids[i])
		if ok {
			gt := globalTransaction.(*model.GlobalTransaction)
			for bs := range gt.BranchSessions {
				branchSessions = append(branchSessions, bs)
			}
		}
	}
	return branchSessions
}

// Update branch session status.
func (driver *driver) UpdateBranchSessionStatus(session *apis.BranchSession, status apis.BranchSession_BranchStatus) error {
	session.Status = status
	return nil
}

// Remove branch session.
func (driver *driver) RemoveBranchSession(globalSession *apis.GlobalSession, session *apis.BranchSession) error {
	globalTransaction, ok := driver.SessionMap.Load(globalSession.XID)
	if ok {
		gt := globalTransaction.(*model.GlobalTransaction)
		delete(gt.BranchSessions, session)
		return nil
	}
	return fmt.Errorf("could not find global transaction xid = %s", session.XID)
}

// AcquireLock Acquire lock boolean.
func (driver *driver) AcquireLock(rowLocks []*apis.RowLock, skipCheckLock bool) bool {
	if rowLocks == nil {
		return true
	}

	for _, rowLock := range rowLocks {
		previousLockTransactionID, loaded := driver.LockMap.LoadOrStore(rowLock.RowKey, rowLock.TransactionID)
		if loaded {
			if previousLockTransactionID == rowLock.TransactionID {
				// Locked by me before
				continue
			} else {
				log.Infof("Global rowLock on [%s:%s] is holding by %d", rowLock.TableName, rowLock.PK, previousLockTransactionID)
				driver.ReleaseLock(rowLocks)
				return false
			}
		}
	}

	return true
}

// ReleaseLock Unlock boolean.
func (driver *driver) ReleaseLock(rowLocks []*apis.RowLock) bool {
	if rowLocks == nil {
		return true
	}

	for _, rowLock := range rowLocks {
		lockedTransactionID, loaded := driver.LockMap.Load(rowLock.RowKey)
		if loaded && lockedTransactionID == rowLock.TransactionID {
			driver.LockMap.Delete(rowLock.RowKey)
		}
	}

	return true
}

// IsLockable Is lockable boolean.
func (driver *driver) IsLockable(xid string, resourceID string, lockKey string) bool {
	rowLocks := storage.CollectRowLocks(lockKey, resourceID, xid)
	for _, rowLock := range rowLocks {
		lockedTransactionID, loaded := driver.LockMap.Load(rowLock.RowKey)
		if loaded && lockedTransactionID != rowLock.TransactionID {
			return false
		}
	}
	return true
}
