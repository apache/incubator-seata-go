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
	"database/sql"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	serr "seata.apache.org/seata-go/pkg/util/errors"
)

func InitAT(cfg undo.Config, asyncCfg AsyncWorkerConfig) {
	atSourceManager := &ATSourceManager{
		resourceCache: sync.Map{},
		basic:         datasource.NewBasicSourceManager(),
		rmRemoting:    rm.GetRMRemotingInstance(),
	}

	undo.InitUndoConfig(cfg)
	atSourceManager.worker = NewAsyncWorker(prometheus.DefaultRegisterer, asyncCfg, atSourceManager)
	rm.GetRmCacheInstance().RegisterResourceManager(atSourceManager)
}

type ATSourceManager struct {
	resourceCache sync.Map
	worker        *AsyncWorker
	basic         *datasource.BasicSourceManager
	rmRemoting    *rm.RMRemoting
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
	a.resourceCache.Store(res.GetResourceId(), res)
	return a.basic.RegisterResource(res)
}

// UnregisterResource unregister a Resource from the Resource Manager
func (a *ATSourceManager) UnregisterResource(res rm.Resource) error {
	return a.basic.UnregisterResource(res)
}

// BranchRollback rollback a branch transaction
func (a *ATSourceManager) BranchRollback(ctx context.Context, branchResource rm.BranchResource) (branch.BranchStatus, error) {
	var dbResource *DBResource
	if resource, ok := a.resourceCache.Load(branchResource.ResourceId); !ok {
		err := fmt.Errorf("DB resource is not exist, resourceId: %s", branchResource.ResourceId)
		return branch.BranchStatusUnknown, err
	} else {
		dbResource, _ = resource.(*DBResource)
	}

	undoMgr, err := undo.GetUndoLogManager(dbResource.dbType)
	if err != nil {
		return branch.BranchStatusUnknown, err
	}

	if err := undoMgr.RunUndo(ctx, branchResource.Xid, branchResource.BranchId, dbResource.db, dbResource.dbName); err != nil {
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

func (a *ATSourceManager) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType,
	db *sql.DB) (datasource.TableMetaCache, error) {
	return a.basic.CreateTableMetaCache(ctx, resID, dbType, db)
}
