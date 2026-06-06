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

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/util/reflectx"
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

func Test_seataATDriver_Open(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open("seata-at-mysql", "root:seata_go@tcp(127.0.0.1:3306)/seata_go_test?multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	_ = initMockAtConnector(t, ctrl, db, func(t *testing.T, ctrl *gomock.Controller) driver.Connector {
		connector := mock.NewMockTestDriverConnector(ctrl)
		connector.EXPECT().Connect(gomock.Any()).Return(nil, fmt.Errorf("connect error"))
		return connector
	})

	conn, err := db.Conn(context.Background())
	assert.NotNil(t, err)
	assert.Nil(t, conn)
}

func Test_seataATDriver_OpenConnector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
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

func Test_seataATPostgresDriver_OpenConnector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeAT, ctrl)
	_ = mockMgr

	db, err := sql.Open(SeataATPostgresDriver, postgresTestDSN)
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

	connector, ok := fieldVal.(*seataATConnector)
	assert.True(t, ok, "need return seata at connector")
	assert.Equal(t, types.DBTypePostgreSQL, connector.dbType)
	assert.Equal(t, "seata_go_test", connector.dbName)
}

func Test_seataXADriver_OpenConnector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMgr := initMockResourceManager(branch.BranchTypeXA, ctrl)
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

func TestDriverDescriptorTableMetaCache(t *testing.T) {
	assert.NotNil(t, mySQLDriverDescriptor.newTableMetaCache)
	assert.NotNil(t, postgresDriverDescriptor.newTableMetaCache)
	assert.Nil(t, oracleDriverDescriptor.newTableMetaCache)
}

func TestParseResourceIDOracleIncludesServiceName(t *testing.T) {
	serviceOne := parseResourceID("oracle://system:pass@localhost:1521/?service name=FREEPDB1", types.DBTypeOracle)
	serviceTwo := parseResourceID("oracle://system:pass@localhost:1521/?SERVICE_NAME=FREEPDB2", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@localhost:1521/FREEPDB1", serviceOne)
	assert.Equal(t, "oracle://system:pass@localhost:1521/FREEPDB2", serviceTwo)
	assert.NotEqual(t, serviceOne, serviceTwo)
}

func TestParseResourceIDOracleIncludesSID(t *testing.T) {
	sidOne := parseResourceID("oracle://system:pass@localhost:1521/?SID=ORCL1", types.DBTypeOracle)
	sidTwo := parseResourceID("oracle://system:pass@localhost:1521/?sid=ORCL2", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@localhost:1521/ORCL1", sidOne)
	assert.Equal(t, "oracle://system:pass@localhost:1521/ORCL2", sidTwo)
	assert.NotEqual(t, sidOne, sidTwo)
}

func TestParseResourceIDOracleMatchesGoOraQueryPrecedence(t *testing.T) {
	queryService := parseResourceID("oracle://system:pass@localhost:1521/PATHDB?SERVICE NAME=QUERYDB", types.DBTypeOracle)
	sid := parseResourceID("oracle://system:pass@localhost:1521/PATHDB?SERVICE NAME=QUERYDB&SID=SIDDB", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@localhost:1521/QUERYDB", queryService)
	assert.Equal(t, "oracle://system:pass@localhost:1521/SIDDB", sid)
}

func TestParseResourceIDOracleIncludesConnStrServiceName(t *testing.T) {
	serviceOne := parseResourceID("oracle://system:pass@localhost:1521/?connStr=(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=localhost)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=FREEPDB1)))", types.DBTypeOracle)
	serviceTwo := parseResourceID("oracle://system:pass@localhost:1521/?connStr=(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=localhost)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=FREEPDB2)))", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@localhost:1521/FREEPDB1", serviceOne)
	assert.Equal(t, "oracle://system:pass@localhost:1521/FREEPDB2", serviceTwo)
	assert.NotEqual(t, serviceOne, serviceTwo)
}

func TestParseResourceIDOracleIncludesConnStrJDBCServiceName(t *testing.T) {
	resourceID := parseResourceID("oracle://system:pass@localhost:1521/?connStr=jdbc:oracle:thin:@//localhost:1521/FREEPDB1", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@localhost:1521/FREEPDB1", resourceID)
}

func TestParseResourceIDOracleIncludesServerQueryIdentity(t *testing.T) {
	serverOne := parseResourceID("oracle://system:pass@placeholder:1521/?SERVER=db1:1521&SERVICE%20NAME=XE", types.DBTypeOracle)
	serverTwo := parseResourceID("oracle://system:pass@placeholder:1521/?SERVER=db2:1521&SERVICE%20NAME=XE", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@db1:1521/XE", serverOne)
	assert.Equal(t, "oracle://system:pass@db2:1521/XE", serverTwo)
	assert.NotEqual(t, serverOne, serverTwo)
}

func TestParseResourceIDOracleIncludesConnStrServerIdentity(t *testing.T) {
	serverOne := parseResourceID("oracle://system:pass@placeholder:1521/?connStr=(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=db1)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=XE)))", types.DBTypeOracle)
	serverTwo := parseResourceID("oracle://system:pass@placeholder:1521/?connStr=(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=db2)(PORT=1521))(CONNECT_DATA=(SERVICE_NAME=XE)))", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@db1:1521/XE", serverOne)
	assert.Equal(t, "oracle://system:pass@db2:1521/XE", serverTwo)
	assert.NotEqual(t, serverOne, serverTwo)
}

func TestParseResourceIDOracleStripsNonIdentityQuery(t *testing.T) {
	resourceID := parseResourceID("oracle://system:pass@localhost:1521/FREEPDB1?TRACE FILE=trace.log", types.DBTypeOracle)

	assert.Equal(t, "oracle://system:pass@localhost:1521/FREEPDB1", resourceID)
}

func TestParseOracleDBNameUsesQueryIdentity(t *testing.T) {
	cases := map[string]string{
		"oracle://system:pass@localhost:1521/?service name=FREEPDB1":                                        "FREEPDB1",
		"oracle://system:pass@localhost:1521/?SID=ORCL1":                                                    "ORCL1",
		"oracle://system:pass@localhost:1521/PATHDB?SERVICE NAME=QUERYDB":                                   "QUERYDB",
		"oracle://system:pass@localhost:1521/PATHDB?SERVICE NAME=QUERYDB&SID=SIDDB":                         "SIDDB",
		"oracle://system:pass@localhost:1521/?connStr=(DESCRIPTION=(CONNECT_DATA=(SERVICE_NAME=FREEPDB1)))": "FREEPDB1",
	}

	for dsn, expected := range cases {
		dbName, err := parseOracleDBName(dsn)
		assert.NoError(t, err)
		assert.Equal(t, expected, dbName)
	}
}
