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
	"database/sql"
	gosql "database/sql"
	"time"
)

type DB struct {
	target *gosql.DB
}

// Close
func (db *DB) Close() error {
	return db.target.Close()
}

// PingContext
func (db *DB) Ping(ctx context.Context) error {
	return db.target.PingContext(ctx)
}

// SetConnMaxIdleTime
func (db *DB) SetConnMaxIdleTime(d time.Duration) {
	db.target.SetConnMaxIdleTime(d)
}

// SetConnMaxLifetime
func (db *DB) SetConnMaxLifetime(d time.Duration) {
	db.target.SetConnMaxLifetime(d)
}

// SetMaxIdleConns
func (db *DB) SetMaxIdleConns(n int) {
	db.target.SetMaxIdleConns(n)
}

// SetMaxOpenConns
func (db *DB) SetMaxOpenConns(n int) {
	db.target.SetMaxOpenConns(n)
}

// Begin
func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	tx, err := db.target.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &Tx{target: tx}, nil
}

// BeginTx
func (db *DB) BeginTx(ctx context.Context, opt *sql.TxOptions) (*Tx, error) {
	tx, err := db.target.BeginTx(ctx, opt)
	if err != nil {
		return nil, err
	}

	return &Tx{target: tx}, nil
}

// QueryContext
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*gosql.Rows, error) {
	return db.target.QueryContext(ctx, query, args...)
}

// QueryRowContext
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *gosql.Row {
	return db.target.QueryRowContext(ctx, query, args...)
}

// ExecContext
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (gosql.Result, error) {
	return db.target.ExecContext(ctx, query, args...)
}

// Conn
func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	conn, err := db.target.Conn(ctx)

	if err != nil {
		return nil, err
	}

	return &Conn{target: conn}, nil
}

// PrepareContext
func (db *DB) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	stmt, err := db.target.PrepareContext(ctx, query)

	if err != nil {
		return nil, err
	}

	return &Stmt{target: stmt}, nil
}

// Stats
func (db *DB) Stats() gosql.DBStats {
	return db.target.Stats()
}
