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

package mock

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func TestNewMockDataSourceManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	assert.NotNil(t, mockDSM)
	assert.NotNil(t, mockDSM.EXPECT())
}

func TestMockDataSourceManager_BranchCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	ctx := context.Background()
	resource := rm.BranchResource{
		ResourceId: "test-resource",
		BranchId:   123,
		Xid:        "test-xid",
	}

	expectedStatus := branch.BranchStatus(branch.BranchStatusPhasetwoCommitted)
	mockDSM.EXPECT().BranchCommit(ctx, resource).Return(expectedStatus, nil)

	status, err := mockDSM.BranchCommit(ctx, resource)
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
}

func TestMockDataSourceManager_BranchRollback(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	ctx := context.Background()
	resource := rm.BranchResource{
		ResourceId: "test-resource",
		BranchId:   456,
		Xid:        "test-xid",
	}

	expectedStatus := branch.BranchStatus(branch.BranchStatusPhasetwoRollbacked)
	mockDSM.EXPECT().BranchRollback(ctx, resource).Return(expectedStatus, nil)

	status, err := mockDSM.BranchRollback(ctx, resource)
	assert.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
}

func TestMockDataSourceManager_BranchRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	ctx := context.Background()
	param := rm.BranchRegisterParam{
		Xid:        "test-xid",
		ResourceId: "test-resource",
		LockKeys:   "key1,key2",
	}

	expectedBranchID := int64(789)
	mockDSM.EXPECT().BranchRegister(ctx, param).Return(expectedBranchID, nil)

	branchID, err := mockDSM.BranchRegister(ctx, param)
	assert.NoError(t, err)
	assert.Equal(t, expectedBranchID, branchID)
}

func TestMockDataSourceManager_BranchReport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	ctx := context.Background()
	param := rm.BranchReportParam{
		Xid:      "test-xid",
		BranchId: 123,
		Status:   branch.BranchStatus(branch.BranchStatusPhasetwoCommitted),
	}

	mockDSM.EXPECT().BranchReport(ctx, param).Return(nil)

	err := mockDSM.BranchReport(ctx, param)
	assert.NoError(t, err)
}

func TestMockDataSourceManager_CreateTableMetaCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	ctx := context.Background()
	resID := "test-resource"
	dbType := types.DBTypeMySQL
	db := &sql.DB{}

	expectedCache := datasource.TableMetaCache(nil)
	mockDSM.EXPECT().CreateTableMetaCache(ctx, resID, dbType, db).Return(expectedCache, nil)

	cache, err := mockDSM.CreateTableMetaCache(ctx, resID, dbType, db)
	assert.NoError(t, err)
	assert.Equal(t, expectedCache, cache)
}

func TestMockDataSourceManager_GetSetBranchType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	expectedBranchType := branch.BranchTypeAT

	mockDSM.SetBranchType(expectedBranchType)
	branchType := mockDSM.GetBranchType()
	assert.Equal(t, expectedBranchType, branchType)
}

func TestMockDataSourceManager_GetCachedResources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	expectedMap := &sync.Map{}

	mockDSM.EXPECT().GetCachedResources().Return(expectedMap)

	resources := mockDSM.GetCachedResources()
	assert.Equal(t, expectedMap, resources)
}

func TestMockDataSourceManager_LockQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	ctx := context.Background()
	param := rm.LockQueryParam{
		Xid:        "test-xid",
		ResourceId: "test-resource",
		LockKeys:   "key1,key2",
	}

	expectedResult := true
	mockDSM.EXPECT().LockQuery(ctx, param).Return(expectedResult, nil)

	result, err := mockDSM.LockQuery(ctx, param)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestMockDataSourceManager_RegisterResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	resource := rm.Resource(nil)

	mockDSM.EXPECT().RegisterResource(resource).Return(nil)

	err := mockDSM.RegisterResource(resource)
	assert.NoError(t, err)
}

func TestMockDataSourceManager_UnregisterResource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDSM := NewMockDataSourceManager(ctrl)
	resource := rm.Resource(nil)

	mockDSM.EXPECT().UnregisterResource(resource).Return(nil)

	err := mockDSM.UnregisterResource(resource)
	assert.NoError(t, err)
}
