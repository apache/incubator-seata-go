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

package factor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestGetUndoExecutorHolder(t *testing.T) {
	tests := []struct {
		name        string
		dbType      types.DBType
		expectError bool
		resetMap    bool
	}{
		{
			name:        "mysql db type - success",
			dbType:      types.DBTypeMySQL,
			expectError: false,
			resetMap:    false,
		},
		{
			name:        "mysql db type - lazy init",
			dbType:      types.DBTypeMySQL,
			expectError: false,
			resetMap:    true,
		},
		{
			name:        "postgresql db type - not implemented",
			dbType:      types.DBTypePostgreSQL,
			expectError: true,
			resetMap:    false,
		},
		{
			name:        "oracle db type - not implemented",
			dbType:      types.DBTypeOracle,
			expectError: true,
			resetMap:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMap := undoExecutorHolderMap
			defer func() {
				undoExecutorHolderMap = originalMap
			}()

			if tt.resetMap {
				undoExecutorHolderMap = nil
			}

			result, err := GetUndoExecutorHolder(tt.dbType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Equal(t, ErrNotImplDBType, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestErrNotImplDBType(t *testing.T) {
	assert.Equal(t, "db type executor not implement", ErrNotImplDBType.Error())
}

func TestGetUndoExecutorHolder_LazyInit(t *testing.T) {
	originalMap := undoExecutorHolderMap
	defer func() {
		undoExecutorHolderMap = originalMap
	}()

	undoExecutorHolderMap = nil

	holder1, err1 := GetUndoExecutorHolder(types.DBTypeMySQL)
	assert.NoError(t, err1)
	assert.NotNil(t, holder1)

	assert.NotNil(t, undoExecutorHolderMap)
	assert.Contains(t, undoExecutorHolderMap, types.DBTypeMySQL)

	holder2, err2 := GetUndoExecutorHolder(types.DBTypeMySQL)
	assert.NoError(t, err2)
	assert.NotNil(t, holder2)
	assert.Equal(t, holder1, holder2)
}

func TestGetUndoExecutorHolder_MultipleDBTypes(t *testing.T) {
	originalMap := undoExecutorHolderMap
	defer func() {
		undoExecutorHolderMap = originalMap
	}()

	undoExecutorHolderMap = nil

	mysqlHolder, err := GetUndoExecutorHolder(types.DBTypeMySQL)
	assert.NoError(t, err)
	assert.NotNil(t, mysqlHolder)

	_, err = GetUndoExecutorHolder(types.DBTypePostgreSQL)
	assert.Error(t, err)
	assert.Equal(t, ErrNotImplDBType, err)

	mysqlHolder2, err := GetUndoExecutorHolder(types.DBTypeMySQL)
	assert.NoError(t, err)
	assert.Equal(t, mysqlHolder, mysqlHolder2)
}