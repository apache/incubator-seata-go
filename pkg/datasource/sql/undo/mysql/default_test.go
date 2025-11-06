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

package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestInitUndoLogManager(t *testing.T) {
	// Call InitUndoLogManager
	InitUndoLogManager()

	// Verify that the manager was registered
	manager, err := undo.GetUndoLogManager(types.DBTypeMySQL)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, types.DBTypeMySQL, manager.DBType())
}

func TestInitUndoLogManager_Multiple(t *testing.T) {
	// Calling InitUndoLogManager multiple times should not panic
	// The second call will be a no-op since the manager is already registered
	InitUndoLogManager()
	InitUndoLogManager()

	// Verify manager is still accessible
	manager, err := undo.GetUndoLogManager(types.DBTypeMySQL)
	assert.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestInitUndoLogManager_ManagerType(t *testing.T) {
	InitUndoLogManager()

	manager, err := undo.GetUndoLogManager(types.DBTypeMySQL)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// Verify it's the correct type
	_, ok := manager.(*undoLogManager)
	assert.True(t, ok, "Manager should be of type *undoLogManager")
}
