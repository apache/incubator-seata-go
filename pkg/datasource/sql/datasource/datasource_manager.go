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
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
)

var (
	atOnce sync.Once
	atMgr  DataSourceManager
	xaMgr  DataSourceManager
	solts  = map[types.DBType]func() TableMetaCache{}
)

// RegisterTableCache
func RegisterTableCache(dbType types.DBType, builder func() TableMetaCache) {
	solts[dbType] = builder
}

func RegisterResourceManager(b branch.BranchType, d DataSourceManager) {
	if b == branch.BranchTypeAT {
		atMgr = d
	}

	if b == branch.BranchTypeXA {
		xaMgr = d
	}
}

func GetDataSourceManager(b branch.BranchType) DataSourceManager {
	if b == branch.BranchTypeAT {
		return atMgr
	}
	if b == branch.BranchTypeXA {
		return xaMgr
	}
	return nil
}

// DataSourceManager
type DataSourceManager interface {
	// Register a Resource to be managed by Resource Manager
	RegisterResource(resource rm.Resource) error
	//  Unregister a Resource from the Resource Manager
	UnregisterResource(resource rm.Resource) error
	// Get all resources managed by this manager
	GetManagedResources() map[string]rm.Resource
	// BranchRollback
	BranchRollback(ctx context.Context, req message.BranchRollbackRequest) (branch.BranchStatus, error)
	// BranchCommit
	BranchCommit(ctx context.Context, req message.BranchCommitRequest) (branch.BranchStatus, error)
	// LockQuery
	LockQuery(ctx context.Context, req message.GlobalLockQueryRequest) (bool, error)
	// BranchRegister
	BranchRegister(ctx context.Context, clientId string, req message.BranchRegisterRequest) (int64, error)
	// BranchReport
	BranchReport(ctx context.Context, req message.BranchReportRequest) error
	// CreateTableMetaCache
	CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error)
}

type entry struct {
	db        *sql.DB
	metaCache TableMetaCache
}

// BasicSourceManager
type BasicSourceManager struct {
	// lock
	lock sync.RWMutex
	// tableMetaCache
	tableMetaCache map[string]*entry
}

func NewBasicSourceManager() *BasicSourceManager {
	return &BasicSourceManager{
		tableMetaCache: make(map[string]*entry, 0),
	}
}

// Commit a branch transaction
// TODO wait finish
func (dm *BasicSourceManager) BranchCommit(ctx context.Context, req message.BranchCommitRequest) (branch.BranchStatus, error) {
	return branch.BranchStatusPhaseoneDone, nil
}

// Rollback a branch transaction
// TODO wait finish
func (dm *BasicSourceManager) BranchRollback(ctx context.Context, req message.BranchRollbackRequest) (branch.BranchStatus, error) {
	return branch.BranchStatusPhaseoneFailed, nil
}

// Branch register long
func (dm *BasicSourceManager) BranchRegister(ctx context.Context, clientId string, req message.BranchRegisterRequest) (int64, error) {
	return 0, nil
}

//  Branch report
func (dm *BasicSourceManager) BranchReport(ctx context.Context, req message.BranchReportRequest) error {
	return nil
}

// Lock query boolean
func (dm *BasicSourceManager) LockQuery(ctx context.Context, branchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return true, nil
}

// Register a   model.Resource to be managed by   model.Resource Manager
func (dm *BasicSourceManager) RegisterResource(resource rm.Resource) error {
	err := rm.GetRMRemotingInstance().RegisterResource(resource)
	if err != nil {
		return err
	}
	return nil
}

//  Unregister a   model.Resource from the   model.Resource Manager
func (dm *BasicSourceManager) UnregisterResource(resource rm.Resource) error {
	return errors.New("unsupport unregister resource")
}

// Get all resources managed by this manager
func (dm *BasicSourceManager) GetManagedResources() *sync.Map {
	return nil
}

// Get the model.BranchType
func (dm *BasicSourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeAT
}

// CreateTableMetaCache
func (dm *BasicSourceManager) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error) {
	dm.lock.Lock()
	defer dm.lock.Unlock()

	res, err := buildResource(ctx, dbType, db)
	if err != nil {
		return nil, err
	}

	dm.tableMetaCache[resID] = res

	// 注册 AT 数据资源
	// dm.resourceMgr.RegisterResource(ATResource)

	return res.metaCache, err
}

// TableMetaCache tables metadata cache, default is open
type TableMetaCache interface {
	// Init
	Init(ctx context.Context, conn *sql.DB) error
	// GetTableMeta
	GetTableMeta(table string) (types.TableMeta, error)
	// Destroy
	Destroy() error
}

// buildResource
func buildResource(ctx context.Context, dbType types.DBType, db *sql.DB) (*entry, error) {
	cache := solts[dbType]()

	if err := cache.Init(ctx, db); err != nil {
		return nil, err
	}

	return &entry{
		db:        db,
		metaCache: cache,
	}, nil
}
