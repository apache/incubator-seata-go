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
	"github.com/pkg/errors"
	"reflect"
	"sync"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
)

var (
	atOnce            sync.Once
	tableMetaCacheMap = map[types.DBType]func(*sql.DB, interface{}) TableMetaCache{}
)

// RegisterTableCache register the table meta cache for at and xa
func RegisterTableCache(dbType types.DBType, factory func(*sql.DB, interface{}) TableMetaCache) {
	tableMetaCacheMap[dbType] = factory
}

func GetTableCache(dbType types.DBType) TableMetaCache {
	if factory, ok := tableMetaCacheMap[dbType]; ok {
		return factory(nil, nil)
	}
	return nil
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
	return rm.GetRMRemotingInstance().RegisterResource(resource)
}

func (dm *BasicSourceManager) UnregisterResource(resource rm.Resource) error {
	return fmt.Errorf("unsupport unregister resource")
}

// CreateTableMetaCache create a table meta cache
func (dm *BasicSourceManager) CreateTableMetaCache(ctx context.Context, resID string, dbType types.DBType, db *sql.DB) (TableMetaCache, error) {
	dm.lock.Lock()
	defer dm.lock.Unlock()

	if existing, ok := dm.tableMetaCache[resID]; ok {
		return existing.metaCache, nil
	}

	res, err := buildResource(ctx, dbType, db)
	if err != nil {
		return nil, err
	}

	dm.tableMetaCache[resID] = res
	return res.metaCache, nil
}

// TableMetaCache tables metadata cache, default is open
type TableMetaCache interface {
	Init(ctx context.Context, conn *sql.DB) error
	GetTableMeta(ctx context.Context, dbName, table string) (*types.TableMeta, error)
	Destroy() error
}

// buildResource
func buildResource(ctx context.Context, dbType types.DBType, db *sql.DB) (*entry, error) {
	factory, ok := tableMetaCacheMap[dbType]
	if !ok {
		return nil, fmt.Errorf("unsupported db type: %v", dbType)
	}

	cfg, err := parseDBConfig(dbType, db)
	if err != nil {
		return nil, fmt.Errorf("parse db config failed: %w", err)
	}

	cache := factory(db, cfg)
	if err := cache.Init(ctx, db); err != nil {
		return nil, fmt.Errorf("init cache failed: %w", err)
	}

	return &entry{db: db, metaCache: cache}, nil
}

func parseDBConfig(dbType types.DBType, db *sql.DB) (interface{}, error) {
	dsn, err := extractDSN(db)
	if err != nil {
		return nil, err
	}

	switch dbType {
	case types.DBTypeMySQL, types.DBTypePostgreSQL:
		return dsn, nil
	default:
		return nil, fmt.Errorf("unsupported db type: %v", dbType)
	}
}

func extractDSN(db *sql.DB) (string, error) {
	if db == nil {
		return "", errors.New("db is nil")
	}

	conn, err := db.Conn(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to obtain connection: %w", err)
	}
	defer conn.Close()

	connVal := reflect.ValueOf(conn).Elem()
	rawConnField := connVal.FieldByName("conn")
	if !rawConnField.IsValid() || rawConnField.IsNil() {
		return "", errors.New("unable to obtain the underlying driver connection")
	}
	rawConn := rawConnField.Interface()

	connType := reflect.TypeOf(rawConn)
	if connType.Kind() == reflect.Ptr {
		connType = connType.Elem()
	}
	fullTypeName := connType.PkgPath() + "." + connType.Name()

	if fullTypeName == "github.com/jackc/pgx/v4/stdlib.Conn" {
		return extractPgxDSN(rawConn)
	}

	if fullTypeName == "github.com/lib/pq.Conn" {
		return extractPQDSN(rawConn)
	}

	if fullTypeName == "github.com/go-sql-driver/mysql.MySQLConn" {
		return extractMySQLDSN(rawConn)
	}

	return "", fmt.Errorf("unsupported drive type: %s", fullTypeName)
}

func extractPgxDSN(rawConn interface{}) (string, error) {
	val := reflect.ValueOf(rawConn).Elem()
	pgxConnField := val.FieldByName("conn")
	if !pgxConnField.IsValid() || pgxConnField.IsNil() {
		return "", errors.New("pgx driver: conn field not found")
	}

	pgxConnVal := pgxConnField.Elem()
	configField := pgxConnVal.FieldByName("config")
	if !configField.IsValid() || configField.IsNil() {
		return "", errors.New("pgx driver: config field not found")
	}
	
	configVal := configField.Elem()
	connStringField := configVal.FieldByName("ConnString")
	if !connStringField.IsValid() || connStringField.Kind() != reflect.String {
		return "", errors.New("pgx driver: ConnString field not found")
	}

	return connStringField.String(), nil
}

func extractPQDSN(rawConn interface{}) (string, error) {
	val := reflect.ValueOf(rawConn).Elem()
	dsnField := val.FieldByName("dsn")
	if !dsnField.IsValid() || dsnField.Kind() != reflect.String {
		return "", errors.New("pq driver: dsn field not found")
	}
	return dsnField.String(), nil
}

func extractMySQLDSN(rawConn interface{}) (string, error) {
	val := reflect.ValueOf(rawConn).Elem()
	dsnField := val.FieldByName("dsn")
	if !dsnField.IsValid() || dsnField.Kind() != reflect.String {
		return "", errors.New("MySQL driver: dsn field not found")
	}
	return dsnField.String(), nil
}
