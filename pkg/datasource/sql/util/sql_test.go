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

package util

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mock driver.Rows implementation
type mockDriverRows struct {
	columns    []string
	data       [][]driver.Value
	currentRow int
	closed     bool
	closeErr   error
	nextErr    error
}

func newMockDriverRows(columns []string, data [][]driver.Value) *mockDriverRows {
	return &mockDriverRows{
		columns:    columns,
		data:       data,
		currentRow: -1,
	}
}

func (m *mockDriverRows) Columns() []string {
	return m.columns
}

func (m *mockDriverRows) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockDriverRows) Next(dest []driver.Value) error {
	if m.nextErr != nil {
		return m.nextErr
	}
	m.currentRow++
	if m.currentRow >= len(m.data) {
		return io.EOF
	}
	copy(dest, m.data[m.currentRow])
	return nil
}

// Mock driver.Rows with NextResultSet support
type mockDriverRowsWithNextResultSet struct {
	*mockDriverRows
	hasNextResultSet    bool
	nextResultSetErr    error
	nextResultSetCalled int
}

func newMockDriverRowsWithNextResultSet(columns []string, data [][]driver.Value, hasNext bool) *mockDriverRowsWithNextResultSet {
	return &mockDriverRowsWithNextResultSet{
		mockDriverRows:   newMockDriverRows(columns, data),
		hasNextResultSet: hasNext,
	}
}

func (m *mockDriverRowsWithNextResultSet) HasNextResultSet() bool {
	return m.hasNextResultSet
}

func (m *mockDriverRowsWithNextResultSet) NextResultSet() error {
	if m.nextResultSetErr != nil {
		return m.nextResultSetErr
	}
	m.nextResultSetCalled++
	m.hasNextResultSet = false
	m.currentRow = -1
	return nil
}

func TestNewScanRows(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id", "name"}, nil)
	scanRows := NewScanRows(mockRows)

	assert.NotNil(t, scanRows)
	assert.Equal(t, mockRows, scanRows.rowsi)
	assert.False(t, scanRows.closed)
	assert.Nil(t, scanRows.lasterr)
}

func TestScanRows_Next_WithData(t *testing.T) {
	data := [][]driver.Value{
		{int64(1), "Alice"},
		{int64(2), "Bob"},
		{int64(3), "Charlie"},
	}
	mockRows := newMockDriverRows([]string{"id", "name"}, data)
	scanRows := NewScanRows(mockRows)

	// First row
	assert.True(t, scanRows.Next())
	assert.Equal(t, int64(1), scanRows.lastcols[0])
	assert.Equal(t, "Alice", scanRows.lastcols[1])

	// Second row
	assert.True(t, scanRows.Next())
	assert.Equal(t, int64(2), scanRows.lastcols[0])
	assert.Equal(t, "Bob", scanRows.lastcols[1])

	// Third row
	assert.True(t, scanRows.Next())
	assert.Equal(t, int64(3), scanRows.lastcols[0])
	assert.Equal(t, "Charlie", scanRows.lastcols[1])

	// No more rows
	assert.False(t, scanRows.Next())
}

func TestScanRows_Next_EmptyResult(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id", "name"}, [][]driver.Value{})
	scanRows := NewScanRows(mockRows)

	assert.False(t, scanRows.Next())
	assert.Equal(t, io.EOF, scanRows.lasterr)
}

// Test ScanRows.Next with error
func TestScanRows_Next_WithError(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	expectedErr := errors.New("database error")
	mockRows.nextErr = expectedErr
	scanRows := NewScanRows(mockRows)

	assert.False(t, scanRows.Next())
	assert.Equal(t, expectedErr, scanRows.lasterr)
}

func TestScanRows_Scan_BasicTypes(t *testing.T) {
	data := [][]driver.Value{
		{int64(123), "test", true, 45.67},
	}
	mockRows := newMockDriverRows([]string{"id", "name", "active", "score"}, data)
	scanRows := NewScanRows(mockRows)

	assert.True(t, scanRows.Next())

	var id int64
	var name string
	var active bool
	var score float64

	err := scanRows.Scan(&id, &name, &active, &score)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)
	assert.Equal(t, "test", name)
	assert.True(t, active)
	assert.InDelta(t, 45.67, score, 0.001)
}

func TestScanRows_Scan_WithNilValues(t *testing.T) {
	data := [][]driver.Value{
		{int64(1), nil},
	}
	mockRows := newMockDriverRows([]string{"id", "name"}, data)
	scanRows := NewScanRows(mockRows)

	assert.True(t, scanRows.Next())

	var id int64
	var name string

	err := scanRows.Scan(&id, &name)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), id)
	assert.Equal(t, "", name) // nil should keep default value
}

