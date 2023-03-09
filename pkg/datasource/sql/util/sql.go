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

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sql provides a generic interface around SQL (or SQL-like)
// databases.
//
// The sql package must be used in conjunction with a database driver.
// See https://golang.org/s/sqldrivers for a list of drivers.
//
// Drivers that do not support context cancellation will not return until
// after the query is completed.
//
// For usage examples, see the wiki page at
// https://golang.org/s/sqlwiki.
package util

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
)

// ScanRows is the result of a query. Its cursor starts before the first row
// of the result set. Use Next to advance from row to row.
type ScanRows struct {
	//dc          *driverConn // owned; must call releaseConn when closed to release
	dc          driver.Conn
	releaseConn func(error)
	rowsi       driver.Rows
	cancel      func() // called when ScanRows is closed, may be nil.
	//closeStmt   *driverStmt // if non-nil, statement to Close on close

	// closemu prevents ScanRows from closing while there
	// is an active streaming result. It is held for read during non-close operations
	// and exclusively during close.
	//
	// closemu guards lasterr and closed.
	closemu sync.RWMutex
	closed  bool
	lasterr error // non-nil only if closed is true

	// lastcols is only used in Scan, Next, and NextResultSet which are expected
	// not to be called concurrently.
	lastcols []driver.Value
}

func NewScanRows(rowsi driver.Rows) *ScanRows {
	return &ScanRows{rowsi: rowsi}
}

// lasterrOrErrLocked returns either lasterr or the provided err.
// rs.closemu must be read-locked.
func (rs *ScanRows) lasterrOrErrLocked(err error) error {
	if rs.lasterr != nil && rs.lasterr != io.EOF {
		return rs.lasterr
	}
	return err
}

// bypassRowsAwaitDone is only used for testing.
// If true, it will not close the ScanRows automatically from the context.
var bypassRowsAwaitDone = false

func (rs *ScanRows) initContextClose(ctx, txctx context.Context) {
	if ctx.Done() == nil && (txctx == nil || txctx.Done() == nil) {
		return
	}
	if bypassRowsAwaitDone {
		return
	}
	ctx, rs.cancel = context.WithCancel(ctx)
	go rs.awaitDone(ctx, txctx)
}

// awaitDone blocks until either ctx or txctx is canceled. The ctx is provided
// from the query context and is canceled when the query ScanRows is closed.
// If the query was issued in a transaction, the transaction's context
// is also provided in txctx to ensure ScanRows is closed if the Tx is closed.
func (rs *ScanRows) awaitDone(ctx, txctx context.Context) {
	var txctxDone <-chan struct{}
	if txctx != nil {
		txctxDone = txctx.Done()
	}
	select {
	case <-ctx.Done():
	case <-txctxDone:
	}
	rs.close(ctx.Err())
}

// Next prepares the next result row for reading with the Scan method. It
// returns true on success, or false if there is no next result row or an error
// happened while preparing it. Err should be consulted to distinguish between
// the two cases.
//
// Every call to Scan, even the first one, must be preceded by a call to Next.
func (rs *ScanRows) Next() bool {
	var doClose, ok bool
	withLock(rs.closemu.RLocker(), func() {
		doClose, ok = rs.nextLocked()
	})
	if doClose {
		rs.Close()
	}
	return ok
}

func (rs *ScanRows) nextLocked() (doClose, ok bool) {
	if rs.closed {
		return false, false
	}

	// Lock the driver connection before calling the driver interface
	// rowsi to prevent a Tx from rolling back the connection at the same time.
	//rs.dc.Lock()
	//defer rs.dc.Unlock()

	if rs.lastcols == nil {
		rs.lastcols = make([]driver.Value, len(rs.rowsi.Columns()))
	}

	rs.lasterr = rs.rowsi.Next(rs.lastcols)
	if rs.lasterr != nil {
		// Close the connection if there is a driver error.
		if rs.lasterr != io.EOF {
			return true, false
		}
		nextResultSet, ok := rs.rowsi.(driver.RowsNextResultSet)
		if !ok {
			return true, false
		}
		// The driver is at the end of the current result set.
		// Test to see if there is another result set after the current one.
		// Only close ScanRows if there is no further result sets to read.
		if !nextResultSet.HasNextResultSet() {
			doClose = true
		}
		return doClose, false
	}
	return false, true
}

// NextResultSet prepares the next result set for reading. It reports whether
// there is further result sets, or false if there is no further result set
// or if there is an error advancing to it. The Err method should be consulted
// to distinguish between the two cases.
//
// After calling NextResultSet, the Next method should always be called before
// scanning. If there are further result sets they may not have rows in the result
// set.
func (rs *ScanRows) NextResultSet() bool {
	var doClose bool
	defer func() {
		if doClose {
			rs.Close()
		}
	}()
	rs.closemu.RLock()
	defer rs.closemu.RUnlock()

	if rs.closed {
		return false
	}

	rs.lastcols = nil
	nextResultSet, ok := rs.rowsi.(driver.RowsNextResultSet)
	if !ok {
		doClose = true
		return false
	}

	// Lock the driver connection before calling the driver interface
	// rowsi to prevent a Tx from rolling back the connection at the same time.
	//rs.dc.Lock()
	//defer rs.dc.Unlock()

	rs.lasterr = nextResultSet.NextResultSet()
	if rs.lasterr != nil {
		doClose = true
		return false
	}
	return true
}

