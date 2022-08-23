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
	"strings"

	"github.com/google/uuid"
)

//go:generate stringer -type=DBType
type DBType int16

type (
	// DBType
	// BranchPhase
	BranchPhase int8
	// IndexType index type
	IndexType int16
)

const (
	_ DBType = iota
	DBTypeUnknown
	DBTypeMySQL
	DBTypePostgreSQL
	DBTypeSQLServer
	DBTypeOracle

	BranchPhase_Unknown = 0
	BranchPhase_Done    = 1
	BranchPhase_Failed  = 2

	// Index_Primary primary index type.
	IndexPrimary = 0
	// Index_Normal normal index type.
	IndexNormal = 1
	// Index_Unique unique index type.
	IndexUnique = 2
	// Index_FullText full text index type.
	IndexFullText = 3
)

func ParseDBType(driverName string) DBType {
	switch strings.ToLower(driverName) {
	case "mysql":
		return DBTypeMySQL
	default:
		return DBTypeUnknown
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

// TransactionContext seata-go‘s context of transaction
type TransactionContext struct {
	// LocalTransID locals transaction id
	LocalTransID string
	// LockKeys
	LockKeys []string
	// DBType db type, eg. MySQL/PostgreSQL/SQLServer
	DBType DBType
	// TxOpt transaction option
	TxOpt driver.TxOptions
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

func NewTxCtx() *TransactionContext {
	return &TransactionContext{
		LockKeys:     make([]string, 0, 4),
		TransType:    ATMode,
		LocalTransID: uuid.New().String(),
		RoundImages:  &RoundRecordImage{},
	}
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

type (
	ExecResult interface {
		GetRows() driver.Rows

		GetResult() driver.Result
	}

	queryResult struct {
		Rows driver.Rows
	}

	writeResult struct {
		Result driver.Result
	}
)

func (r *queryResult) GetRows() driver.Rows {
	return r.Rows
}

func (r *queryResult) GetResult() driver.Result {
	panic("writeResult no support")
}

func (r *writeResult) GetRows() driver.Rows {
	panic("writeResult no support")
}

func (r *writeResult) GetResult() driver.Result {
	return r.Result
}

type option struct {
	rows driver.Rows
	ret  driver.Result
}

type Option func(*option)

func WithRows(rows driver.Rows) Option {
	return func(o *option) {
		o.rows = rows
	}
}

func WithResult(ret driver.Result) Option {
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
	if o.rows != nil {
		return &queryResult{Rows: o.rows}
	}

	panic("not expect result, impossible run into here")
}
