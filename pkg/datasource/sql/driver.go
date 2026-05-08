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
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
	go_ora "github.com/sijms/go-ora/v2"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	mysql2 "seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource/mysql"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

var (
	oracleConnStrSIDPattern     = regexp.MustCompile(`(?i)\bSID\s*=\s*([^\s)]+)`)
	oracleConnStrServicePattern = regexp.MustCompile(`(?i)SERVICE[_[:space:]]*NAME\s*=\s*([^\s)]+)`)
	oracleConnStrHostPattern    = regexp.MustCompile(`(?i)\bHOST\s*=\s*([^\s)]+)`)
	oracleConnStrPortPattern    = regexp.MustCompile(`(?i)\bPORT\s*=\s*([^\s)]+)`)
	oracleJDBCServerPattern     = regexp.MustCompile(`(?i)@//([^/]+)`)
)

const (
	// SeataATMySQLDriver MySQL driver for AT mode
	SeataATMySQLDriver = "seata-at-mysql"
	// SeataXAMySQLDriver MySQL driver for XA mode
	SeataXAMySQLDriver = "seata-xa-mysql"
	// SeataXAOracleDriver Oracle driver for XA mode
	SeataXAOracleDriver = "seata-xa-oracle"
)

func initDriver() {
	sql.Register(SeataATMySQLDriver, &seataATDriver{
		seataDriver: &seataDriver{
			branchType: branch.BranchTypeAT,
			transType:  types.ATMode,
			target:     mysql.MySQLDriver{},
			driverName: "mysql",
		},
	})

	sql.Register(SeataXAMySQLDriver, &seataXADriver{
		seataDriver: &seataDriver{
			branchType: branch.BranchTypeXA,
			transType:  types.XAMode,
			target:     mysql.MySQLDriver{},
			driverName: "mysql",
		},
	})

	sql.Register(SeataXAOracleDriver, &seataXADriver{
		seataDriver: &seataDriver{
			branchType: branch.BranchTypeXA,
			transType:  types.XAMode,
			target:     &go_ora.OracleDriver{},
			driverName: "oracle",
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
	if d.getTargetDriverName() == "mysql" {
		cfg, _ := mysql.ParseDSN(name)
		_connector.cfg = cfg
	}

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
	if d.getTargetDriverName() == "mysql" {
		cfg, _ := mysql.ParseDSN(name)
		_connector.cfg = cfg
	}

	return &seataXAConnector{
		seataConnector: _connector,
	}, nil
}

type seataDriver struct {
	branchType branch.BranchType
	transType  types.TransactionMode
	target     driver.Driver
	driverName string
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
			log.Errorf("open connector: %v", err)
			return nil, err
		}
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
	var cfg *mysql.Config
	options := []dbOption{
		withResourceID(parseResourceID(dataSourceName, dbType)),
		withTarget(db),
		withBranchType(d.branchType),
		withDBType(dbType),
		withDBName(parseDBName(dataSourceName, dbType)),
		withConnector(connector),
	}
	res, err := newResource(options...)
	if err != nil {
		log.Errorf("create new resource: %v", err)
		return nil, err
	}
	if shouldUseMySQLTableCache(dbType) {
		cfg, _ = mysql.ParseDSN(dataSourceName)
		datasource.RegisterTableCache(types.DBTypeMySQL, mysql2.NewTableMetaInstance(db, cfg))
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
	if d.driverName != "" {
		return d.driverName
	}
	switch d.target.(type) {
	case mysql.MySQLDriver, *mysql.MySQLDriver:
		return "mysql"
	case *go_ora.OracleDriver:
		return "oracle"
	default:
		return ""
	}
}

func dbTypeDriverName(dbType types.DBType) string {
	switch dbType {
	case types.DBTypeMySQL:
		return "mysql"
	case types.DBTypeMARIADB:
		return "mariadb"
	case types.DBTypeOracle:
		return "oracle"
	default:
		return ""
	}
}

func shouldUseMySQLTableCache(dbType types.DBType) bool {
	return dbType == types.DBTypeMySQL || dbType == types.DBTypeMARIADB
}

func parseDBName(dataSourceName string, dbType types.DBType) string {
	switch dbType {
	case types.DBTypeMySQL, types.DBTypeMARIADB:
		cfg, err := mysql.ParseDSN(dataSourceName)
		if err != nil {
			return ""
		}
		return cfg.DBName
	case types.DBTypeOracle:
		u, err := url.Parse(dataSourceName)
		if err != nil {
			return ""
		}
		return oracleDatabaseName(u)
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
	if dbType == types.DBTypeOracle {
		if res := oracleResourceID(dsn); res != "" {
			return strings.ReplaceAll(res, ",", "|")
		}
	}
	i := strings.Index(dsn, "?")
	res := dsn
	if i > 0 {
		res = dsn[:i]
	}
	return strings.ReplaceAll(res, ",", "|")
}

func oracleResourceID(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	if dbName := oracleDatabaseName(u); dbName != "" {
		if server := oracleServerName(u.Query()); server != "" {
			u.Host = server
		}
		u.Path = "/" + dbName
		u.RawQuery = ""
		return u.String()
	}
	return ""
}

func oracleDatabaseName(u *url.URL) string {
	if dbName := oracleQueryDatabaseName(u.Query()); dbName != "" {
		return dbName
	}
	if dbName := oracleConnStrDatabaseName(u.Query()); dbName != "" {
		return dbName
	}
	return strings.Trim(u.Path, "/")
}

func oracleQueryDatabaseName(query url.Values) string {
	return oracleQueryValue(query, "SID", "SERVICE NAME")
}

func oracleConnStrDatabaseName(query url.Values) string {
	for key, values := range query {
		if normalizeOracleQueryKey(key) != "CONNSTR" {
			continue
		}
		for _, value := range values {
			if dbName := oracleDatabaseNameFromConnStr(value); dbName != "" {
				return dbName
			}
		}
	}
	return ""
}

func oracleServerName(query url.Values) string {
	if server := oracleQueryValue(query, "SERVER"); server != "" {
		return server
	}
	for key, values := range query {
		if normalizeOracleQueryKey(key) != "CONNSTR" {
			continue
		}
		for _, value := range values {
			if server := oracleServerNameFromConnStr(value); server != "" {
				return server
			}
		}
	}
	return ""
}

func oracleServerNameFromConnStr(connStr string) string {
	if server := oracleConnStrPatternValue(oracleJDBCServerPattern, connStr); server != "" {
		return server
	}
	host := oracleConnStrPatternValue(oracleConnStrHostPattern, connStr)
	if host == "" {
		return ""
	}
	if port := oracleConnStrPatternValue(oracleConnStrPortPattern, connStr); port != "" {
		return host + ":" + port
	}
	return host
}

func oracleDatabaseNameFromConnStr(connStr string) string {
	connStr = strings.TrimSpace(connStr)
	if connStr == "" {
		return ""
	}
	if dbName := oracleConnStrPatternValue(oracleConnStrSIDPattern, connStr); dbName != "" {
		return dbName
	}
	if dbName := oracleConnStrPatternValue(oracleConnStrServicePattern, connStr); dbName != "" {
		return dbName
	}
	return oracleConnStrPathDatabaseName(connStr)
}

func oracleConnStrPatternValue(pattern *regexp.Regexp, connStr string) string {
	matches := pattern.FindStringSubmatch(connStr)
	if len(matches) < 2 {
		return ""
	}
	return oracleCleanDatabaseName(matches[1])
}

func oracleConnStrPathDatabaseName(connStr string) string {
	idx := strings.LastIndex(connStr, "/")
	if idx < 0 || idx == len(connStr)-1 {
		return ""
	}
	dbName := connStr[idx+1:]
	if end := strings.IndexAny(dbName, "?)"); end >= 0 {
		dbName = dbName[:end]
	}
	return oracleCleanDatabaseName(dbName)
}

func oracleCleanDatabaseName(dbName string) string {
	return strings.Trim(strings.TrimSpace(dbName), `"'`)
}

func oracleQueryValue(query url.Values, names ...string) string {
	for _, name := range names {
		for key, values := range query {
			if normalizeOracleQueryKey(key) != name {
				continue
			}
			for _, value := range values {
				if value = strings.TrimSpace(value); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func normalizeOracleQueryKey(key string) string {
	key = strings.ReplaceAll(key, "_", " ")
	return strings.ToUpper(strings.Join(strings.Fields(key), " "))
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
