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

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

// ATConn Database connection proxy object under XA transaction model
// Conn is assumed to be stateful.
type ATConn struct {
	*Conn
}

func (c *ATConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}
	return c.Conn.PrepareContext(ctx, query)
}

// QueryContext
func (c *ATConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	ret, err := c.createNewTxOnExecIfNeed(ctx, func() (types.ExecResult, error) {
		executor, err := exec.BuildExecutor(c.res.dbType, c.txCtx.TransactionMode, query)
		if err != nil {
			return nil, err
		}

		execCtx := &types.ExecContext{
			TxCtx:                c.txCtx,
			Query:                query,
			NamedValues:          args,
			Conn:                 c.targetConn,
			DBName:               c.dbName,
			IsSupportsSavepoints: true,
			IsAutoCommit:         c.GetAutoCommit(),
		}

		return executor.ExecWithNamedValue(ctx, execCtx,
			func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				ret, err := c.Conn.QueryContext(ctx, query, args)
				if err != nil {
					return nil, err
				}
				return types.NewResult(types.WithRows(ret)), nil
			})
	})
	if err != nil {
		return nil, err
	}
	return ret.GetRows(), nil
}

// ExecContext
func (c *ATConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	ret, err := c.createNewTxOnExecIfNeed(ctx, func() (types.ExecResult, error) {
		executor, err := exec.BuildExecutor(c.res.dbType, c.txCtx.TransactionMode, query)
		if err != nil {
			return nil, err
		}

		execCtx := &types.ExecContext{
			TxCtx:                c.txCtx,
			Query:                query,
			NamedValues:          args,
			Conn:                 c.targetConn,
			DBName:               c.dbName,
			IsSupportsSavepoints: true,
			IsAutoCommit:         c.GetAutoCommit(),
		}

		ret, err := executor.ExecWithNamedValue(ctx, execCtx,
			func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
				ret, err := c.Conn.ExecContext(ctx, query, args)
				if err != nil {
					return nil, err
				}
				return types.NewResult(types.WithResult(ret)), nil
			})

		return ret, err
	})
	if err != nil {
		return nil, err
	}
	return ret.GetResult(), nil
}

// BeginTx
func (c *ATConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.autoCommit = false

	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts
	c.txCtx.ResourceID = c.res.resourceID

	if tm.IsGlobalTx(ctx) {
		c.txCtx.XID = tm.GetXID(ctx)
		c.txCtx.TransactionMode = types.ATMode
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

func (c *ATConn) createNewTxOnExecIfNeed(ctx context.Context, f func() (types.ExecResult, error)) (types.ExecResult, error) {
	var (
		tx  driver.Tx
		err error
	)

	if c.txCtx.TransactionMode != types.Local && tm.IsGlobalTx(ctx) && c.autoCommit {
		tx, err = c.BeginTx(ctx, driver.TxOptions{Isolation: driver.IsolationLevel(gosql.LevelDefault)})
		if err != nil {
			return nil, err
		}
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

	ret, err := f()
	if err != nil {
		return nil, err
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}

	return ret, nil
}
