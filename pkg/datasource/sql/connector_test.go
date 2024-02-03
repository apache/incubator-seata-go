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

type initConnectorFunc func(t *testing.T, ctrl *gomock.Controller) driver.Connector

func initMockConnector(t *testing.T, ctrl *gomock.Controller) driver.Connector {
	mockConn := mock.NewMockTestDriverConn(ctrl)

	connector := mock.NewMockTestDriverConnector(ctrl)
	connector.EXPECT().Connect(gomock.Any()).AnyTimes().Return(mockConn, nil)
	mockConn.EXPECT().QueryContext(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
			rows := &mysqlMockRows{}
			rows.data = [][]interface{}{
				{"8.0.29"},
			}
			return rows, nil
		})
	return connector
}

func initMockAtConnector(t *testing.T, ctrl *gomock.Controller, db *sql.DB, f initConnectorFunc) driver.Connector {
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
	reflectx.SetUnexportedField(v.FieldByName("target"), f(t, ctrl))

	return fieldVal.(driver.Connector)
}

func Test_seataATConnector_Connect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATMySQLDriver, "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	proxyConnector := initMockAtConnector(t, ctrl, db, initMockConnector)
	conn, err := proxyConnector.Connect(context.Background())
	assert.NoError(t, err)

	atConn, ok := conn.(*ATConn)
	assert.True(t, ok, "need return seata at connection")
	assert.True(t, atConn.txCtx.TransactionMode == types.Local, "init need local tx")
}

func initMockXaConnector(t *testing.T, ctrl *gomock.Controller, db *sql.DB, f initConnectorFunc) driver.Connector {
	v := reflect.ValueOf(db)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName("connector")
	fieldVal := reflectx.GetUnexportedField(field)

	atConnector, ok := fieldVal.(*seataXAConnector)
	assert.True(t, ok, "need return seata xa connector")

	v = reflect.ValueOf(atConnector)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	reflectx.SetUnexportedField(v.FieldByName("target"), f(t, ctrl))

	return fieldVal.(driver.Connector)
}

func Test_seataXAConnector_Connect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeXA, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataXAMySQLDriver, "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	proxyConnector := initMockXaConnector(t, ctrl, db, initMockConnector)
	conn, err := proxyConnector.Connect(context.Background())
	assert.NoError(t, err)

	xaConn, ok := conn.(*XAConn)
	assert.True(t, ok, "need return seata xa connection")
	assert.True(t, xaConn.txCtx.TransactionMode == types.Local, "init need local tx")
}
