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
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	sqlparser "seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

// nonRetryableATError preserves the commit error without exposing the retry signal to database/sql.
type nonRetryableATError struct{ cause error }

func (e nonRetryableATError) Error() string { return e.cause.Error() }

func (e nonRetryableATError) Is(target error) bool {
	return target != driver.ErrBadConn && errors.Is(e.cause, target)
}

func (e nonRetryableATError) As(target any) bool { return errors.As(e.cause, target) }

// ATConn is a database connection proxy for AT transactions.
// Conn is assumed to be stateful.
type ATConn struct {
	*Conn
}

// ErrATTransactionContextConflict indicates an XID incompatible with an active AT-driver transaction.
var ErrATTransactionContextConflict = errors.New("seata AT: transaction context conflict")

var (
	errATPreparedMultiSQLUnsupported = errors.New("seata AT: prepared multi-SQL is unsupported; use Exec or ExecContext")
	parseATPreparedSQL               = func(query string) (*types.ParseContext, error) {
		return sqlparser.ParseSQLForDB(query, types.DBTypeMySQL)
	}
)

func rejectATPreparedMultiSQL(dbType types.DBType, query string) error {
	if dbType != types.DBTypeMySQL || !strings.Contains(query, ";") {
		return nil
	}

	parseCtx, err := parseATPreparedSQL(query)
	if err != nil || parseCtx == nil {
		// This is a best-effort Prepare guard, not the AT execution-time
		// validation boundary. Preserve Prepare compatibility for SQL that
		// the parser cannot handle; BuildExecutor parses it again before
		// AT execution.
		return nil
	}

	if len(parseCtx.MultiStmt) > 1 {
		return errATPreparedMultiSQLUnsupported
	}

	return nil
}

func (c *ATConn) Prepare(query string) (driver.Stmt, error) {
	if c.invalid {
		return nil, driver.ErrBadConn
	}
	if err := rejectATPreparedMultiSQL(c.dbType, query); err != nil {
		return nil, err
	}

	stmt, err := c.Conn.Prepare(query)
	return c.attachStatement(stmt, err, false)
}

func (c *ATConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if c.invalid {
		return nil, driver.ErrBadConn
	}
	if err := rejectATPreparedMultiSQL(c.dbType, query); err != nil {
		return nil, err
	}

	stmt, err := c.Conn.PrepareContext(ctx, query)
	return c.attachStatement(stmt, err, c.autoCommit && tm.IsGlobalTx(ctx))
}

func (c *ATConn) attachStatement(stmt driver.Stmt, err error, preparedWithXID bool) (driver.Stmt, error) {
	if err != nil {
		return nil, err
	}
	atStmt := stmt.(*Stmt)
	atStmt.atConn = c
	atStmt.preparedWithXID = preparedWithXID
	atStmt.txCtx = nil
	return atStmt, nil
}

func (c *ATConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	ret, err := c.createTxAndExecIfNeeded(ctx, func() (types.ExecResult, error) {
		return c.execute(ctx, query, args,
			func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				result, err := util.CtxDriverExecWithPrepareFallback(ctx, c.targetConn, query, args)
				if err != nil {
					return nil, err
				}
				return types.NewResult(types.WithResult(result)), nil
			})
	})
	if err != nil {
		return nil, err
	}
	return ret.GetResult(), nil
}

func (c *ATConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	ret, err := c.createTxAndQueryIfNeeded(ctx, func() (types.ExecResult, error) {
		return c.execute(ctx, query, args,
			func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				rows, err := util.CtxDriverQueryWithPrepareFallback(ctx, c.targetConn, query, args)
				if err != nil {
					return nil, err
				}
				return types.NewResult(types.WithRows(rows)), nil
			})
	})
	if err != nil {
		return nil, err
	}
	return ret.GetRows(), nil
}

func (c *ATConn) execute(ctx context.Context, query string, args []driver.NamedValue, run exec.CallbackWithNamedValue) (types.ExecResult, error) {
	executor, err := exec.BuildExecutor(c.res.dbType, c.txCtx.TransactionMode, query)
	if err != nil {
		return nil, err
	}
	execCtx := c.newExecContext(c.txCtx, query, nil, args)
	return executor.ExecWithNamedValue(ctx, execCtx, run)
}

