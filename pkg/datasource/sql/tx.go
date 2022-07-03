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

	"github.com/pkg/errors"
	"github.com/seata/seata-go-datasource/sql/exec"
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

// Tx
type Tx struct {
	ctx    *types.TransactionContext
	target *gosql.Tx
}

// init
func (tx *Tx) init() error {
	return nil
}

// Commit do commit action
// case 1. no open global-transaction, just do local transaction commit
// case 2. not need flush undolog, is XA mode, do local transaction commit
// case 3. need run AT transaction
func (tx *Tx) Commit() error {
	if tx.ctx.TransType == types.Local {
		return tx.commitOnLocal()
	}

	// flush undo log if need, is XA mode
	if tx.ctx.TransType == types.XAMode {
		return tx.commitOnXA()
	}

	return tx.commitOnAT()
}

// commitOnLocal
func (tx *Tx) commitOnLocal() error {
	return tx.target.Commit()
}

// commitOnXA
func (tx *Tx) commitOnXA() error {
	return nil
}

// commitOnAT
func (tx *Tx) commitOnAT() error {
	// if TX-Mode is AT, run regis this transaction branch
	if err := tx.regis(tx.ctx); err != nil {
		return err
	}

	undoLogMgr, err := undo.GetUndoLogManager(tx.ctx.DBType)
	if err != nil {
		return err
	}

	if err := undoLogMgr.FlushUndoLog(tx.ctx, tx.target); err != nil {
		if rerr := tx.report(false); rerr != nil {
			return errors.WithStack(rerr)
		}
		return errors.WithStack(err)
	}

	if err := tx.commitOnLocal(); err != nil {
		if rerr := tx.report(false); rerr != nil {
			return errors.WithStack(rerr)
		}
		return errors.WithStack(err)
	}

	tx.report(true)
	return nil
}

// Rollback
func (tx *Tx) Rollback() error {
	err := tx.target.Rollback()

	if err != nil {
		if tx.ctx.OpenGlobalTrsnaction() && tx.ctx.IsBranchRegistered() {
			tx.report(false)
		}
	}

	return err
}

// regis
// TODO
func (tx *Tx) regis(ctx *types.TransactionContext) error {
	if !ctx.HasUndoLog() || !ctx.HasLockKey() {
		return nil
	}

	return nil
}

// report
// TODO
func (tx *Tx) report(success bool) error {
	return nil
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

	executor, err := exec.BuildExecutor(tx.ctx.DBType, query)
	if err != nil {
		return nil, err
	}

	ret, err := executor.Exec(tx.ctx, func(ctx context.Context, query string, args ...interface{}) (types.ExecResult, error) {
		ret, err := tx.target.ExecContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}

		return types.NewResult(types.WithResult(ret)), nil
	})

	if err != nil {
		return nil, err
	}

	return ret.GetResult(), nil
}

// PrepareContext
func (tx *Tx) PrepareContext(ctx context.Context, query string, args ...interface{}) (*Stmt, error) {
	stmt, err := tx.target.PrepareContext(ctx, query)

	if err != nil {
		return nil, err
	}

	return &Stmt{target: stmt, query: query, ctx: tx.ctx}, nil
}

// Stmt
func (tx *Tx) Stmt(ctx context.Context, stmt *gosql.Stmt) (*Stmt, error) {
	newStmt := tx.target.StmtContext(ctx, stmt)

	return &Stmt{target: newStmt, ctx: tx.ctx}, nil
}
