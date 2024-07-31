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

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
)

// Conn is a connection to a database. It is not used concurrently
// by multiple goroutines.
//
// Conn is assumed to be stateful.
type Conn struct {
	res        *DBResource
	txCtx      *types.TransactionContext
	targetConn driver.Conn
	autoCommit bool
	dbName     string
	dbType     types.DBType
}

// ResetSession is called prior to executing a query on the connection
// if the connection has been used before. If the driver returns ErrBadConn
// the connection is discarded.
func (c *Conn) ResetSession(ctx context.Context) error {
	conn, ok := c.targetConn.(driver.SessionResetter)
	if !ok {
		return driver.ErrSkip
	}

	c.autoCommit = true
	c.txCtx = types.NewTxCtx()
	return conn.ResetSession(ctx)
}

// Prepare returns a prepared statement, bound to this connection.
func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	s, err := c.targetConn.Prepare(query)
	if err != nil {
		return nil, err
	}

	return &Stmt{
		conn:  c,
		stmt:  s,
		query: query,
		res:   c.res,
		txCtx: c.txCtx,
	}, nil
}

// PrepareContext
func (c *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	conn, ok := c.targetConn.(driver.ConnPrepareContext)
	if !ok {
		stmt, err := c.targetConn.Prepare(query)
		if err != nil {
			return nil, err
		}

		return &Stmt{stmt: stmt, query: query, res: c.res, txCtx: c.txCtx}, nil
	}

	s, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}

	return &Stmt{
		conn:  c,
		stmt:  s,
		query: query,
		res:   c.res,
		txCtx: c.txCtx,
	}, nil
}

// Exec warning: if you want to use global transaction, please use ExecContext function
func (c *Conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	conn, ok := c.targetConn.(driver.Execer)
	if !ok {
		return nil, driver.ErrSkip
	}

	ret, err := conn.Exec(query, args)
	if err != nil {
		return nil, err
	}

	return types.NewResult(types.WithResult(ret)).GetResult(), nil
}

// ExecContext
func (c *Conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	targetConn, ok := c.targetConn.(driver.ExecerContext)
	if !ok {
		values := make([]driver.Value, 0, len(args))
		for i := range args {
			values = append(values, args[i].Value)
		}
		return c.Exec(query, values)
	}

	ret, err := targetConn.ExecContext(ctx, query, args)
	if err != nil {
		return nil, err
	}
	return types.NewResult(types.WithResult(ret)).GetResult(), nil
}

// Query
func (c *Conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	conn, ok := c.targetConn.(driver.Queryer)
	if !ok {
		return nil, driver.ErrSkip
	}

	executor, err := exec.BuildExecutor(c.res.dbType, c.txCtx.TransactionMode, query)
	if err != nil {
		return nil, err
	}

	execCtx := &types.ExecContext{
		TxCtx:  c.txCtx,
		Query:  query,
		Values: args,
	}

	ret, err := executor.ExecWithValue(context.Background(), execCtx,
		func(ctx context.Context, query string, args []driver.NamedValue) (types.ExecResult, error) {
			ret, err := conn.Query(query, util.NamedValueToValue(args))
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

// QueryContext
func (c *Conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	conn, ok := c.targetConn.(driver.QueryerContext)
	if !ok {
		values := make([]driver.Value, 0, len(args))

		for i := range args {
			values = append(values, args[i].Value)
		}

		return c.Query(query, values)
	}

	ret, err := conn.QueryContext(ctx, query, args)
	if err != nil {
		return nil, err
	}

	return types.NewResult(types.WithRows(ret)).GetRows(), nil
}

// Begin starts and returns a new transaction.
//
// Deprecated: Drivers should implement ConnBeginTx instead (or additionally).
func (c *Conn) Begin() (driver.Tx, error) {
	c.autoCommit = false

	tx, err := c.targetConn.Begin()
	if err != nil {
		return nil, err
	}

	if c.txCtx == nil {
		c.txCtx = types.NewTxCtx()
		c.txCtx.DBType = c.res.dbType
		c.txCtx.TxOpt = driver.TxOptions{}
	}

	return newTx(
		withDriverConn(c),
		withTxCtx(c.txCtx),
		withOriginTx(tx),
	)
}

// BeginTx Open a transaction and judge whether the current transaction needs to open a
//
//	global transaction according to tranCtx. If so, it needs to be included in the transaction management of seata
func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.txCtx.TransactionMode == types.XAMode {
		c.autoCommit = false
		return newTx(
			withDriverConn(c),
			withTxCtx(c.txCtx),
			withOriginTx(nil),
		)
	}

	if conn, ok := c.targetConn.(driver.ConnBeginTx); ok {
		tx, err := conn.BeginTx(ctx, opts)
		if err != nil {
			return nil, err
		}

		return newTx(
			withDriverConn(c),
			withTxCtx(c.txCtx),
			withOriginTx(tx),
		)
	}

	txi, err := c.Begin()
	if err != nil {
		return nil, err
	}
	return newTx(
		withDriverConn(c),
		withTxCtx(c.txCtx),
		withOriginTx(txi),
	)
}

func (c *Conn) GetAutoCommit() bool {
	return c.autoCommit
}

// Close invalidates and potentially stops any current
// prepared statements and transactions, marking this
// connection as no longer in use.
//
// Because the sql package maintains a free pool of
// connections and only calls Close when there's a surplus of
// idle connections, it shouldn't be necessary for drivers to
// do their own connection caching.
//
// Drivers must ensure all network calls made by Close
// do not block indefinitely (e.g. apply a timeout).
func (c *Conn) Close() error {
	c.txCtx = types.NewTxCtx()
	return c.targetConn.Close()
}
