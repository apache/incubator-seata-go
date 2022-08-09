package rm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/stretchr/testify/assert"
	"reflect"
	"sync"
	"testing"
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
	type args struct {
		v interface{}
	}

	tests := struct {
		name string
		args args
		want bool
	}{
		"test1", args{userProvider}, true,
	}

	t.Run(tests.name, func(t *testing.T) {
		assert.Equalf(t, tests.want, IsTwoPhaseAction(tests.args.v), "IsTwoPhaseAction(%v)", tests.args.v)
	})

}

func TestParseTwoPhaseAction(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *TwoPhaseAction
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTwoPhaseAction(tt.args.v)
			if !tt.wantErr(t, err, fmt.Sprintf("ParseTwoPhaseAction(%v)", tt.args.v)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ParseTwoPhaseAction(%v)", tt.args.v)
		})
	}
}

func TestParseTwoPhaseActionByInterface(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *TwoPhaseAction
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTwoPhaseActionByInterface(tt.args.v)
			if !tt.wantErr(t, err, fmt.Sprintf("ParseTwoPhaseActionByInterface(%v)", tt.args.v)) {
				return
			}
			assert.Equalf(t, tt.want, got, "ParseTwoPhaseActionByInterface(%v)", tt.args.v)
		})
	}
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
	tests := []struct {
		name    string
		args    args
		want    int64
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RMRemoting{}
			got, err := rm.BranchRegister(tt.args.branchType, tt.args.resourceId, tt.args.clientId, tt.args.xid, tt.args.applicationData, tt.args.lockKeys)
			if !tt.wantErr(t, err, fmt.Sprintf("BranchRegister(%v, %v, %v, %v, %v, %v)", tt.args.branchType, tt.args.resourceId, tt.args.clientId, tt.args.xid, tt.args.applicationData, tt.args.lockKeys)) {
				return
			}
			assert.Equalf(t, tt.want, got, "BranchRegister(%v, %v, %v, %v, %v, %v)", tt.args.branchType, tt.args.resourceId, tt.args.clientId, tt.args.xid, tt.args.applicationData, tt.args.lockKeys)
		})
	}
}

func TestRMRemoting_BranchReport(t *testing.T) {
	type args struct {
		branchType      branch.BranchType
		xid             string
		branchId        int64
		status          branch.BranchStatus
		applicationData string
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RMRemoting{}
			tt.wantErr(t, rm.BranchReport(tt.args.branchType, tt.args.xid, tt.args.branchId, tt.args.status, tt.args.applicationData), fmt.Sprintf("BranchReport(%v, %v, %v, %v, %v)", tt.args.branchType, tt.args.xid, tt.args.branchId, tt.args.status, tt.args.applicationData))
		})
	}
}

func TestRMRemoting_LockQuery(t *testing.T) {
	type args struct {
		branchType branch.BranchType
		resourceId string
		xid        string
		lockKeys   string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RMRemoting{}
			got, err := rm.LockQuery(tt.args.branchType, tt.args.resourceId, tt.args.xid, tt.args.lockKeys)
			if !tt.wantErr(t, err, fmt.Sprintf("LockQuery(%v, %v, %v, %v)", tt.args.branchType, tt.args.resourceId, tt.args.xid, tt.args.lockKeys)) {
				return
			}
			assert.Equalf(t, tt.want, got, "LockQuery(%v, %v, %v, %v)", tt.args.branchType, tt.args.resourceId, tt.args.xid, tt.args.lockKeys)
		})
	}
}

func TestRMRemoting_RegisterResource(t *testing.T) {
	type args struct {
		resource Resource
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RMRemoting{}
			tt.wantErr(t, r.RegisterResource(tt.args.resource), fmt.Sprintf("RegisterResource(%v)", tt.args.resource))
		})
	}
}

func TestRMRemoting_onRegisterRMFailure(t *testing.T) {
	type args struct {
		response message.RegisterRMResponse
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RMRemoting{}
			r.onRegisterRMFailure(tt.args.response)
		})
	}
}

func TestRMRemoting_onRegisterRMSuccess(t *testing.T) {
	type args struct {
		response message.RegisterRMResponse
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RMRemoting{}
			r.onRegisterRMSuccess(tt.args.response)
		})
	}
}

func TestRMRemoting_onRegisterTMFailure(t *testing.T) {
	type args struct {
		response message.RegisterTMResponse
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RMRemoting{}
			r.onRegisterTMFailure(tt.args.response)
		})
	}
}

func TestRMRemoting_onRegisterTMSuccess(t *testing.T) {
	type args struct {
		response message.RegisterTMResponse
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RMRemoting{}
			r.onRegisterTMSuccess(tt.args.response)
		})
	}
}

