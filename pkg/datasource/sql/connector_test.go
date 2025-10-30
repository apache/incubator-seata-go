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
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/util/reflectx"
)

type initConnectorFunc func(t *testing.T, ctrl *gomock.Controller, config dbTestConfig) driver.Connector

func initMockConnector(t *testing.T, ctrl *gomock.Controller, config dbTestConfig) driver.Connector {
	mockConn := mock.NewMockTestDriverConn(ctrl)
	baseMockConn(t, mockConn, config)

	connector := mock.NewMockTestDriverConnector(ctrl)
	connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
	return connector
}

func initMockAtConnector(t *testing.T, ctrl *gomock.Controller, db *sql.DB, config dbTestConfig, f initConnectorFunc) driver.Connector {
	v := reflect.ValueOf(db)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")
	fieldVal := reflectx.GetUnexportedField(field)

	atConnector, ok := fieldVal.(*seataATConnector)
	assert.True(t, ok, "need return seata at connector")

	v = reflect.ValueOf(atConnector)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	reflectx.SetUnexportedField(v.FieldByName("target"), f(t, ctrl, config))

	return fieldVal.(driver.Connector)
}

func Test_seataATConnector_Connect(t *testing.T) {

	for _, config := range getAllDBTestConfigs() {
		t.Run(config.name, func(t *testing.T) {

			if config.name == "PostgreSQL" {
				t.Skip("PostgreSQL AT mode is not supported yet, skip test")
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
			_ = mockMgr

			db, err := sql.Open(config.atDriverName, config.dsn)
			if err != nil {
				t.Fatalf("failed to open %s AT connector: %v", config.name, err)
			}
			defer func() {
				if db != nil {
					db.Close()
				}
			}()

			proxyConnector := initMockAtConnector(t, ctrl, db, config, initMockConnector)
			conn, err := proxyConnector.Connect(context.Background())
			assert.NoError(t, err, "failed to connect %s AT connector", config.name)

			atConn, ok := conn.(*ATConn)
			assert.True(t, ok, "%s should return seata AT connection", config.name)
			assert.Equal(t, types.Local, atConn.txCtx.TransactionMode, "%s init transaction mode should be Local", config.name)
		})
	}
}

func initMockXaConnector(t *testing.T, ctrl *gomock.Controller, db *sql.DB, config dbTestConfig, f initConnectorFunc) driver.Connector {
	v := reflect.ValueOf(db)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")
	fieldVal := reflectx.GetUnexportedField(field)

	xaConnector, ok := fieldVal.(*seataXAConnector)
	assert.True(t, ok, "need return seata xa connector")

	v = reflect.ValueOf(xaConnector)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	reflectx.SetUnexportedField(v.FieldByName("target"), f(t, ctrl, config))

	return fieldVal.(driver.Connector)
}

func Test_seataXAConnector_Connect(t *testing.T) {
	for _, config := range getAllDBTestConfigs() {
		t.Run(config.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMgr := initMockResourceManager(branch.BranchTypeXA, ctrl)
			_ = mockMgr

			db, err := sql.Open(config.xaDriverName, config.dsn)
			if err != nil {
				t.Fatalf("failed to open %s XA connector: %v", config.name, err)
			}
			defer func() {
				if db != nil {
					db.Close()
				}
			}()

			proxyConnector := initMockXaConnector(t, ctrl, db, config, initMockConnector)
			conn, err := proxyConnector.Connect(context.Background())
			assert.NoError(t, err, "failed to connect %s XA connector", config.name)

			xaConn, ok := conn.(*XAConn)
			assert.True(t, ok, "%s should return seata XA connection", config.name)
			assert.Equal(t, types.Local, xaConn.txCtx.TransactionMode, "%s init transaction mode should be Local", config.name)
		})
	}
}
