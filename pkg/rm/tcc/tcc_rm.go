package tcc

import (
	"context"
	"fmt"
	"github.com/seata/seata-go/pkg/protocol"
	"github.com/seata/seata-go/pkg/rm/tcc/api"
	"github.com/seata/seata-go/pkg/rpc/getty"
	"github.com/seata/seata-go/pkg/utils/log"
	"sync"
)

import (
	"github.com/seata/seata-go/pkg/common/model"
	"github.com/seata/seata-go/pkg/rm"
)

var (
	tCCResourceManager     *TCCResourceManager
	onceTCCResourceManager = &sync.Once{}
)

func init() {
	rm.RegisterResourceManager(GetTCCResourceManagerInstance())
}

func GetTCCResourceManagerInstance() *TCCResourceManager {
	if tCCResourceManager == nil {
		onceTCCResourceManager.Do(func() {
			tCCResourceManager = &TCCResourceManager{
				resourceManagerMap: sync.Map{},
				rmRemoting:         rm.GetRMRemotingInstance(),
			}
		})
	}
	return tCCResourceManager
}

type TCCResourceManager struct {
	rmRemoting *rm.RMRemoting
	// resourceID -> resource
	resourceManagerMap sync.Map
}

// register transaction branch
func (t *TCCResourceManager) BranchRegister(ctx context.Context, branchType model.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	request := protocol.BranchRegisterRequest{
		Xid:             xid,
		BranchType:      t.GetBranchType(),
		ResourceId:      resourceId,
		LockKey:         lockKeys,
		ApplicationData: []byte(applicationData),
	}
	response, err := getty.GetGettyRemotingClient().SendSyncRequest(request)
	if err != nil || response == nil {
		log.Errorf("BranchRegister error: %v, res %v", err.Error(), response)
		return 0, err
	}
	return response.(protocol.BranchRegisterResponse).BranchId, nil
}

func (t *TCCResourceManager) BranchReport(ctx context.Context, ranchType model.BranchType, xid string, branchId int64, status model.BranchStatus, applicationData string) error {
	//TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) LockQuery(ctx context.Context, ranchType model.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) UnregisterResource(resource model.Resource) error {
	//TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) RegisterResource(resource model.Resource) error {
	if _, ok := resource.(*TCCResource); !ok {
		panic(fmt.Sprintf("register tcc resource error, TCCResource is needed, param %v", resource))
	}
	t.resourceManagerMap.Store(resource.GetResourceId(), resource)
	return t.rmRemoting.RegisterResource(resource)
}

func (t *TCCResourceManager) GetManagedResources() sync.Map {
	return t.resourceManagerMap
}

// Commit a branch transaction
func (t *TCCResourceManager) BranchCommit(ctx context.Context, ranchType model.BranchType, xid string, branchID int64, resourceID string, applicationData []byte) (model.BranchStatus, error) {
	var tccResource *TCCResource
	if resource, ok := t.resourceManagerMap.Load(resourceID); !ok {
		err := fmt.Errorf("CC resource is not exist, resourceId: %s", resourceID)
		return 0, err
	} else {
		tccResource, _ = resource.(*TCCResource)
	}

	err := tccResource.TCCServiceBean.Commit(ctx, t.getBusinessActionContext(xid, branchID, resourceID, applicationData))
	if err != nil {
		return model.BranchStatusPhasetwoCommitFailedRetryable, err
	}
	return model.BranchStatusPhasetwoCommitted, err
}

func (t *TCCResourceManager) getBusinessActionContext(xid string, branchID int64, resourceID string, applicationData []byte) api.BusinessActionContext {
	return api.BusinessActionContext{
		Xid:        xid,
		BranchId:   string(branchID),
		ActionName: resourceID,
		// todo get ActionContext
		//ActionContext:,
	}
}

// Rollback a branch transaction
func (t *TCCResourceManager) BranchRollback(ctx context.Context, ranchType model.BranchType, xid string, branchID int64, resourceID string, applicationData []byte) (model.BranchStatus, error) {
	var tccResource *TCCResource
	if resource, ok := t.resourceManagerMap.Load(resourceID); !ok {
		err := fmt.Errorf("CC resource is not exist, resourceId: %s", resourceID)
		return 0, err
	} else {
		tccResource, _ = resource.(*TCCResource)
	}

	err := tccResource.TCCServiceBean.Rollback(ctx, t.getBusinessActionContext(xid, branchID, resourceID, applicationData))
	if err != nil {
		return model.BranchStatusPhasetwoRollbacked, err
	}
	return model.BranchStatusPhasetwoRollbackFailedRetryable, err
}

func (t *TCCResourceManager) GetBranchType() model.BranchType {
	return model.BranchTypeTCC
}
