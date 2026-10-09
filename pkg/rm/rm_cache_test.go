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
	"errors"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
)

type nilResourceManager struct{}

func (*nilResourceManager) BranchCommit(_ context.Context, _ BranchResource) (branch.BranchStatus, error) {
	return branch.BranchStatusUnknown, nil
}

func (*nilResourceManager) BranchRollback(_ context.Context, _ BranchResource) (branch.BranchStatus, error) {
	return branch.BranchStatusUnknown, nil
}

func (*nilResourceManager) BranchRegister(_ context.Context, _ BranchRegisterParam) (int64, error) {
	return 0, nil
}

func (*nilResourceManager) BranchReport(_ context.Context, _ BranchReportParam) error { return nil }
func (*nilResourceManager) LockQuery(_ context.Context, _ LockQueryParam) (bool, error) {
	return false, nil
}
func (*nilResourceManager) RegisterResource(_ Resource) error   { panic("typed nil manager called") }
func (*nilResourceManager) UnregisterResource(_ Resource) error { return nil }
func (*nilResourceManager) GetCachedResources() *sync.Map       { panic("typed nil manager called") }
func (*nilResourceManager) GetBranchType() branch.BranchType    { return branch.BranchTypeTCC }

type nilResource struct{}

func (*nilResource) GetResourceGroupId() string       { return "" }
func (*nilResource) GetResourceId() string            { return "" }
func (*nilResource) GetBranchType() branch.BranchType { return branch.BranchTypeTCC }

func TestResourceManagerCache_RegisterCachedResources(t *testing.T) {
	ctl := gomock.NewController(t)
	resourceManager := NewMockResourceManager(ctl)
	resource1 := NewMockResource(ctl)
	resource2 := NewMockResource(ctl)
	resources := &sync.Map{}
	resources.Store("resource-1", resource1)
	resources.Store("resource-2", resource2)

	resourceManager.EXPECT().GetCachedResources().Return(resources)
	resourceManager.EXPECT().RegisterResource(resource1).Return(nil)
	resourceManager.EXPECT().RegisterResource(resource2).Return(nil)

	cache := &ResourceManagerCache{}
	cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)

	assert.NoError(t, cache.RegisterCachedResources())
}

func TestResourceManagerCache_RegisterCachedResourcesContinuesAfterError(t *testing.T) {
	ctl := gomock.NewController(t)
	resourceManager := NewMockResourceManager(ctl)
	resource1 := NewMockResource(ctl)
	resource2 := NewMockResource(ctl)
	resources := &sync.Map{}
	resources.Store("resource-1", resource1)
	resources.Store("resource-2", resource2)
	registrationErr := errors.New("registration failed")

	resourceManager.EXPECT().GetCachedResources().Return(resources)
	resourceManager.EXPECT().RegisterResource(resource1).Return(registrationErr)
	resourceManager.EXPECT().RegisterResource(resource2).Return(nil)

	cache := &ResourceManagerCache{}
	cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)

	assert.ErrorIs(t, cache.RegisterCachedResources(), registrationErr)
}

func TestResourceManagerCache_RegisterCachedResourcesInvalidEntries(t *testing.T) {
	t.Run("nil cached resources", func(t *testing.T) {
		ctl := gomock.NewController(t)
		resourceManager := NewMockResourceManager(ctl)
		resourceManager.EXPECT().GetCachedResources().Return(nil)

		cache := &ResourceManagerCache{}
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)
		assert.NoError(t, cache.RegisterCachedResources())
	})

	t.Run("invalid resource manager entry", func(t *testing.T) {
		cache := &ResourceManagerCache{}
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, "not a resource manager")
		assert.ErrorContains(t, cache.RegisterCachedResources(), "invalid resource manager cache entry")
	})

	t.Run("typed nil resource manager", func(t *testing.T) {
		cache := &ResourceManagerCache{}
		var resourceManager *nilResourceManager
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, ResourceManager(resourceManager))
		assert.ErrorContains(t, cache.RegisterCachedResources(), "invalid resource manager cache entry")
	})

	t.Run("invalid cached resource entry", func(t *testing.T) {
		ctl := gomock.NewController(t)
		resourceManager := NewMockResourceManager(ctl)
		resources := &sync.Map{}
		resources.Store("invalid", "not a resource")
		resourceManager.EXPECT().GetCachedResources().Return(resources)

		cache := &ResourceManagerCache{}
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)
		assert.ErrorContains(t, cache.RegisterCachedResources(), "invalid cached resource")
	})

	t.Run("typed nil cached resource", func(t *testing.T) {
		ctl := gomock.NewController(t)
		resourceManager := NewMockResourceManager(ctl)
		resources := &sync.Map{}
		var resource *nilResource
		resources.Store("nil", Resource(resource))
		resourceManager.EXPECT().GetCachedResources().Return(resources)

		cache := &ResourceManagerCache{}
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)
		assert.ErrorContains(t, cache.RegisterCachedResources(), "invalid cached resource")
	})
}

func TestResourceManagerCache_RegisterCachedResourcesMultipleManagers(t *testing.T) {
	ctl := gomock.NewController(t)
	resource1 := NewMockResource(ctl)
	resource2 := NewMockResource(ctl)
	resources1 := &sync.Map{}
	resources1.Store("resource-1", resource1)
	resources2 := &sync.Map{}
	resources2.Store("resource-2", resource2)

	manager1 := NewMockResourceManager(ctl)
	manager1.EXPECT().GetCachedResources().Return(resources1)
	manager1.EXPECT().RegisterResource(resource1).Return(nil)
	manager2 := NewMockResourceManager(ctl)
	manager2.EXPECT().GetCachedResources().Return(resources2)
	manager2.EXPECT().RegisterResource(resource2).Return(nil)

	cache := &ResourceManagerCache{}
	cache.resourceManagerMap.Store(branch.BranchTypeTCC, manager1)
	cache.resourceManagerMap.Store(branch.BranchTypeAT, manager2)
	assert.NoError(t, cache.RegisterCachedResources())
}

func TestGetRmCacheInstance(t *testing.T) {
	ctl := gomock.NewController(t)

	mockResourceManager := NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().GetBranchType().Return(branch.BranchTypeTCC)

	tests := struct {
		name string
		want *ResourceManagerCache
	}{"test1", &ResourceManagerCache{}}

	t.Run(tests.name, func(t *testing.T) {
		GetRmCacheInstance().RegisterResourceManager(mockResourceManager)
		actual := GetRmCacheInstance().GetResourceManager(branch.BranchTypeTCC)
		assert.Equalf(t, mockResourceManager, actual, "GetRmCacheInstance()")
	})
}

func TestResourceManagerCache_UnregisterResourceManager(t *testing.T) {
	ctl := gomock.NewController(t)

	mockResourceManager := NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().GetBranchType().Return(branch.BranchTypeSAGA).AnyTimes()

	cache := GetRmCacheInstance()
	cache.RegisterResourceManager(mockResourceManager)

	cache.UnregisterResourceManager(branch.BranchTypeSAGA)

	assert.Panics(t, func() {
		cache.GetResourceManager(branch.BranchTypeSAGA)
	})
}
