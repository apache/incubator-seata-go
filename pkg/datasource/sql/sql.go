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
	gosql "database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"strings"
	"unsafe"

	"github.com/seata/seata-go-datasource/sql/datasource"
	"github.com/seata/seata-go-datasource/sql/types"
	"github.com/seata/seata-go/pkg/protocol/branch"
)

// Open opens a database specified by its database driver name and a
// driver-specific data source name, usually consisting of at least a
// database name and connection information.
//
// Most users will open a database via a driver-specific connection
// helper function that returns a *DB. No database drivers are included
// in the Go standard library. See https://golang.org/s/sqldrivers for
// a list of third-party drivers.
//
// Open may just validate its arguments without creating a connection
// to the database. To verify that the data source name is valid, call
// Ping.
//
// The returned DB is safe for concurrent use by multiple goroutines
// and maintains its own pool of idle connections. Thus, the Open
// function should be called just once. It is rarely necessary to
// close a DB.
func Open(driverName, dataSourceName string, opts ...seataOption) (*gosql.DB, error) {

	targetDriver := strings.ReplaceAll(driverName, "seata-", "")

	db, err := gosql.Open(targetDriver, dataSourceName)
	if err != nil {
		return nil, err
	}

	v := reflect.ValueOf(db)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")

	connector, _ := GetUnexportedField(field).(driver.Connector)

	dbType := types.ParseDBType(targetDriver)
	if dbType == types.DBType_Unknown {
		return nil, errors.New("unsuppoer db type")
	}

	proxy, err := regisResource(connector, dbType, db, dataSourceName, opts...)
	if err != nil {
		return nil, err
	}

	SetUnexportedField(field, proxy)
	return db, nil
}

func regisResource(connector driver.Connector, dbType types.DBType, db *gosql.DB,
	dataSourceName string, opts ...seataOption) (driver.Connector, error) {

	conf := loadConfig()
	for i := range opts {
		opts[i](conf)
	}

	if err := conf.validate(); err != nil {
		return connector, err
	}

	options := []dbOption{
		withGroupID(conf.GroupID),
		withResourceID(parseResourceID(dataSourceName)),
		withConf(conf),
		withTarget(db),
		withDBType(dbType),
	}

	res, err := newResource(options...)
	if err != nil {
		return nil, err
	}

	if err := datasource.GetDataSourceManager(conf.BranchType).RegisterResource(res); err != nil {
		return nil, err
	}

	return &seataConnector{
		res:    res,
		target: connector,
		conf:   conf,
	}, nil
}

type (
	seataOption func(cfg *seataServerConfig)

	// seataServerConfig
	seataServerConfig struct {
		// GroupID
		GroupID string `yaml:"groupID"`
		// BranchType
		BranchType branch.BranchType
		// Endpoints
		Endpoints []string `yaml:"endpoints" json:"endpoints"`
	}
)

func (c *seataServerConfig) validate() error {
	return nil
}

// loadConfig
// TODO wait finish
func loadConfig() *seataServerConfig {
	// 先设置默认配置

	// 从默认文件获取
	return &seataServerConfig{
		GroupID:    "DEFAULT_GROUP",
		BranchType: branch.BranchTypeAT,
		Endpoints:  []string{"127.0.0.1:8888"},
	}
}

func parseResourceID(dsn string) string {
	i := strings.Index(dsn, "?")

	res := dsn

	if i > 0 {
		res = dsn[:i]
	}

	return strings.ReplaceAll(res, ",", "|")
}

func GetUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func SetUnexportedField(field reflect.Value, value interface{}) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).
		Elem().
		Set(reflect.ValueOf(value))
}