func TestResourceManagerCache_GetResourceManager(t *testing.T) {
	type fields struct {
		resourceManagerMap sync.Map
	}
	type args struct {
		branchType branch.BranchType
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   ResourceManager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &ResourceManagerCache{
				resourceManagerMap: tt.fields.resourceManagerMap,
			}
			assert.Equalf(t, tt.want, d.GetResourceManager(tt.args.branchType), "GetResourceManager(%v)", tt.args.branchType)
		})
	}
}

func TestResourceManagerCache_RegisterResourceManager(t *testing.T) {
	type fields struct {
		resourceManagerMap sync.Map
	}
	type args struct {
		resourceManager ResourceManager
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &ResourceManagerCache{
				resourceManagerMap: tt.fields.resourceManagerMap,
			}
			d.RegisterResourceManager(tt.args.resourceManager)
		})
	}
}

func TestTwoPhaseAction_Commit(t1 *testing.T) {
	type fields struct {
		twoPhaseService    interface{}
		actionName         string
		prepareMethodName  string
		prepareMethod      *reflect.Value
		commitMethodName   string
		commitMethod       *reflect.Value
		rollbackMethodName string
		rollbackMethod     *reflect.Value
	}
	type args struct {
		ctx                   context.Context
		businessActionContext *tm.BusinessActionContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TwoPhaseAction{
				twoPhaseService:    tt.fields.twoPhaseService,
				actionName:         tt.fields.actionName,
				prepareMethodName:  tt.fields.prepareMethodName,
				prepareMethod:      tt.fields.prepareMethod,
				commitMethodName:   tt.fields.commitMethodName,
				commitMethod:       tt.fields.commitMethod,
				rollbackMethodName: tt.fields.rollbackMethodName,
				rollbackMethod:     tt.fields.rollbackMethod,
			}
			got, err := t.Commit(tt.args.ctx, tt.args.businessActionContext)
			if !tt.wantErr(t1, err, fmt.Sprintf("Commit(%v, %v)", tt.args.ctx, tt.args.businessActionContext)) {
				return
			}
			assert.Equalf(t1, tt.want, got, "Commit(%v, %v)", tt.args.ctx, tt.args.businessActionContext)
		})
	}
}

func TestTwoPhaseAction_GetActionName(t1 *testing.T) {
	type fields struct {
		twoPhaseService    interface{}
		actionName         string
		prepareMethodName  string
		prepareMethod      *reflect.Value
		commitMethodName   string
		commitMethod       *reflect.Value
		rollbackMethodName string
		rollbackMethod     *reflect.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TwoPhaseAction{
				twoPhaseService:    tt.fields.twoPhaseService,
				actionName:         tt.fields.actionName,
				prepareMethodName:  tt.fields.prepareMethodName,
				prepareMethod:      tt.fields.prepareMethod,
				commitMethodName:   tt.fields.commitMethodName,
				commitMethod:       tt.fields.commitMethod,
				rollbackMethodName: tt.fields.rollbackMethodName,
				rollbackMethod:     tt.fields.rollbackMethod,
			}
			assert.Equalf(t1, tt.want, t.GetActionName(), "GetActionName()")
		})
	}
}

func TestTwoPhaseAction_GetTwoPhaseService(t1 *testing.T) {
	type fields struct {
		twoPhaseService    interface{}
		actionName         string
		prepareMethodName  string
		prepareMethod      *reflect.Value
		commitMethodName   string
		commitMethod       *reflect.Value
		rollbackMethodName string
		rollbackMethod     *reflect.Value
	}
	tests := []struct {
		name   string
		fields fields
		want   interface{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TwoPhaseAction{
				twoPhaseService:    tt.fields.twoPhaseService,
				actionName:         tt.fields.actionName,
				prepareMethodName:  tt.fields.prepareMethodName,
				prepareMethod:      tt.fields.prepareMethod,
				commitMethodName:   tt.fields.commitMethodName,
				commitMethod:       tt.fields.commitMethod,
				rollbackMethodName: tt.fields.rollbackMethodName,
				rollbackMethod:     tt.fields.rollbackMethod,
			}
			assert.Equalf(t1, tt.want, t.GetTwoPhaseService(), "GetTwoPhaseService()")
		})
	}
}

