package rm

import (
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

func init() {
	rmFacadeInstance = &ResourceManagerFacade{}
}

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
func RegisterResource(resourceManager model2.ResourceManager) {
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
func (d *ResourceManagerFacade) BranchCommit(branchType model2.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (model2.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchCommit(branchType, xid, branchId, resourceId, applicationData)
}

// Rollback a branch transaction
func (d *ResourceManagerFacade) BranchRollback(branchType model2.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (model2.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchRollback(branchType, xid, branchId, resourceId, applicationData)
}

// Branch register long
func (d *ResourceManagerFacade) BranchRegister(branchType model2.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return d.GetResourceManager(branchType).BranchRegister(branchType, resourceId, clientId, xid, applicationData, lockKeys)
}

//  Branch report
func (d *ResourceManagerFacade) BranchReport(branchType model2.BranchType, xid string, branchId int64, status model2.BranchStatus, applicationData string) error {
	return d.GetResourceManager(branchType).BranchReport(branchType, xid, branchId, status, applicationData)
}

// Lock query boolean
func (d *ResourceManagerFacade) LockQuery(branchType model2.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return d.GetResourceManager(branchType).LockQuery(branchType, resourceId, xid, lockKeys)
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
