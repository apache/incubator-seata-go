package lock

import (
	"github.com/dk-lockdown/seata-golang/tc/session"
)

type ILockManager interface {
	/**
	 * Acquire lock boolean.
	 *
	 * @param branchSession the branch session
	 * @return the boolean
	 * @throws TransactionException the transaction exception
	 */
	AcquireLock(branchSession *session.BranchSession) (bool, error)

	/**
	 * Un lock boolean.
	 *
	 * @param branchSession the branch session
	 * @return the boolean
	 * @throws TransactionException the transaction exception
	 */
	ReleaseLock(branchSession *session.BranchSession) (bool, error)

	/**
	 * GlobalSession 是没有锁的，所有的锁都在 BranchSession 上，因为 BranchSession 才
	 * 持有资源，释放 GlobalSession 锁是指释放它所有的 BranchSession 上的锁
	 * Un lock boolean.
	 *
	 * @param globalSession the global session
	 * @return the boolean
	 * @throws TransactionException the transaction exception
	 */
	ReleaseGlobalSessionLock(globalSession *session.GlobalSession) (bool, error)

	/**
	 * Is lockable boolean.
	 *
	 * @param xid        the xid
	 * @param resourceId the resource id
	 * @param lockKey    the lock key
	 * @return the boolean
	 * @throws TransactionException the transaction exception
	 */
	IsLockable(xid string, resourceId string, lockKey string) bool

	/**
	 * Clean all locks.
	 *
	 * @throws TransactionException the transaction exception
	 */
	CleanAllLocks()

	GetLockKeyCount() int64
}