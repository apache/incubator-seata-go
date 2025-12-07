/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/enum"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	"reflect"
	"sync"
)

var (
	sagaResourceManager     *SagaResourceManager
	onceSagaReosurceManager = &sync.Once{}
)

func InitSaga() {

	rm.GetRmCacheInstance().RegisterResourceManager(GetSagaResourceManagerInstance())
}

func GetSagaResourceManagerInstance() *SagaResourceManager {
	if sagaResourceManager == nil {
		onceSagaReosurceManager.Do(func() {

			sagaResourceManager = &SagaResourceManager{
				resourceManagerMap: sync.Map{},
				rmRemoting:         rm.GetRMRemotingInstance(),
			}
		})
	}
	return sagaResourceManager
}

type SagaResource struct {
	ResourceGroupId string `default:"DEFAULT"`
	AppName         string
	actionName      string
	//一阶段提交的方法
	prepareMethod reflect.Method
	//与tcc事务不同saga事务只有回滚才需要二阶段
	rollbackMethod      reflect.Method
	rollbackArgsClasses reflect.Kind
	*rm.SagaAction
}

func ParseSagaResource(v interface{}) (*SagaResource, error) {
	t, err := rm.ParseSagaTwoPhaseAction(v)
	if err != nil {
		log.Errorf("%#v is not tcc two phase service, %s", v, err.Error())
		return nil, err
	}
	return &SagaResource{
		// todo read from config
		ResourceGroupId: `default:"DEFAULT"`,
		AppName:         "seata-go-mock-app-name",
		SagaAction:      t,
	}, nil
}

func (t *SagaResource) GetResourceId() string {
	return t.SagaAction.GetSagaActionName()
}

func (t *SagaResource) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}

func (saga *SagaResource) GetResourceGroupId() string {
	return saga.ResourceGroupId
}

func (saga *SagaResource) GetAppName() string {
	return saga.AppName
}

func (saga *SagaResource) ActionName() string {
	return saga.actionName
}

func (saga *SagaResource) SetActionName(actionName string) {
	saga.actionName = actionName
}

func (saga *SagaResource) PrepareMethod() reflect.Method {
	return saga.prepareMethod
}

func (saga *SagaResource) SetPrepareMethod(prepareMethod reflect.Method) {
	saga.prepareMethod = prepareMethod
}

func (saga *SagaResource) RollbackMethod() reflect.Method {
	return saga.rollbackMethod
}

func (saga *SagaResource) SetRollbackMethod(rollbackMethod reflect.Method) {
	saga.rollbackMethod = rollbackMethod
}

func (saga *SagaResource) RollbackArgsClasses() reflect.Kind {
	return saga.rollbackArgsClasses
}

func (saga *SagaResource) SetRollbackArgsClasses(rollbackArgsClasses reflect.Kind) {
	saga.rollbackArgsClasses = rollbackArgsClasses
}

//需要单独抽象一套出来 但是没法沿用tcc的接口
type SagaResourceManager struct {
	rmRemoting         *rm.RMRemoting
	resourceManagerMap sync.Map
}

func (saga *SagaResourceManager) getBusinessAction(xid string, branchID int64, resourceID string, applicationData []byte) *tm.BusinessActionContext {
	actionMap := make(map[string]interface{}, 2)
	if len(applicationData) > 0 {
		var sagaContext map[string]interface{}
		if err := json.Unmarshal(applicationData, &sagaContext); err != nil {
			fmt.Errorf("application data failed load %s", applicationData)
		}
		if v, ok := sagaContext[constant.ActionContext]; ok {
			actionMap = v.(map[string]interface{})
		}
	}

	return &tm.BusinessActionContext{
		Xid:           xid,
		BranchId:      branchID,
		ActionName:    resourceID,
		ActionContext: actionMap,
	}
}

func (saga *SagaResourceManager) BranchCommit(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	var sagaReSource *SagaResource
	if resource, ok := saga.resourceManagerMap.Load(branchResource.ResourceId); !ok {
		err := fmt.Errorf("TCC Resource is not exist, resourceId: %s", branchResource.ResourceId)
		return 0, err
	} else {
		sagaReSource, _ = resource.(*SagaResource)
	}

	//需要拿到此时action的名称
	businessContext := saga.getBusinessAction(branchResource.Xid, branchResource.BranchId, branchResource.ResourceId, branchResource.ApplicationData)
	//携带事务回滚，提交的状态
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, branchResource.Xid)
	//设置回滚事务时，防悬挂
	tm.SetFencePhase(ctx, enum.FencePhaseAction)
	tm.SetBusinessActionContext(ctx, businessContext)
	//getBusinessActionContext
	//一阶段直接提交
	_, err := sagaReSource.SagaAction.Action(ctx, businessContext)

	return branch.BranchStatusPhasetwoCommitted, err
}

func (saga *SagaResourceManager) BranchRollback(ctx context.Context, resource rm.BranchResource) (branch.BranchStatus, error) {

	var sagaResource *SagaResource
	if resource, ok := saga.resourceManagerMap.Load(resource.ResourceId); !ok {
		err := fmt.Errorf("xxxx")
		return 0, err
	} else {
		sagaResource, _ = resource.(*SagaResource)
	}

	businessActionContext := saga.getBusinessAction(resource.Xid, resource.BranchId, resource.ResourceId, resource.ApplicationData)

	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, resource.Xid)
	//设置回滚事务时，防悬挂
	tm.SetFencePhase(ctx, enum.FencePhaseCompensationAction)
	tm.SetBusinessActionContext(ctx, businessActionContext)
	_, err := sagaResource.SagaAction.Compensation(ctx, businessActionContext)
	return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
}

func (saga *SagaResourceManager) BranchRegister(ctx context.Context, param rm.BranchRegisterParam) (int64, error) {
	return saga.rmRemoting.BranchRegister(param)
}

// BranchReport branch transaction report the status
func (saga *SagaResourceManager) BranchReport(ctx context.Context, param rm.BranchReportParam) error {
	return nil
}

// LockQuery lock query boolean
func (saga *SagaResourceManager) LockQuery(ctx context.Context, param rm.LockQueryParam) (bool, error) {
	return false, nil
}

// GetCachedResources get all resources managed by this manager
func (saga *SagaResourceManager) GetCachedResources() *sync.Map {
	return nil
}

// RegisterResource register a Resource to be managed by Resource Manager
func (saga *SagaResourceManager) RegisterResource(res rm.Resource) error {
	return nil
}

// UnregisterResource unregister a Resource from the Resource Manager
func (saga *SagaResourceManager) UnregisterResource(res rm.Resource) error {
	return nil
}

func (saga *SagaResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeSAGA
}
