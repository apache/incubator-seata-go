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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
)

func TestOpen(t *testing.T) {
	conn, err := (&FenceDriver{}).Open("")
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func TestOpenConnector(t *testing.T) {
	mysqlDriver := &mysql.MySQLDriver{}
	d := &FenceDriver{TargetDriver: mysqlDriver}

	openDBStub := gomonkey.ApplyFunc(sql.OpenDB, func(_ driver.Connector) *sql.DB {
		return &sql.DB{}
	})

	openConnectorStub := gomonkey.ApplyMethod(
		reflect.TypeOf(mysqlDriver),
		"OpenConnector",
		func(_ *mysql.MySQLDriver, name string) (connector driver.Connector, re error) {
			return nil, nil
		},
	)

	connector, err := d.OpenConnector("mock")
	assert.Nil(t, err)
	seataConnector, ok := connector.(*SeataFenceConnector)
	assert.True(t, ok)
	assert.Equal(t, d.TargetDB, seataConnector.TargetDB)

	openDBStub.Reset()
	openConnectorStub.Reset()
}

func TestConnect(t *testing.T) {
	c := &dsnConnector{
		driver: &mysql.MySQLDriver{},
	}

	openStub := gomonkey.ApplyMethod(
		reflect.TypeOf(c.driver),
		"Open",
		func(_ *mysql.MySQLDriver, name string) (driver.Conn, error) {
			return nil, nil
		},
	)

	conn, err := c.Connect(context.Background())
	assert.Nil(t, err)
	assert.Nil(t, conn)

	openStub.Reset()

}

func TestDriver(t *testing.T) {
	c := &dsnConnector{
		driver: &mysql.MySQLDriver{},
	}

	assert.Equal(t, c.driver, c.Driver())
}
