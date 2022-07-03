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
)

var (
	_dataSourceMgr = &DataSourceManager{
		lock:          sync.RWMutex{},
		resourceCache: make(map[string]*resource),
	}

	solts = map[types.DBType]func() TableMetaCache{}
)

func GetDataSourceManager() *DataSourceManager {
	return _dataSourceMgr
}

type resource struct {
	db        *sql.DB
	metaCache TableMetaCache
}

// dataSourceManager
type DataSourceManager struct {
	lock sync.RWMutex
	// resourceCache
	resourceCache map[string]*resource
	// tablemetaCache
	tablemetaCache TableMetaCache
}

// RegisResource
func (dm *DataSourceManager) RegisResource(resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error) {
	dm.lock.Lock()
	defer dm.lock.Unlock()

	res, err := buildResource(dbType, db)
	if err != nil {
		return nil, err
	}

	dm.resourceCache[resID] = res

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
func buildResource(dbType types.DBType, db *sql.DB) (*resource, error) {

	cache := solts[dbType]()

	if err := cache.init(db); err != nil {
		return nil, err
	}

	return &resource{
		db:        db,
		metaCache: cache,
	}, nil
}
