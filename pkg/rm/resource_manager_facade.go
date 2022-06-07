package rm

import (
	"context"
	"fmt"
	"sync"
)

import (
	model2 "github.com/seata/seata-go/pkg/common/model"
)

var (
	// BranchType -> ResourceManager
	resourceManagerMap sync.Map
	// singletone ResourceManagerFacade
	rmFacadeInstance *ResourceManagerFacade
	onceRMFacade     = &sync.Once{}
)

func GetResourceManagerFacadeInstance() *ResourceManagerFacade {
	if rmFacadeInstance == nil {
		onceRMFacade.Do(func() {
			rmFacadeInstance = &ResourceManagerFacade{}
		})
	}
	return rmFacadeInstance
}

type ResourceManagerFacade struct {
}

// 将事务管理器注册到这里
func RegisterResourceManager(resourceManager model2.ResourceManager) {
	resourceManagerMap.Store(resourceManager.GetBranchType(), resourceManager)
}

func (*ResourceManagerFacade) GetResourceManager(branchType model2.BranchType) model2.ResourceManager {
	rm, ok := resourceManagerMap.Load(branchType)
	if !ok {
		panic(fmt.Sprintf("No ResourceManager for BranchType: %v", branchType))
	}
	return rm.(model2.ResourceManager)
}

// Commit a branch transaction
func (d *ResourceManagerFacade) BranchCommit(ctx context.Context, branchType model2.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (model2.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchCommit(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Rollback a branch transaction
func (d *ResourceManagerFacade) BranchRollback(ctx context.Context, branchType model2.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (model2.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchRollback(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Branch register long
func (d *ResourceManagerFacade) BranchRegister(ctx context.Context, branchType model2.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return d.GetResourceManager(branchType).BranchRegister(ctx, branchType, resourceId, clientId, xid, applicationData, lockKeys)
}

//  Branch report
func (d *ResourceManagerFacade) BranchReport(ctx context.Context, branchType model2.BranchType, xid string, branchId int64, status model2.BranchStatus, applicationData string) error {
	return d.GetResourceManager(branchType).BranchReport(ctx, branchType, xid, branchId, status, applicationData)
}

// Lock query boolean
func (d *ResourceManagerFacade) LockQuery(ctx context.Context, branchType model2.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return d.GetResourceManager(branchType).LockQuery(ctx, branchType, resourceId, xid, lockKeys)
}

// Register a   model.Resource to be managed by   model.Resource Manager
func (d *ResourceManagerFacade) RegisterResource(resource model2.Resource) error {
	return d.GetResourceManager(resource.GetBranchType()).RegisterResource(resource)
}

//  Unregister a   model.Resource from the   model.Resource Manager
func (d *ResourceManagerFacade) UnregisterResource(resource model2.Resource) error {
	return d.GetResourceManager(resource.GetBranchType()).UnregisterResource(resource)
}

// Get all resources managed by this manager
func (d *ResourceManagerFacade) GetManagedResources() sync.Map {
	return resourceManagerMap
}

// Get the model.BranchType
func (d *ResourceManagerFacade) GetBranchType() model2.BranchType {
	panic("DefaultResourceManager isn't a real ResourceManager")
}
