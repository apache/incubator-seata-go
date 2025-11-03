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

package base

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

func TestGetUndoLogTableName(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		expected string
	}{
		{
			name:     "default table name",
			config:   "",
			expected: " undo_log ",
		},
		{
			name:     "custom table name",
			config:   "custom_undo_log",
			expected: "custom_undo_log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalConfig := undo.UndoConfig.LogTable
			defer func() {
				undo.UndoConfig.LogTable = originalConfig
			}()

			undo.UndoConfig.LogTable = tt.config
			result := getUndoLogTableName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCheckUndoLogTableExistSql(t *testing.T) {
	originalConfig := undo.UndoConfig.LogTable
	defer func() {
		undo.UndoConfig.LogTable = originalConfig
	}()

	undo.UndoConfig.LogTable = "test_undo_log"
	result := getCheckUndoLogTableExistSql()
	expected := "SELECT 1 FROM test_undo_log LIMIT 1"
	assert.Equal(t, expected, result)
}

func TestGetInsertUndoLogSql(t *testing.T) {
	originalConfig := undo.UndoConfig.LogTable
	defer func() {
		undo.UndoConfig.LogTable = originalConfig
	}()

	undo.UndoConfig.LogTable = "test_undo_log"
	result := getInsertUndoLogSql()
	expected := "INSERT INTO test_undo_log(branch_id,xid,context,rollback_info,log_status,log_created,log_modified) VALUES (?, ?, ?, ?, ?, now(6), now(6))"
	assert.Equal(t, expected, result)
}

func TestGetSelectUndoLogSql(t *testing.T) {
	originalConfig := undo.UndoConfig.LogTable
	defer func() {
		undo.UndoConfig.LogTable = originalConfig
	}()

	undo.UndoConfig.LogTable = "test_undo_log"
	result := getSelectUndoLogSql()
	expected := "SELECT `branch_id`,`xid`,`context`,`rollback_info`,`log_status` FROM test_undo_log WHERE branch_id = ? AND xid = ? FOR UPDATE"
	assert.Equal(t, expected, result)
}

func TestGetDeleteUndoLogSql(t *testing.T) {
	originalConfig := undo.UndoConfig.LogTable
	defer func() {
		undo.UndoConfig.LogTable = originalConfig
	}()

	undo.UndoConfig.LogTable = "test_undo_log"
	result := getDeleteUndoLogSql()
	expected := "DELETE FROM test_undo_log WHERE branch_id = ? AND xid = ?"
	assert.Equal(t, expected, result)
}

func TestNewBaseUndoLogManager(t *testing.T) {
	manager := NewBaseUndoLogManager()
	assert.NotNil(t, manager)
	assert.IsType(t, &BaseUndoLogManager{}, manager)
}

func TestBaseUndoLogManager_Init(t *testing.T) {
	manager := NewBaseUndoLogManager()
	assert.NotPanics(t, func() {
		manager.Init()
	})
}

func TestBaseUndoLogManager_canUndo(t *testing.T) {
	manager := NewBaseUndoLogManager()

	tests := []struct {
		name  string
		state int32
		want  bool
	}{
		{
			name:  "normal status - can undo",
			state: UndoLogStatusNormal,
			want:  true,
		},
		{
			name:  "global finished status - cannot undo",
			state: UndoLogStatusGlobalFinished,
			want:  false,
		},
		{
			name:  "invalid status",
			state: 999,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.canUndo(tt.state)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestBaseUndoLogManager_UnmarshalContext(t *testing.T) {
	manager := NewBaseUndoLogManager()

	tests := []struct {
		name        string
		input       []byte
		expected    map[string]string
		expectError bool
	}{
		{
			name:  "valid json",
			input: []byte(`{"key1":"value1","key2":"value2"}`),
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expectError: false,
		},
		{
			name:        "invalid json",
			input:       []byte(`{"key1":value1`),
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty json",
			input:       []byte(`{}`),
			expected:    map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.UnmarshalContext(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBaseUndoLogManager_getSerializer(t *testing.T) {
	manager := NewBaseUndoLogManager()

	tests := []struct {
		name     string
		context  map[string]string
		expected string
	}{
		{
			name:     "nil context",
			context:  nil,
			expected: "",
		},
		{
			name:     "empty context",
			context:  map[string]string{},
			expected: "",
		},
		{
			name: "context with serializer",
			context: map[string]string{
				"serializerKey": "json",
			},
			expected: "json",
		},
		{
			name: "context without serializer",
			context: map[string]string{
				"otherKey": "value",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getSerializer(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBaseUndoLogManager_getBatchDeleteUndoLogSql(t *testing.T) {
	manager := NewBaseUndoLogManager()

	originalConfig := undo.UndoConfig.LogTable
	defer func() {
		undo.UndoConfig.LogTable = originalConfig
	}()
	undo.UndoConfig.LogTable = "test_undo_log"

	tests := []struct {
		name        string
		xid         []string
		branchID    []int64
		expectError bool
		expected    string
	}{
		{
			name:        "empty xid",
			xid:         []string{},
			branchID:    []int64{1, 2},
			expectError: true,
		},
		{
			name:        "empty branchID",
			xid:         []string{"xid1", "xid2"},
			branchID:    []int64{},
			expectError: true,
		},
		{
			name:        "both empty",
			xid:         []string{},
			branchID:    []int64{},
			expectError: true,
		},
		{
			name:     "valid inputs",
			xid:      []string{"xid1", "xid2"},
			branchID: []int64{1, 2},
			expected: " DELETE FROM test_undo_log WHERE branch_id IN  (?,?)  AND xid IN  (?,?) ",
		},
		{
			name:     "single values",
			xid:      []string{"xid1"},
			branchID: []int64{1},
			expected: " DELETE FROM test_undo_log WHERE branch_id IN  (?)  AND xid IN  (?) ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.getBatchDeleteUndoLogSql(tt.xid, tt.branchID)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "xid or branch_id can't nil")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBaseUndoLogManager_appendInParam(t *testing.T) {
	manager := NewBaseUndoLogManager()

	tests := []struct {
		name     string
		size     int
		expected string
	}{
		{
			name:     "zero size",
			size:     0,
			expected: "",
		},
		{
			name:     "negative size",
			size:     -1,
			expected: "",
		},
		{
			name:     "single parameter",
			size:     1,
			expected: " (?) ",
		},
		{
			name:     "multiple parameters",
			size:     3,
			expected: " (?,?,?) ",
		},
		{
			name:     "five parameters",
			size:     5,
			expected: " (?,?,?,?,?) ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			manager.appendInParam(tt.size, &builder)
			result := builder.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInt64Slice2Str(t *testing.T) {
	tests := []struct {
		name        string
		values      interface{}
		sep         string
		expected    string
		expectError bool
	}{
		{
			name:     "valid int64 slice",
			values:   []int64{1, 2, 3, 4, 5},
			sep:      ",",
			expected: "1,2,3,4,5",
		},
		{
			name:     "single value",
			values:   []int64{42},
			sep:      ",",
			expected: "42",
		},
		{
			name:     "empty slice",
			values:   []int64{},
			sep:      ",",
			expected: "",
		},
		{
			name:     "different separator",
			values:   []int64{1, 2, 3},
			sep:      "|",
			expected: "1|2|3",
		},
		{
			name:        "invalid type",
			values:      []string{"1", "2", "3"},
			sep:         ",",
			expectError: true,
		},
		{
			name:        "nil value",
			values:      nil,
			sep:         ",",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Int64Slice2Str(tt.values, tt.sep)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "param type is fault")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBaseUndoLogManager_encodeDecodeUndoLogCtx(t *testing.T) {
	manager := NewBaseUndoLogManager()

	testContext := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	encoded := manager.encodeUndoLogCtx(testContext)
	assert.NotNil(t, encoded)

	decoded := manager.decodeUndoLogCtx(encoded)
	assert.Equal(t, testContext, decoded)
}

func TestBaseUndoLogManager_getRollbackInfo(t *testing.T) {
	manager := NewBaseUndoLogManager()

	tests := []struct {
		name           string
		rollbackInfo   []byte
		undoContext    map[string]string
		expectedOutput []byte
		expectError    bool
	}{
		{
			name:         "no compression",
			rollbackInfo: []byte("test data"),
			undoContext:  map[string]string{},
			expectedOutput: []byte("test data"),
		},
		{
			name:         "context without compressor",
			rollbackInfo: []byte("test data"),
			undoContext: map[string]string{
				"otherKey": "value",
			},
			expectedOutput: []byte("test data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.getRollbackInfo(tt.rollbackInfo, tt.undoContext)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, result)
			}
		})
	}
}