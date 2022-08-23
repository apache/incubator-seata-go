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
	"sync"
)

type seataConnector struct {
	conf   *seataServerConfig
	res    *DBResource
	once   sync.Once
	driver driver.Driver
	target driver.Connector
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
	conn, err := c.target.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return &Conn{targetConn: conn, res: c.res}, nil
}

// Driver returns the underlying Driver of the Connector,
// mainly to maintain compatibility with the Driver method
// on sql.DB.
func (c *seataConnector) Driver() driver.Driver {
	c.once.Do(func() {
		d := c.target.Driver()
		c.driver = d
	})

	return &SeataDriver{target: c.driver}
}
