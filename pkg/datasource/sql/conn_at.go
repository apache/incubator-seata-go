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

type ATConn struct {
	*Conn
}

func (c *ATConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if c.createTxCtxIfAbsent(ctx) {
		defer func() {
			c.txCtx = nil
		}()
	}

	return c.Conn.PrepareContext(ctx, query)
}

// ExecContext
func (c *ATConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.createTxCtxIfAbsent(ctx) {
		defer func() {
			c.txCtx = nil
		}()
	}

	return c.Conn.ExecContext(ctx, query, args)
}

// BeginTx
func (c *ATConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.txCtx = types.NewTxCtx()
	c.txCtx.DBType = c.res.dbType
	c.txCtx.TxOpt = opts

	if IsGlobalTx(ctx) {
		c.txCtx.XaID = tm.GetXID(ctx)
		c.txCtx.TransType = c.txType
	}

	return c.Conn.BeginTx(ctx, opts)
}

func (c *ATConn) createTxCtxIfAbsent(ctx context.Context) bool {
	var onceTx bool

	if IsGlobalTx(ctx) && c.txCtx == nil {
		c.txCtx = types.NewTxCtx()
		c.txCtx.DBType = c.res.dbType
		c.txCtx.XaID = tm.GetXID(ctx)
		c.txCtx.TransType = types.ATMode
		c.autoCommit = true
		onceTx = true
	}

	if c.txCtx == nil {
		c.txCtx = types.NewTxCtx()
		onceTx = true
	}

	return onceTx
}
