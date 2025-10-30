// pkg/datasource/sql/mock_conn.go
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

package mock

import (
	"context"
	"database/sql/driver"
	"io"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"strings"
)

type GenericMockConn struct {
	dbType types.DBType
}

func NewGenericMockConn(dbType types.DBType) *GenericMockConn {
	return &GenericMockConn{dbType: dbType}
}

func (m *GenericMockConn) Ping(ctx context.Context) error {
	return nil
}

func (m *GenericMockConn) Close() error {
	return nil
}

func (m *GenericMockConn) Begin() (driver.Tx, error) {
	return &GenericMockTx{}, nil
}

func (m *GenericMockConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return &GenericMockTx{}, nil
}

func (m *GenericMockConn) Prepare(query string) (driver.Stmt, error) {
	return &GenericMockStmt{
		dbType: m.dbType,
		query:  query,
	}, nil
}

func (m *GenericMockConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return &GenericMockStmt{
		dbType: m.dbType,
		query:  query,
	}, nil
}

func (m *GenericMockConn) Exec(query string, args []driver.Value) (driver.Result, error) {
	return &GenericMockResult{}, nil
}

func (m *GenericMockConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return &GenericMockResult{}, nil
}

func (m *GenericMockConn) Query(query string, args []driver.Value) (driver.Rows, error) {
	if m.dbType == types.DBTypePostgreSQL &&
		strings.Contains(query, "pg_catalog.pg_sequences") &&
		strings.Contains(query, "test_id_seq") {
		return NewPostgreSQLSequenceRows(), nil
	}
	return NewGenericMockRows(nil, nil), nil
}

func (m *GenericMockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if m.dbType == types.DBTypePostgreSQL &&
		strings.Contains(query, "pg_catalog.pg_sequences") &&
		strings.Contains(query, "test_id_seq") {
		return NewPostgreSQLSequenceRows(), nil
	}
	return NewGenericMockRows(nil, nil), nil
}

func (m *GenericMockConn) ResetSession(ctx context.Context) error {
	return nil
}

type GenericMockTx struct{}

func (m *GenericMockTx) Commit() error   { return nil }
func (m *GenericMockTx) Rollback() error { return nil }

type GenericMockStmt struct {
	dbType types.DBType
	query  string
}

func (m *GenericMockStmt) Close() error  { return nil }
func (m *GenericMockStmt) NumInput() int { return 0 }
func (m *GenericMockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &GenericMockResult{}, nil
}
func (m *GenericMockStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return &GenericMockResult{}, nil
}
func (m *GenericMockStmt) Query(args []driver.Value) (driver.Rows, error) {
	if m.dbType == types.DBTypePostgreSQL &&
		strings.Contains(m.query, "pg_catalog.pg_sequences") &&
		strings.Contains(m.query, "test_id_seq") {
		return NewPostgreSQLSequenceRows(), nil
	}
	return NewGenericMockRows(nil, nil), nil
}
func (m *GenericMockStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if m.dbType == types.DBTypePostgreSQL &&
		strings.Contains(m.query, "pg_catalog.pg_sequences") &&
		strings.Contains(m.query, "test_id_seq") {
		return NewPostgreSQLSequenceRows(), nil
	}
	return NewGenericMockRows(nil, nil), nil
}

type GenericMockResult struct{}

func (m *GenericMockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *GenericMockResult) RowsAffected() (int64, error) { return 0, nil }

type PostgreSQLSequenceRows struct {
	columns []string
	data    [][]driver.Value
	index   int
}

func NewPostgreSQLSequenceRows() *PostgreSQLSequenceRows {
	return &PostgreSQLSequenceRows{
		columns: []string{"increment_by"},
		data:    [][]driver.Value{{int64(1)}},
	}
}

func (r *PostgreSQLSequenceRows) Columns() []string {
	return r.columns
}

func (r *PostgreSQLSequenceRows) Close() error {
	return nil
}

func (r *PostgreSQLSequenceRows) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}
	for i, v := range r.data[r.index] {
		dest[i] = v
	}
	r.index++
	return nil
}
