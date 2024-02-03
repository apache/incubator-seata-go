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
	"fmt"
	"sync"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
)

var (
	atOnce            sync.Once
	tableMetaCacheMap = map[types.DBType]TableMetaCache{}
)

// RegisterTableCache register the table meta cache for at and xa
func RegisterTableCache(dbType types.DBType, tableMetaCache TableMetaCache) {
	tableMetaCacheMap[dbType] = tableMetaCache
}

func GetTableCache(dbType types.DBType) TableMetaCache {
	return tableMetaCacheMap[dbType]
}

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
	CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error)
}

type entry struct {
	db        *sql.DB
	metaCache TableMetaCache
}

// BasicSourceManager the basic source manager for xa and at
type BasicSourceManager struct {
	lock sync.RWMutex
	// tableMetaCache
	// todo do not put meta cache here
	tableMetaCache map[string]*entry
}

func NewBasicSourceManager() *BasicSourceManager {
	return &BasicSourceManager{
		tableMetaCache: make(map[string]*entry, 0),
	}
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

// CreateTableMetaCache create a table meta cache
func (dm *BasicSourceManager) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error) {
	dm.lock.Lock()
	defer dm.lock.Unlock()

	res, err := buildResource(ctx, dbType, db)
	if err != nil {
		return nil, err
	}

	dm.tableMetaCache[resID] = res
	return res.metaCache, err
}

// TableMetaCache tables metadata cache, default is open
type TableMetaCache interface {
	Init(ctx context.Context, conn *sql.DB) error
	GetTableMeta(ctx context.Context, dbName, table string) (*types.TableMeta, error)
	Destroy() error
}

// buildResource
func buildResource(ctx context.Context, dbType types.DBType, db *sql.DB) (*entry, error) {
	cache := tableMetaCacheMap[dbType]
	if err := cache.Init(ctx, db); err != nil {
		return nil, err
	}

	return &entry{
		db:        db,
		metaCache: cache,
	}, nil
}