// Err returns the error, if any, that was encountered during iteration.
// Err may be called after an explicit or implicit Close.
func (rs *ScanRows) Err() error {
	rs.closemu.RLock()
	defer rs.closemu.RUnlock()
	return rs.lasterrOrErrLocked(nil)
}

var errRowsClosed = errors.New("sql: ScanRows are closed")

// Scan copies the columns in the current row into the values pointed
// at by dest. The number of values in dest must be the same as the
// number of columns in ScanRows.
//
// Scan converts columns read from the database into the following
// common Go types and special types provided by the sql package:
//
//	*string
//	*[]byte
//	*int, *int8, *int16, *int32, *int64
//	*uint, *uint8, *uint16, *uint32, *uint64
//	*bool
//	*float32, *float64
//	*interface{}
//	*RawBytes
//	*ScanRows (cursor value)
//	any type implementing Scanner (see Scanner docs)
//
// In the most simple case, if the type of the value from the source
// column is an integer, bool or string type T and dest is of type *T,
// Scan simply assigns the value through the pointer.
//
// Scan also converts between string and numeric types, as long as no
// information would be lost. While Scan stringifies all numbers
// scanned from numeric database columns into *string, scans into
// numeric types are checked for overflow. For example, a float64 with
// value 300 or a string with value "300" can scan into a uint16, but
// not into a uint8, though float64(255) or "255" can scan into a
// uint8. One exception is that scans of some float64 numbers to
// strings may lose information when stringifying. In general, scan
// floating point columns into *float64.
//
// If a dest argument has type *[]byte, Scan saves in that argument a
// copy of the corresponding data. The copy is owned by the caller and
// can be modified and held indefinitely. The copy can be avoided by
// using an argument of type *RawBytes instead; see the documentation
// for RawBytes for restrictions on its use.
//
// If an argument has type *interface{}, Scan copies the value
// provided by the underlying driver without conversion. When scanning
// from a source value of type []byte to *interface{}, a copy of the
// slice is made and the caller owns the result.
//
// Source values of type time.Time may be scanned into values of type
// *time.Time, *interface{}, *string, or *[]byte. When converting to
// the latter two, time.RFC3339Nano is used.
//
// Source values of type bool may be scanned into types *bool,
// *interface{}, *string, *[]byte, or *RawBytes.
//
// For scanning into *bool, the source may be true, false, 1, 0, or
// string inputs parseable by strconv.ParseBool.
//
// Scan can also convert a cursor returned from a query, such as
// "select cursor(select * from my_table) from dual", into a
// *ScanRows value that can itself be scanned from. The parent
// select query will close any cursor *ScanRows if the parent *ScanRows is closed.
//
// If any of the first arguments implementing Scanner returns an error,
// that error will be wrapped in the returned error
func (rs *ScanRows) Scan(dest ...interface{}) error {
	rs.closemu.RLock()

	if rs.lasterr != nil && rs.lasterr != io.EOF {
		rs.closemu.RUnlock()
		return rs.lasterr
	}
	if rs.closed {
		err := rs.lasterrOrErrLocked(errRowsClosed)
		rs.closemu.RUnlock()
		return err
	}
	rs.closemu.RUnlock()

	if rs.lastcols == nil {
		return errors.New("sql: Scan called without calling Next")
	}
	if len(dest) != len(rs.lastcols) {
		return fmt.Errorf("sql: expected %d destination arguments in Scan, not %d", len(rs.lastcols), len(dest))
	}
	for i, sv := range rs.lastcols {
		if sv == nil {
			continue
		}
		// the type of dest may be NullString, NullInt64, int64, etc, we should call its Scan()
		ty := reflect.TypeOf(dest[i])
		fn, ok := ty.MethodByName("Scan")
		if !ok {
			err := convertAssignRows(dest[i], sv, rs)
			if err != nil {
				return fmt.Errorf(`sql: Scan error on column index %d, name %q: %w`, i, rs.rowsi.Columns()[i], err)
			}
		} else {
			res := fn.Func.Call([]reflect.Value{reflect.ValueOf(dest[i]), reflect.ValueOf(sv)})
			if len(res) > 0 && !res[0].IsNil() {
				return fmt.Errorf(`sql: Scan error on column index %d, name %q: %v`, i, rs.rowsi.Columns()[i], res[0].Elem().String())
			}
		}
	}
	return nil
}

// rowsCloseHook returns a function so tests may install the
// hook through a test only mutex.
var rowsCloseHook = func() func(*ScanRows, *error) { return nil }

// Close closes the ScanRows, preventing further enumeration. If Next is called
// and returns false and there are no further result sets,
// the ScanRows are closed automatically and it will suffice to check the
// result of Err. Close is idempotent and does not affect the result of Err.
func (rs *ScanRows) Close() error {
	//return rs.close(nil)
	return nil
}

func (rs *ScanRows) close(err error) error {
	rs.closemu.Lock()
	defer rs.closemu.Unlock()

	if rs.closed {
		return nil
	}
	rs.closed = true

	if rs.lasterr == nil {
		rs.lasterr = err
	}

	err = rs.rowsi.Close()
	//withLock(rs.dc, func() {
	//	err = rs.rowsi.Close()
	//})
	if fn := rowsCloseHook(); fn != nil {
		fn(rs, &err)
	}
	if rs.cancel != nil {
		rs.cancel()
	}

	//if rs.closeStmt != nil {
	//	rs.closeStmt.Close()
	//}
	rs.releaseConn(err)
	return err
}

// withLock runs while holding lk.
func withLock(lk sync.Locker, fn func()) {
	lk.Lock()
	defer lk.Unlock() // in case fn panics
	fn()
}
