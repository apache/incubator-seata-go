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
	"fmt"
	"sync"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
)

var (
	// singletone ResourceManagerCache
	rmCacheInstance *ResourceManagerCache
	onceRMFacade    = &sync.Once{}
)

func GetRmCacheInstance() *ResourceManagerCache {
	if rmCacheInstance == nil {
		onceRMFacade.Do(func() {
			rmCacheInstance = &ResourceManagerCache{}
		})
	}
	return rmCacheInstance
}

type ResourceManagerCache struct {
	// BranchType -> ResourceManagerCache
	resourceManagerMap sync.Map
}

func (d *ResourceManagerCache) RegisterResourceManager(resourceManager ResourceManager) {
	d.resourceManagerMap.Store(resourceManager.GetBranchType(), resourceManager)
}

func (d *ResourceManagerCache) UnregisterResourceManager(branchType branch.BranchType) {
	d.resourceManagerMap.Delete(branchType)
}

func (d *ResourceManagerCache) GetResourceManager(branchType branch.BranchType) ResourceManager {
	rm, ok := d.resourceManagerMap.Load(branchType)
	if !ok {
		panic(fmt.Sprintf("No ResourceManagerCache for BranchType: %v", branchType))
	}
	return rm.(ResourceManager)
}

// RegisterCachedResources re-registers all resources with their resource managers.
// It is used after a remoting session is recreated because TC loses the old
// resource-to-session association when the connection closes.
func (d *ResourceManagerCache) RegisterCachedResources() error {
	var registrationErrs []error

	d.resourceManagerMap.Range(func(_, value interface{}) bool {
		resourceManager, ok := value.(ResourceManager)
		if !ok || resourceManager == nil {
			registrationErrs = append(registrationErrs, fmt.Errorf("invalid resource manager cache entry: %T", value))
			return true
		}

		resources := resourceManager.GetCachedResources()
		if resources == nil {
			return true
		}
		resources.Range(func(_, value interface{}) bool {
			resource, ok := value.(Resource)
			if !ok || resource == nil {
				registrationErrs = append(registrationErrs, fmt.Errorf("invalid cached resource: %T", value))
				return true
			}
			if err := resourceManager.RegisterResource(resource); err != nil {
				registrationErrs = append(registrationErrs, err)
			}
			return true
		})
		return true
	})

	return errors.Join(registrationErrs...)
}
