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

package types

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/branch"
)

func TestParseIndexType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected IndexType
	}{
		{"PRIMARY_KEY", "PRIMARY_KEY", IndexTypePrimaryKey},
		{"primary_key lowercase", "primary_key", IndexTypeNull},
		{"NULL", "NULL", IndexTypeNull},
		{"empty string", "", IndexTypeNull},
		{"unknown type", "UNKNOWN", IndexTypeNull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseIndexType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexType_MarshalText(t *testing.T) {
	tests := []struct {
		name     string
		input    IndexType
		expected []byte
	}{
		{"IndexTypePrimaryKey", IndexTypePrimaryKey, []byte("PRIMARY_KEY")},
		{"IndexTypeNull", IndexTypeNull, []byte("NULL")},
		{"Unknown type", IndexType(999), []byte("NULL")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.input.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIndexType_UnmarshalText(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    IndexType
		expectError bool
	}{
		{"PRIMARY_KEY", []byte("PRIMARY_KEY"), IndexTypePrimaryKey, false},
		{"NULL", []byte("NULL"), IndexTypeNull, false},
		{"invalid type", []byte("INVALID"), IndexTypeNull, true},
		{"empty", []byte(""), IndexTypeNull, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var indexType IndexType
			err := indexType.UnmarshalText(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid index type")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, indexType)
			}
		})
	}
}

func TestDBTypeConstants(t *testing.T) {
	assert.Equal(t, DBType(1), DBTypeUnknown)
	assert.Equal(t, DBType(2), DBTypeMySQL)
	assert.Equal(t, DBType(3), DBTypePostgreSQL)
	assert.Equal(t, DBType(4), DBTypeSQLServer)
	assert.Equal(t, DBType(5), DBTypeOracle)
	assert.Equal(t, DBType(6), DBTypeMARIADB)
}

func TestParseDBType(t *testing.T) {
	tests := []struct {
		name       string
		driverName string
		expected   DBType
	}{
		{"mysql", "mysql", DBTypeMySQL},
		{"MySQL uppercase", "MySQL", DBTypeMySQL},
		{"MYSQL", "MYSQL", DBTypeMySQL},
		{"postgres", "postgres", DBTypeUnknown},
		{"unknown", "unknown", DBTypeUnknown},
		{"empty", "", DBTypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDBType(tt.driverName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionModeConstants(t *testing.T) {
	assert.Equal(t, TransactionMode(1), Local)
	assert.Equal(t, TransactionMode(2), XAMode)
	assert.Equal(t, TransactionMode(3), ATMode)
}

func TestTransactionMode_BranchType(t *testing.T) {
	tests := []struct {
		name     string
		mode     TransactionMode
		expected branch.BranchType
	}{
		{"XAMode", XAMode, branch.BranchTypeXA},
		{"ATMode", ATMode, branch.BranchTypeAT},
		{"Local", Local, branch.BranchTypeUnknow},
		{"Unknown", TransactionMode(99), branch.BranchTypeUnknow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mode.BranchType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTxCtx(t *testing.T) {
	ctx := NewTxCtx()

	assert.NotNil(t, ctx)
	assert.NotNil(t, ctx.LockKeys)
	assert.Equal(t, 0, len(ctx.LockKeys))
	assert.Equal(t, Local, ctx.TransactionMode)
	assert.NotEmpty(t, ctx.LocalTransID)
	assert.NotNil(t, ctx.RoundImages)
}

func TestTransactionContext_HasUndoLog(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *TransactionContext
		expected bool
	}{
		{
			name: "AT mode with images",
			ctx: &TransactionContext{
				TransactionMode: ATMode,
				RoundImages:     &RoundRecordImage{}, // Assuming empty is false for HasUndoLog
			},
			expected: false, // Empty RoundRecordImage
		},
		{
			name: "XA mode",
			ctx: &TransactionContext{
				TransactionMode: XAMode,
				RoundImages:     &RoundRecordImage{},
			},
			expected: false,
		},
		{
			name: "Local mode",
			ctx: &TransactionContext{
				TransactionMode: Local,
				RoundImages:     &RoundRecordImage{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.HasUndoLog()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionContext_HasLockKey(t *testing.T) {
	tests := []struct {
		name     string
		ctx      *TransactionContext
		expected bool
	}{
		{
			name: "with lock keys",
			ctx: &TransactionContext{
				LockKeys: map[string]struct{}{
					"key1": {},
					"key2": {},
				},
			},
			expected: true,
		},
		{
			name: "empty lock keys",
			ctx: &TransactionContext{
				LockKeys: map[string]struct{}{},
			},
			expected: false,
		},
		{
			name: "nil lock keys",
			ctx: &TransactionContext{
				LockKeys: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.HasLockKey()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionContext_OpenGlobalTransaction(t *testing.T) {
	tests := []struct {
		name     string
		mode     TransactionMode
		expected bool
	}{
		{"Local", Local, false},
		{"XAMode", XAMode, true},
		{"ATMode", ATMode, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &TransactionContext{
				TransactionMode: tt.mode,
			}
			result := ctx.OpenGlobalTransaction()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionContext_IsBranchRegistered(t *testing.T) {
	tests := []struct {
		name     string
		branchID uint64
		expected bool
	}{
		{"registered", 123, true},
		{"not registered", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &TransactionContext{
				BranchID: tt.branchID,
			}
			result := ctx.IsBranchRegistered()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryResult(t *testing.T) {
	rows := &mockRows{}
	result := &queryResult{Rows: rows}

	assert.Equal(t, rows, result.GetRows())
	assert.Panics(t, func() {
		result.GetResult()
	})
}

func TestWriteResult(t *testing.T) {
	sqlResult := &mockResult{}
	result := &writeResult{Result: sqlResult}

	assert.Equal(t, sqlResult, result.GetResult())
	assert.Panics(t, func() {
		result.GetRows()
	})
}

func TestNewResult(t *testing.T) {
	t.Run("with rows", func(t *testing.T) {
		rows := &mockRows{}
		result := NewResult(WithRows(rows))

		queryRes, ok := result.(*queryResult)
		assert.True(t, ok)
		assert.Equal(t, rows, queryRes.Rows)
	})

	t.Run("with result", func(t *testing.T) {
		sqlResult := &mockResult{}
		result := NewResult(WithResult(sqlResult))

		writeRes, ok := result.(*writeResult)
		assert.True(t, ok)
		assert.Equal(t, sqlResult, writeRes.Result)
	})

	t.Run("with both (result takes precedence)", func(t *testing.T) {
		rows := &mockRows{}
		sqlResult := &mockResult{}
		result := NewResult(WithRows(rows), WithResult(sqlResult))

		writeRes, ok := result.(*writeResult)
		assert.True(t, ok)
		assert.Equal(t, sqlResult, writeRes.Result)
	})

	t.Run("with neither (panics)", func(t *testing.T) {
		assert.Panics(t, func() {
			NewResult()
		})
	})
}

func TestWithRows(t *testing.T) {
	rows := &mockRows{}
	opt := &option{}

	WithRows(rows)(opt)
	assert.Equal(t, rows, opt.rows)
}

func TestWithResult(t *testing.T) {
	result := &mockResult{}
	opt := &option{}

	WithResult(result)(opt)
	assert.Equal(t, result, opt.ret)
}

// Mock types for testing
type mockRows struct{}

func (m *mockRows) Columns() []string              { return nil }
func (m *mockRows) Close() error                   { return nil }
func (m *mockRows) Next(dest []driver.Value) error { return nil }

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 0, nil }

func TestBranchPhaseConstants(t *testing.T) {
	assert.Equal(t, 0, BranchPhase_Unknown)
	assert.Equal(t, 1, BranchPhase_Done)
	assert.Equal(t, 2, BranchPhase_Failed)
}

func TestIndexConstants(t *testing.T) {
	// IndexPrimary starts from where DBType iota left off (after DBTypeMARIADB which is 6)
	// But since there are also BranchPhase constants in between, we need to check actual values
	assert.Equal(t, IndexType(10), IndexPrimary)
	assert.Equal(t, IndexType(11), IndexNormal)
	assert.Equal(t, IndexType(12), IndexUnique)
	assert.Equal(t, IndexType(13), IndexFullText)
}
