package rm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/dubbo/client/service"

	"github.com/stretchr/testify/assert"
)

var (
	testResourceManager     *TestResourceManager
	onceTestResourceManager = &sync.Once{}
)

func TestGetRMRemotingInstance(t *testing.T) {
	tests := []struct {
		name string
		want *RMRemoting
	}{
		{"test1", &RMRemoting{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetRMRemotingInstance(), "GetRMRemotingInstance()")
		})
	}
}

func TestGetRmCacheInstance(t *testing.T) {

	resourceMapState := sync.Map{}
	resourceMapState.Store(branch.BranchTypeTCC, GetTestResourceManagerInstance())

	GetRmCacheInstance().RegisterResourceManager(GetTestResourceManagerInstance())
	tests := struct {
		name string
		want *ResourceManagerCache
	}{"test1", &ResourceManagerCache{resourceManagerMap: resourceMapState}}

	t.Run(tests.name, func(t *testing.T) {

		excepted, _ := tests.want.resourceManagerMap.Load(branch.BranchTypeTCC)
		actual, _ := GetRmCacheInstance().resourceManagerMap.Load(branch.BranchTypeTCC)
		assert.Equalf(t, excepted, actual, "GetRmCacheInstance()")
	})

}

func TestIsTwoPhaseAction(t *testing.T) {

	userProvider := &TwoPhaseDemoService{}
	userProvider1 := &service.UserProviderInstance
	type args struct {
		v interface{}
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test1", args{userProvider}, true},
		{"test2", args{userProvider1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, IsTwoPhaseAction(tt.args.v), "IsTwoPhaseAction(%v)", tt.args.v)
		})
	}

}

func TestParseTwoPhaseAction(t *testing.T) {

	type args struct {
		v interface{}
	}

	userProvider := &TwoPhaseDemoService{}
	twoPhaseAction, _ := ParseTwoPhaseAction(userProvider)
	args1 := args{userProvider}

	tests := struct {
		name    string
		args    args
		want    *TwoPhaseAction
		wantErr assert.ErrorAssertionFunc
	}{"test1", args1, twoPhaseAction, assert.NoError}
	t.Run(tests.name, func(t *testing.T) {
		got, err := ParseTwoPhaseAction(tests.args.v)
		if !tests.wantErr(t, err, fmt.Sprintf("ParseTwoPhaseAction(%v)", tests.args.v)) {
			return
		}
		assert.Equalf(t, tests.want.GetTwoPhaseService(), got.GetTwoPhaseService(), "ParseTwoPhaseAction(%v)", tests.args.v)
	})

}

func TestParseTwoPhaseActionByInterface(t *testing.T) {
	type args struct {
		v interface{}
	}

	userProvider := &service.UserProvider{}
	twoPhaseAction, _ := ParseTwoPhaseAction(userProvider)
	args1 := args{userProvider}

	tests := struct {
		name    string
		args    args
		want    *TwoPhaseAction
		wantErr assert.ErrorAssertionFunc
	}{"test1", args1, twoPhaseAction, assert.NoError}

	t.Run(tests.name, func(t *testing.T) {
		got, err := ParseTwoPhaseActionByInterface(tests.args.v)
		if !tests.wantErr(t, err, fmt.Sprintf("ParseTwoPhaseActionByInterface(%v)", tests.args.v)) {
			return
		}
		assert.Equalf(t, tests.want, got, "ParseTwoPhaseActionByInterface(%v)", tests.args.v)
	})
}

func TestRMRemoting_BranchRegister(t *testing.T) {
	type args struct {
		branchType      branch.BranchType
		resourceId      string
		clientId        string
		xid             string
		applicationData string
		lockKeys        string
	}

	args1 := args{branch.BranchTypeTCC, "1232324", "56679", "123345", "TestExtraData", ""}

	tt := struct {
		name    string
		args    args
		want    int64
		wantErr assert.ErrorAssertionFunc
	}{"test1", args1, 56679, assert.NoError}

	t.Run(tt.name, func(t *testing.T) {
		rm := RMRemoting{}
		func() {
			listener, err := net.Listen("tcp", ":8091")
			if err != nil {
				t.Error(err)
			}

			go http.Serve(listener, nil)
			time.Sleep(10 * time.Second)

			got, err := rm.BranchRegister(tt.args.branchType, tt.args.resourceId, tt.args.clientId, tt.args.xid, tt.args.applicationData, tt.args.lockKeys)
			if !tt.wantErr(t, err, fmt.Sprintf("BranchRegister(%v, %v, %v, %v, %v, %v)", tt.args.branchType, tt.args.resourceId, tt.args.clientId, tt.args.xid, tt.args.applicationData, tt.args.lockKeys)) {
				return
			}
			assert.Equalf(t, tt.want, got, "BranchRegister(%v, %v, %v, %v, %v, %v)", tt.args.branchType, tt.args.resourceId, tt.args.clientId, tt.args.xid, tt.args.applicationData, tt.args.lockKeys)

			return
		}()

	})

}

