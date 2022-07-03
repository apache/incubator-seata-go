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
	"time"

	"github.com/seata/seata-go-datasource/sql/datasource"
	"github.com/seata/seata-go-datasource/sql/types"
	"github.com/seata/seata-go-datasource/sql/undo"
)

type dbOption func(db *DB)

func withResourceID(id string) dbOption {
	return func(db *DB) {
		db.resourceID = id
	}
}

func withTableMetaCache(c datasource.TableMetaCache) dbOption {
	return func(db *DB) {
		db.metaCache = c
	}
}

func withDBType(dt types.DBType) dbOption {
	return func(db *DB) {
		db.dbType = dt
	}
}

func withTarget(source *gosql.DB) dbOption {
	return func(db *DB) {
		db.target = source
	}
}

func withConf(conf *seataServerConfig) dbOption {
	return func(db *DB) {
		db.conf = *conf
	}
}

func newDB(opts ...dbOption) *DB {
	db := new(DB)

	for i := range opts {
		opts[i](db)
	}

	return db
}

// DB proxy sql.DB, enchance database/sql.DB to add distribute transaction ability
type DB struct {
	resourceID string
	// conf
	conf seataServerConfig
	// target
	target *gosql.DB
	// dbType
	dbType types.DBType
	// undoLogMgr
	undoLogMgr undo.UndoLogManager
	// metaCache
	metaCache datasource.TableMetaCache
}

// Close
func (db *DB) Close() error {
	return db.target.Close()
}

// Ping
func (db *DB) Ping() error {
	return db.target.PingContext(context.Background())
}

// PingContext
func (db *DB) PingContext(ctx context.Context) error {
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

// Begin only turn on local transaction, and distribute transaction need to be called BeginTx func
func (db *DB) Begin() (*Tx, error) {
	tx, err := db.target.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	txCtx := types.NewTxContext(
		types.WithTransType(types.Local),
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

// BeginTx
func (db *DB) BeginTx(ctx context.Context, opt *types.TxOptions) (*Tx, error) {
	tx, err := db.target.BeginTx(ctx, &gosql.TxOptions{
		Isolation: opt.Isolation,
		ReadOnly:  opt.ReadOnly,
	})
	if err != nil {
		return nil, err
	}

	txCtx := types.NewTxContext(
		types.WithTxOptions(opt),
		types.WithTransType(opt.TransType),
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
