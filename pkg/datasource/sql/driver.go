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
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"io"
	"reflect"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/util/log"
	"strings"
)

const (
	// SeataATMySQLDriver MySQL driver for AT mode
	SeataATMySQLDriver = "seata-at-mysql"
	// SeataXAMySQLDriver MySQL driver for XA mode
	SeataXAMySQLDriver = "seata-xa-mysql"
	// SeataATPostgreSQLDriver PostgreSQL driver for AT mode
	SeataATPostgreSQLDriver = "seata-at-postgres"
	// SeataXAPostgreSQLDriver PostgreSQL driver for XA mode
	SeataXAPostgreSQLDriver = "seata-xa-postgres"
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

	sql.Register(SeataATPostgreSQLDriver, &seataATDriver{
		seataDriver: &seataDriver{
			branchType: branch.BranchTypeAT,
			transType:  types.ATMode,
			target:     &stdlib.Driver{},
		},
	})

	sql.Register(SeataXAPostgreSQLDriver, &seataXADriver{
		seataDriver: &seataDriver{
			branchType: branch.BranchTypeXA,
			transType:  types.XAMode,
			target:     &stdlib.Driver{},
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

	dbType := types.ParseDBType(d.getTargetDriverName())
	switch dbType {
	case types.DBTypeMySQL:
		mysqlCfg, err := mysql.ParseDSN(name)
		if err != nil {
			return nil, fmt.Errorf("parse mysql dsn error: %w", err)
		}
		_connector.cfg = mysqlCfg
	case types.DBTypePostgreSQL:
		pgxCfg, err := pgx.ParseConfig(name)
		if err != nil {
			return nil, fmt.Errorf("parse postgresql dsn error: %w", err)
		}
		_connector.cfg = pgxCfg
	}
	_connector.dbType = dbType

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
	driverCtx, ok := d.target.(driver.DriverContext)
	log.Infof("driver %T supports DriverContext? %v", d.target, ok)

	if ok {
		c, err = driverCtx.OpenConnector(name)
		if err != nil {
			log.Errorf("open connector via DriverContext: %v", err)
			return nil, err
		}
	} else {
		log.Infof("driver %T does not support DriverContext, use dsnConnector (lazy connect)", d.target)
	}

	dbType := types.ParseDBType(d.getTargetDriverName())
	if dbType == types.DBTypeUnknown {
		return nil, fmt.Errorf("unsupport conn type %s", d.getTargetDriverName())
	}

	proxy, err := d.getOpenConnectorProxy(c, dbType, sql.OpenDB(c), name)
	if err != nil {
		log.Errorf("register resource: %v", err)
		return nil, err
	}

	return proxy, nil
}

func (d *seataDriver) getOpenConnectorProxy(connector driver.Connector, dbType types.DBType,
	db *sql.DB, dataSourceName string) (driver.Connector, error) {

	var cfg interface{}
	var dbName string

	switch dbType {
	case types.DBTypeMySQL:
		mysqlCfg, err := mysql.ParseDSN(dataSourceName)
		if err != nil {
			log.Errorf("parse mysql dsn error: %v", err)
			return nil, err
		}
		cfg = mysqlCfg
		dbName = mysqlCfg.DBName
	case types.DBTypePostgreSQL:
		pgxCfg, err := pgx.ParseConfig(dataSourceName)
		if err != nil {
			log.Errorf("parse postgresql dsn error: %v", err)
			return nil, err
		}
		log.Infof("PostgreSQL parsed config: %+v", pgxCfg)

		dbName = pgxCfg.Database
		cfg = pgxCfg
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	options := []dbOption{
		withResourceID(parseResourceID(dataSourceName, dbType)),
		withTarget(db),
		withBranchType(d.branchType),
		withDBType(dbType),
		withDBName(dbName),
		withConnector(connector),
	}
	res, err := newResource(options...)
	if err != nil {
		log.Errorf("create new resource: %v", err)
		return nil, err
	}

	if err = datasource.GetDataSourceManager(d.branchType).RegisterResource(res); err != nil {
		log.Errorf("register resource: %v", err)
		return nil, err
	}

	return &seataConnector{
		res:    res,
		target: connector,
		cfg:    cfg,
	}, nil
}

func (d *seataDriver) getTargetDriverName() string {
	switch d.target.(type) {
	case mysql.MySQLDriver:
		return "mysql"
	case *stdlib.Driver:
		return "postgres"
	default:
		return ""
	}
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

func parseResourceID(dsn string, dbType types.DBType) string {
	i := strings.Index(dsn, "?")
	res := dsn

	if i > 0 {
		res = dsn[:i]
	}

	var schema string

	if dbType == types.DBTypePostgreSQL && i > 0 {
		queryParams := strings.Split(dsn[i+1:], "&")
		for _, param := range queryParams {
			kv := strings.SplitN(param, "=", 2)
			if len(kv) == 2 && strings.TrimSpace(kv[0]) == "search_path" {
				schema = kv[1]
				break
			}
		}
	}

	if schema != "" {
		res = res + "|schema=" + schema
	}

	return strings.ReplaceAll(res, ",", "|")
}

func selectDBVersion(ctx context.Context, conn driver.Conn) (string, error) {
	if conn == nil {
		log.Errorf("selectDBVersion: conn is nil, cannot query version")
		return "", errors.New("database connection is nil")
	}

	var rowsi driver.Rows
	var err error

	log.Infof("conn type: %T, supports QueryerContext? %v", conn, conn.(driver.QueryerContext) != nil)
	log.Infof("conn type: %T, supports Queryer? %v", conn, conn.(driver.Queryer) != nil)

	queryerCtx, ok := conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = conn.(driver.Queryer)
	}
	if ok {
		query := "SELECT VERSION()"
		if _, ok := conn.(*stdlib.Conn); ok {
			query = "SELECT version()"
		}
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, query, nil)
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
