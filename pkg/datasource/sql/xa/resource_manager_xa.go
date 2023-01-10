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

package xa

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/seata/seata-go/pkg/util/log"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
)

type ResourceManagerXAConfig struct {
	TwoPhaseHoldTime time.Duration `json:"two_phase_hold_time" yaml:"xa_two_phase_hold_time" koanf:"xa_two_phase_hold_time"`
}

func (cfg *ResourceManagerXAConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.TwoPhaseHoldTime, prefix+".two_phase_hold_time", time.Millisecond*1000, "Undo log table name.")
}

type ResourceManagerXA struct {
	config        ResourceManagerXAConfig
	resourceCache sync.Map
	basic         *datasource.BasicSourceManager
	rmRemoting    *rm.RMRemoting
}

func NewXAResourceManager(config ResourceManagerXAConfig) *ResourceManagerXA {
	xaSourceManager := &ResourceManagerXA{
		resourceCache: sync.Map{},
		basic:         datasource.NewBasicSourceManager(),
		rmRemoting:    rm.GetRMRemotingInstance(),
	}

	rm.GetRmCacheInstance().RegisterResourceManager(xaSourceManager)

	go xaSourceManager.xaTwoPhaseTimeoutChecker()

	return xaSourceManager
}

func (x *ResourceManagerXA) xaTwoPhaseTimeoutChecker() {
}

func (x *ResourceManagerXA) GetBranchType() branch.BranchType {
	return branch.BranchTypeXA
}

func (x *ResourceManagerXA) GetCachedResources() *sync.Map {
	return &x.resourceCache
}

func (x *ResourceManagerXA) RegisterResource(res rm.Resource) error {
	x.resourceCache.Store(res.GetResourceId(), res)
	return x.basic.RegisterResource(res)
}

func (x *ResourceManagerXA) UnregisterResource(resource rm.Resource) error {
	return x.basic.UnregisterResource(resource)
}

func (x *ResourceManagerXA) xaIDBuilder(xid string, branchId int64) XAXid {
	return XaIdBuild(xid, branchId)
}

func (x *ResourceManagerXA) BranchCommit(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	xaID := x.xaIDBuilder(branchResource.Xid, branchResource.BranchId)
	resource, ok := x.resourceCache.Load(branchResource.ResourceId)
	if !ok {
		err := fmt.Errorf("unknow resource for xa, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return branch.BranchStatusPhasetwoRollbackFailedUnretryable, err
	}

	dbResource, ok := resource.(DbResourceXA)
	if !ok {
		err := fmt.Errorf("unknow resource for xa, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return branch.BranchStatusPhasetwoRollbackFailedUnretryable, err
	}

	return 0, nil
}

func (x *ResourceManagerXA) BranchRollback(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	return 0, nil
}

func (x *ResourceManagerXA) LockQuery(ctx context.Context, param rm.LockQueryParam) (bool, error) {
	return false, nil
}

func (x *ResourceManagerXA) BranchRegister(ctx context.Context, req rm.BranchRegisterParam) (int64, error) {
	return x.rmRemoting.BranchRegister(req)
}

func (x *ResourceManagerXA) BranchReport(ctx context.Context, param rm.BranchReportParam) error {
	return x.rmRemoting.BranchReport(param)
}

func (x *ResourceManagerXA) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType,
	db *sql.DB) (datasource.TableMetaCache, error) {
	return x.basic.CreateTableMetaCache(ctx, resID, dbType, db)
}
