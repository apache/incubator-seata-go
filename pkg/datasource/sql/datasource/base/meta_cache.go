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

package base

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v4"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

type trigger interface {
	LoadOne(ctx context.Context, dbName string, table string, conn *sql.Conn) (*types.TableMeta, error)
	LoadAll(ctx context.Context, dbName string, conn *sql.Conn, tables ...string) ([]types.TableMeta, error)
}

type entry struct {
	value      types.TableMeta
	lastAccess time.Time
}

// BaseTableMetaCache
type BaseTableMetaCache struct {
	lock           sync.RWMutex
	expireDuration time.Duration
	capacity       int32
	size           int32
	cache          map[string]*entry
	cancel         context.CancelFunc
	trigger        trigger
	db             *sql.DB
	cfg            interface{}
}

// NewBaseCache
func NewBaseCache(capacity int32, expireDuration time.Duration, trigger trigger, db *sql.DB, cfg interface{}) *BaseTableMetaCache {
	ctx, cancel := context.WithCancel(context.Background())

	c := &BaseTableMetaCache{
		lock:           sync.RWMutex{},
		capacity:       capacity,
		size:           0,
		expireDuration: expireDuration,
		cache:          map[string]*entry{},
		cancel:         cancel,
		trigger:        trigger,
		cfg:            cfg,
		db:             db,
	}

	c.Init(ctx)

	return c
}

func (c *BaseTableMetaCache) enforceCapacity() {
	if c.capacity <= 0 || int32(len(c.cache)) <= c.capacity {
		return
	}

	type lruEntry struct {
		key        string
		lastAccess time.Time
	}

	entries := make([]lruEntry, 0, len(c.cache))
	for key, entry := range c.cache {
		entries = append(entries, lruEntry{
			key:        key,
			lastAccess: entry.lastAccess,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].lastAccess.Before(entries[j].lastAccess)
	})

	removeCount := int32(len(c.cache)) - c.capacity
	for i := int32(0); i < removeCount; i++ {
		delete(c.cache, entries[i].key)
	}
}

// init
func (c *BaseTableMetaCache) Init(ctx context.Context) error {
	go c.refresh(ctx)
	go c.scanExpire(ctx)

	return nil
}

// refresh
func (c *BaseTableMetaCache) refresh(ctx context.Context) {
	f := func(ctx context.Context) bool {
		if c.db == nil || c.cfg == nil || c.cache == nil {
			return true
		}

		select {
		case <-ctx.Done():
			log.Printf("refresh: Received a cancel signal, preparing to exit")
			return false
		default:
		}

		tables := make([]string, 0, len(c.cache))
		for table := range c.cache {
			tables = append(tables, table)
		}

		dbName, err := c.getDBName()
		if err != nil {
			return true
		}

		dbType, err := c.getDBType()
		if err != nil {
			log.Printf("refresh: failed to get db type, err: %v", err)
			return true
		}

		conn, err := c.db.Conn(ctx)
		if err != nil {
			return true
		}
		defer conn.Close()
		tableMetas, err := c.trigger.LoadAll(ctx, dbName, conn, tables...)
		if err != nil {
			return true
		}

		c.lock.Lock()
		defer c.lock.Unlock()
		for i := range tableMetas {
			tableMetas[i].DBType = dbType
			upperTableName := strings.ToUpper(tableMetas[i].TableName)
			c.cache[upperTableName] = &entry{
				value:      tableMetas[i],
				lastAccess: time.Now(),
			}
		}
		c.enforceCapacity()
		return true
	}

	if !f(ctx) {
		return
	}

	ticker := time.NewTicker(c.expireDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("refresh: Context cancellation detected, exiting loop")
			return
		case <-ticker.C:
			if !f(ctx) {
				return
			}
		}
	}
}

// scanExpire
func (c *BaseTableMetaCache) scanExpire(ctx context.Context) {
	ticker := time.NewTicker(c.expireDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.lock.Lock()

			cur := time.Now()
			for k, entry := range c.cache {
				if cur.Sub(entry.lastAccess) > c.expireDuration {
					delete(c.cache, k)
				}
			}

			c.lock.Unlock()
		}
	}
}

// GetTableMeta
func (c *BaseTableMetaCache) GetTableMeta(ctx context.Context, dbName, tableName string, conn *sql.Conn) (types.TableMeta, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	upperTableName := strings.ToUpper(tableName)
	e, ok := c.cache[upperTableName]
	if ok {
		e.lastAccess = time.Now()
		c.cache[upperTableName] = e
		return e.value, nil
	}

	dbType, err := c.getDBType()
	if err != nil {
		return types.TableMeta{}, fmt.Errorf("get db type failed: %w", err)
	}

	meta, err := c.trigger.LoadOne(ctx, dbName, upperTableName, conn)
	if err != nil {
		return types.TableMeta{}, err
	}
	if meta == nil || meta.IsEmpty() {
		return types.TableMeta{}, fmt.Errorf("not found table metadata for %s", tableName)
	}

	meta.DBType = dbType

	c.cache[upperTableName] = &entry{
		value:      *meta,
		lastAccess: time.Now(),
	}
	c.enforceCapacity()
	return *meta, nil
}

func (c *BaseTableMetaCache) getDBType() (types.DBType, error) {
	switch cfg := c.cfg.(type) {
	case *mysql.Config:
		return types.DBTypeMySQL, nil
	case string:
		if _, err := pgx.ParseConfig(cfg); err != nil {
			return types.DBTypeUnknown, fmt.Errorf("invalid postgresql dsn: %w", err)
		}
		return types.DBTypePostgreSQL, nil
	default:
		return types.DBTypeUnknown, fmt.Errorf("unsupported config type: %T", cfg)
	}
}

func (c *BaseTableMetaCache) Destroy() error {
	c.cancel()
	return nil
}

func (c *BaseTableMetaCache) getDBName() (string, error) {
	switch cfg := c.cfg.(type) {
	case *mysql.Config:
		return cfg.DBName, nil
	case string:
		pgxCfg, err := pgx.ParseConfig(cfg)
		if err != nil {
			return "", fmt.Errorf("failed to parse postgresql dsn: %w", err)
		}
		return pgxCfg.Database, nil
	default:
		return "", fmt.Errorf("unsupported config type: %T", cfg)
	}
}
