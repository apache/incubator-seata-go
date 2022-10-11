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
	"flag"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
)

const (
	_defaultResourceSize    = 16
	_undoLogDeleteLimitSize = 1000
)

func init() {
	atSourceManager := &ATSourceManager{
		resourceCache: sync.Map{},
		basic:         datasource.NewBasicSourceManager(),
	}

	fs := flag.NewFlagSet("", flag.PanicOnError)
	asyncWorkerConf := AsyncWorkerConfig{}
	asyncWorkerConf.RegisterFlags(fs)
	_ = fs.Parse([]string{})

	atSourceManager.worker = NewAsyncWorker(prometheus.DefaultRegisterer, asyncWorkerConf, atSourceManager)
	datasource.RegisterResourceManager(branch.BranchTypeAT, atSourceManager)
}

type ATSourceManager struct {
	resourceCache sync.Map
	worker        *AsyncWorker
	basic         *datasource.BasicSourceManager
}

// Register a Resource to be managed by Resource Manager
func (mgr *ATSourceManager) RegisterResource(res rm.Resource) error {
	mgr.resourceCache.Store(res.GetResourceId(), res)

	return mgr.basic.RegisterResource(res)
}

// Unregister a Resource from the Resource Manager
func (mgr *ATSourceManager) UnregisterResource(res rm.Resource) error {
	return mgr.basic.UnregisterResource(res)
}

// Get all resources managed by this manager
func (mgr *ATSourceManager) GetManagedResources() map[string]rm.Resource {
	ret := make(map[string]rm.Resource)

	mgr.resourceCache.Range(func(key, value interface{}) bool {
		ret[key.(string)] = value.(rm.Resource)
		return true
	})

	return ret
}

// BranchRollback Rollback the corresponding transactions according to the request
func (mgr *ATSourceManager) BranchRollback(ctx context.Context, req message.BranchRollbackRequest) (branch.BranchStatus, error) {
	val, ok := mgr.resourceCache.Load(req.ResourceId)

	if !ok {
		return branch.BranchStatusPhaseoneFailed, fmt.Errorf("resource %s not found", req.ResourceId)
	}

	res := val.(*DBResource)

	undoMgr, err := undo.GetUndoLogManager(res.dbType)
	if err != nil {
		return branch.BranchStatusUnknown, err
	}

	conn, err := res.target.Conn(ctx)
	if err != nil {
		return branch.BranchStatusUnknown, err
	}

	if err := undoMgr.RunUndo(req.Xid, req.BranchId, conn); err != nil {
		transErr, ok := err.(*types.TransactionError)
		if !ok {
			return branch.BranchStatusPhaseoneFailed, err
		}

		if transErr.Code() == types.ErrorCodeBranchRollbackFailedUnretriable {
			return branch.BranchStatusPhasetwoRollbackFailedUnretryable, nil
		}

		return branch.BranchStatusPhasetwoRollbackFailedRetryable, nil
	}

	return branch.BranchStatusPhasetwoRollbacked, nil
}

// BranchCommit
func (mgr *ATSourceManager) BranchCommit(ctx context.Context, req message.BranchCommitRequest) (branch.BranchStatus, error) {
	mgr.worker.BranchCommit(ctx, req)
	return branch.BranchStatusPhaseoneDone, nil
}

// LockQuery
func (mgr *ATSourceManager) LockQuery(ctx context.Context, req message.GlobalLockQueryRequest) (bool, error) {
	return false, nil
}

// BranchRegister
func (mgr *ATSourceManager) BranchRegister(ctx context.Context, clientId string, req message.BranchRegisterRequest) (int64, error) {
	return 0, nil
}

// BranchReport
func (mgr *ATSourceManager) BranchReport(ctx context.Context, req message.BranchReportRequest) error {
	return nil
}

// CreateTableMetaCache
func (mgr *ATSourceManager) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType,
	db *sql.DB) (datasource.TableMetaCache, error) {
	return mgr.basic.CreateTableMetaCache(ctx, resID, dbType, db)
}
