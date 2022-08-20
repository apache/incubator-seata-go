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
	"reflect"
	"strings"
	"unsafe"

	"github.com/seata/seata-go/pkg/common/log"

	"github.com/go-sql-driver/mysql"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/protocol/branch"
)

const (
	SeataMySQLDriver = "seata-mysql"
)

func init() {
	sql.Register(SeataMySQLDriver, &SeataDriver{
		target: mysql.MySQLDriver{},
	})
}

type SeataDriver struct {
	target driver.Driver
}

func (d *SeataDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.target.Open(name)
	if err != nil {
		log.Errorf("open target connection: %w", err)
		return nil, err
	}

	v := reflect.ValueOf(conn)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")
	proxy, err := d.OpenConnector(name)
	if err != nil {
		log.Errorf("open connector: %w", err)
		return nil, err
	}

	SetUnexportedField(field, proxy)
	return conn, nil
}

func (d *SeataDriver) OpenConnector(name string) (c driver.Connector, err error) {
	c = &dsnConnector{dsn: name, driver: d.target}
	if driverCtx, ok := d.target.(driver.DriverContext); ok {
		c, err = driverCtx.OpenConnector(name)
		if err != nil {
			log.Errorf("open connector: %w", err)
			return nil, err
		}
	}

	dbType := types.ParseDBType(d.getTargetDriverName())
	if dbType == types.DBTypeUnknown {
		return nil, fmt.Errorf("unsupport conn type %s", d.getTargetDriverName())
	}

	proxy, err := registerResource(c, dbType, sql.OpenDB(c), name)
	if err != nil {
		log.Errorf("register resource: %w", err)
		return nil, err
	}

	return proxy, nil
}

func (d *SeataDriver) getTargetDriverName() string {
	return strings.ReplaceAll(SeataMySQLDriver, "seata-", "")
}

type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

func (t *dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return t.driver.Open(t.dsn)
}

func (t *dsnConnector) Driver() driver.Driver {
	return t.driver
}

func registerResource(connector driver.Connector, dbType types.DBType, db *sql.DB,
	dataSourceName string, opts ...seataOption) (driver.Connector, error) {
	conf := loadConfig()
	for i := range opts {
		opts[i](conf)
	}

	if err := conf.validate(); err != nil {
		log.Errorf("invalid conf: %w", err)
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
		log.Errorf("create new resource: %w", err)
		return nil, err
	}

	if err = datasource.GetDataSourceManager(conf.BranchType).RegisterResource(res); err != nil {
		log.Errorf("regisiter resource: %w", err)
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
