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
	"fmt"
	"sync"
)

import (
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/protocol/resource"
	"github.com/seata/seata-go/pkg/remoting/getty"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/rm/common/remoting"
	"github.com/seata/seata-go/pkg/rm/tcc/api"
)

var (
	tCCResourceManager     *TCCResourceManager
	onceTCCResourceManager = &sync.Once{}
)

type TCCResource struct {
	TCCServiceBean  TCCService
	ResourceGroupId string `default:"DEFAULT"`
	AppName         string
	ActionName      string
}

func (t *TCCResource) GetResourceGroupId() string {
	return t.ResourceGroupId
}

func (t *TCCResource) GetResourceId() string {
	return t.ActionName
}

func (t *TCCResource) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}

func init() {
	rm.RegisterResourceManager(GetTCCResourceManagerInstance())
}

func GetTCCResourceManagerInstance() *TCCResourceManager {
	if tCCResourceManager == nil {
		onceTCCResourceManager.Do(func() {
			tCCResourceManager = &TCCResourceManager{
				resourceManagerMap: sync.Map{},
				rmRemoting:         remoting.GetRMRemotingInstance(),
			}
		})
	}
	return tCCResourceManager
}

type TCCResourceManager struct {
	rmRemoting *remoting.RMRemoting
	// resourceID -> resource
	resourceManagerMap sync.Map
}

// register transaction branch
func (t *TCCResourceManager) BranchRegister(ctx context.Context, branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	request := message.BranchRegisterRequest{
		Xid:             xid,
		BranchType:      t.GetBranchType(),
		ResourceId:      resourceId,
		LockKey:         lockKeys,
		ApplicationData: []byte(applicationData),
	}
	resp, err := getty.GetGettyRemotingClient().SendSyncRequest(request)
	if err != nil || resp == nil {
		log.Errorf("BranchRegister error: %v, res %v", err.Error(), resp)
		return 0, err
	}
	return resp.(message.BranchRegisterResponse).BranchId, nil
}

func (t *TCCResourceManager) BranchReport(ctx context.Context, ranchType branch.BranchType, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	//TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) LockQuery(ctx context.Context, ranchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) UnregisterResource(resource resource.Resource) error {
	//TODO implement me
	panic("implement me")
}

func (t *TCCResourceManager) RegisterResource(resource resource.Resource) error {
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
func (t *TCCResourceManager) BranchCommit(ctx context.Context, ranchType branch.BranchType, xid string, branchID int64, resourceID string, applicationData []byte) (branch.BranchStatus, error) {
	var tccResource *TCCResource
	if resource, ok := t.resourceManagerMap.Load(resourceID); !ok {
		err := fmt.Errorf("TCC resource is not exist, resourceId: %s", resourceID)
		return 0, err
	} else {
		tccResource, _ = resource.(*TCCResource)
	}

	err := tccResource.TCCServiceBean.Commit(ctx, t.getBusinessActionContext(xid, branchID, resourceID, applicationData))
	if err != nil {
		return branch.BranchStatusPhasetwoCommitFailedRetryable, err
	}
	return branch.BranchStatusPhasetwoCommitted, err
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
func (t *TCCResourceManager) BranchRollback(ctx context.Context, ranchType branch.BranchType, xid string, branchID int64, resourceID string, applicationData []byte) (branch.BranchStatus, error) {
	var tccResource *TCCResource
	if resource, ok := t.resourceManagerMap.Load(resourceID); !ok {
		err := fmt.Errorf("CC resource is not exist, resourceId: %s", resourceID)
		return 0, err
	} else {
		tccResource, _ = resource.(*TCCResource)
	}

	err := tccResource.TCCServiceBean.Rollback(ctx, t.getBusinessActionContext(xid, branchID, resourceID, applicationData))
	if err != nil {
		return branch.BranchStatusPhasetwoRollbacked, err
	}
	return branch.BranchStatusPhasetwoRollbackFailedRetryable, err
}

func (t *TCCResourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeTCC
}
