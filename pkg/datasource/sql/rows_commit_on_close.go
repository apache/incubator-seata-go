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

package sql

import (
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"sync/atomic"

	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type closeState int32

const (
	stateOpen closeState = iota
	stateClosing
	stateClosed
)

// RowsCommitOnClose wraps driver.Rows and commits the transaction
type RowsCommitOnClose struct {
	rows driver.Rows
	tx   driver.Tx

	state int32 // atomic closeState
	err   error
}

// Close implements driver.Rows.Close.
func (r *RowsCommitOnClose) Close() error {
	if !atomic.CompareAndSwapInt32(&r.state, int32(stateOpen), int32(stateClosing)) {
		return r.err
	}

	var rowErr error
	if r.rows != nil {
		rowErr = r.rows.Close()
	}

	var txErr error
	if r.tx != nil {
		txErr = r.tx.Commit()
		if txErr != nil {
			log.Errorf("RowsCommitOnClose: commit failed: %v", txErr)
		}
		r.tx = nil
	}

	var errs []error
	if rowErr != nil {
		errs = append(errs, rowErr)
	}
	if txErr != nil {
		errs = append(errs, txErr)
	}
	r.err = errors.Join(errs...)

	atomic.StoreInt32(&r.state, int32(stateClosed))
	return r.err
}

// Columns implements driver.Rows.Columns.
func (r *RowsCommitOnClose) Columns() []string {
	return r.rows.Columns()
}

// Next implements driver.Rows.Next.
func (r *RowsCommitOnClose) Next(dest []driver.Value) error {
	err := r.rows.Next(dest)
	if err == io.EOF {
		_ = r.Close()
	}
	return err
}

// preparedRowsCommitOnClose forwards the optional interfaces of prepared query results.
type preparedRowsCommitOnClose struct{ *RowsCommitOnClose }

func (r *preparedRowsCommitOnClose) Next(dest []driver.Value) error {
	err := r.rows.Next(dest)
	if err == io.EOF && !r.HasNextResultSet() {
		_ = r.Close()
	}
	return err
}

func (r *preparedRowsCommitOnClose) HasNextResultSet() bool {
	rows, ok := r.rows.(driver.RowsNextResultSet)
	return atomic.LoadInt32(&r.state) == int32(stateOpen) && ok && rows.HasNextResultSet()
}

func (r *preparedRowsCommitOnClose) NextResultSet() error {
	if rows, ok := r.rows.(driver.RowsNextResultSet); ok {
		return rows.NextResultSet()
	}
	return io.EOF
}

func (r *preparedRowsCommitOnClose) ColumnTypeScanType(index int) reflect.Type {
	if rows, ok := r.rows.(driver.RowsColumnTypeScanType); ok {
		return rows.ColumnTypeScanType(index)
	}
	return reflect.TypeOf((*any)(nil)).Elem()
}

func (r *preparedRowsCommitOnClose) ColumnTypeDatabaseTypeName(index int) string {
	if rows, ok := r.rows.(driver.RowsColumnTypeDatabaseTypeName); ok {
		return rows.ColumnTypeDatabaseTypeName(index)
	}
	return ""
}

func (r *preparedRowsCommitOnClose) ColumnTypeLength(index int) (int64, bool) {
	if rows, ok := r.rows.(driver.RowsColumnTypeLength); ok {
		return rows.ColumnTypeLength(index)
	}
	return 0, false
}

func (r *preparedRowsCommitOnClose) ColumnTypeNullable(index int) (bool, bool) {
	if rows, ok := r.rows.(driver.RowsColumnTypeNullable); ok {
		return rows.ColumnTypeNullable(index)
	}
	return false, false
}

func (r *preparedRowsCommitOnClose) ColumnTypePrecisionScale(index int) (int64, int64, bool) {
	if rows, ok := r.rows.(driver.RowsColumnTypePrecisionScale); ok {
		return rows.ColumnTypePrecisionScale(index)
	}
	return 0, 0, false
}