type TestResource struct {
	ResourceGroupId string `default:"DEFAULT"`
	AppName         string
	*TwoPhaseAction
}

func (t *TestResource) GetResourceGroupId() string {
	return t.ResourceGroupId
}

func (t *TestResource) GetResourceId() string {
	return t.TwoPhaseAction.GetActionName()
}

func (t *TestResource) GetBranchType() branch.BranchType {
	return 3
}

type TwoPhaseDemoService struct {
}

func (t *TwoPhaseDemoService) Prepare(ctx context.Context, params ...interface{}) (bool, error) {
	return false, fmt.Errorf("execute two phase prepare method, param %v", params)
}

func (t *TwoPhaseDemoService) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return true, fmt.Errorf("execute two phase commit method, xid %v", businessActionContext.Xid)
}

func (t *TwoPhaseDemoService) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	return false, fmt.Errorf("execute two phase rollback method, xid %v", businessActionContext.Xid)
}

func (t *TwoPhaseDemoService) GetActionName() string {
	return "TwoPhaseDemoService"
}

func GetTestResourceManagerInstance() *TestResourceManager {
	if testResourceManager == nil {
		onceTestResourceManager.Do(func() {
			testResourceManager = &TestResourceManager{
				resourceManagerMap: sync.Map{},
				rmRemoting:         GetRMRemotingInstance(),
			}
		})
	}
	return testResourceManager
}

type TestResourceManager struct {
	rmRemoting *RMRemoting
	// resourceID -> resource
	resourceManagerMap sync.Map
}

// register transaction branch
func (t *TestResourceManager) BranchRegister(ctx context.Context, branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return t.rmRemoting.BranchRegister(3, resourceId, clientId, xid, applicationData, lockKeys)
}

func (t *TestResourceManager) BranchReport(ctx context.Context, ranchType branch.BranchType, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	//TODO implement me
	panic("implement me")
}

func (t *TestResourceManager) LockQuery(ctx context.Context, ranchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TestResourceManager) UnregisterResource(resource Resource) error {
	//TODO implement me
	panic("implement me")
}

func (t *TestResourceManager) RegisterResource(resource Resource) error {
	if _, ok := resource.(*TestResource); !ok {
		panic(fmt.Sprintf("register tcc resource error, TCCResource is needed, param %v", resource))
	}
	t.resourceManagerMap.Store(resource.GetResourceId(), resource)
	return t.rmRemoting.RegisterResource(resource)
}

func (t *TestResourceManager) GetCachedResources() *sync.Map {
	return &t.resourceManagerMap
}

// Commit a branch transaction
func (t *TestResourceManager) BranchCommit(ctx context.Context, ranchType branch.BranchType, xid string, branchID int64, resourceID string, applicationData []byte) (branch.BranchStatus, error) {
	var tccResource *TestResource
	if resource, ok := t.resourceManagerMap.Load(resourceID); !ok {
		err := fmt.Errorf("TCC resource is not exist, resourceId: %s", resourceID)
		return 0, err
	} else {
		tccResource, _ = resource.(*TestResource)
	}

	_, err := tccResource.TwoPhaseAction.Commit(ctx, t.getBusinessActionContext(xid, branchID, resourceID, applicationData))
	if err != nil {
		return branch.BranchStatusPhasetwoCommitFailedRetryable, err
	}
	return branch.BranchStatusPhasetwoCommitted, err
}

func (t *TestResourceManager) getBusinessActionContext(xid string, branchID int64, resourceID string, applicationData []byte) *tm.BusinessActionContext {
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
func (t *TestResourceManager) BranchRollback(ctx context.Context, ranchType branch.BranchType, xid string, branchID int64, resourceID string, applicationData []byte) (branch.BranchStatus, error) {
	var tccResource *TestResource
	if resource, ok := t.resourceManagerMap.Load(resourceID); !ok {
		err := fmt.Errorf("CC resource is not exist, resourceId: %s", resourceID)
		return 0, err
	} else {
		tccResource, _ = resource.(*TestResource)
	}

	_, err := tccResource.TwoPhaseAction.Rollback(ctx, t.getBusinessActionContext(xid, branchID, resourceID, applicationData))
	if err != nil {
		return branch.BranchStatusPhasetwoRollbacked, err
	}
	return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
}

func (t *TestResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}