func TestTwoPhaseAction_Prepare(t1 *testing.T) {
	type fields struct {
		twoPhaseService    interface{}
		actionName         string
		prepareMethodName  string
		prepareMethod      *reflect.Value
		commitMethodName   string
		commitMethod       *reflect.Value
		rollbackMethodName string
		rollbackMethod     *reflect.Value
	}
	type args struct {
		ctx    context.Context
		params []interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TwoPhaseAction{
				twoPhaseService:    tt.fields.twoPhaseService,
				actionName:         tt.fields.actionName,
				prepareMethodName:  tt.fields.prepareMethodName,
				prepareMethod:      tt.fields.prepareMethod,
				commitMethodName:   tt.fields.commitMethodName,
				commitMethod:       tt.fields.commitMethod,
				rollbackMethodName: tt.fields.rollbackMethodName,
				rollbackMethod:     tt.fields.rollbackMethod,
			}
			got, err := t.Prepare(tt.args.ctx, tt.args.params...)
			if !tt.wantErr(t1, err, fmt.Sprintf("Prepare(%v, %v)", tt.args.ctx, tt.args.params)) {
				return
			}
			assert.Equalf(t1, tt.want, got, "Prepare(%v, %v)", tt.args.ctx, tt.args.params)
		})
	}
}

func TestTwoPhaseAction_Rollback(t1 *testing.T) {
	type fields struct {
		twoPhaseService    interface{}
		actionName         string
		prepareMethodName  string
		prepareMethod      *reflect.Value
		commitMethodName   string
		commitMethod       *reflect.Value
		rollbackMethodName string
		rollbackMethod     *reflect.Value
	}
	type args struct {
		ctx                   context.Context
		businessActionContext *tm.BusinessActionContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TwoPhaseAction{
				twoPhaseService:    tt.fields.twoPhaseService,
				actionName:         tt.fields.actionName,
				prepareMethodName:  tt.fields.prepareMethodName,
				prepareMethod:      tt.fields.prepareMethod,
				commitMethodName:   tt.fields.commitMethodName,
				commitMethod:       tt.fields.commitMethod,
				rollbackMethodName: tt.fields.rollbackMethodName,
				rollbackMethod:     tt.fields.rollbackMethod,
			}
			got, err := t.Rollback(tt.args.ctx, tt.args.businessActionContext)
			if !tt.wantErr(t1, err, fmt.Sprintf("Rollback(%v, %v)", tt.args.ctx, tt.args.businessActionContext)) {
				return
			}
			assert.Equalf(t1, tt.want, got, "Rollback(%v, %v)", tt.args.ctx, tt.args.businessActionContext)
		})
	}
}

func Test_getActionName(t *testing.T) {
	type args struct {
		v interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getActionName(tt.args.v), "getActionName(%v)", tt.args.v)
		})
	}
}

func Test_getCommitMethod(t *testing.T) {
	type args struct {
		t reflect.StructField
		f reflect.Value
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 *reflect.Value
		want2 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := getCommitMethod(tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want, got, "getCommitMethod(%v, %v)", tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want1, got1, "getCommitMethod(%v, %v)", tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want2, got2, "getCommitMethod(%v, %v)", tt.args.t, tt.args.f)
		})
	}
}

func Test_getPrepareAction(t *testing.T) {
	type args struct {
		t reflect.StructField
		f reflect.Value
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 *reflect.Value
		want2 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := getPrepareAction(tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want, got, "getPrepareAction(%v, %v)", tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want1, got1, "getPrepareAction(%v, %v)", tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want2, got2, "getPrepareAction(%v, %v)", tt.args.t, tt.args.f)
		})
	}
}

func Test_getRollbackMethod(t *testing.T) {
	type args struct {
		t reflect.StructField
		f reflect.Value
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 *reflect.Value
		want2 bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2 := getRollbackMethod(tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want, got, "getRollbackMethod(%v, %v)", tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want1, got1, "getRollbackMethod(%v, %v)", tt.args.t, tt.args.f)
			assert.Equalf(t, tt.want2, got2, "getRollbackMethod(%v, %v)", tt.args.t, tt.args.f)
		})
	}
}

func Test_isRegisterSuccess(t *testing.T) {
	type args struct {
		response interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, isRegisterSuccess(tt.args.response), "isRegisterSuccess(%v)", tt.args.response)
		})
	}
}

func Test_isReportSuccess(t *testing.T) {
	type args struct {
		response interface{}
	}
	tests := []struct {
		name string
		args args
		want message.ResultCode
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, isReportSuccess(tt.args.response), "isReportSuccess(%v)", tt.args.response)
		})
	}
}

func Test_parseTwoPhaseActionByTwoPhaseInterface(t *testing.T) {
	type args struct {
		v TwoPhaseInterface
	}
	tests := []struct {
		name string
		args args
		want *TwoPhaseAction
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, parseTwoPhaseActionByTwoPhaseInterface(tt.args.v), "parseTwoPhaseActionByTwoPhaseInterface(%v)", tt.args.v)
		})
	}
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
