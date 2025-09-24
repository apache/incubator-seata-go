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

package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func TestRegisterCommonHook(t *testing.T) {
	originalLen := len(commonHook)

	testHook := &mockSQLHook{sqlType: types.SQLTypeSelect}
	RegisterCommonHook(testHook)

	assert.Equal(t, originalLen+1, len(commonHook))
	assert.Equal(t, testHook, commonHook[originalLen])

	// Clean up
	commonHook = commonHook[:originalLen]
}

func TestCleanCommonHook(t *testing.T) {
	// Add some hooks
	testHook := &mockSQLHook{sqlType: types.SQLTypeSelect}
	RegisterCommonHook(testHook)

	assert.True(t, len(commonHook) > 0)

	CleanCommonHook()

	assert.Equal(t, 0, len(commonHook))
}

func TestRegisterHook(t *testing.T) {
	// Test registering a hook with a valid SQL type
	testHook := &mockSQLHook{sqlType: types.SQLTypeSelect}

	// Make sure the slot is initially empty or create it
	originalLen := len(hookSolts[types.SQLTypeSelect])
	RegisterHook(testHook)

	assert.Equal(t, originalLen+1, len(hookSolts[types.SQLTypeSelect]))
	assert.Equal(t, testHook, hookSolts[types.SQLTypeSelect][originalLen])

	// Test registering a hook with unknown SQL type (should not be added)
	unknownHook := &mockSQLHook{sqlType: types.SQLTypeUnknown}
	originalSelectLen := len(hookSolts[types.SQLTypeSelect])
	RegisterHook(unknownHook)

	// Length should remain the same
	assert.Equal(t, originalSelectLen, len(hookSolts[types.SQLTypeSelect]))
}

func TestSQLHookInterface(t *testing.T) {
	hook := &mockSQLHook{
		sqlType: types.SQLTypeInsert,
	}

	assert.Equal(t, types.SQLType(types.SQLTypeInsert), hook.Type())

	ctx := context.Background()
	execCtx := &types.ExecContext{}

	// Test Before method
	err := hook.Before(ctx, execCtx)
	assert.NoError(t, err)
	assert.True(t, hook.beforeCalled)

	// Test After method
	err = hook.After(ctx, execCtx)
	assert.NoError(t, err)
	assert.True(t, hook.afterCalled)
}
