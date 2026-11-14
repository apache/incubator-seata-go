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
	"reflect"
	"testing"

	"seata.apache.org/seata-go/v2/pkg/rm"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/mock"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/util/reflectx"
)

func initMockResourceManager(branchType branch.BranchType, ctrl *gomock.Controller) *mock.MockDataSourceManager {
	mockResourceMgr := mock.NewMockDataSourceManager(ctrl)
	mockResourceMgr.SetBranchType(branchType)
	mockResourceMgr.EXPECT().BranchRegister(gomock.Any(), gomock.Any()).AnyTimes().Return(int64(0), nil)
	rm.GetRmCacheInstance().RegisterResourceManager(mockResourceMgr)
	mockResourceMgr.EXPECT().RegisterResource(gomock.Any()).AnyTimes().Return(nil)

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

	connector, ok := fieldVal.(*seataATConnector)
	require.True(t, ok, "need return seata at connector")
	assert.Equal(t, types.DBTypeMySQL, connector.res.dbType)
	require.NotNil(t, connector.res.undoLogMgr)
	assert.Equal(t, types.DBTypeMySQL, connector.res.undoLogMgr.DBType())
	assert.Equal(t, SeataATMySQLDriver, db.Driver().(*seataATDriver).name)
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
	require.True(t, ok, "need return seata at connector")
	assert.Equal(t, types.DBTypePostgreSQL, connector.dbType)
	assert.Equal(t, "seata_go_test", connector.dbName)
	require.NotNil(t, connector.res.undoLogMgr)
	assert.Equal(t, types.DBTypePostgreSQL, connector.res.undoLogMgr.DBType())
	assert.Equal(t, SeataATPostgresDriver, db.Driver().(*seataATDriver).name)
}

func TestATOpenConnectorRejectsUnknownDriverAndBadDSN(t *testing.T) {
	manager := &ATSourceManager{rmRemoting: &registrationRemoting{register: func(rm.Resource) error {
		t.Fatal("invalid connector must not register a resource")
		return nil
	}}}
	registerResourceManagerForTest(t, manager)

	connector, err := (&seataATDriver{name: "unknown"}).OpenConnector("")
	require.ErrorContains(t, err, "unknown AT driver")
	require.Nil(t, connector)

	db, err := sql.Open(SeataATMySQLDriver, "invalid dsn")
	require.Error(t, err)
	require.Nil(t, db)
}

type closeTrackingMetaCache struct {
	resourceTestMetaReader
	destroyed bool
}

func (c *closeTrackingMetaCache) Destroy() error {
	c.destroyed = true
	return nil
}

func TestATOpenConnectorRegistrationFailureClosesUnpublishedResource(t *testing.T) {
	dsn := "root:pass@tcp(mock:3306)/test"
	registerErr := errors.New("TC unavailable")
	var published *DBResource
	manager := &ATSourceManager{}
	manager.rmRemoting = &registrationRemoting{register: func(resource rm.Resource) error {
		cached, exists := manager.resourceCache.Load(resource.GetResourceId())
		require.Equal(t, published != nil, exists)
		if exists {
			require.Same(t, published, cached)
		}
		return registerErr
	}}
	registerResourceManagerForTest(t, manager)

	d := &seataDriver{branchType: branch.BranchTypeAT, transType: types.ATMode, descriptor: mySQLDriverDescriptor}
	metaCache := &closeTrackingMetaCache{}
	d.descriptor.newTableMetaCache = func(*sql.DB, string) datasource.TableMetaCache { return metaCache }
	target := disconnectedTestConnector{}
	for _, replace := range []bool{false, true} {
		if replace {
			published = &DBResource{resourceID: parseResourceID(dsn)}
			manager.resourceCache.Store(published.resourceID, published)
		}
		pool := sql.OpenDB(target)
		metaCache.destroyed = false
		connector, err := d.getOpenConnectorProxy(target, types.DBTypeMySQL, pool, dsn)
		require.ErrorIs(t, err, registerErr)
		require.Nil(t, connector)
		require.True(t, metaCache.destroyed)
		require.ErrorContains(t, pool.Ping(), "database is closed")
		cached, exists := manager.resourceCache.Load(parseResourceID(dsn))
		require.Equal(t, replace, exists)
		if exists {
			require.Same(t, published, cached)
		}
	}
}

func TestATOpenConnectorRequiresUndoManager(t *testing.T) {
	d := &seataDriver{branchType: branch.BranchTypeAT}
	target := disconnectedTestConnector{}
	pool := sql.OpenDB(target)
	t.Cleanup(func() { _ = pool.Close() })
	connector, err := d.getOpenConnectorProxy(target, types.DBTypeOracle, pool, "")
	require.ErrorContains(t, err, "resolve AT undo log manager for DBTypeOracle")
	require.Nil(t, connector)
	require.ErrorContains(t, pool.Ping(), "database is closed")
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