func TestScanRows_Scan_WithoutNext(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)

	var id int64
	err := scanRows.Scan(&id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Scan called without calling Next")
}

func TestScanRows_Scan_WrongArgCount(t *testing.T) {
	data := [][]driver.Value{
		{int64(1), "test"},
	}
	mockRows := newMockDriverRows([]string{"id", "name"}, data)
	scanRows := NewScanRows(mockRows)

	assert.True(t, scanRows.Next())

	var id int64
	err := scanRows.Scan(&id) // Should expect 2 arguments
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected 2 destination arguments")
}

func TestScanRows_Scan_WhenClosed(t *testing.T) {
	data := [][]driver.Value{
		{int64(1), "test"},
	}
	mockRows := newMockDriverRows([]string{"id", "name"}, data)
	scanRows := NewScanRows(mockRows)
	scanRows.closed = true

	var id int64
	var name string
	err := scanRows.Scan(&id, &name)
	assert.Error(t, err)
	assert.Equal(t, errRowsClosed, err)
}

func TestScanRows_Err(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)

	// No error initially
	assert.NoError(t, scanRows.Err())

	// Set an error
	expectedErr := errors.New("test error")
	scanRows.lasterr = expectedErr
	assert.Equal(t, expectedErr, scanRows.Err())
}

func TestScanRows_Close(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}

	err := scanRows.close(nil)
	assert.NoError(t, err)
	assert.True(t, mockRows.closed)
}

func TestScanRows_Close_Idempotent(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}

	// First close
	err1 := scanRows.close(nil)
	assert.NoError(t, err1)

	// Second close should not fail
	err2 := scanRows.close(nil)
	assert.NoError(t, err2)
}

func TestScanRows_Close_WithError(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	expectedErr := errors.New("close error")
	mockRows.closeErr = expectedErr
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}

	err := scanRows.close(nil)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestScanRows_NextResultSet(t *testing.T) {
	mockRows := newMockDriverRowsWithNextResultSet(
		[]string{"id"},
		[][]driver.Value{{int64(1)}},
		true,
	)
	scanRows := NewScanRows(mockRows)

	result := scanRows.NextResultSet()
	assert.True(t, result)
	assert.Equal(t, 1, mockRows.nextResultSetCalled)
}

func TestScanRows_NextResultSet_NoNext(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)

	// mockDriverRows doesn't implement RowsNextResultSet
	assert.False(t, scanRows.NextResultSet())
}

func TestScanRows_NextResultSet_WithError(t *testing.T) {
	expectedErr := errors.New("next result set error")
	mockRows := newMockDriverRowsWithNextResultSet(
		[]string{"id"},
		nil,
		false,
	)
	mockRows.nextResultSetErr = expectedErr
	scanRows := NewScanRows(mockRows)

	assert.False(t, scanRows.NextResultSet())
	assert.Equal(t, expectedErr, scanRows.lasterr)
}

func TestScanRows_NextResultSet_WhenClosed(t *testing.T) {
	mockRows := newMockDriverRowsWithNextResultSet([]string{"id"}, nil, false)
	scanRows := NewScanRows(mockRows)
	scanRows.closed = true

	assert.False(t, scanRows.NextResultSet())
}

func TestScanRows_ContextCancellation(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, [][]driver.Value{{int64(1)}})
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}

	ctx, cancel := context.WithCancel(context.Background())
	scanRows.initContextClose(ctx, nil)

	// Cancel context
	cancel()
	time.Sleep(10 * time.Millisecond) // Wait for goroutine to process

	// Check if rows are closed
	scanRows.closemu.RLock()
	closed := scanRows.closed
	scanRows.closemu.RUnlock()

	assert.True(t, closed)
}

func TestScanRows_TransactionContext(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, [][]driver.Value{{int64(1)}})
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}

	ctx := context.Background()
	txctx, txcancel := context.WithCancel(context.Background())
	scanRows.initContextClose(ctx, txctx)

	// Cancel transaction context
	txcancel()
	time.Sleep(10 * time.Millisecond) // Wait for goroutine to process

	// Check if rows are closed
	scanRows.closemu.RLock()
	closed := scanRows.closed
	scanRows.closemu.RUnlock()

	assert.True(t, closed)
}

