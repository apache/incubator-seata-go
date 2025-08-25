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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/util/reflectx"
)

func initMockResourceManager(branchType branch.BranchType, ctrl *gomock.Controller) *mock.MockDataSourceManager {
	mockResourceMgr := mock.NewMockDataSourceManager(ctrl)
	mockResourceMgr.SetBranchType(branchType)
	mockResourceMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).AnyTimes().Return(int64(0), nil)
	rm.GetRmCacheInstance().RegisterResourceManager(mockResourceMgr)
	mockResourceMgr.EXPECT().RegisterResource(gomock.Any()).AnyTimes().Return(nil)
	mockResourceMgr.EXPECT().CreateTableMetaCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	return mockResourceMgr
}

func assertConnectorType(t *testing.T, db *sql.DB, expectedType reflect.Type, format string, args ...interface{}) {
	t.Helper()

	v := reflect.ValueOf(db)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")
	assert.True(t, field.IsValid(), "sql.DB should have 'connector' field")

	fieldVal := reflectx.GetUnexportedField(field)
	assert.NotNil(t, fieldVal, "connector field value should not be nil")

	errMsg := fmt.Sprintf(format, args...)
	assert.True(t, reflect.TypeOf(fieldVal) == expectedType, errMsg)
}

func Test_seataATDriver_Open(t *testing.T) {
	for _, config := range getAllDBTestConfigs() {
		t.Run(config.name, func(t *testing.T) {
			if config.dbType == types.DBTypePostgreSQL {
				t.Skipf("AT mode for PostgreSQL is not implemented yet, skip test: %s", config.name)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
			_ = mockMgr

			db, err := sql.Open(config.atDriverName, config.dsn)
			if err != nil {
				t.Fatalf("failed to open %s AT driver: %v", config.name, err)
			}
			defer func() {
				if db != nil {
					db.Close()
				}
			}()

			_ = initMockAtConnector(
				t,
				ctrl,
				db,
				config,
				func(t *testing.T, ctrl *gomock.Controller, config dbTestConfig) driver.Connector {
					v := reflect.ValueOf(db)
					if v.Kind() == reflect.Ptr {
						v = v.Elem()
					}

					field := v.FieldByName("connector")
					fieldVal := reflectx.GetUnexportedField(field)

					driverVal, ok := fieldVal.(driver.Connector).Driver().(*seataATDriver)
					assert.True(t, ok, "%s need seata AT driver", config.name)

					vv := reflect.ValueOf(driverVal)
					if vv.Kind() == reflect.Ptr {
						vv = vv.Elem()
					}
					field = vv.FieldByName("target")

					mockDriver := mock.NewMockTestDriver(ctrl)
					reflectx.SetUnexportedField(field, mockDriver)

					return &dsnConnector{driver: driverVal}
				},
			)

			conn, err := db.Conn(context.Background())
			assert.NotNil(t, err, "%s should return error when getting conn", config.name)
			assert.Nil(t, conn, "%s conn should be nil when error occurs", config.name)
		})
	}
}

func Test_seataATDriver_OpenConnector(t *testing.T) {
	for _, config := range getAllDBTestConfigs() {
		t.Run(config.name, func(t *testing.T) {
			if config.dbType == types.DBTypePostgreSQL {
				t.Skipf("AT mode for PostgreSQL is not implemented yet, skip connector test: %s", config.name)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
			_ = mockMgr

			db, err := sql.Open(config.atDriverName, config.dsn)
			if err != nil {
				t.Fatalf("failed to open %s AT driver for connector test: %v", config.name, err)
			}
			defer func() {
				if db != nil {
					db.Close()
				}
			}()

			assertConnectorType(
				t,
				db,
				reflect.TypeOf(&seataATConnector{}),
				"%s should return seata AT connector", config.name,
			)
		})
	}
}

func Test_seataXADriver_OpenConnector(t *testing.T) {
	for _, config := range getAllDBTestConfigs() {
		t.Run(config.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMgr := initMockResourceManager(branch.BranchTypeXA, ctrl)
			_ = mockMgr

			db, err := sql.Open(config.xaDriverName, config.dsn)
			if err != nil {
				t.Fatalf("failed to open %s XA driver for connector test: %v", config.name, err)
			}
			defer func() {
				if db != nil {
					db.Close()
				}
			}()

			assertConnectorType(
				t,
				db,
				reflect.TypeOf(&seataXAConnector{}),
				"%s should return seata XA connector", config.name,
			)
		})
	}
}
