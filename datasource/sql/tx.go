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
	"context"
	gosql "database/sql"

	"github.com/seata/seata-go-datasource/sql/types"
	"github.com/seata/seata-go-datasource/sql/undo"
)

type txOption func(tx *Tx)

func newProxyTx(opts ...txOption) (*Tx, error) {
	tx := new(Tx)

	for i := range opts {
		opts[i](tx)
	}

	if err := tx.init(); err != nil {
		return nil, err
	}

	return tx, nil
}

func withOriginTx(tx *gosql.Tx) txOption {
	return func(t *Tx) {
		t.target = tx
	}
}

func withCtx(ctx *types.TransactionContext) txOption {
	return func(t *Tx) {
		t.ctx = ctx
	}
}

func withHooks(hooks []SQLHook) txOption {
	return func(t *Tx) {
		t.hooks = hooks
	}
}

// Tx
type Tx struct {
	ctx    *types.TransactionContext
	hooks  []SQLHook
	target *gosql.Tx
}

// init
func (tx *Tx) init() error {
	return nil
}

// Commit
func (tx *Tx) Commit() error {
	branchID, err := tx.regis()
	if err != nil {
		return err
	}

	tx.ctx.BranchID = branchID

	// flush undo log if need
	if !tx.needFlushUndoLog() {
		return tx.target.Commit()
	}

	undoLogMgr, err := undo.GetUndoLogManager(tx.ctx.DBType)
	if err != nil {
		return err
	}

	undoLogMgr.FlushUndoLog(tx.ctx, tx.target)

	// do report
	return nil
}

func (tx *Tx) regis() (string, error) {
	return "", nil
}

func (tx *Tx) needFlushUndoLog() bool {
	return false
}

// Rollback
func (tx *Tx) Rollback() error {
	return tx.target.Rollback()
}

// QueryContext
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*gosql.Rows, error) {
	rows, err := tx.target.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, err
	}

	return rows, nil
}

// QueryRowContext
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *gosql.Row {
	row := tx.target.QueryRowContext(ctx, query, args...)

	return row
}

// ExecContext
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (gosql.Result, error) {

	executor, err := buildExecutor(query)
	if err != nil {
		return nil, err
	}

	ret, err := executor.Exec(tx.ctx, func(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
		return tx.target.ExecContext(ctx, query, args...)
	})

	if err != nil {
		return nil, err
	}

	return ret.(gosql.Result), nil
}

// PrepareContext
func (tx *Tx) PrepareContext(ctx context.Context, query string, args ...interface{}) (*Stmt, error) {
	stmt, err := tx.target.PrepareContext(ctx, query)

	if err != nil {
		return nil, err
	}

	return &Stmt{target: stmt, query: query}, nil
}

func (tx *Tx) Stmt(ctx context.Context, stmt *gosql.Stmt) (*Stmt, error) {
	newStmt := tx.target.StmtContext(ctx, stmt)

	return &Stmt{target: newStmt}, nil
}
