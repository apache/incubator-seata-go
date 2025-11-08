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

package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSeataError_Error tests the Error() method of SeataError
func TestSeataError_Error(t *testing.T) {
	tests := []struct {
		name     string
		code     TransactionErrorCode
		message  string
		parent   error
		expected string
	}{
		{
			name:     "error with parent error",
			code:     TransactionErrorCodeBeginFailed,
			message:  "transaction failed",
			parent:   errors.New("database connection lost"),
			expected: "SeataError code 1, msg transaction failed, parent msg is database connection lost",
		},
		{
			name:     "error without parent error",
			code:     TransactionErrorCodeLockKeyConflict,
			message:  "timeout occurred",
			parent:   nil,
			expected: "SeataError code 2, msg timeout occurred, parent msg is %!s(<nil>)",
		},
		{
			name:     "error with empty message",
			code:     TransactionErrorCodeUnknown,
			message:  "",
			parent:   errors.New("some error"),
			expected: "SeataError code 0, msg , parent msg is some error",
		},
		{
			name:     "error with zero code and no parent",
			code:     TransactionErrorCodeUnknown,
			message:  "empty error",
			parent:   nil,
			expected: "SeataError code 0, msg empty error, parent msg is %!s(<nil>)",
		},
		{
			name:     "error with formatted parent error",
			code:     TransactionErrorCode(100),
			message:  "rollback failed",
			parent:   fmt.Errorf("nested error: %w", errors.New("root cause")),
			expected: "SeataError code 100, msg rollback failed, parent msg is nested error: root cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seataErr := SeataError{
				Code:    tt.code,
				Message: tt.message,
				Parent:  tt.parent,
			}

			got := seataErr.Error()
			assert.Equal(t, tt.expected, got)
		})
	}
}

// TestNew tests the New() constructor function
func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		code   TransactionErrorCode
		msg    string
		parent error
	}{
		{
			name:   "create error with all fields",
			code:   TransactionErrorCodeBeginFailed,
			msg:    "test message",
			parent: errors.New("parent error"),
		},
		{
			name:   "create error without parent",
			code:   TransactionErrorCodeLockKeyConflict,
			msg:    "test message 2",
			parent: nil,
		},
		{
			name:   "create error with empty message",
			code:   TransactionErrorCodeIO,
			msg:    "",
			parent: errors.New("parent"),
		},
		{
			name:   "create error with zero code",
			code:   TransactionErrorCodeUnknown,
			msg:    "zero code message",
			parent: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.code, tt.msg, tt.parent)

			assert.NotNil(t, got)
			assert.Equal(t, tt.code, got.Code)
			assert.Equal(t, tt.msg, got.Message)
			assert.Equal(t, tt.parent, got.Parent)
		})
	}
}

// TestNew_ReturnsPointer tests that New() returns a pointer
func TestNew_ReturnsPointer(t *testing.T) {
	err1 := New(TransactionErrorCodeBeginFailed, "message1", nil)
	err2 := New(TransactionErrorCodeBeginFailed, "message1", nil)

	// Two separate calls should return different pointers (compare addresses)
	if err1 == err2 {
		t.Error("New() should return different pointers for different calls")
	}
}

// TestSeataError_AsError tests that SeataError implements error interface
func TestSeataError_AsError(t *testing.T) {
	var _ error = SeataError{}
	var _ error = &SeataError{}

	seataErr := New(TransactionErrorCodeBeginFailed, "test", nil)
	var err error = seataErr

	assert.NotNil(t, err, "SeataError should implement error interface")
	assert.NotEmpty(t, err.Error(), "SeataError.Error() should return non-empty string")
}

// TestSeataError_ErrorWithNestedSeataError tests nested SeataError
func TestSeataError_ErrorWithNestedSeataError(t *testing.T) {
	parentErr := New(TransactionErrorCodeBeginFailed, "parent error", nil)
	childErr := New(TransactionErrorCodeLockKeyConflict, "child error", parentErr)

	expectedMsg := "SeataError code 2, msg child error, parent msg is SeataError code 1, msg parent error, parent msg is %!s(<nil>)"
	assert.Equal(t, expectedMsg, childErr.Error())
}

// TestSeataError_Fields tests direct field access
func TestSeataError_Fields(t *testing.T) {
	code := TransactionErrorCodeBranchRegisterFailed
	msg := "test message"
	parent := errors.New("parent error")

	seataErr := SeataError{
		Code:    code,
		Message: msg,
		Parent:  parent,
	}

	assert.Equal(t, code, seataErr.Code)
	assert.Equal(t, msg, seataErr.Message)
	assert.Equal(t, parent, seataErr.Parent)
}

