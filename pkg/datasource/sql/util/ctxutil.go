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

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"context"
	"database/sql/driver"
	"errors"
)

func ctxDriverPrepare(ctx context.Context, ci driver.Conn, query string) (driver.Stmt, error) {
	if ciCtx, is := ci.(driver.ConnPrepareContext); is {
		return ciCtx.PrepareContext(ctx, query)
	}
	si, err := ci.Prepare(query)
	if err == nil {
		select {
		default:
		case <-ctx.Done():
			si.Close()
			return nil, ctx.Err()
		}
	}
	return si, err
}

func ctxDriverExec(ctx context.Context, execerCtx driver.ExecerContext, execer driver.Execer, query string, nvdargs []driver.NamedValue) (driver.Result, error) {
	if execerCtx != nil {
		return execerCtx.ExecContext(ctx, query, nvdargs)
	}
	dargs, err := namedValueToValue(nvdargs)
	if err != nil {
		return nil, err
	}

	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return execer.Exec(query, dargs)
}

func CtxDriverQuery(ctx context.Context, queryerCtx driver.QueryerContext, queryer driver.Queryer, query string, nvdargs []driver.NamedValue) (driver.Rows, error) {
	if queryerCtx != nil {
		return queryerCtx.QueryContext(ctx, query, nvdargs)
	}
	dargs, err := namedValueToValue(nvdargs)
	if err != nil {
		return nil, err
	}

	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return queryer.Query(query, dargs)
}

func ctxDriverStmtExec(ctx context.Context, si driver.Stmt, nvdargs []driver.NamedValue) (driver.Result, error) {
	if siCtx, is := si.(driver.StmtExecContext); is {
		return siCtx.ExecContext(ctx, nvdargs)
	}
	dargs, err := namedValueToValue(nvdargs)
	if err != nil {
		return nil, err
	}

	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return si.Exec(dargs)
}

func ctxDriverStmtQuery(ctx context.Context, si driver.Stmt, nvdargs []driver.NamedValue) (driver.Rows, error) {
	if siCtx, is := si.(driver.StmtQueryContext); is {
		return siCtx.QueryContext(ctx, nvdargs)
	}
	dargs, err := namedValueToValue(nvdargs)
	if err != nil {
		return nil, err
	}

	select {
	default:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return si.Query(dargs)
}

func namedValueToValue(named []driver.NamedValue) ([]driver.Value, error) {
	dargs := make([]driver.Value, len(named))
	for n, param := range named {
		if len(param.Name) > 0 {
			return nil, errors.New("sql: driver does not support the use of Named Parameters")
		}
		dargs[n] = param.Value
	}
	return dargs, nil
}

type rowsWithStmt struct {
	driver.Rows
	stmt driver.Stmt
}

// NewRowsWithStmt wraps rows and closes both rows and stmt when the returned rows are closed.
func NewRowsWithStmt(rows driver.Rows, stmt driver.Stmt) driver.Rows {
	return &rowsWithStmt{Rows: rows, stmt: stmt}
}

func (r *rowsWithStmt) Close() error {
	var rowsErr error
	if r.Rows != nil {
		rowsErr = r.Rows.Close()
	}

	var stmtErr error
	if r.stmt != nil {
		stmtErr = r.stmt.Close()
	}

	return errors.Join(rowsErr, stmtErr)
}

// CtxDriverExecWithPrepareFallback first tries the connection-level Exec path.
// If the driver returns driver.ErrSkip, it prepares and executes the statement
// directly on the underlying driver connection.
func CtxDriverExecWithPrepareFallback(ctx context.Context, conn driver.Conn, query string, args []driver.NamedValue) (driver.Result, error) {
	var execerContext driver.ExecerContext
	if execer, ok := conn.(driver.ExecerContext); ok {
		execerContext = execer
	}

	var execer driver.Execer
	if e, ok := conn.(driver.Execer); ok {
		execer = e
	}

	if execerContext != nil || execer != nil {
		result, err := ctxDriverExec(ctx, execerContext, execer, query, args)
		if err == nil {
			return result, nil
		}

		if !errors.Is(err, driver.ErrSkip) {
			return nil, err
		}
	}

	stmt, err := ctxDriverPrepare(ctx, conn, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	return ctxDriverStmtExec(ctx, stmt, args)
}

func CtxDriverQueryWithPrepareFallback(ctx context.Context, conn driver.Conn, query string, args []driver.NamedValue) (driver.Rows, error) {
	var queryerContext driver.QueryerContext
	if queryer, ok := conn.(driver.QueryerContext); ok {
		queryerContext = queryer
	}

	var queryer driver.Queryer
	if q, ok := conn.(driver.Queryer); ok {
		queryer = q
	}

	if queryerContext != nil || queryer != nil {
		rows, err := CtxDriverQuery(ctx, queryerContext, queryer, query, args)
		if err == nil {
			return rows, nil
		}

		if !errors.Is(err, driver.ErrSkip) {
			return nil, err
		}
	}

	stmt, err := ctxDriverPrepare(ctx, conn, query)
	if err != nil {
		return nil, err
	}

	rows, err := ctxDriverStmtQuery(ctx, stmt, args)
	if err != nil {
		_ = stmt.Close()
		return nil, err
	}

	return NewRowsWithStmt(rows, stmt), nil
}
