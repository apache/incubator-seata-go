package tcc

import (
	"sync"
)

import (
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/rm"
)

type TCCRm struct {
	rmRemoting rm.RMRemoting
	// resourceID -> resource
	resourceManagerMap sync.Map
}

func (t *TCCRm) RegisterResource(resource model.Resource) error {
	t.resourceManagerMap.Store(resource.GetResourceId(), resource)
	t.rmRemoting.RegisterResource(resource)
	return nil
}

func (t *TCCRm) GetManagedResources() sync.Map {
	return t.resourceManagerMap
}

// Commit a branch transaction
func (t *TCCRm) BranchCommit(branchType model.BranchType, xid, branchId int64, resourceId, applicationData string) (model.BranchStatus, error) {
	// TODO
	return 0, nil
}

// Rollback a branch transaction
func (t *TCCRm) BranchRollback(branchType model.BranchType, xid string, branchId int64, resourceId, applicationData string) (model.BranchStatus, error) {
	// TODO
	return 0, nil
}

func (t *TCCRm) GetBranchType() model.BranchType {
	return model.BranchTypeTCC
}
