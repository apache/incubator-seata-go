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
	"errors"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
)

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
