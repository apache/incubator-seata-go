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
	"database/sql"

	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/protocol/branch"
)

type dbOption func(db *DBResource)

func withGroupID(id string) dbOption {
	return func(db *DBResource) {
		db.groupID = id
	}
}

func withResourceID(id string) dbOption {
	return func(db *DBResource) {
		db.resourceID = id
	}
}

func withTableMetaCache(c datasource.TableMetaCache) dbOption {
	return func(db *DBResource) {
		db.metaCache = c
	}
}

func withDBType(dt types.DBType) dbOption {
	return func(db *DBResource) {
		db.dbType = dt
	}
}

func withTarget(source *sql.DB) dbOption {
	return func(db *DBResource) {
		db.db = source
	}
}

func withDBName(dbName string) dbOption {
	return func(db *DBResource) {
		db.dbName = dbName
	}
}

func withConf(conf *seataServerConfig) dbOption {
	return func(db *DBResource) {
		db.conf = *conf
	}
}

func newResource(opts ...dbOption) (*DBResource, error) {
	db := new(DBResource)

	for i := range opts {
		opts[i](db)
	}

	return db, db.init()
}

// DBResource proxy sql.DB, enchance database/sql.DB to add distribute transaction ability
type DBResource struct {
	// groupID
	groupID string
	// resourceID
	resourceID string
	// conf
	conf seataServerConfig
	// db
	db     *sql.DB
	dbName string
	// dbType
	dbType types.DBType
	// undoLogMgr
	undoLogMgr undo.UndoLogManager
	// metaCache
	metaCache datasource.TableMetaCache
}

func (db *DBResource) init() error {
	return nil
}

// todo do not put meta data to rm
//func (db *DBResource) init() error {
//	mgr := datasource.GetDataSourceManager(db.GetBranchType())
//	metaCache, err := mgr.CreateTableMetaCache(context.Background(), db.resourceID, db.dbType, db.db)
//	if err != nil {
//		return err
//	}
//
//	db.metaCache = metaCache
//
//	return nil
//}

func (db *DBResource) GetResourceGroupId() string {
	return db.groupID
}

func (db *DBResource) GetResourceId() string {
	return db.resourceID
}

func (db *DBResource) GetBranchType() branch.BranchType {
	return db.conf.BranchType
}

type SqlDBProxy struct {
	db     *sql.DB
	dbName string
}

func (s *SqlDBProxy) GetDB() *sql.DB {
	return s.db
}

func (s *SqlDBProxy) GetDBName() string {
	return s.dbName
}
