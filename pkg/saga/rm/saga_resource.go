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
	"fmt"

	"seata.apache.org/seata-go/pkg/protocol/branch"
)

type SagaResource struct {
	resourceGroupId string
	applicationId   string
}

func (r *SagaResource) GetResourceGroupId() string {
	return r.resourceGroupId
}

func (r *SagaResource) SetResourceGroupId(resourceGroupId string) {
	r.resourceGroupId = resourceGroupId
}

func (r *SagaResource) GetResourceId() string {
	return fmt.Sprintf("%s#%s", r.applicationId, r.resourceGroupId)
}

func (r *SagaResource) GetBranchType() branch.BranchType {
	return branch.BranchTypeSAGA
}

func (r *SagaResource) GetApplicationId() string {
	return r.applicationId
}

func (r *SagaResource) SetApplicationId(applicationId string) {
	r.applicationId = applicationId
}
