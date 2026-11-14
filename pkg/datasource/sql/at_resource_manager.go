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

package sql

import (
	"context"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	serr "seata.apache.org/seata-go/v2/pkg/util/errors"
)

func InitAT(cfg undo.Config, asyncCfg AsyncWorkerConfig) {
	atSourceManager := &ATSourceManager{
		resourceCache: sync.Map{},
		basic:         datasource.NewBasicSourceManager(),
		rmRemoting:    rm.GetRMRemotingInstance(),
	}

	undo.InitUndoConfig(cfg)
	atSourceManager.worker = NewAsyncWorker(context.Background(), prometheus.DefaultRegisterer, asyncCfg, atSourceManager)
	rm.GetRmCacheInstance().RegisterResourceManager(atSourceManager)
}

type ATSourceManager struct {
	resourceCache sync.Map
	worker        *AsyncWorker
	basic         *datasource.BasicSourceManager
	rmRemoting    rm.RMRemoting

	registrationMu sync.Mutex
	replayEpoch    uint64
}

func (a *ATSourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeAT
}

// GetCachedResources get all resources managed by this manager
func (a *ATSourceManager) GetCachedResources() *sync.Map {
	return &a.resourceCache
}

// RegisterResource register a Resource to be managed by Resource Manager
func (a *ATSourceManager) RegisterResource(res rm.Resource) error {
	const maxRegistrationAttempts = 3

	a.registrationMu.Lock()
	epoch := a.replayEpoch
	a.registrationMu.Unlock()

	for attempt := 0; attempt < maxRegistrationAttempts; attempt++ {
		if err := a.rmRemoting.RegisterResource(res); err != nil {
			return err
		}
		a.registrationMu.Lock()
		if epoch == a.replayEpoch {
			a.resourceCache.Store(res.GetResourceId(), res)
			a.registrationMu.Unlock()
			return nil
		}
		// A reconnect may have scanned the cache before this resource was published.
		epoch = a.replayEpoch
		a.registrationMu.Unlock()
	}
	return fmt.Errorf("AT resource registration interrupted by reconnect after %d attempts", maxRegistrationAttempts)
}

// PrepareResourceReplay marks the start of reconnect replay before the cache scan.
// The returned function only registers with the TC because cached resources
// may be replaced while replay is in progress.
func (a *ATSourceManager) PrepareResourceReplay() func(rm.Resource) error {
	a.registrationMu.Lock()
	a.replayEpoch++
	a.registrationMu.Unlock()
	return a.rmRemoting.RegisterResource
}

// UnregisterResource unregister a Resource from the Resource Manager
func (a *ATSourceManager) UnregisterResource(res rm.Resource) error {
	return a.basic.UnregisterResource(res)
}

// BranchRollback rollback a branch transaction
func (a *ATSourceManager) BranchRollback(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	resource, ok := a.resourceCache.Load(branchResource.ResourceId)
	if !ok {
		err := fmt.Errorf("DB resource is not exist, resourceId: %s", branchResource.ResourceId)
		return branch.BranchStatusUnknown, err
	}
	dbResource, ok := resource.(*DBResource)
	if !ok || dbResource == nil {
		return branch.BranchStatusUnknown, fmt.Errorf("invalid AT resource, resourceId: %s", branchResource.ResourceId)
	}
	if dbResource.undoLogMgr == nil {
		return branch.BranchStatusUnknown, fmt.Errorf("no Undo Log Manager for AT resource, resourceId: %s", branchResource.ResourceId)
	}

	if err := dbResource.undoLogMgr.RunUndo(ctx, branchResource.Xid, branchResource.BranchId, dbResource.db, dbResource.dbName, dbResource.metaCache); err != nil {
		transErr, ok := err.(*serr.SeataError)
		if !ok {
			return branch.BranchStatusPhaseoneFailed, err
		}

		if transErr.Code == serr.TransactionErrorCodeBranchRollbackFailedUnretriable {
			return branch.BranchStatusPhasetwoRollbackFailedUnretryable, nil
		}

		return branch.BranchStatusPhasetwoRollbackFailedRetryable, nil
	}

	return branch.BranchStatusPhasetwoRollbacked, nil
}

// BranchCommit commit the branch transaction
func (a *ATSourceManager) BranchCommit(ctx context.Context, resource rm.BranchResource) (branch.BranchStatus, error) {
	a.worker.BranchCommit(ctx, resource)
	return branch.BranchStatusPhasetwoCommitted, nil
}

func (a *ATSourceManager) LockQuery(ctx context.Context, param rm.LockQueryParam) (bool, error) {
	return a.rmRemoting.LockQuery(param)
}

// BranchRegister branch transaction register
func (a *ATSourceManager) BranchRegister(ctx context.Context, req rm.BranchRegisterParam) (int64, error) {
	return a.rmRemoting.BranchRegister(req)
}

// BranchReport Report status of transaction branch
func (a *ATSourceManager) BranchReport(ctx context.Context, param rm.BranchReportParam) error {
	return a.rmRemoting.BranchReport(param)
}
