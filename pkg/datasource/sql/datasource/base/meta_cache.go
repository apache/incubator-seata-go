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
	"sync"
	"sync/atomic"
	"time"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

type (
	// trigger
	trigger interface {
		LoadOne(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) (*types.TableMeta, error)

		LoadAll(ctx context.Context, conn *sql.Conn, keys ...types.TableMetaKey) (map[types.TableMetaKey]types.TableMeta, error)
	}

	entry struct {
		value types.TableMeta
		// lastAccess is unix nanoseconds and must only be accessed through
		// sync/atomic: the cache-hit fast path in GetTableMeta updates it
		// while holding no lock at all, so a plain field would race against
		// scanExpire and concurrent readers.
		lastAccess int64
	}
)

// BaseTableMetaCache
type BaseTableMetaCache struct {
	lock            sync.RWMutex
	cancel          context.CancelFunc
	expireDuration  time.Duration
	refreshInterval time.Duration
	capity          int32
	size            int32
	cache           map[types.TableMetaKey]*entry
	trigger         trigger
	db              *sql.DB
	dbName          string
}

// NewBaseCache
func NewBaseCache(ctx context.Context, capity int32, expireDuration time.Duration, trigger trigger, db *sql.DB, dbName string) *BaseTableMetaCache {
	// WithCancel derives a cancellable child even from context.Background(),
	// so Destroy() can always stop the background goroutines; nothing in
	// production calls Destroy yet, which is the remaining lifetime gap.
	ctx, cancel := context.WithCancel(ctx)
	c := &BaseTableMetaCache{
		lock:            sync.RWMutex{},
		capity:          capity,
		size:            0,
		expireDuration:  expireDuration,
		refreshInterval: time.Minute,
		cache:           map[types.TableMetaKey]*entry{},
		trigger:         trigger,
		dbName:          dbName,
		db:              db,
		cancel:          cancel,
	}

	c.Init(ctx)

	return c
}

// Init
func (c *BaseTableMetaCache) Init(ctx context.Context) error {
	if c.db == nil {
		return nil
	}

	go c.refresh(ctx)
	go c.scanExpire(ctx)
	return nil
}

// refresh
func (c *BaseTableMetaCache) refresh(ctx context.Context) {
	f := func() {
		// Get cached identities with read lock
		c.lock.RLock()
		if c.db == nil || c.cache == nil || len(c.cache) == 0 {
			c.lock.RUnlock()
			return
		}

		keys := make([]types.TableMetaKey, 0, len(c.cache))
		for key := range c.cache {
			keys = append(keys, key)
		}
		c.lock.RUnlock()

		// Load metadata without lock (I/O operation)
		conn, err := c.db.Conn(ctx)
		if err != nil {
			return
		}
		defer conn.Close()
		refreshed, err := c.trigger.LoadAll(ctx, conn, keys...)
		if err != nil {
			return
		}

		// Update cache with write lock
		c.lock.Lock()
		defer c.lock.Unlock()

		for key, meta := range refreshed {
			if cached, ok := c.cache[key]; ok {
				cached.value = meta
			}
		}
	}

	f()

	ticker := time.NewTicker(c.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			f()
		case <-ctx.Done():
			return
		}
	}
}

// scanExpire
func (c *BaseTableMetaCache) scanExpire(ctx context.Context) {
	ticker := time.NewTicker(c.expireDuration)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			f := func() {
				c.lock.Lock()
				defer c.lock.Unlock()

				cur := time.Now()
				for k := range c.cache {
					entry := c.cache[k]

					// lastAccess is unix nanos (atomic, see the entry comment);
					// a wall-clock jump can only make expiry marginally early or
					// late, which is acceptable for a cache heuristic.
					if cur.UnixNano()-atomic.LoadInt64(&entry.lastAccess) > c.expireDuration.Nanoseconds() {
						delete(c.cache, k)
					}
				}
			}

			f()
		case <-ctx.Done():
			return
		}
	}
}

// GetTableMeta
func (c *BaseTableMetaCache) GetTableMeta(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) (types.TableMeta, error) {
	// The caller hands over a wrapper-opened conn it never closes, so this
	// method owns it on every return path — including the cache-hit fast
	// path below (a hit-path early return used to leak the pooled
	// *sql.Conn until the pool ran dry).
	defer conn.Close()

	// Read path: serve cache hits under the read lock instead of queueing
	// behind the write lock, which could be held across information_schema
	// I/O — previously even hits serialized behind a cold load.
	c.lock.RLock()
	v, ok := c.cache[key]
	var cached types.TableMeta
	if ok {
		// Copy the payload while still holding the read lock: refresh()
		// replaces entry.value under the write lock, so an unlocked read
		// could observe a torn TableMeta.
		cached = v.value
	}
	c.lock.RUnlock()
	if ok {
		// Lock-free touch: lastAccess is atomic, see the entry comment.
		atomic.StoreInt64(&v.lastAccess, time.Now().UnixNano())
		return cached, nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	v, ok = c.cache[key]
	if !ok {
		meta, err := c.trigger.LoadOne(ctx, key, conn)
		if err != nil {
			return types.TableMeta{}, err
		}

		if meta != nil && !meta.IsEmpty() {
			e := &entry{value: *meta}
			atomic.StoreInt64(&e.lastAccess, time.Now().UnixNano())
			c.cache[key] = e

			return *meta, nil
		}

		return types.TableMeta{}, fmt.Errorf("not found table metadata")
	}

	atomic.StoreInt64(&v.lastAccess, time.Now().UnixNano())

	return v.value, nil
}

// Destroy stops the refresh and scanExpire goroutines. The derived context
// is cancellable regardless of the parent passed to NewBaseCache, so calling
// Destroy always works; note that no production wiring invokes it today
// (datasource managers never call cache Destroy), so in practice the
// goroutines live for the process lifetime.
func (c *BaseTableMetaCache) Destroy() error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}
