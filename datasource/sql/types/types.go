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
	gosql "database/sql"
)

// DBType
type DBType int16

const (
	_ DBType = iota
	MySQL
	PostgreSQL
	SQLServer
	Oracle
)

// TransactionType
type TransactionType int8

const (
	_ TransactionType = iota
	TransactionTypeXA
	TransactionTypeAT
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
func WithTxOptions(opt *gosql.TxOptions) ctxOption {
	return func(tx *TransactionContext) {
		tx.TxOpt = opt
	}
}

// TransactionContext
type TransactionContext struct {
	// dbType
	DBType DBType
	// txOpt
	TxOpt *gosql.TxOptions
	// TransType
	TransType TransactionType
	// ResourceID
	ResourceID string
	// BranchID
	BranchID string
	// XaID
	XaID string
	// GlobalLockRequire
	GlobalLockRequire bool
	// RecordImages
	RecordImages RoundRecordImage
}

// RoundRecordImage
type RoundRecordImage struct {
	BeforeImages RecordImage
	AfterImages  RecordImage
}

// RecordImage
type RecordImage struct {
	Table   string
	SQLType string
	Rows    []RowImage
}

type RowImage struct {
	Name  string
	Type  gosql.ColumnType
	Value interface{}
}
