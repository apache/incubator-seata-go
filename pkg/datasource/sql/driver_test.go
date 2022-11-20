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

	"github.com/seata/seata-go/pkg/rm"

	"github.com/golang/mock/gomock"
	"github.com/seata/seata-go/pkg/datasource/sql/mock"
	"github.com/seata/seata-go/pkg/util/reflectx"
	"github.com/stretchr/testify/assert"
)

func initMockResourceManager(t *testing.T, ctrl *gomock.Controller) *mock.MockDataSourceManager {
	mockResourceMgr := mock.NewMockDataSourceManager(ctrl)
	rm.GetRmCacheInstance().RegisterResourceManager(mockResourceMgr)
	mockResourceMgr.EXPECT().RegisterResource(gomock.Any()).AnyTimes().Return(nil)
	mockResourceMgr.EXPECT().CreateTableMetaCache(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	return mockResourceMgr
}

func Test_seataATDriver_Open(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(t, ctrl)
	_ = mockMgr

	db, err := sql.Open("seata-at-mysql", "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {

		v := reflect.ValueOf(db)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		field := v.FieldByName("connector")
		fieldVal := reflectx.GetUnexportedField(field)

		driverVal, ok := fieldVal.(driver.Connector).Driver().(*seataATDriver)
		assert.True(t, ok, "need seata at driver")

		vv := reflect.ValueOf(driverVal)
		if vv.Kind() == reflect.Ptr {
			vv = vv.Elem()
		}
		field = vv.FieldByName("target")

		mockDriver := mock.NewMockTestDriver(ctrl)
		mockDriver.EXPECT().Open(gomock.Any()).Return(mock.NewMockTestDriverConn(ctrl), nil)

		reflectx.SetUnexportedField(field, mockDriver)

		connector := &dsnConnector{
			driver: driverVal,
		}
		return connector
	})

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)

	v := reflect.ValueOf(conn)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("dc")
	fieldVal := reflectx.GetUnexportedField(field)

	vv := reflect.ValueOf(fieldVal)
	if vv.Kind() == reflect.Ptr {
		vv = vv.Elem()
	}

	field = vv.FieldByName("ci")
	fieldVal = reflectx.GetUnexportedField(field)

	_, ok := fieldVal.(*ATConn)
	assert.True(t, ok, "need return seata at connection")
}

func Test_seataATDriver_OpenConnector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(t, ctrl)
	_ = mockMgr

	db, err := sql.Open("seata-at-mysql", "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	v := reflect.ValueOf(db)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")
	fieldVal := reflectx.GetUnexportedField(field)

	_, ok := fieldVal.(*seataATConnector)
	assert.True(t, ok, "need return seata at connector")
}

func Test_seataXADriver_OpenConnector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(t, ctrl)
	_ = mockMgr

	db, err := sql.Open("seata-xa-mysql", "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	v := reflect.ValueOf(db)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")
	fieldVal := reflectx.GetUnexportedField(field)

	_, ok := fieldVal.(*seataXAConnector)
	assert.True(t, ok, "need return seata xa connector")
}
