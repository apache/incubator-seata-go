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
	"sync"

	"seata.apache.org/seata-go/pkg/protocol/branch"
)

// Resource that can be managed by Resource Manager and involved into global transaction
type Resource interface {
	GetResourceGroupId() string
	GetResourceId() string
	GetBranchType() branch.BranchType
}

// BranchResource contains branch to commit or rollback
type BranchResource struct {
	BranchType      branch.BranchType
	Xid             string
	BranchId        int64
	ResourceId      string
	ApplicationData []byte
}

// ResourceManagerInbound Control a branch transaction commit or rollback
type ResourceManagerInbound interface {
	// BranchCommit commit a branch transaction
	BranchCommit(ctx context.Context, resource BranchResource) (branch.BranchStatus, error)
	// BranchRollback rollback a branch transaction
	BranchRollback(ctx context.Context, resource BranchResource) (branch.BranchStatus, error)
}

// BranchRegisterParam Branch register function param for ResourceManager
type BranchRegisterParam struct {
	BranchType      branch.BranchType
	ResourceId      string
	ClientId        string
	Xid             string
	ApplicationData string
	LockKeys        string
}

// BranchReportParam Branch report function param for ResourceManager
type BranchReportParam struct {
	BranchType      branch.BranchType
	Xid             string
	BranchId        int64
	Status          branch.BranchStatus
	ApplicationData string
}

// LockQueryParam Lock query function param for ResourceManager
type LockQueryParam struct {
	BranchType branch.BranchType
	ResourceId string
	Xid        string
	LockKeys   string
}

// ResourceManagerOutbound Resource Manager: send outbound request to TC
type ResourceManagerOutbound interface {
	// BranchRegister rm register the branch transaction
	BranchRegister(ctx context.Context, param BranchRegisterParam) (int64, error)
	// BranchReport branch transaction report the status
	BranchReport(ctx context.Context, param BranchReportParam) error
	// LockQuery lock query boolean
	LockQuery(ctx context.Context, param LockQueryParam) (bool, error)
}

// ResourceManager Resource Manager: common behaviors
type ResourceManager interface {
	ResourceManagerInbound
	ResourceManagerOutbound

	// RegisterResource register a resource to be managed by resource manager
	RegisterResource(resource Resource) error
	// UnregisterResource unregister a resource from the Resource Manager
	UnregisterResource(resource Resource) error
	// GetCachedResources get all resources managed by this manager
	GetCachedResources() *sync.Map
	// GetBranchType get the branch type
	GetBranchType() branch.BranchType
}

type ResourceManagerGetter interface {
	GetResourceManager(branchType branch.BranchType) ResourceManager
}
