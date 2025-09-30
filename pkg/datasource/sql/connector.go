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
	"database/sql/driver"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v4"
	"sync"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type seataATConnector struct {
	*seataConnector
	transType types.TransactionMode
}

func (c *seataATConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.seataConnector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	_conn, _ := conn.(*Conn)

	return &ATConn{
		Conn: _conn,
	}, nil
}

func (c *seataATConnector) Driver() driver.Driver {
	return &seataATDriver{
		seataDriver: c.seataConnector.Driver().(*seataDriver),
	}
}

type seataXAConnector struct {
	*seataConnector
	transType types.TransactionMode
}

func (c *seataXAConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.seataConnector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	_conn, _ := conn.(*Conn)

	return &XAConn{
		Conn: _conn,
	}, nil
}

func (c *seataXAConnector) Driver() driver.Driver {
	return &seataXADriver{
		seataDriver: c.seataConnector.Driver().(*seataDriver),
	}
}

// A Connector represents a driver in a fixed configuration
// and can create any number of equivalent Conns for use
// by multiple goroutines.
//
// A Connector can be passed to sql.OpenDB, to allow drivers
// to implement their own sql.DB constructors, or returned by
// DriverContext's OpenConnector method, to allow drivers
// access to context and to avoid repeated parsing of driver
// configuration.
//
// If a Connector implements io.Closer, the sql package's DB.Close
// method will call Close and return error (if any).
type seataConnector struct {
	transType types.TransactionMode
	res       *DBResource
	once      sync.Once
	driver    driver.Driver
	target    driver.Connector
	cfg       interface{}
	dbType    types.DBType
}

// Connect returns a connection to the database.
// Connect may return a cached connection (one previously
// closed), but doing so is unnecessary; the sql package
// maintains a pool of idle connections for efficient re-use.
//
// The provided context.Context is for dialing purposes only
// (see net.DialContext) and should not be stored or used for
// other purposes. A default timeout should still be used
// when dialing as a connection pool may call Connect
// asynchronously to any query.
//
// The returned connection is only used by one goroutine at a
// time.
func (c *seataConnector) Connect(ctx context.Context) (driver.Conn, error) {

	if mockDriverConn, ok := c.target.(*Conn); ok {
		mockDriverConn.res = c.res
		mockDriverConn.dbName = c.getDBName()
		mockDriverConn.dbType = c.dbType
		if mockDriverConn.txCtx == nil {
			mockDriverConn.txCtx = types.NewTxCtx()
		}
		if mockDriverConn.autoCommit == false {
			mockDriverConn.autoCommit = true
		}
		return mockDriverConn, nil
	}

	conn, err := c.target.Connect(ctx)
	if err != nil {
		return nil, err
	}

	dbName := c.getDBName()
	return &Conn{
		targetConn: conn,
		res:        c.res,
		txCtx:      types.NewTxCtx(),
		autoCommit: true,
		dbName:     dbName,
		dbType:     c.dbType,
	}, nil
}

func (c *seataConnector) getDBName() string {
	switch c.dbType {
	case types.DBTypeMySQL:
		if mysqlCfg, ok := c.cfg.(*mysql.Config); ok {
			return mysqlCfg.DBName
		}
	case types.DBTypePostgreSQL:
		switch cfg := c.cfg.(type) {
		case *pgx.ConnConfig:
			return cfg.Database
		case string:
			parsedCfg, err := pgx.ParseConfig(cfg)
			if err == nil {
				return parsedCfg.Database
			}
		}
	}
	return ""
}

// Driver returns the underlying Driver of the Connector,
// mainly to maintain compatibility with the Driver method
// on sql.DB.
func (c *seataConnector) Driver() driver.Driver {
	c.once.Do(func() {
		d := c.target.Driver()
		c.driver = d
	})

	return &seataDriver{
		target: c.driver,
	}
}
