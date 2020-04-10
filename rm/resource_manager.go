package rm

import (
	"github.com/dk-lockdown/seata-golang/meta"
	"github.com/dk-lockdown/seata-golang/model"
)

type IResourceManagerInbound interface {
	/**
	 * Commit a branch transaction.
	 *
	 * @param branchType      the branch type
	 * @param xid             Transaction id.
	 * @param branchId        Branch id.
	 * @param resourceId      Resource id.
	 * @param applicationData Application data bind with this branch.
	 * @return Status of the branch after committing.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 *                              out.
	 */
	BranchCommit(branchType meta.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (meta.BranchStatus, error)

	/**
	 * Rollback a branch transaction.
	 *
	 * @param branchType      the branch type
	 * @param xid             Transaction id.
	 * @param branchId        Branch id.
	 * @param resourceId      Resource id.
	 * @param applicationData Application data bind with this branch.
	 * @return Status of the branch after rollbacking.
	 * @throws TransactionException Any exception that fails this will be wrapped with TransactionException and thrown
	 *                              out.
	 */
	BranchRollback(branchType meta.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (meta.BranchStatus, error)
}

type IResourceManagerOutbound interface {
	/**
	 * Branch register long.
	 *
	 * @param branchType the branch type
	 * @param resourceId the resource id
	 * @param clientId   the client id
	 * @param xid        the xid
	 * @param applicationData the context
	 * @param lockKeys   the lock keys
	 * @return the long
	 * @throws TransactionException the transaction exception
	 */
	BranchRegister(branchType meta.BranchType, resourceId string, clientId string, xid string, applicationData []byte, lockKeys string) (int64, error)

	/**
	 * Branch report.
	 *
	 * @param branchType      the branch type
	 * @param xid             the xid
	 * @param branchId        the branch id
	 * @param status          the status
	 * @param applicationData the application data
	 * @throws TransactionException the transaction exception
	 */
	BranchReport(branchType meta.BranchType, xid string, branchId int64, status meta.BranchStatus, applicationData []byte) error

	/**
	 * Lock query boolean.
	 *
	 * @param branchType the branch type
	 * @param resourceId the resource id
	 * @param xid        the xid
	 * @param lockKeys   the lock keys
	 * @return the boolean
	 * @throws TransactionException the transaction exception
	 */
	LockQuery(branchType meta.BranchType, resourceId string, xid string, lockKeys string) (bool, error)
}

type IResourceManager interface {
	IResourceManagerInbound
	IResourceManagerOutbound

	/**
	 * Register a Resource to be managed by Resource Manager.
	 *
	 * @param resource The resource to be managed.
	 */
	registerResource(resource model.IResource)

	/**
	 * Unregister a Resource from the Resource Manager.
	 *
	 * @param resource The resource to be removed.
	 */
	unregisterResource(resource model.IResource)

	/**
	 * Get all resources managed by this manager.
	 *
	 * @return resourceId -> Resource Map
	 */
	getManagedResources() map[string]model.IResource

	/**
	 * Get the BranchType.
	 *
	 * @return The BranchType of ResourceManager.
	 */
	getBranchType() meta.BranchType
}