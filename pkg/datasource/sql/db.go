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
	"database/sql/driver"
	"fmt"
	"sync"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/datasource/sql/xa"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/util/log"
)

type dbOption func(db *DBResource)

func withDsn(dsn string) dbOption {
	return func(db *DBResource) {
		db.dsn = dsn
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

func withBranchType(dt branch.BranchType) dbOption {
	return func(db *DBResource) {
		db.branchType = dt
	}
}

func withTarget(source *sql.DB) dbOption {
	return func(db *DBResource) {
		db.db = source
	}
}

func withConnector(ci driver.Connector) dbOption {
	return func(db *DBResource) {
		db.connector = ci
	}
}

func withDBName(dbName string) dbOption {
	return func(db *DBResource) {
		db.dbName = dbName
	}
}

func withConf(conf *XAConnConf) dbOption {
	return func(db *DBResource) {
		db.xaConnConf = conf
	}
}

func newResource(opts ...dbOption) (*DBResource, error) {
	db := new(DBResource)

	for i := range opts {
		opts[i](db)
	}

	db.init()
	return db, nil
}

// DBResource proxy sql.DB, enchance database/sql.DB to add distribute transaction ability
type DBResource struct {
	xaConnConf *XAConnConf
	// only use by mysql
	dbVersion  string
	dsn        string
	resourceID string
	db         *sql.DB
	connector  driver.Connector
	dbName     string
	dbType     types.DBType
	undoLogMgr undo.UndoLogManager
	branchType branch.BranchType

	// for xa
	metaCache    datasource.TableMetaCache
	shouldBeHeld bool
	keeper       sync.Map
}

func (db *DBResource) GetResourceGroupId() string {
	panic("implement me")
}

func (db *DBResource) init() {
	ctx := context.Background()
	conn, err := db.connector.Connect(ctx)
	if err != nil {
		log.Errorf("connect: %w", err)
	}
	version, err := selectDBVersion(ctx, conn)
	if err != nil {
		log.Errorf("select db version: %w", err)
	}
	db.SetDbVersion(version)
	db.checkDbVersion()
}

func (db *DBResource) GetResourceId() string {
	return db.resourceID
}

func (db *DBResource) GetBranchType() branch.BranchType {
	return db.branchType
}

func (db *DBResource) GetDB() *sql.DB {
	return db.db
}

func (db *DBResource) GetDBName() string {
	return db.dbName
}

func (db *DBResource) GetDbType() types.DBType {
	return db.dbType
}

func (db *DBResource) SetDbType(dbType types.DBType) {
	db.dbType = dbType
}

func (db *DBResource) SetDbVersion(v string) {
	db.dbVersion = v
}

func (db *DBResource) GetDbVersion() string {
	return db.dbVersion
}

func (db *DBResource) IsShouldBeHeld() bool {
	return db.shouldBeHeld
}

// Hold the xa connection.
func (db *DBResource) Hold(xaBranchID string, v interface{}) error {
	_, exist := db.keeper.Load(xaBranchID)
	if !exist {
		db.keeper.Store(xaBranchID, v)
		return nil
	}
	return nil
}

func (db *DBResource) Release(xaBranchID string) {
	db.keeper.Delete(xaBranchID)
}

func (db *DBResource) Lookup(xaBranchID string) (interface{}, bool) {
	return db.keeper.Load(xaBranchID)
}

func (db *DBResource) GetKeeper() *sync.Map {
	return &db.keeper
}

func (db *DBResource) ConnectionForXA(ctx context.Context, xaXid XAXid) (*XAConn, error) {
	xaBranchXid := xaXid.String()
	tmpConn, ok := db.Lookup(xaBranchXid)
	if ok && tmpConn != nil {
		connectionProxyXa, isConnectionProxyXa := tmpConn.(*XAConn)
		if !isConnectionProxyXa {
			return nil, fmt.Errorf("get connection proxy xa from cache error, xid:%s", xaXid.String())
		}
		return connectionProxyXa, nil
	}

	// why here need a new connection?
	// 1. because there maybe a rm cluster
	// 2. the first phase select a rm1, and store the connection is the keeper
	// 3. tc request the second phase. but the rm1 is shutdown, so the tc select another rm (like rm2)
	// 4. so when the second phase request coming to rm2, rm2 must not store the connection.
	// 5. the rm2 get the second phase do the two thing.
	// 	  1. in mysql version >= 8.0.29, mysql support the xa transaction commit by another connection. so just commit
	//    2. when the version < 8.0.29. so just make the transaction rollback
	newDriverConn, err := db.connector.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("get xa new connection failure, xid:%s, err:%v", xaXid.String(), err)
	}
	xaResource, err := xa.CreateXAResource(newDriverConn, types.DBTypeMySQL)
	if err != nil {
		return nil, fmt.Errorf("create xa resoruce err:%w", err)
	}
	xaConn := &XAConn{
		Conn: &Conn{
			targetConn: newDriverConn,
			res:        db,
		},
		xaBranchXid: XaIdBuild(xaXid.GetGlobalXid(), xaXid.GetBranchId()),
		xaResource:  xaResource,
	}
	return xaConn, nil
}

func (db *DBResource) checkDbVersion() error {
	switch db.dbType {
	case types.DBTypeMySQL:
		currentVersion, err := util.ConvertDbVersion(db.dbVersion)
		if err != nil {
			return fmt.Errorf("new connection xa proxy convert db version:%s err:%v", db.GetDbVersion(), err)
		}

		shouldKeptVersion, err := util.ConvertDbVersion("8.0.29")
		if err != nil {
			return fmt.Errorf("new connection xa proxy convert db version 8.0.29 err:%v", err)
		}

		if currentVersion < shouldKeptVersion {
			db.shouldBeHeld = true
		}
	case types.DBTypeMARIADB:
		db.shouldBeHeld = true
	}
	return nil
}
