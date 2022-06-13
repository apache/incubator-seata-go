package rm

import (
	"context"
	"fmt"
	"sync"
)

import (
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/resource"
)

var (
	// BranchType -> ResourceManager
	resourceManagerMap sync.Map
	// singletone ResourceManager
	rmFacadeInstance *ResourceManager
	onceRMFacade     = &sync.Once{}
)

func GetResourceManagerInstance() *ResourceManager {
	if rmFacadeInstance == nil {
		onceRMFacade.Do(func() {
			rmFacadeInstance = &ResourceManager{}
		})
	}
	return rmFacadeInstance
}

type ResourceManager struct {
}

// 将事务管理器注册到这里
func RegisterResourceManager(resourceManager resource.ResourceManager) {
	resourceManagerMap.Store(resourceManager.GetBranchType(), resourceManager)
}

func (*ResourceManager) GetResourceManager(branchType branch.BranchType) resource.ResourceManager {
	rm, ok := resourceManagerMap.Load(branchType)
	if !ok {
		panic(fmt.Sprintf("No ResourceManager for BranchType: %v", branchType))
	}
	return rm.(resource.ResourceManager)
}

// Commit a branch transaction
func (d *ResourceManager) BranchCommit(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (branch.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchCommit(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Rollback a branch transaction
func (d *ResourceManager) BranchRollback(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (branch.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchRollback(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Branch register long
func (d *ResourceManager) BranchRegister(ctx context.Context, branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return d.GetResourceManager(branchType).BranchRegister(ctx, branchType, resourceId, clientId, xid, applicationData, lockKeys)
}

//  Branch report
func (d *ResourceManager) BranchReport(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	return d.GetResourceManager(branchType).BranchReport(ctx, branchType, xid, branchId, status, applicationData)
}

// Lock query boolean
func (d *ResourceManager) LockQuery(ctx context.Context, branchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return d.GetResourceManager(branchType).LockQuery(ctx, branchType, resourceId, xid, lockKeys)
}

// Register a   model.Resource to be managed by   model.Resource Manager
func (d *ResourceManager) RegisterResource(resource resource.Resource) error {
	return d.GetResourceManager(resource.GetBranchType()).RegisterResource(resource)
}

//  Unregister a   model.Resource from the   model.Resource Manager
func (d *ResourceManager) UnregisterResource(resource resource.Resource) error {
	return d.GetResourceManager(resource.GetBranchType()).UnregisterResource(resource)
}

// Get all resources managed by this manager
func (d *ResourceManager) GetManagedResources() sync.Map {
	return resourceManagerMap
}

// Get the model.BranchType
func (d *ResourceManager) GetBranchType() branch.BranchType {
	panic("DefaultResourceManager isn't a real ResourceManager")
}
