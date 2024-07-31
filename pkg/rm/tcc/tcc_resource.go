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

package tcc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"seata.apache.org/seata-go/pkg/constant"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	tCCResourceManager     *TCCResourceManager
	onceTCCResourceManager = &sync.Once{}
)

type TCCResource struct {
	ResourceGroupId string `default:"DEFAULT"`
	AppName         string
	*rm.TwoPhaseAction
}

func ParseTCCResource(v interface{}) (*TCCResource, error) {
	t, err := rm.ParseTwoPhaseAction(v)
	if err != nil {
		log.Errorf("%#v is not tcc two phase service, %s", v, err.Error())
		return nil, err
	}
	return &TCCResource{
		// todo read from config
		ResourceGroupId: `default:"DEFAULT"`,
		AppName:         "seata-go-mock-app-name",
		TwoPhaseAction:  t,
	}, nil
}

func (t *TCCResource) GetResourceGroupId() string {
	return t.ResourceGroupId
}

func (t *TCCResource) GetResourceId() string {
	return t.TwoPhaseAction.GetActionName()
}

func (t *TCCResource) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}

func InitTCC() {
	rm.GetRmCacheInstance().RegisterResourceManager(GetTCCResourceManagerInstance())
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

// BranchRegister register transaction branch
func (t *TCCResourceManager) BranchRegister(ctx context.Context, param rm.BranchRegisterParam) (int64, error) {
	return t.rmRemoting.BranchRegister(param)
}

// BranchReport report status of transaction branch
func (t *TCCResourceManager) BranchReport(ctx context.Context, param rm.BranchReportParam) error {
	return t.rmRemoting.BranchReport(param)
}

// LockQuery query lock status of transaction branch
func (t *TCCResourceManager) LockQuery(ctx context.Context, param rm.LockQueryParam) (bool, error) {
	// TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) UnregisterResource(resource rm.Resource) error {
	// TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) RegisterResource(resource rm.Resource) error {
	if _, ok := resource.(*TCCResource); !ok {
		panic(fmt.Sprintf("register tcc resource error, TCCResource is needed, param %v", resource))
	}
	t.resourceManagerMap.Store(resource.GetResourceId(), resource)
	return t.rmRemoting.RegisterResource(resource)
}

func (t *TCCResourceManager) GetCachedResources() *sync.Map {
	return &t.resourceManagerMap
}

// Commit a branch transaction
func (t *TCCResourceManager) BranchCommit(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	var tccResource *TCCResource
	if resource, ok := t.resourceManagerMap.Load(branchResource.ResourceId); !ok {
		err := fmt.Errorf("TCC resource is not exist, resourceId: %s", branchResource.ResourceId)
		return 0, err
	} else {
		tccResource, _ = resource.(*TCCResource)
	}

	businessActionContext := t.getBusinessActionContext(branchResource.Xid, branchResource.BranchId, branchResource.ResourceId, branchResource.ApplicationData)

	// to set up the fence phase
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, branchResource.Xid)
	tm.SetFencePhase(ctx, enum.FencePhaseCommit)
	tm.SetBusinessActionContext(ctx, businessActionContext)

	_, err := tccResource.TwoPhaseAction.Commit(ctx, businessActionContext)
	if err != nil {
		return branch.BranchStatusPhasetwoCommitFailedRetryable, err
	}
	return branch.BranchStatusPhasetwoCommitted, err
}

func (t *TCCResourceManager) getBusinessActionContext(xid string, branchID int64, resourceID string, applicationData []byte) *tm.BusinessActionContext {
	actionContextMap := make(map[string]interface{}, 2)
	if len(applicationData) > 0 {
		var tccContext map[string]interface{}
		if err := json.Unmarshal(applicationData, &tccContext); err != nil {
			panic("application data failed to unmarshl as json")
		}
		if v, ok := tccContext[constant.ActionContext]; ok {
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
func (t *TCCResourceManager) BranchRollback(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	var tccResource *TCCResource
	if resource, ok := t.resourceManagerMap.Load(branchResource.ResourceId); !ok {
		err := fmt.Errorf("CC resource is not exist, resourceId: %s", branchResource.ResourceId)
		return 0, err
	} else {
		tccResource, _ = resource.(*TCCResource)
	}

	businessActionContext := t.getBusinessActionContext(branchResource.Xid, branchResource.BranchId, branchResource.ResourceId, branchResource.ApplicationData)

	// to set up the fence phase
	ctx = tm.InitSeataContext(ctx)
	tm.SetXID(ctx, branchResource.Xid)
	tm.SetFencePhase(ctx, enum.FencePhaseRollback)
	tm.SetBusinessActionContext(ctx, businessActionContext)

	_, err := tccResource.TwoPhaseAction.Rollback(ctx, businessActionContext)
	if err != nil {
		return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
	}
	return branch.BranchStatusPhasetwoRollbacked, err
}

func (t *TCCResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}
