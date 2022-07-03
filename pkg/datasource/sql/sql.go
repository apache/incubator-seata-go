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
	"errors"
	"strings"

	"github.com/seata/seata-go-datasource/sql/datasource"
	"github.com/seata/seata-go-datasource/sql/types"
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
func Open(driverName, dataSourceName string, opts ...seataOption) (*DB, error) {

	db, err := gosql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	conf := loadConfig()
	for i := range opts {
		opts[i](conf)
	}

	dbType := types.ParseDBType(dataSourceName)

	if dbType == types.DBType_Unknown {
		return nil, errors.New("unsuppoer db type")
	}

	mgr := datasource.GetDataSourceManager()

	c, err := mgr.RegisResource(parseResourceID(dataSourceName), dbType, db)
	if err != nil {
		return nil, err
	}

	return newDB(
		withResourceID(parseResourceID(dataSourceName)),
		withConf(conf),
		withTableMetaCache(c),
		withDBType(dbType),
		withTarget(db),
	), nil
}

type (
	seataOption func(cfg *seataServerConfig)

	// seataServerConfig
	seataServerConfig struct {
		// Endpoints
		Endpoints []string `yaml:"endpoints" json:"endpoints"`
	}
)

// loadConfig
// TODO wait finish
func loadConfig() *seataServerConfig {
	// 先设置默认配置

	// 从默认文件获取
	return nil
}

func parseResourceID(dsn string) string {
	i := strings.Index(dsn, "?")

	res := dsn[:i]
	res = strings.ReplaceAll(res, ",", "|")

	return res
}
