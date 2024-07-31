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

package fence

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	// SeataFenceMySQLDriver MySQL driver for fence
	SeataFenceMySQLDriver = "seata-fence-mysql"
)

func init() {
	sql.Register(SeataFenceMySQLDriver, &FenceDriver{
		TargetDriver: &mysql.MySQLDriver{},
	})
}

type FenceDriver struct {
	TargetDriver driver.Driver
	TargetDB     *sql.DB
}

func (fd *FenceDriver) Open(name string) (driver.Conn, error) {
	return nil, errors.New("operation unsupported")
}

func (fd *FenceDriver) OpenConnector(name string) (connector driver.Connector, re error) {
	connector = &dsnConnector{dsn: name, driver: fd.TargetDriver}
	if driverCtx, ok := fd.TargetDriver.(driver.DriverContext); ok {
		connector, re = driverCtx.OpenConnector(name)
		if re != nil {
			log.Errorf("open connector: %w", re)
			return nil, re
		}
	}

	fd.TargetDB = sql.OpenDB(connector)

	return &SeataFenceConnector{
		TargetConnector: connector,
		TargetDB:        fd.TargetDB,
	}, nil
}

type dsnConnector struct {
	dsn    string
	driver driver.Driver
}

func (connector *dsnConnector) Connect(_ context.Context) (driver.Conn, error) {
	return connector.driver.Open(connector.dsn)
}

func (connector *dsnConnector) Driver() driver.Driver {
	return connector.driver
}

type SeataFenceConnector struct {
	TargetConnector driver.Connector
	TargetDB        *sql.DB
}

func (connector *SeataFenceConnector) Connect(ctx context.Context) (driver.Conn, error) {
	targetConn, err := connector.TargetConnector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &FenceConn{
		TargetConn: targetConn,
		TargetDB:   connector.TargetDB,
	}, nil
}

func (connector *SeataFenceConnector) Driver() driver.Driver {
	return connector.TargetConnector.Driver()
}
