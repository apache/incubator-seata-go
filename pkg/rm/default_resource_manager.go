package rm

import (
	"fmt"
	"sync"
)

import (
	"github.com/seata/seata-go/pkg/model"
)

var (
	// BranchType -> ResourceManager
	resourceManagerMap sync.Map
	// singletone defaultResourceManager
	defaultRM *defaultResourceManager
)

func init() {
	defaultRM = &defaultResourceManager{}
}

type defaultResourceManager struct {
}

// 将事务管理器注册到这里
func RegisterResource(resourceManager model.ResourceManager) {
	resourceManagerMap.Store(resourceManager.GetBranchType(), resourceManager)
}

func GetDefaultResourceManager() model.ResourceManager {
	return defaultRM
}

func (*defaultResourceManager) getResourceManager(branchType model.BranchType) model.ResourceManager {
	rm, ok := resourceManagerMap.Load(branchType)
	if !ok {
		panic(fmt.Sprintf("No ResourceManager for BranchType: %v", branchType))
	}
	return rm.(model.ResourceManager)
}

// Commit a branch transaction
func (d *defaultResourceManager) BranchCommit(branchType model.BranchType, xid, branchId int64, resourceId, applicationData string) (model.BranchStatus, error) {
	return d.getResourceManager(branchType).BranchCommit(branchType, xid, branchId, resourceId, applicationData)
}

// Rollback a branch transaction
func (d *defaultResourceManager) BranchRollback(branchType model.BranchType, xid string, branchId int64, resourceId, applicationData string) (model.BranchStatus, error) {
	return d.getResourceManager(branchType).BranchRollback(branchType, xid, branchId, resourceId, applicationData)
}

// Branch register long
func (d *defaultResourceManager) BranchRegister(branchType model.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return d.getResourceManager(branchType).BranchRegister(branchType, resourceId, clientId, xid, applicationData, lockKeys)
}

//  Branch report
func (d *defaultResourceManager) BranchReport(branchType model.BranchType, xid string, branchId int64, status model.BranchStatus, applicationData string) error {
	return d.getResourceManager(branchType).BranchReport(branchType, xid, branchId, status, applicationData)
}

// Lock query boolean
func (d *defaultResourceManager) LockQuery(branchType model.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return d.getResourceManager(branchType).LockQuery(branchType, resourceId, xid, lockKeys)
}

// Register a   model.Resource to be managed by   model.Resource Manager
func (d *defaultResourceManager) RegisterResource(resource model.Resource) error {
	return d.getResourceManager(resource.GetBranchType()).RegisterResource(resource)
}

//  Unregister a   model.Resource from the   model.Resource Manager
func (d *defaultResourceManager) UnregisterResource(resource model.Resource) error {
	return d.getResourceManager(resource.GetBranchType()).UnregisterResource(resource)
}

// Get all resources managed by this manager
func (d *defaultResourceManager) GetManagedResources() map[string]model.Resource {
	var allResource map[string]model.Resource
	resourceManagerMap.Range(func(branchType, resourceManager interface{}) bool {
		rs := resourceManager.(model.ResourceManager).GetManagedResources()
		for k, v := range rs {
			rs[k] = v
		}
		return true
	})
	return allResource
}

// Get the model.BranchType
func (d *defaultResourceManager) GetBranchType() model.BranchType {
	panic("DefaultResourceManager isn't a real ResourceManager")
}
