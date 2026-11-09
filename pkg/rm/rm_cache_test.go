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
	"seata.apache.org/seata-go/v2/pkg/protocol/connection"
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

type testConnection struct{}

func (testConnection) RemoteAddr() string { return "127.0.0.1:8091" }
func (testConnection) IsClosed() bool     { return false }

type fakeConnectionAwareRMRemoting struct {
	registered map[Resource]connection.Connection
	errs       map[Resource]error
}

func (f *fakeConnectionAwareRMRemoting) BranchRegister(_ BranchRegisterParam) (int64, error) {
	return 0, nil
}

func (f *fakeConnectionAwareRMRemoting) BranchReport(_ BranchReportParam) error { return nil }
func (f *fakeConnectionAwareRMRemoting) LockQuery(_ LockQueryParam) (bool, error) {
	return false, nil
}
func (f *fakeConnectionAwareRMRemoting) RegisterResource(_ Resource) error { return nil }
func (f *fakeConnectionAwareRMRemoting) RegisterResourceOnConnection(
	resource Resource, conn connection.Connection) error {
	f.registered[resource] = conn
	return f.errs[resource]
}

func useRMRemotingForTest(t *testing.T, remoting RMRemoting) {
	t.Helper()
	previous := rmRemoting
	rmRemoting = remoting
	t.Cleanup(func() {
		rmRemoting = previous
	})
}

func TestResourceManagerCache_RegisterCachedResources(t *testing.T) {
	ctl := gomock.NewController(t)
	resourceManager := NewMockResourceManager(ctl)
	resource1 := NewMockResource(ctl)
	resource2 := NewMockResource(ctl)
	resources := &sync.Map{}
	resources.Store("resource-1", resource1)
	resources.Store("resource-2", resource2)

	resourceManager.EXPECT().GetCachedResources().Return(resources)
	conn := testConnection{}
	remoting := &fakeConnectionAwareRMRemoting{registered: make(map[Resource]connection.Connection)}
	useRMRemotingForTest(t, remoting)

	cache := &ResourceManagerCache{}
	cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)

	assert.NoError(t, cache.RegisterCachedResources(conn))
	assert.Equal(t, conn, remoting.registered[resource1])
	assert.Equal(t, conn, remoting.registered[resource2])
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
	conn := testConnection{}
	remoting := &fakeConnectionAwareRMRemoting{
		registered: make(map[Resource]connection.Connection),
		errs:       map[Resource]error{resource1: registrationErr},
	}
	useRMRemotingForTest(t, remoting)

	cache := &ResourceManagerCache{}
	cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)

	assert.ErrorIs(t, cache.RegisterCachedResources(conn), registrationErr)
}

func TestResourceManagerCache_RegisterCachedResourcesInvalidEntries(t *testing.T) {
	useRMRemotingForTest(t, &fakeConnectionAwareRMRemoting{
		registered: make(map[Resource]connection.Connection),
	})

	t.Run("nil cached resources", func(t *testing.T) {
		ctl := gomock.NewController(t)
		resourceManager := NewMockResourceManager(ctl)
		resourceManager.EXPECT().GetCachedResources().Return(nil)

		cache := &ResourceManagerCache{}
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)
		assert.NoError(t, cache.RegisterCachedResources(testConnection{}))
	})

	t.Run("invalid resource manager entry", func(t *testing.T) {
		cache := &ResourceManagerCache{}
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, "not a resource manager")
		assert.ErrorContains(t, cache.RegisterCachedResources(testConnection{}), "invalid resource manager cache entry")
	})

	t.Run("typed nil resource manager", func(t *testing.T) {
		cache := &ResourceManagerCache{}
		var resourceManager *nilResourceManager
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, ResourceManager(resourceManager))
		assert.ErrorContains(t, cache.RegisterCachedResources(testConnection{}), "invalid resource manager cache entry")
	})

	t.Run("invalid cached resource entry", func(t *testing.T) {
		ctl := gomock.NewController(t)
		resourceManager := NewMockResourceManager(ctl)
		resources := &sync.Map{}
		resources.Store("invalid", "not a resource")
		resourceManager.EXPECT().GetCachedResources().Return(resources)

		cache := &ResourceManagerCache{}
		cache.resourceManagerMap.Store(branch.BranchTypeTCC, resourceManager)
		assert.ErrorContains(t, cache.RegisterCachedResources(testConnection{}), "invalid cached resource")
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
		assert.ErrorContains(t, cache.RegisterCachedResources(testConnection{}), "invalid cached resource")
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
	conn := testConnection{}
	manager2 := NewMockResourceManager(ctl)
	manager2.EXPECT().GetCachedResources().Return(resources2)
	remoting := &fakeConnectionAwareRMRemoting{registered: make(map[Resource]connection.Connection)}
	useRMRemotingForTest(t, remoting)

	cache := &ResourceManagerCache{}
	cache.resourceManagerMap.Store(branch.BranchTypeTCC, manager1)
	cache.resourceManagerMap.Store(branch.BranchTypeAT, manager2)
	assert.NoError(t, cache.RegisterCachedResources(conn))
	assert.Equal(t, conn, remoting.registered[resource1])
	assert.Equal(t, conn, remoting.registered[resource2])
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
