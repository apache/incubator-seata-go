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

package fence

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

type FenceConn struct {
	TargetConn driver.Conn
	TargetDB   *sql.DB
}

func (c *FenceConn) ResetSession(ctx context.Context) error {
	resetter, ok := c.TargetConn.(driver.SessionResetter)
	if !ok {
		return driver.ErrSkip
	}

	return resetter.ResetSession(ctx)
}

func (c *FenceConn) Prepare(query string) (driver.Stmt, error) {
	return c.TargetConn.Prepare(query)
}

func (c *FenceConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.TargetConn.Prepare(query)
}

func (c *FenceConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	execer, ok := c.TargetConn.(driver.Execer)
	if !ok {
		return nil, driver.ErrSkip
	}

	return execer.Exec(query, args)
}

func (c *FenceConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	execerContext, ok := c.TargetConn.(driver.ExecerContext)
	if !ok {
		values := make([]driver.Value, 0, len(args))
		for i := range args {
			values = append(values, args[i].Value)
		}
		return c.Exec(query, values)
	}

	return execerContext.ExecContext(ctx, query, args)
}

func (c *FenceConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	queryer, ok := c.TargetConn.(driver.Queryer)
	if !ok {
		return nil, driver.ErrSkip
	}

	return queryer.Query(query, args)
}

func (c *FenceConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	QueryerContext, ok := c.TargetConn.(driver.QueryerContext)
	if !ok {
		values := make([]driver.Value, 0, len(args))

		for i := range args {
			values = append(values, args[i].Value)
		}

		return c.Query(query, values)
	}

	return QueryerContext.QueryContext(ctx, query, args)
}

func (c *FenceConn) Begin() (driver.Tx, error) {
	return nil, errors.New("operation unsupport")
}

func (c *FenceConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	beginer, ok := c.TargetConn.(driver.ConnBeginTx)
	if !ok {
		return nil, errors.New("operation unsupported")
	}

	tx, err := beginer.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	if !tm.IsSeataContext(ctx) {
		return nil, errors.New("there is not seata context")
	}

	// check if have been begin fence tx
	if tm.IsFenceTxBegin(ctx) {
		return tx, nil
	}

	tm.SetFenceTxBeginedFlag(ctx, true)

	fenceTx, err := c.TargetDB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if err := fenceTx.Rollback(); err != nil {
				log.Error(err)
			}

			// although it have not any db operations yet, is still rollback to avoid leak tx.
			if err := tx.Rollback(); err != nil {
				log.Error(err)
			}
		}
	}()

	// do fence operations
	emptyCallback := func() error {
		return nil
	}

	if err := WithFence(ctx, fenceTx, emptyCallback); err != nil {
		return nil, err
	}

	return &FenceTx{
		Ctx:           ctx,
		TargetTx:      tx,
		TargetFenceTx: fenceTx,
	}, nil
}

func (c *FenceConn) Close() error {
	return c.TargetConn.Close()
}
