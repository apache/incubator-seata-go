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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestNewMockTableMetaCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := NewMockTableMetaCache(ctrl)
	assert.NotNil(t, mockCache)
	assert.NotNil(t, mockCache.EXPECT())
}

func TestMockTableMetaCache_GetTableMeta(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := NewMockTableMetaCache(ctrl)
	ctx := context.Background()
	tableName := "test_table"
	key := types.TableMetaKey{DBName: "test_db", TableName: tableName}

	expectedTableMeta := &types.TableMeta{
		TableName: tableName,
		Columns: map[string]types.ColumnMeta{
			"id": {
				ColumnName: "id",
				ColumnType: "BIGINT",
			},
		},
	}

	mockCache.EXPECT().GetTableMeta(ctx, key).Return(expectedTableMeta, nil)

	tableMeta, err := mockCache.GetTableMeta(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, expectedTableMeta, tableMeta)
	assert.Equal(t, tableName, tableMeta.TableName)
}

func TestMockTableMetaCache_ResolveTableMetaKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := NewMockTableMetaCache(ctrl)
	ctx := context.Background()
	ref := types.TableRef{TableName: "test_table"}
	key := types.TableMetaKey{DBName: "test_db", TableName: "test_table"}

	mockCache.EXPECT().ResolveTableMetaKey(ctx, nil, ref).Return(key, nil)

	got, err := mockCache.ResolveTableMetaKey(ctx, nil, ref)
	assert.NoError(t, err)
	assert.Equal(t, key, got)
}

func TestMockTableMetaCache_Destroy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := NewMockTableMetaCache(ctrl)

	mockCache.EXPECT().Destroy().Return(nil)

	err := mockCache.Destroy()
	assert.NoError(t, err)
}
