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

func TestGetUndoExecutorHolder_MySQL(t *testing.T) {
	// Test getting MySQL undo executor holder
	holder, err := GetUndoExecutorHolder(types.DBTypeMySQL)

	assert.NoError(t, err)
	assert.NotNil(t, holder)
}

func TestGetUndoExecutorHolder_UnsupportedDBType(t *testing.T) {
	// Test unsupported database types
	testCases := []struct {
		name   string
		dbType types.DBType
	}{
		{"Unknown", types.DBTypeUnknown},
		{"PostgreSQL", types.DBTypePostgreSQL},
		{"SQLServer", types.DBTypeSQLServer},
		{"Oracle", types.DBTypeOracle},
		{"MariaDB", types.DBTypeMARIADB},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			holder, err := GetUndoExecutorHolder(tc.dbType)

			assert.Error(t, err)
			assert.Nil(t, holder)
			assert.Equal(t, ErrNotImplDBType, err)
		})
	}
}

func TestGetUndoExecutorHolder_LazyInit(t *testing.T) {
	// Reset the map to test lazy initialization
	undoExecutorHolderMap = nil

	holder1, err1 := GetUndoExecutorHolder(types.DBTypeMySQL)
	assert.NoError(t, err1)
	assert.NotNil(t, holder1)

	holder2, err2 := GetUndoExecutorHolder(types.DBTypeMySQL)
	assert.NoError(t, err2)
	assert.NotNil(t, holder2)
	assert.IsType(t, holder1, holder2)
}

func TestGetUndoExecutorHolder_MapNotNil(t *testing.T) {
	// Ensure the map is initialized
	_, _ = GetUndoExecutorHolder(types.DBTypeMySQL)

	assert.NotNil(t, undoExecutorHolderMap)
	assert.Contains(t, undoExecutorHolderMap, types.DBTypeMySQL)
}

func TestGetUndoExecutorHolder_InvalidDBType(t *testing.T) {
	// Test with an invalid/custom DB type value
	invalidDBType := types.DBType(999)

	holder, err := GetUndoExecutorHolder(invalidDBType)

	assert.Error(t, err)
	assert.Nil(t, holder)
	assert.Equal(t, ErrNotImplDBType, err)
}

func TestErrNotImplDBType(t *testing.T) {
	// Test the error constant
	assert.NotNil(t, ErrNotImplDBType)
	assert.Equal(t, "db type executor not implement", ErrNotImplDBType.Error())
}
