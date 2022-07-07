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
	"database/sql"
	"sync"

	"github.com/seata/seata-go-datasource/sql/types"
	"github.com/seata/seata-go/pkg/rm"
)

var (
	once           sync.Once
	_dataSourceMgr DataSourceManager
	solts          = map[types.DBType]func() TableMetaCache{}
)

type DataSourceManager interface {
	// RegisDB
	RegisDB(resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error)
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

func (dm *dataSourceManager) GetResourceMgr() *rm.ResourceManager {
	return dm.resourceMgr
}

// RegisResource
func (dm *dataSourceManager) RegisDB(resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error) {
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