// TestSeataError_LargeCode tests with large error code values
func TestSeataError_LargeCode(t *testing.T) {
	tests := []struct {
		name string
		code TransactionErrorCode
	}{
		{"max int32 value", TransactionErrorCode(2147483647)},
		{"negative value", TransactionErrorCode(-1)},
		{"large negative", TransactionErrorCode(-999999)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, "test", nil)
			assert.Equal(t, tt.code, err.Code)

			// Should not panic
			assert.NotPanics(t, func() {
				_ = err.Error()
			})
		})
	}
}

// TestSeataError_LongMessage tests with long error messages
func TestSeataError_LongMessage(t *testing.T) {
	longMsg := string(make([]byte, 10000))
	err := New(TransactionErrorCodeBeginFailed, longMsg, nil)

	assert.Equal(t, longMsg, err.Message, "Long message not preserved correctly")

	// Should not panic with long message
	assert.NotPanics(t, func() {
		_ = err.Error()
	})
}

// TestSeataError_SpecialCharacters tests messages with special characters
func TestSeataError_SpecialCharacters(t *testing.T) {
	specialMsgs := []string{
		"message with\nnewline",
		"message with\ttab",
		"message with 中文字符",
		"message with emoji 🚀",
		"message with \"quotes\"",
		"message with 'single quotes'",
	}

	for _, msg := range specialMsgs {
		t.Run(msg, func(t *testing.T) {
			err := New(TransactionErrorCodeBeginFailed, msg, nil)
			assert.Equal(t, msg, err.Message, "Special character message not preserved")

			// Should not panic
			assert.NotPanics(t, func() {
				_ = err.Error()
			})
		})
	}
}

// TestSeataError_WithRealErrorCodes tests with actual error codes from the package
func TestSeataError_WithRealErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		code TransactionErrorCode
	}{
		{"Unknown", TransactionErrorCodeUnknown},
		{"BeginFailed", TransactionErrorCodeBeginFailed},
		{"LockKeyConflict", TransactionErrorCodeLockKeyConflict},
		{"IO", TransactionErrorCodeIO},
		{"BranchRollbackFailedRetriable", TransactionErrorCodeBranchRollbackFailedRetriable},
		{"BranchRollbackFailedUnretriable", TransactionErrorCodeBranchRollbackFailedUnretriable},
		{"BranchRegisterFailed", TransactionErrorCodeBranchRegisterFailed},
		{"BranchReportFailed", TransactionErrorCodeBranchReportFailed},
		{"LockableCheckFailed", TransactionErrorCodeLockableCheckFailed},
		{"BranchTransactionNotExist", TransactionErrorCodeBranchTransactionNotExist},
		{"GlobalTransactionNotExist", TransactionErrorCodeGlobalTransactionNotExist},
		{"GlobalTransactionNotActive", TransactionErrorCodeGlobalTransactionNotActive},
		{"GlobalTransactionStatusInvalid", TransactionErrorCodeGlobalTransactionStatusInvalid},
		{"FailedToSendBranchCommitRequest", TransactionErrorCodeFailedToSendBranchCommitRequest},
		{"FailedToSendBranchRollbackRequest", TransactionErrorCodeFailedToSendBranchRollbackRequest},
		{"FailedToAddBranch", TransactionErrorCodeFailedToAddBranch},
		{"FailedLockGlobalTranscation", TransactionErrorCodeFailedLockGlobalTranscation},
		{"FailedWriteSession", TransactionErrorCodeFailedWriteSession},
		{"FailedStore", FailedStore},
		{"LockKeyConflictFailFast", LockKeyConflictFailFast},
		{"TccFenceDbDuplicateKeyError", TccFenceDbDuplicateKeyError},
		{"RollbackFenceError", RollbackFenceError},
		{"CommitFenceError", CommitFenceError},
		{"TccFenceDbError", TccFenceDbError},
		{"PrepareFenceError", PrepareFenceError},
		{"FenceBusinessError", FenceBusinessError},
		{"FencePhaseError", FencePhaseError},
		{"SQLUndoDirtyError", SQLUndoDirtyError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, "test message for "+tt.name, nil)
			assert.NotNil(t, err)
			assert.Equal(t, tt.code, err.Code)
			assert.Contains(t, err.Error(), "test message for "+tt.name)
		})
	}
}

// TestSeataError_NilParentFormatting tests the specific formatting of nil parent
func TestSeataError_NilParentFormatting(t *testing.T) {
	err := New(TransactionErrorCodeBeginFailed, "test", nil)
	errMsg := err.Error()

	// Verify that nil parent is formatted as %!s(<nil>) due to the %s format verb
	assert.Contains(t, errMsg, "parent msg is %!s(<nil>)")
}

// TestSeataError_ErrorChaining tests error chaining with multiple levels
func TestSeataError_ErrorChaining(t *testing.T) {
	rootErr := errors.New("root cause")
	level1Err := New(TransactionErrorCodeIO, "level 1", rootErr)
	level2Err := New(TransactionErrorCodeBeginFailed, "level 2", level1Err)
	level3Err := New(TransactionErrorCodeLockKeyConflict, "level 3", level2Err)

	errMsg := level3Err.Error()
	assert.Contains(t, errMsg, "level 3")
	assert.Contains(t, errMsg, "level 2")
}

