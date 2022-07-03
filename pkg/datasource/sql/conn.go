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
)

type Conn struct {
	target *gosql.Conn
}

// BeginTx
func (c *Conn) BeginTx(ctx context.Context, opts *types.TxOptions) (*Tx, error) {
	tx, err := c.target.BeginTx(ctx, &gosql.TxOptions{
		Isolation: opts.Isolation,
		ReadOnly:  opts.ReadOnly,
	})

	if err != nil {
		return nil, err
	}

	txCtx := types.NewTxContext(
		types.WithTxOptions(opts),
		types.WithTransType(opts.TransType),
	)

	proxyTx, err := newProxyTx(
		withCtx(txCtx),
		withOriginTx(tx),
	)

	if err != nil {
		return nil, err
	}

	return proxyTx, nil
}

// PingContext
func (c *Conn) PingContext(ctx context.Context) error {
	return c.target.PingContext(ctx)
}

// ExecContext
func (c *Conn) ExecContext(ctx context.Context, query string, args ...interface{}) (gosql.Result, error) {
	return c.target.ExecContext(ctx, query, args...)
}

// PrepareContext
func (c *Conn) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := c.target.PrepareContext(ctx, query)

	if err != nil {
		return nil, err
	}

	return &Stmt{target: stmt}, nil
}

// QueryContext
func (c *Conn) QueryContext(ctx context.Context, query string, args ...interface{}) (*gosql.Rows, error) {
	return c.target.QueryContext(ctx, query, args...)
}

// QueryRowContext
func (c *Conn) QueryRowContext(ctx context.Context, query string, args ...interface{}) *gosql.Row {
	return c.target.QueryRowContext(ctx, query, args...)
}

// Raw
func (c *Conn) Raw(f func(driverConn interface{}) error) error {
	return c.target.Raw(f)
}
