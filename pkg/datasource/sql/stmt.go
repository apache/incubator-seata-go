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
	"database/sql/driver"
	"errors"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

type Stmt struct {
	conn            *Conn
	atConn          *ATConn
	preparedWithXID bool
	res             *DBResource
	txCtx           *types.TransactionContext
	query           string
	stmt            driver.Stmt
}

func (s *Stmt) executeAT(ctx context.Context, args []driver.NamedValue, returnsRows bool, run exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if err := s.atConn.validateATExecutionContext(ctx); err != nil {
		return nil, err
	}
	if !s.atConn.autoCommit {
		return s.atConn.execute(ctx, s.query, args, run)
	}

	hasXID := tm.IsGlobalTx(ctx)
	// The old PostgreSQL executor could enter AT from the execution XID
	// alone. Keep those statements from silently executing in local mode.
	hadATIntent := s.preparedWithXID || (s.res.dbType == types.DBTypePostgreSQL && hasXID)
	if !hadATIntent && !(returnsRows && hasXID) {
		return s.atConn.execute(ctx, s.query, args, run)
	}

	parsed, err := parser.ParseSQLForDB(s.query, s.res.dbType)
	if returnsRows && hasXID && err == nil && isPlainSelectForUpdate(parsed) {
		return s.atConn.createTxAndQueryIfNeeded(ctx, true, func() (types.ExecResult, error) {
			return s.atConn.execute(ctx, s.query, args, run)
		})
	}
	if hadATIntent {
		if err != nil || parsed == nil || parsed.SelectStmt == nil || parsed.SQLType != types.SQLTypeSelect {
			return nil, errors.New("seata AT: execute this prepared statement in a transaction started with BeginTx")
		}
		return run(ctx, s.query, args)
	}
	return s.atConn.execute(ctx, s.query, args, run)
}

func isPlainSelectForUpdate(parsed *types.ParseContext) bool {
	if parsed == nil || parsed.SQLType != types.SQLTypeSelectForUpdate || parsed.SelectStmt == nil {
		return false
	}
	stmt := parsed.SelectStmt
	// The auxiliary query needs a table and does not preserve INTO or OF.
	return stmt.From != nil && stmt.SelectIntoOpt == nil && stmt.LockInfo != nil && len(stmt.LockInfo.Tables) == 0
}

// Close closes the statement.
//
// As of Go 1.1, a Stmt will not be closed if it's in use
// by any queries.
//
// Drivers must ensure all network calls made by Close
// do not block indefinitely (e.g. apply a timeout).
func (s *Stmt) Close() error {
	s.txCtx = nil
	return s.stmt.Close()
}

// NumInput returns the number of placeholder parameters.
//
// If NumInput returns >= 0, the sql package will sanity check
// argument counts from callers and return errors to the caller
// before the statement's Exec or Query methods are called.
//
// NumInput may also return -1, if the driver doesn't know
// its number of placeholders. In that case, the sql package
// will not sanity check Exec or Query argument counts.
func (s *Stmt) NumInput() int {
	return s.stmt.NumInput()
}

// Query executes a query that may return rows, such as a
// SELECT.
//
// Deprecated: Drivers should implement StmtQueryContext instead (or additionally).
func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	if s.atConn != nil {
		return s.QueryContext(context.Background(), util.ValueToNamedValue(args))
	}
	executor, err := exec.BuildExecutor(s.res.dbType, s.txCtx.TransactionMode, s.query)
	if err != nil {
		return nil, err
	}

	execCtx := s.conn.newExecContext(s.txCtx, s.query, args, nil)

	ret, err := executor.ExecWithValue(context.Background(), execCtx,
		func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
			ret, err := s.stmt.Query(util.NamedValueToValue(args))
			if err != nil {
				return nil, err
			}

			return types.NewResult(types.WithRows(ret)), nil
		})
	if err != nil {
		return nil, err
	}

	return ret.GetRows(), nil
}

// QueryContext StmtQueryContext enhances the Stmt interface by providing Query with context.
// QueryContext executes a query that may return rows, such as a  SELECT.
// QueryContext must honor the context timeout and return when it is canceled.
func (s *Stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if s.atConn != nil {
		ret, err := s.executeAT(ctx, args, true, func(ctx context.Context, _ string, args []driver.NamedValue) (types.ExecResult, error) {
			rows, err := util.CtxDriverStmtQuery(ctx, s.stmt, args)
			if err != nil {
				return nil, err
			}
			return types.NewResult(types.WithRows(rows)), nil
		})
		if err != nil {
			return nil, err
		}
		return ret.GetRows(), nil
	}
	stmt, ok := s.stmt.(driver.StmtQueryContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	executor, err := exec.BuildExecutor(s.res.dbType, s.txCtx.TransactionMode, s.query)
	if err != nil {
		return nil, err
	}

	execCtx := s.conn.newExecContext(s.txCtx, s.query, nil, args)

	ret, err := executor.ExecWithNamedValue(ctx, execCtx,
		func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
			ret, err := stmt.QueryContext(ctx, args)
			if err != nil {
				return nil, err
			}

			return types.NewResult(types.WithRows(ret)), nil
		})
	if err != nil {
		return nil, err
	}

	return ret.GetRows(), nil
}

// Exec executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// Deprecated: Drivers should implement StmtExecContext instead (or additionally).
func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.atConn != nil {
		return s.ExecContext(context.Background(), util.ValueToNamedValue(args))
	}
	// in transaction, need run Executor
	executor, err := exec.BuildExecutor(s.res.dbType, s.txCtx.TransactionMode, s.query)
	if err != nil {
		return nil, err
	}

	execCtx := s.conn.newExecContext(s.txCtx, s.query, args, nil)

	ret, err := executor.ExecWithValue(context.Background(), execCtx,
		func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
			ret, err := s.stmt.Exec(util.NamedValueToValue(args))
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

// ExecContext executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// ExecContext must honor the context timeout and return when it is canceled.
func (s *Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if s.atConn != nil {
		ret, err := s.executeAT(ctx, args, false, func(ctx context.Context, _ string, args []driver.NamedValue) (types.ExecResult, error) {
			result, err := util.CtxDriverStmtExec(ctx, s.stmt, args)
			if err != nil {
				return nil, err
			}
			return types.NewResult(types.WithResult(result)), nil
		})
		if err != nil {
			return nil, err
		}
		return ret.GetResult(), nil
	}
	stmt, ok := s.stmt.(driver.StmtExecContext)
	if !ok {
		return nil, driver.ErrSkip
	}

	// in transaction, need run Executor
	executor, err := exec.BuildExecutor(s.res.dbType, s.txCtx.TransactionMode, s.query)
	if err != nil {
		return nil, err
	}

	execCtx := s.conn.newExecContext(s.txCtx, s.query, nil, args)

	ret, err := executor.ExecWithNamedValue(ctx, execCtx,
		func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
			ret, err := stmt.ExecContext(ctx, args)
			if err != nil {
				return nil, err
			}

			return types.NewResult(types.WithResult(ret)), nil
		})
	if err != nil {
		return nil, err
	}

	return ret.GetResult(), err
}
