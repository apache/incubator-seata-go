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

// ATConn Database connection proxy object under XA transaction model
// Conn is assumed to be stateful.
type ATConn struct {
	*Conn
}

var (
	errATPreparedMultiSQLUnsupported = errors.New("seata AT: prepared multi-SQL is unsupported; use Exec or ExecContext")
	parseATPreparedSQL               = sqlparser.DoParser
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
	if err := rejectATPreparedMultiSQL(c.dbType, query); err != nil {
		return nil, err
	}

	return c.Conn.Prepare(query)
}

func (c *ATConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if err := rejectATPreparedMultiSQL(c.dbType, query); err != nil {
		return nil, err
	}

	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}
	return c.Conn.PrepareContext(ctx, query)
}

// ExecContext
func (c *ATConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	ret, err := c.createTxAndExecIfNeeded(ctx, func() (types.ExecResult, error) {
		executor, err := exec.BuildExecutor(c.res.dbType, c.txCtx.TransactionMode, query)
		if err != nil {
			return nil, err
		}

		execCtx := c.newExecContext(c.txCtx, query, nil, args)

		ret, err := executor.ExecWithNamedValue(ctx, execCtx,
			func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				result, err := util.CtxDriverExecWithPrepareFallback(ctx, c.targetConn, query, args)
				if err != nil {
					return nil, err
				}

				return types.NewResult(types.WithResult(result)), nil
			})

		if err != nil {
			return nil, err
		}
		return ret, nil
	})
	if err != nil {
		return nil, err
	}
	return ret.GetResult(), nil
}

// QueryContext
func (c *ATConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	ret, err := c.createTxAndQueryIfNeeded(ctx, func() (types.ExecResult, error) {
		executor, err := exec.BuildExecutor(c.res.dbType, c.txCtx.TransactionMode, query)
		if err != nil {
			return nil, err
		}

		execCtx := c.newExecContext(c.txCtx, query, nil, args)

		ret, err := executor.ExecWithNamedValue(ctx, execCtx,
			func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				rows, err := c.Conn.QueryContext(ctx, query, args)
				if err == nil {
					return types.NewResult(types.WithRows(rows)), nil
				}

				// If skip fast-path error, fallback to prepared statement
				if strings.Contains(err.Error(), "skip fast-path") {
					stmt, prepErr := c.Conn.Prepare(query)
					if prepErr != nil {
						return nil, prepErr
					}

					if stmtQueryCtx, ok := stmt.(driver.StmtQueryContext); ok {
						rows, err = stmtQueryCtx.QueryContext(ctx, args)
					} else {
						dargs := make([]driver.Value, len(args))
						for i, arg := range args {
							dargs[i] = arg.Value
						}
						rows, err = stmt.Query(dargs)
					}

					if err != nil {
						stmt.Close()
						return nil, err
					}

					// Wrap rows with statement to close both together
					wrappedRows := util.NewRowsWithStmt(rows, stmt)
					return types.NewResult(types.WithRows(wrappedRows)), nil
				}

				return nil, err
			})

		if err != nil {
			return nil, err
		}
		return ret, nil
	})
	if err != nil {
		return nil, err
	}
	return ret.GetRows(), nil
}

// BeginTx
func (c *ATConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.autoCommit = false

	// Only create new txCtx if not already in a global transaction (auto-commit case)
	if c.txCtx.XID == "" {
		c.txCtx = types.NewTxCtx()
		c.txCtx.DBType = c.res.dbType
		c.txCtx.TxOpt = opts
		c.txCtx.ResourceID = c.res.resourceID

		if tm.IsGlobalTx(ctx) {
			c.txCtx.XID = tm.GetXID(ctx)
			c.txCtx.TransactionMode = types.ATMode
		}
	}

	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &ATTx{tx: tx.(*Tx)}, nil
}

func (c *ATConn) createOnceTxContext(ctx context.Context) bool {
	onceTx := tm.IsGlobalTx(ctx) && c.autoCommit

	if onceTx {
		c.txCtx = types.NewTxCtx()
		c.txCtx.DBType = c.res.dbType
		c.txCtx.ResourceID = c.res.resourceID
		c.txCtx.XID = tm.GetXID(ctx)
		c.txCtx.TransactionMode = types.ATMode
		c.txCtx.GlobalLockRequire = true
	}

	return onceTx
}

// createTxAndExecIfNeeded creates a transaction for execution context and commits it after execution
func (c *ATConn) createTxAndExecIfNeeded(ctx context.Context, f func() (types.ExecResult, error)) (types.ExecResult, error) {
	var (
		tx  driver.Tx
		err error
	)

	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		tx, err = c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			return nil, err
		}
		defer func() {
			recoverErr := recover()
			if recoverErr != nil {
				log.Errorf("at exec panic, recoverErr:%v", recoverErr)
				if tx != nil {
					rollbackErr := tx.Rollback()
					if rollbackErr != nil {
						log.Errorf("conn at rollback error:%v", rollbackErr)
					}
				}
			}
		}()
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
	var (
		tx  driver.Tx
		err error
	)

	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		tx, err = c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			return nil, err
		}
		defer func() {
			recoverErr := recover()
			if recoverErr != nil {
				log.Errorf("at exec panic, recoverErr:%v", recoverErr)
				if tx != nil {
					rollbackErr := tx.Rollback()
					if rollbackErr != nil {
						log.Errorf("conn at rollback error:%v", rollbackErr)
					}
				}
			}
		}()
	}

	ret, err := f()
	if err != nil {
		return nil, c.rollbackCreatedTx(tx, err)
	}

	// For QueryContext, wrap rows to commit on close
	var activeTx driver.Tx
	if c.txCtx.LocalTx != nil {
		activeTx = c.txCtx.LocalTx
	} else if tx != nil {
		activeTx = tx
	}

	if activeTx != nil {
		if rows, ok := ret.(types.ExecResult); ok {
			if dr := rows.GetRows(); dr != nil {
				wrappedRows := &RowsCommitOnClose{rows: dr, tx: activeTx}
				return types.NewResult(types.WithRows(wrappedRows)), nil
			}
		}
	}

	return ret, nil
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