// Begin starts a local transaction with the same connection state as BeginTx.
func (c *ATConn) Begin() (driver.Tx, error) {
	return c.beginTx(context.Background(), driver.TxOptions{}, c.Conn.Begin)
}

func (c *ATConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.beginTx(ctx, opts, func() (driver.Tx, error) {
		return c.Conn.BeginTx(ctx, opts)
	})
}

func (c *ATConn) beginTx(ctx context.Context, opts driver.TxOptions, begin func() (driver.Tx, error)) (driver.Tx, error) {
	if c.invalid {
		return nil, driver.ErrBadConn
	}
	if !c.autoCommit {
		return nil, errors.New("seata AT: a transaction is already active on this connection")
	}

	previousCtx := c.txCtx
	c.autoCommit = false
	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts
	c.txCtx.ResourceID = c.res.resourceID
	if tm.IsGlobalTx(ctx) {
		c.txCtx.XID = tm.GetXID(ctx)
		c.txCtx.TransactionMode = types.ATMode
	}

	tx, err := begin()
	if err != nil {
		c.txCtx, c.autoCommit = previousCtx, true
		return nil, err
	}
	return &ATTx{tx: tx.(*Tx)}, nil
}

func (c *ATConn) validateATExecutionContext(ctx context.Context) error {
	if c.invalid {
		return driver.ErrBadConn
	}
	if c.autoCommit || !tm.IsGlobalTx(ctx) {
		return nil
	}
	if c.txCtx.TransactionMode == types.Local {
		return fmt.Errorf("%w: SQL context carries an XID but the transaction began in local mode", ErrATTransactionContextConflict)
	}
	if tm.GetXID(ctx) != c.txCtx.XID {
		return fmt.Errorf("%w: SQL context XID differs from the XID bound at BeginTx", ErrATTransactionContextConflict)
	}
	return nil
}

func (c *ATConn) beginATExecution(ctx context.Context) (driver.Tx, error) {
	if err := c.validateATExecutionContext(ctx); err != nil {
		return nil, err
	}
	if c.autoCommit && tm.IsGlobalTx(ctx) {
		tx, err := c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			return nil, err
		}
		c.txCtx.GlobalLockRequire = true
		return tx, nil
	}
	return nil, nil
}

// createTxAndExecIfNeeded creates a transaction for execution context and commits it after execution
func (c *ATConn) createTxAndExecIfNeeded(ctx context.Context, f func() (types.ExecResult, error)) (types.ExecResult, error) {
	tx, err := c.beginATExecution(ctx)
	if err != nil {
		return nil, err
	}
	if tx != nil {
		defer c.rollbackOnPanic(tx)
	}

	ret, err := f()
	if err != nil {
		return nil, c.rollbackCreatedTx(tx, err)
	}

	// For ExecContext, commit the transaction if it was created
	if tx != nil {
		if err := tx.Commit(); err != nil {
			if errors.Is(err, driver.ErrBadConn) {
				return nil, nonRetryableATError{cause: err}
			}
			return nil, err
		}
	}

	return ret, nil
}

// createTxAndQueryIfNeeded creates a transaction for query context and wraps the rows to commit on close
func (c *ATConn) createTxAndQueryIfNeeded(ctx context.Context, f func() (types.ExecResult, error)) (types.ExecResult, error) {
	tx, err := c.beginATExecution(ctx)
	if err != nil {
		return nil, err
	}
	if tx != nil {
		defer c.rollbackOnPanic(tx)
	}

	ret, err := f()
	if err != nil {
		return nil, c.rollbackCreatedTx(tx, err)
	}

	if tx != nil {
		rows := &RowsCommitOnClose{rows: ret.GetRows(), tx: tx}
		return types.NewResult(types.WithRows(rows)), nil
	}
	return ret, nil
}

func (c *ATConn) rollbackOnPanic(tx driver.Tx) {
	if recovered := recover(); recovered != nil {
		if !c.autoCommit {
			if err := tx.Rollback(); err != nil {
				log.Errorf("conn at rollback error:%v", err)
			}
		}
		panic(recovered)
	}
}

func (c *ATConn) rollbackCreatedTx(tx driver.Tx, execErr error) error {
	if tx == nil || execErr == nil {
		return execErr
	}

	rollbackErr := tx.Rollback()
	if rollbackErr != nil {
		log.Errorf("conn at rollback error:%v", rollbackErr)
		return errors.Join(execErr, rollbackErr)
	}

	return execErr
}
