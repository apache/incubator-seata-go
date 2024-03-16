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
	"fmt"
	"strings"

	"github.com/google/uuid"

	"seata.apache.org/seata-go/pkg/protocol/branch"
)

type DBType int16

type (
	// BranchPhase
	BranchPhase int8
	// IndexType index type
	IndexType int16
)

const (
	IndexTypeNull       IndexType = 0
	IndexTypePrimaryKey IndexType = 1
)

func ParseIndexType(str string) IndexType {
	if str == "PRIMARY_KEY" {
		return IndexTypePrimaryKey
	}
	return IndexTypeNull
}

func (i IndexType) MarshalText() (text []byte, err error) {
	switch i {
	case IndexTypePrimaryKey:
		return []byte("PRIMARY_KEY"), nil
	}
	return []byte("NULL"), nil
}

func (i *IndexType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "PRIMARY_KEY":
		*i = IndexTypePrimaryKey
		return nil
	case "NULL":
		*i = IndexTypeNull
		return nil
	default:
		return fmt.Errorf("invalid index type")
	}
}

const (
	_ DBType = iota
	DBTypeUnknown
	DBTypeMySQL
	DBTypePostgreSQL
	DBTypeSQLServer
	DBTypeOracle
	DBTypeMARIADB

	BranchPhase_Unknown = 0
	BranchPhase_Done    = 1
	BranchPhase_Failed  = 2

	// IndexPrimary primary index type.
	IndexPrimary IndexType = iota
	// IndexNormal normal index type.
	IndexNormal
	// IndexUnique unique index type.
	IndexUnique
	// IndexFullText full text index type.
	IndexFullText
)

func ParseDBType(driverName string) DBType {
	switch strings.ToLower(driverName) {
	case "mysql":
		return DBTypeMySQL
	default:
		return DBTypeUnknown
	}
}

type TransactionMode int8

const (
	_ TransactionMode = iota
	Local
	XAMode
	ATMode
)

func (t TransactionMode) BranchType() branch.BranchType {
	switch t {
	case XAMode:
		return branch.BranchTypeXA
	case ATMode:
		return branch.BranchTypeAT
	default:
		return branch.BranchTypeUnknow
	}
}

// TransactionContext seata-go‘s context of transaction
type TransactionContext struct {
	// LocalTransID locals transaction id
	LocalTransID string
	// LockKeys
	LockKeys map[string]struct{}
	// DBType db type, eg. MySQL/PostgreSQL/SQLServer
	DBType DBType
	// TxOpt transaction option
	TxOpt driver.TxOptions
	// TransactionMode transaction mode, eg. XA/AT
	TransactionMode TransactionMode
	// ResourceID resource id, database-table
	ResourceID string
	// BranchID transaction branch unique id
	BranchID uint64
	// XID global transaction id
	XID string
	// GlobalLockRequire
	GlobalLockRequire bool
	// RoundImages when run in AT mode, record before and after Row image
	RoundImages *RoundRecordImage
}

// ExecContext
type ExecContext struct {
	TxCtx *TransactionContext
	Query string
	// todo delete
	ParseContext *ParseContext
	NamedValues  []driver.NamedValue
	// todo delete
	Values []driver.Value
	// todo delete
	MetaDataMap map[string]TableMeta
	Conn        driver.Conn
	DBName      string
	DBType      DBType
	// todo set values for these 4 param
	IsAutoCommit         bool
	IsSupportsSavepoints bool
	IsRequireGlobalLock  bool
}

func NewTxCtx() *TransactionContext {
	return &TransactionContext{
		LockKeys:        make(map[string]struct{}, 0),
		TransactionMode: Local,
		LocalTransID:    uuid.New().String(),
		RoundImages:     &RoundRecordImage{},
	}
}

// HasUndoLog
func (t *TransactionContext) HasUndoLog() bool {
	return t.TransactionMode == ATMode && !t.RoundImages.IsEmpty()
}

// HasLockKey
func (t *TransactionContext) HasLockKey() bool {
	return len(t.LockKeys) != 0
}

func (t *TransactionContext) OpenGlobalTransaction() bool {
	return t.TransactionMode != Local
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