func TestScanRows_BypassRowsAwaitDone(t *testing.T) {
	// Save original value
	originalBypass := bypassRowsAwaitDone
	defer func() {
		bypassRowsAwaitDone = originalBypass
	}()

	bypassRowsAwaitDone = true

	mockRows := newMockDriverRows([]string{"id"}, [][]driver.Value{{int64(1)}})
	scanRows := NewScanRows(mockRows)

	ctx, cancel := context.WithCancel(context.Background())
	scanRows.initContextClose(ctx, nil)

	cancel()
	time.Sleep(10 * time.Millisecond)

	// Should not be closed because bypass is true
	scanRows.closemu.RLock()
	closed := scanRows.closed
	scanRows.closemu.RUnlock()

	assert.False(t, closed)
}

func TestWithLock(t *testing.T) {
	var mu sync.Mutex
	counter := 0

	withLock(&mu, func() {
		counter++
	})

	assert.Equal(t, 1, counter)
}

func TestWithLock_WithPanic(t *testing.T) {
	var mu sync.Mutex

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic")
		}
	}()

	withLock(&mu, func() {
		panic("test panic")
	})
}

func TestScanRows_LasterrOrErrLocked(t *testing.T) {
	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)

	// No lasterr
	err := scanRows.lasterrOrErrLocked(errors.New("test error"))
	assert.Equal(t, "test error", err.Error())

	// With lasterr
	scanRows.lasterr = errors.New("last error")
	err = scanRows.lasterrOrErrLocked(errors.New("test error"))
	assert.Equal(t, "last error", err.Error())

	// With io.EOF as lasterr
	scanRows.lasterr = io.EOF
	err = scanRows.lasterrOrErrLocked(errors.New("test error"))
	assert.Equal(t, "test error", err.Error())
}

func TestScanRows_Scan_WithConversion(t *testing.T) {
	data := [][]driver.Value{
		{"123", "45.67", "true"},
	}
	mockRows := newMockDriverRows([]string{"id", "score", "active"}, data)
	scanRows := NewScanRows(mockRows)

	assert.True(t, scanRows.Next())

	var id int64
	var score float64
	var active bool

	err := scanRows.Scan(&id, &score, &active)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)
	assert.InDelta(t, 45.67, score, 0.001)
	assert.True(t, active)
}

func TestScanRows_MultipleResultSets(t *testing.T) {
	// Create mock rows with data for first result set
	mockRows := newMockDriverRowsWithNextResultSet(
		[]string{"id"},
		[][]driver.Value{{int64(1)}},
		true,
	)
	scanRows := NewScanRows(mockRows)

	// First result set - read the row
	assert.True(t, scanRows.Next())
	var id int64
	err := scanRows.Scan(&id)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), id)

	// No more rows in first result set
	assert.False(t, scanRows.Next())

	// Move to next result set
	assert.True(t, scanRows.NextResultSet())
	assert.Equal(t, 1, mockRows.nextResultSetCalled)
}

func TestScanRows_CloseHook(t *testing.T) {
	// Save original hook
	originalHook := rowsCloseHook
	defer func() {
		rowsCloseHook = originalHook
	}()

	hookCalled := false
	rowsCloseHook = func() func(*ScanRows, *error) {
		return func(rs *ScanRows, err *error) {
			hookCalled = true
		}
	}

	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}

	scanRows.close(nil)

	assert.True(t, hookCalled)
}

func TestScanRows_WithCancelFunction(t *testing.T) {
	cancelCalled := false
	mockRows := newMockDriverRows([]string{"id"}, nil)
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}
	scanRows.cancel = func() {
		cancelCalled = true
	}

	scanRows.close(nil)

	assert.True(t, cancelCalled)
}

func TestScanRows_ConcurrentAccess(t *testing.T) {
	data := [][]driver.Value{
		{int64(1), "row1"},
		{int64(2), "row2"},
		{int64(3), "row3"},
	}
	mockRows := newMockDriverRows([]string{"id", "name"}, data)
	scanRows := NewScanRows(mockRows)

	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Concurrent Next calls
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if scanRows.Next() {
				var id int64
				var name string
				if err := scanRows.Scan(&id, &name); err != nil {
					errChan <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Logf("Concurrent error (expected in some cases): %v", err)
	}
}

func TestScanRows_Next_HasNextResultSetFalse(t *testing.T) {
	mockRows := newMockDriverRowsWithNextResultSet(
		[]string{"id"},
		[][]driver.Value{},
		false,
	)
	scanRows := NewScanRows(mockRows)
	scanRows.releaseConn = func(error) {}

	assert.False(t, scanRows.Next())
}
