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
	"database/sql"
	"strings"
)

type (
	// DBType
	DBType int16
	// BranchPhase
	BranchPhase int8
	// IndexType index type
	IndexType int16
)

const (
	_ DBType = iota
	DBType_Unknown
	DBType_MySQL
	DBType_PostgreSQL
	DBType_SQLServer
	DBType_Oracle

	BranchPhase_Unknown = 0
	BranchPhase_Done    = 1
	BranchPhase_Failed  = 2

	// Index_Primary primary index type.
	Index_Primary = 0
	// Index_Normal normal index type.
	Index_Normal = 1
	// Index_Unique unique index type.
	Index_Unique = 2
	// Index_FullText full text index type.
	Index_FullText = 3
)

func ParseDBType(driverName string) DBType {
	switch strings.ToLower(driverName) {
	case "mysql":
		return DBType_MySQL
	default:
		return DBType_Unknown
	}
}

// TransactionType
type TransactionType int8

const (
	_ TransactionType = iota
	Local
	XAMode
	ATMode
)

type ctxOption func(tx *TransactionContext)

// NewTxContext
func NewTxContext(opts ...ctxOption) *TransactionContext {
	tx := new(TransactionContext)

	for i := range opts {
		opts[i](tx)
	}

	return tx
}

// WithTransType
func WithTransType(t TransactionType) ctxOption {
	return func(tx *TransactionContext) {
		tx.TransType = t
	}
}

// WithTxOptions
func WithTxOptions(opt *TxOptions) ctxOption {
	return func(tx *TransactionContext) {
		tx.TxOpt = opt
	}
}

// TransactionContext seata-go‘s context of transaction
type TransactionContext struct {
	// LockKeys
	LockKeys []string
	// DBType db type, eg. MySQL/PostgreSQL/SQLServer
	DBType DBType
	// TxOpt transaction option
	TxOpt *TxOptions
	// TransType transaction mode, eg. XA/AT
	TransType TransactionType
	// ResourceID resource id, database-table
	ResourceID string
	// BranchID transaction branch unique id
	BranchID uint64
	// XaID XA id
	XaID string
	// GlobalLockRequire
	GlobalLockRequire bool
	// RoundImages when run in AT mode, record before and after Row image
	RoundImages *RoundRecordImage
}

// HasUndoLog
func (t *TransactionContext) HasUndoLog() bool {
	return t.TransType == ATMode && !t.RoundImages.IsEmpty()
}

// HasLockKey
func (t *TransactionContext) HasLockKey() bool {
	return len(t.LockKeys) != 0
}

func (t *TransactionContext) OpenGlobalTrsnaction() bool {
	return t.TransType != Local
}

func (t *TransactionContext) IsBranchRegistered() bool {
	return t.BranchID != 0
}

// TxOptions
type TxOptions struct {
	// Isolation is the transaction isolation level.
	// If zero, the driver or database's default level is used.
	Isolation sql.IsolationLevel
	// ReadOnly
	ReadOnly bool
	// TransType
	TransType TransactionType
}

type (
	ExecResult interface {
		GetRow() *sql.Row

		GetRows() *sql.Rows

		GetResult() sql.Result
	}

	queryRowResult struct {
		Row *sql.Row
	}

	queryResult struct {
		Rows *sql.Rows
	}

	writeResult struct {
		Result sql.Result
	}
)

func (r *queryRowResult) GetRow() *sql.Row {
	return r.Row
}

func (r *queryRowResult) GetRows() *sql.Rows {
	panic("writeResult no support")
}

func (r *queryRowResult) GetResult() sql.Result {
	panic("writeResult no support")
}

func (r *queryResult) GetRow() *sql.Row {
	panic("writeResult no support")
}

func (r *queryResult) GetRows() *sql.Rows {
	return r.Rows
}

func (r *queryResult) GetResult() sql.Result {
	panic("writeResult no support")
}

func (r *writeResult) GetRow() *sql.Row {
	panic("writeResult no support")
}

func (r *writeResult) GetRows() *sql.Rows {
	panic("writeResult no support")
}

func (r *writeResult) GetResult() sql.Result {
	return r.Result
}

type option struct {
	row  *sql.Row
	rows *sql.Rows
	ret  sql.Result
}

type Option func(*option)

func WithRow(row *sql.Row) Option {
	return func(o *option) {
		o.row = row
	}
}

func WithRows(rows *sql.Rows) Option {
	return func(o *option) {
		o.rows = rows
	}
}

func WithResult(ret sql.Result) Option {
	return func(o *option) {
		o.ret = ret
	}
}

func NewResult(opts ...Option) ExecResult {
	o := &option{}

	for i := range opts {
		opts[i](o)
	}

	if o.ret != nil {
		return &writeResult{Result: o.ret}
	}

	if o.row != nil {
		return &queryRowResult{Row: o.row}
	}

	if o.rows != nil {
		return &queryResult{Rows: o.rows}
	}

	panic("not expect result, impossible run into here")
}
