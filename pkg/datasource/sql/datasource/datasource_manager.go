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

	"github.com/seata/seata-go-datasource/sql/types"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/resource"
	"github.com/seata/seata-go/pkg/rm"
)

var (
	once           sync.Once
	_dataSourceMgr DataSourceManager
	solts          = map[types.DBType]func() TableMetaCache{}
)

type DataSourceManager interface {
	resource.ResourceManager
	// CreateTableMetaCache
	CreateTableMetaCache(resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error)
}

func GetDataSourceManager() DataSourceManager {
	once.Do(func() {
		_dataSourceMgr = &dataSourceManager{
			resourceMgr:   rm.GetResourceManagerInstance(),
			lock:          sync.RWMutex{},
			resourceCache: make(map[string]*entry),
		}

	})
	return _dataSourceMgr
}

type entry struct {
	db        *sql.DB
	metaCache TableMetaCache
}

// dataSourceManager
type dataSourceManager struct {
	lock        sync.RWMutex
	resourceMgr *rm.ResourceManager
	// resourceCache
	resourceCache map[string]*entry
	// tablemetaCache
	tablemetaCache TableMetaCache
}

func (dm *dataSourceManager) RegisterResourceManager(resourceManager resource.ResourceManager) {
	dm.resourceMgr.RegisterResourceManager(resourceManager)
}

func (dm *dataSourceManager) GetResourceManager(branchType branch.BranchType) resource.ResourceManager {
	return dm.resourceMgr.GetResourceManager(branchType)
}

// Commit a branch transaction
func (dm *dataSourceManager) BranchCommit(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (branch.BranchStatus, error) {
	return dm.resourceMgr.BranchCommit(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Rollback a branch transaction
func (dm *dataSourceManager) BranchRollback(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, resourceId string, applicationData []byte) (branch.BranchStatus, error) {
	return dm.resourceMgr.BranchRollback(ctx, branchType, xid, branchId, resourceId, applicationData)
}

// Branch register long
func (dm *dataSourceManager) BranchRegister(ctx context.Context, branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	return dm.resourceMgr.BranchRegister(ctx, branchType, resourceId, clientId, xid, applicationData, lockKeys)
}

//  Branch report
func (dm *dataSourceManager) BranchReport(ctx context.Context, branchType branch.BranchType, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	return dm.resourceMgr.BranchReport(ctx, branchType, xid, branchId, status, applicationData)
}

// Lock query boolean
func (dm *dataSourceManager) LockQuery(ctx context.Context, branchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return dm.resourceMgr.LockQuery(ctx, branchType, resourceId, xid, lockKeys)
}

// Register a   model.Resource to be managed by   model.Resource Manager
func (dm *dataSourceManager) RegisterResource(resource resource.Resource) error {
	return dm.resourceMgr.RegisterResource(resource)
}

//  Unregister a   model.Resource from the   model.Resource Manager
func (dm *dataSourceManager) UnregisterResource(resource resource.Resource) error {
	return errors.New("unsupport unregister resource")
}

// Get all resources managed by this manager
func (dm *dataSourceManager) GetManagedResources() *sync.Map {
	return dm.GetManagedResources()
}

// Get the model.BranchType
func (dm *dataSourceManager) GetBranchType() branch.BranchType {
	return branch.BranchTypeAT
}

// CreateTableMetaCache
func (dm *dataSourceManager) CreateTableMetaCache(resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error) {
	dm.lock.Lock()
	defer dm.lock.Unlock()

	res, err := buildResource(dbType, db)
	if err != nil {
		return nil, err
	}

	dm.resourceCache[resID] = res

	// 注册 AT 数据资源
	// dm.resourceMgr.RegisterResource(ATResource)

	return res.metaCache, err
}

// TableMetaCache tables metadata cache, default is open
type TableMetaCache interface {
	// Init
	init(conn *sql.DB) error
	// GetTableMeta
	GetTableMeta(table string) (types.TableMeta, error)
	// Destory
	Destory() error
}

// buildResource
func buildResource(dbType types.DBType, db *sql.DB) (*entry, error) {

	cache := solts[dbType]()

	if err := cache.init(db); err != nil {
		return nil, err
	}

	return &entry{
		db:        db,
		metaCache: cache,
	}, nil
}