// TestSeataError_MultipleInstancesIndependent tests that multiple error instances are independent
func TestSeataError_MultipleInstancesIndependent(t *testing.T) {
	err1 := New(TransactionErrorCodeBeginFailed, "error 1", nil)
	err2 := New(TransactionErrorCodeLockKeyConflict, "error 2", nil)

	// Modify err1 should not affect err2
	err1.Message = "modified"
	assert.Equal(t, "modified", err1.Message)
	assert.Equal(t, "error 2", err2.Message)
}

// TestSeataError_ZeroValue tests zero value of SeataError
func TestSeataError_ZeroValue(t *testing.T) {
	var err SeataError

	assert.Equal(t, TransactionErrorCode(0), err.Code)
	assert.Equal(t, "", err.Message)
	assert.Nil(t, err.Parent)

	// Should not panic on zero value
	assert.NotPanics(t, func() {
		_ = err.Error()
	})
}

// TestSeataError_PointerVsValue tests pointer vs value receiver behavior
func TestSeataError_PointerVsValue(t *testing.T) {
	// Value type
	errValue := SeataError{
		Code:    TransactionErrorCodeBeginFailed,
		Message: "value error",
		Parent:  nil,
	}

	// Pointer type
	errPointer := &SeataError{
		Code:    TransactionErrorCodeBeginFailed,
		Message: "pointer error",
		Parent:  nil,
	}

	// Both should implement error interface
	var _ error = errValue
	var _ error = errPointer

	assert.Contains(t, errValue.Error(), "value error")
	assert.Contains(t, errPointer.Error(), "pointer error")
}

// TestNew_WithDifferentParentTypes tests New with various parent error types
func TestNew_WithDifferentParentTypes(t *testing.T) {
	tests := []struct {
		name   string
		parent error
	}{
		{
			name:   "standard error",
			parent: errors.New("standard error"),
		},
		{
			name:   "formatted error",
			parent: fmt.Errorf("formatted: %s", "error"),
		},
		{
			name:   "wrapped error",
			parent: fmt.Errorf("wrapped: %w", errors.New("inner")),
		},
		{
			name:   "seata error as parent",
			parent: New(TransactionErrorCodeIO, "parent seata error", nil),
		},
		{
			name:   "nil parent",
			parent: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(TransactionErrorCodeBeginFailed, "test", tt.parent)
			assert.NotNil(t, err)
			assert.Equal(t, tt.parent, err.Parent)
			assert.NotPanics(t, func() {
				_ = err.Error()
			})
		})
	}
}

// TestSeataError_ConcurrentAccess tests concurrent access to SeataError
func TestSeataError_ConcurrentAccess(t *testing.T) {
	err := New(TransactionErrorCodeBeginFailed, "concurrent test", nil)

	done := make(chan bool)

	// Multiple goroutines reading the error
	for i := 0; i < 10; i++ {
		go func() {
			_ = err.Error()
			_ = err.Code
			_ = err.Message
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSeataError_ErrorStringFormat tests the exact format of error string
func TestSeataError_ErrorStringFormat(t *testing.T) {
	tests := []struct {
		name     string
		code     TransactionErrorCode
		message  string
		parent   error
		contains []string
	}{
		{
			name:    "format check with parent",
			code:    TransactionErrorCode(42),
			message: "test message",
			parent:  errors.New("parent message"),
			contains: []string{
				"SeataError code 42",
				"msg test message",
				"parent msg is parent message",
			},
		},
		{
			name:    "format check without parent",
			code:    TransactionErrorCode(99),
			message: "another test",
			parent:  nil,
			contains: []string{
				"SeataError code 99",
				"msg another test",
				"parent msg is",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, tt.message, tt.parent)
			errStr := err.Error()

			for _, substr := range tt.contains {
				assert.Contains(t, errStr, substr)
			}
		})
	}
}

// BenchmarkSeataError_Error benchmarks the Error() method
func BenchmarkSeataError_Error(b *testing.B) {
	err := New(TransactionErrorCodeBeginFailed, "benchmark message", errors.New("parent"))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// BenchmarkNew benchmarks the New() constructor
func BenchmarkNew(b *testing.B) {
	parent := errors.New("parent error")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = New(TransactionErrorCodeBeginFailed, "message", parent)
	}
}

// BenchmarkSeataError_ErrorNoParent benchmarks Error() without parent
func BenchmarkSeataError_ErrorNoParent(b *testing.B) {
	err := New(TransactionErrorCodeBeginFailed, "benchmark message", nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

// BenchmarkNew_NoParent benchmarks New() without parent
func BenchmarkNew_NoParent(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = New(TransactionErrorCodeBeginFailed, "message", nil)
	}
}
