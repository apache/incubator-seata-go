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

package rm

import (
	"context"
	"fmt"
	"sync"

	"github.com/seata/seata-go/pkg/protocol/resource"

	"github.com/seata/seata-go/pkg/protocol/branch"
)

var (
	// singletone ResourceManager
	rmFacadeInstance *ResourceManager
	onceRMFacade     = &sync.Once{}
)

func GetResourceManagerInstance() *ResourceManager {
	if rmFacadeInstance == nil {
		onceRMFacade.Do(func() {
			rmFacadeInstance = &ResourceManager{}
		})
	}
	return rmFacadeInstance
}

type ResourceManager struct {
	// BranchType -> ResourceManager
	resourceManagerMap sync.Map
}

func (d *ResourceManager) RegisterResourceManager(resourceManager resource.ResourceManager) {
	d.resourceManagerMap.Store(resourceManager.GetBranchType(), resourceManager)
}

func (d *ResourceManager) GetResourceManager(branchType branch.BranchType) resource.ResourceManager {
	rm, ok := d.resourceManagerMap.Load(branchType)
	if !ok {
		panic(fmt.Sprintf("No ResourceManager for BranchType: %v", branchType))
	}
	return rm.(resource.ResourceManager)
}

// Commit a branch transaction
func (d *ResourceManager) BranchCommit(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (branch.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchCommit(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Rollback a branch transaction
func (d *ResourceManager) BranchRollback(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (branch.BranchStatus, error) {
	return d.GetResourceManager(branchType).BranchRollback(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Branch register long
func (d *ResourceManager) BranchRegister(ctx context.Context, branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return d.GetResourceManager(branchType).BranchRegister(ctx, branchType, resourceId, clientId, xid, applicationData, lockKeys)
}

//  Branch report
func (d *ResourceManager) BranchReport(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	return d.GetResourceManager(branchType).BranchReport(ctx, branchType, xid, branchId, status, applicationData)
}

// Lock query boolean
func (d *ResourceManager) LockQuery(ctx context.Context, branchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return d.GetResourceManager(branchType).LockQuery(ctx, branchType, resourceId, xid, lockKeys)
}

// Register a   model.Resource to be managed by   model.Resource Manager
func (d *ResourceManager) RegisterResource(resource resource.Resource) error {
	return d.GetResourceManager(resource.GetBranchType()).RegisterResource(resource)
}

//  Unregister a   model.Resource from the   model.Resource Manager
func (d *ResourceManager) UnregisterResource(resource resource.Resource) error {
	return d.GetResourceManager(resource.GetBranchType()).UnregisterResource(resource)
}

// Get all resources managed by this manager
func (d *ResourceManager) GetManagedResources() *sync.Map {
	return &d.resourceManagerMap
}

// Get the model.BranchType
func (d *ResourceManager) GetBranchType() branch.BranchType {
	panic("DefaultResourceManager isn't a real ResourceManager")
}
