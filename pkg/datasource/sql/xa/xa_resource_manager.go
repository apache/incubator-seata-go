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
	"sync"
	"time"

	sql2 "github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/util/log"
)

type XAConfig struct {
	TwoPhaseHoldTime time.Duration `json:"two_phase_hold_time" yaml:"xa_two_phase_hold_time" koanf:"xa_two_phase_hold_time"`
}

func (cfg *XAConfig) RegisterFlagsWithPrefix(prefix string, f *flag.FlagSet) {
	f.DurationVar(&cfg.TwoPhaseHoldTime, prefix+".two_phase_hold_time", time.Millisecond*1000, "Undo log table name.")
}

func InitXA(config XAConfig) *ResourceManagerXA {
	xaSourceManager := &ResourceManagerXA{
		resourceCache: sync.Map{},
		basic:         datasource.NewBasicSourceManager(),
		rmRemoting:    rm.GetRMRemotingInstance(),
		config:        config,
	}

	rm.GetRmCacheInstance().RegisterResourceManager(xaSourceManager)

	go xaSourceManager.xaTwoPhaseTimeoutChecker()

	return xaSourceManager
}

type ResourceManagerXA struct {
	config        XAConfig
	resourceCache sync.Map
	basic         *datasource.BasicSourceManager
	rmRemoting    *rm.RMRemoting
}

func (xaManager *ResourceManagerXA) xaTwoPhaseTimeoutChecker() {
	var dbResource *sql2.DBResource
	xaManager.resourceCache.Range(func(key, value any) bool {
		if source, ok := value.(*sql2.DBResource); ok {
			dbResource = source
		}
		return false
	})

	if dbResource.IsShouldBeHeld() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-ticker.C:
				xaManager.resourceCache.Range(func(key, value any) bool {
					source, ok := value.(*sql2.DBResource)
					if !ok {
						return true
					}
					if source.IsShouldBeHeld() {
						return true
					}

					connectionXA, isConnectionXA := source.GetKeeper().(*ConnectionProxyXA)
					if !isConnectionXA {
						return true
					}

					if time.Now().Sub(connectionXA.GetPrepareTime()) > xaManager.config.TwoPhaseHoldTime {
						if err := connectionXA.CloseForce(); err != nil {
							log.Errorf("Force close the xa xid:%s physical connection fail", connectionXA.xid)
						}
					}
					return true
				})
			}
		}

	}

}

func (xaManager *ResourceManagerXA) GetBranchType() branch.BranchType {
	return branch.BranchTypeXA
}

func (xaManager *ResourceManagerXA) GetCachedResources() *sync.Map {
	return &xaManager.resourceCache
}

func (xaManager *ResourceManagerXA) RegisterResource(res rm.Resource) error {
	xaManager.resourceCache.Store(res.GetResourceId(), res)
	return xaManager.basic.RegisterResource(res)
}

func (xaManager *ResourceManagerXA) UnregisterResource(resource rm.Resource) error {
	return xaManager.basic.UnregisterResource(resource)
}

func (xaManager *ResourceManagerXA) xaIDBuilder(xid string, branchId int64) XAXid {
	return XaIdBuild(xid, branchId)
}

func (xaManager *ResourceManagerXA) finishBranch(ctx context.Context, xaID XAXid, branchResource rm.BranchResource) (*ConnectionProxyXA, error) {
	resource, ok := xaManager.resourceCache.Load(branchResource.ResourceId)
	if !ok {
		err := fmt.Errorf("unknow resource for rollback xa, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return nil, err
	}

	dbResource, ok := resource.(sql2.DBResource)
	if !ok {
		err := fmt.Errorf("unknow resource for rollback xa, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return nil, err
	}

	connectionProxyXA, err := dbResource.ConnectionForXA(ctx, xaID)
	if err != nil {
		err := fmt.Errorf("get connection for rollback xa, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return nil, err
	}

	return connectionProxyXA, nil
}

func (xaManager *ResourceManagerXA) BranchCommit(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	xaID := xaManager.xaIDBuilder(branchResource.Xid, branchResource.BranchId)
	connectionProxyXA, err := xaManager.finishBranch(ctx, xaID, branchResource)
	if err != nil {
		return branch.BranchStatusPhasetwoRollbackFailedUnretryable, err
	}

	if commitErr := connectionProxyXA.XaCommit(ctx, xaID.String(), branchResource.BranchId); commitErr != nil {
		err := fmt.Errorf("rollback xa, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return branch.BranchStatusPhasetwoCommitFailedUnretryable, err
	}

	log.Infof("%s was committed", xaID.String())
	return branch.BranchStatusPhasetwoCommitted, nil
}

func (xaManager *ResourceManagerXA) BranchRollback(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	xaID := xaManager.xaIDBuilder(branchResource.Xid, branchResource.BranchId)
	connectionProxyXA, err := xaManager.finishBranch(ctx, xaID, branchResource)
	if err != nil {
		return branch.BranchStatusPhasetwoRollbackFailedUnretryable, err
	}

	if rollbackErr := connectionProxyXA.XaRollbackByBranchId(ctx, xaID.String(), branchResource.BranchId); rollbackErr != nil {
		err := fmt.Errorf("rollback xa, resourceId: %s", branchResource.ResourceId)
		log.Errorf(err.Error())
		return branch.BranchStatusPhasetwoRollbackFailedUnretryable, err
	}

	log.Infof("%s was rollback", xaID.String())
	return branch.BranchStatusPhasetwoRollbacked, nil
}

func (xaManager *ResourceManagerXA) LockQuery(ctx context.Context, param rm.LockQueryParam) (bool, error) {
	return false, nil
}

func (xaManager *ResourceManagerXA) BranchRegister(ctx context.Context, req rm.BranchRegisterParam) (int64, error) {
	return xaManager.rmRemoting.BranchRegister(req)
}

func (xaManager *ResourceManagerXA) BranchReport(ctx context.Context, param rm.BranchReportParam) error {
	return xaManager.rmRemoting.BranchReport(param)
}

func (xaManager *ResourceManagerXA) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType, db *sql.DB) (datasource.TableMetaCache, error) {
	return xaManager.basic.CreateTableMetaCache(ctx, resID, dbType, db)
}
