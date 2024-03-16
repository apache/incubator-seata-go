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
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/go-sql-driver/mysql"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	mysql2 "seata.apache.org/seata-go/pkg/datasource/sql/datasource/mysql"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	// SeataATMySQLDriver MySQL driver for AT mode
	SeataATMySQLDriver = "seata-at-mysql"
	// SeataXAMySQLDriver MySQL driver for XA mode
	SeataXAMySQLDriver = "seata-xa-mysql"
)

func initDriver() {
	sql.Register(SeataATMySQLDriver, &seataATDriver{
		seataDriver: &seataDriver{
			branchType: branch.BranchTypeAT,
			transType:  types.ATMode,
			target:     mysql.MySQLDriver{},
		},
	})

	sql.Register(SeataXAMySQLDriver, &seataXADriver{
		seataDriver: &seataDriver{
			branchType: branch.BranchTypeXA,
			transType:  types.XAMode,
			target:     mysql.MySQLDriver{},
		},
	})
}

type seataATDriver struct {
	*seataDriver
}

func (d *seataATDriver) OpenConnector(name string) (c driver.Connector, err error) {
	connector, err := d.seataDriver.OpenConnector(name)
	if err != nil {
		return nil, err
	}

	_connector, _ := connector.(*seataConnector)
	_connector.transType = types.ATMode
	cfg, _ := mysql.ParseDSN(name)
	_connector.cfg = cfg

	return &seataATConnector{
		seataConnector: _connector,
	}, nil
}

type seataXADriver struct {
	*seataDriver
}

func (d *seataXADriver) OpenConnector(name string) (c driver.Connector, err error) {
	connector, err := d.seataDriver.OpenConnector(name)
	if err != nil {
		return nil, err
	}

	_connector, _ := connector.(*seataConnector)
	_connector.transType = types.XAMode
	cfg, _ := mysql.ParseDSN(name)
	_connector.cfg = cfg

	return &seataXAConnector{
		seataConnector: _connector,
	}, nil
}

type seataDriver struct {
	branchType branch.BranchType
	transType  types.TransactionMode
	target     driver.Driver
}

// Open never be called, because seataDriver implemented dri.DriverContext interface.
// reference package: datasource/sql [https://cs.opensource.google/go/go/+/master:src/database/sql/sql.go;l=813]
// and maybe the sql.BD will be call Driver() method, but it obtain the Driver is fron Connector that is proxed by seataConnector.
func (d *seataDriver) Open(name string) (driver.Conn, error) {
	return nil, errors.New(("operation unsupport."))
}

func (d *seataDriver) OpenConnector(name string) (c driver.Connector, err error) {
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

	proxy, err := d.getOpenConnectorProxy(c, dbType, sql.OpenDB(c), name)
	if err != nil {
		log.Errorf("register resource: %w", err)
		return nil, err
	}

	return proxy, nil
}

func (d *seataDriver) getOpenConnectorProxy(connector driver.Connector, dbType types.DBType,
	db *sql.DB, dataSourceName string) (driver.Connector, error) {
	cfg, _ := mysql.ParseDSN(dataSourceName)
	options := []dbOption{
		withResourceID(parseResourceID(dataSourceName)),
		withTarget(db),
		withBranchType(d.branchType),
		withDBType(dbType),
		withDBName(cfg.DBName),
		withConnector(connector),
	}
	res, err := newResource(options...)
	if err != nil {
		log.Errorf("create new resource: %w", err)
		return nil, err
	}
	datasource.RegisterTableCache(types.DBTypeMySQL, mysql2.NewTableMetaInstance(db))
	if err = datasource.GetDataSourceManager(d.branchType).RegisterResource(res); err != nil {
		log.Errorf("regisiter resource: %w", err)
		return nil, err
	}
	return &seataConnector{
		res:    res,
		target: connector,
		cfg:    cfg,
	}, nil
}

func (d *seataDriver) getTargetDriverName() string {
	return "mysql"
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

func parseResourceID(dsn string) string {
	i := strings.Index(dsn, "?")
	res := dsn
	if i > 0 {
		res = dsn[:i]
	}
	return strings.ReplaceAll(res, ",", "|")
}

func selectDBVersion(ctx context.Context, conn driver.Conn) (string, error) {
	var rowsi driver.Rows
	var err error

	queryerCtx, ok := conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, "SELECT VERSION()", nil)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return "", err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return "", fmt.Errorf("invalid conn")
	}

	dest := make([]driver.Value, 1)
	var version string
	if err = rowsi.Next(dest); err != nil {
		if err == io.EOF {
			return version, nil
		}
		return "", err
	}
	if len(dest) != 1 {
		return "", errors.New("get db version is not column 1")
	}

	switch reflect.TypeOf(dest[0]).Kind() {
	case reflect.Slice, reflect.Array:
		val := reflect.ValueOf(dest[0]).Bytes()
		version = string(val)
	case reflect.String:
		version = reflect.ValueOf(dest[0]).String()
	default:
		return "", errors.New("get db version is not a string")
	}

	return version, nil
}
