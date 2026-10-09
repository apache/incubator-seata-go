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

package datasource

import (
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func GetDataSourceManager(branchType branch.BranchType) DataSourceManager {
	resourceManager := rm.GetRmCacheInstance().GetResourceManager(branchType)
	if resourceManager == nil {
		return nil
	}
	if d, ok := resourceManager.(DataSourceManager); ok {
		return d
	}
	return nil
}

type DataSourceManager interface {
	rm.ResourceManager
}

// BasicSourceManager the basic source manager for xa and at
type BasicSourceManager struct{}

func NewBasicSourceManager() *BasicSourceManager {
	return &BasicSourceManager{}
}

// RegisterResource register a model.Resource to be managed by model.Resource Manager
func (dm *BasicSourceManager) RegisterResource(resource rm.Resource) error {
	err := rm.GetRMRemotingInstance().RegisterResource(resource)
	if err != nil {
		return err
	}
	return nil
}

func (dm *BasicSourceManager) UnregisterResource(resource rm.Resource) error {
	return fmt.Errorf("unsupport unregister resource")
}

// TableMetaCache tables metadata cache, default is open
type TableMetaCache interface {
	types.TableMetaReader
	Destroy() error
}
