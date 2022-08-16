package rm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/tm"
)

var (
	resourceManager         *testResourceManager
	onceTestResourceManager = &sync.Once{}
)

func TestGetRMRemotingInstance(t *testing.T) {
	tests := struct {
		name string
		want *RMRemoting
	}{"test1", &RMRemoting{}}

	t.Run(tests.name, func(t *testing.T) {
		assert.Equalf(t, tests.want, GetRMRemotingInstance(), "GetRMRemotingInstance()")
	})
}

func TestGetRmCacheInstance(t *testing.T) {
	tests := struct {
		name string
		want *ResourceManagerCache
	}{"test1", &ResourceManagerCache{}}

	t.Run(tests.name, func(t *testing.T) {
		GetRmCacheInstance().RegisterResourceManager(GetTestResourceManagerInstance())
		actual := GetRmCacheInstance().GetResourceManager(branch.BranchTypeTCC)
		assert.Equalf(t, GetTestResourceManagerInstance(), actual, "GetRmCacheInstance()")
	})

}

type testResource struct {
	ResourceGroupId string `default:"DEFAULT"`
	AppName         string
	*TwoPhaseAction
}

func (t *testResource) GetResourceGroupId() string {
	return t.ResourceGroupId
}

func (t *testResource) GetResourceId() string {
	return t.TwoPhaseAction.GetActionName()
}

func (t *testResource) GetBranchType() branch.BranchType {
	return 3
}

type twoPhaseDemoService struct {
}

func (t *twoPhaseDemoService) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	return false, fmt.Errorf("execute two phase prepare method, param %v", params)
}

func (t *twoPhaseDemoService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
}

func (t *twoPhaseDemoService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return false, fmt.Errorf("execute two phase rollback method, xid %v", businessActionContext.Xid)
}

func (t *twoPhaseDemoService) GetActionName() string {
	return "twoPhaseDemoService"
}

func GetTestResourceManagerInstance() *testResourceManager {
	if resourceManager == nil {
		onceTestResourceManager.Do(func() {
			resourceManager = &testResourceManager{
				resourceManagerMap: sync.Map{},
				rmRemoting:         GetRMRemotingInstance(),
			}
		})
	}
	return resourceManager
}

type testResourceManager struct {
	rmRemoting *RMRemoting
	// resourceID -> resource
	resourceManagerMap sync.Map
}

// BranchRegister register transaction branch
func (t *testResourceManager) BranchRegister(ctx context.Context, param BranchRegisterParam) (int64, error) {
	return t.rmRemoting.BranchRegister(param)
}

// BranchReport report status of transaction branch
func (t *testResourceManager) BranchReport(ctx context.Context, param BranchReportParam) error {
	return t.rmRemoting.BranchReport(param)
}

// LockQuery query lock status of transaction branch
func (t *testResourceManager) LockQuery(ctx context.Context, param LockQueryParam) (bool, error) {
	panic("implement me")
}

func (t *testResourceManager) UnregisterResource(resource Resource) error {
	panic("implement me")
}

func (t *testResourceManager) RegisterResource(resource Resource) error {
	if _, ok := resource.(*testResource); !ok {
		panic(fmt.Sprintf("register tcc resource error, TCCResource is needed, param %v", resource))
	}
	t.resourceManagerMap.Store(resource.GetResourceId(), resource)
	return t.rmRemoting.RegisterResource(resource)
}

func (t *testResourceManager) GetCachedResources() *sync.Map {
	return &t.resourceManagerMap
}

// Commit a branch transaction
func (t *testResourceManager) BranchCommit(ctx context.Context, resource BranchResource) (branch.BranchStatus, error) {
	var tccResource *testResource
	if resource, ok := t.resourceManagerMap.Load(resource.ResourceId); !ok {
		err := fmt.Errorf("TCC resource is not exist, resourceId: %s", resource)
		return 0, err
	} else {
		tccResource, _ = resource.(*testResource)
	}

	_, err := tccResource.TwoPhaseAction.Commit(ctx, t.getBusinessActionContext(resource.Xid, resource.BranchId, resource.ResourceId, resource.ApplicationData))
	if err != nil {
		return branch.BranchStatusPhasetwoCommitFailedRetryable, err
	}
	return branch.BranchStatusPhasetwoCommitted, err
}

func (t *testResourceManager) getBusinessActionContext(xid string, branchID int64, resourceID string, applicationData []byte) *tm.BusinessActionContext {
	var actionContextMap = make(map[string]interface{}, 2)
	if len(applicationData) > 0 {
		var tccContext map[string]interface{}
		if err := json.Unmarshal(applicationData, &tccContext); err != nil {
			panic("application data failed to unmarshl as json")
		}
		if v, ok := tccContext[common.ActionContext]; ok {
			actionContextMap = v.(map[string]interface{})
		}
	}

	return &tm.BusinessActionContext{
		Xid:           xid,
		BranchId:      branchID,
		ActionName:    resourceID,
		ActionContext: actionContextMap,
	}
}

// Rollback a branch transaction
func (t *testResourceManager) BranchRollback(ctx context.Context, resource BranchResource) (branch.BranchStatus, error) {
	var tccResource *testResource
	if resource, ok := t.resourceManagerMap.Load(resource.ResourceId); !ok {
		err := fmt.Errorf("CC resource is not exist, resourceId: %s", resource)
		return 0, err
	} else {
		tccResource, _ = resource.(*testResource)
	}

	_, err := tccResource.TwoPhaseAction.Rollback(ctx, t.getBusinessActionContext(resource.Xid, resource.BranchId, resource.ResourceId, resource.ApplicationData))
	if err != nil {
		return branch.BranchStatusPhasetwoRollbacked, err
	}
	return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
}

func (t *testResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}
