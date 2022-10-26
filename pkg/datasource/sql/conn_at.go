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

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/tm"
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

	return c.Conn.QueryContext(ctx, query, args)
}

// ExecContext
func (c *ATConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.createOnceTxContext(ctx) {
		defer func() {
			c.txCtx = types.NewTxCtx()
		}()
	}

	return c.Conn.ExecContext(ctx, query, args)
}

// BeginTx
func (c *ATConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.autoCommit = false

	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts

	if IsGlobalTx(ctx) {
		c.txCtx.XaID = tm.GetXID(ctx)
		c.txCtx.TransType = types.ATMode
	}

	tx, err := c.Conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &ATTx{tx: tx.(*Tx)}, nil
}

func (c *ATConn) createOnceTxContext(ctx context.Context) bool {
	onceTx := IsGlobalTx(ctx) && c.autoCommit

	if onceTx {
		c.txCtx = types.NewTxCtx()
		c.txCtx.DBType = c.res.dbType
		c.txCtx.XaID = tm.GetXID(ctx)
		c.txCtx.TransType = types.ATMode
	}

	return onceTx
}
