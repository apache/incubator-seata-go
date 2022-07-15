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
	"errors"
	"sync"
	"time"

	"github.com/seata/seata-go-datasource/sql/types"
)

type (
	trigger interface {
		LoadOne(table string) (types.TableMeta, error)

		LoadAll() ([]types.TableMeta, error)
	}

	entry struct {
		value      types.TableMeta
		lastAccess time.Time
	}
)

// BaseTableMetaCache
type BaseTableMetaCache struct {
	lock           sync.RWMutex
	expireDuration time.Duration
	capity         int32
	size           int32
	cache          map[string]*entry
	cancel         context.CancelFunc
	trigger        trigger
}

// NewBaseCache
func NewBaseCache(capity int32, expireDuration time.Duration, trigger trigger) (*BaseTableMetaCache, error) {
	ctx, cancel := context.WithCancel(context.Background())

	c := &BaseTableMetaCache{
		lock:           sync.RWMutex{},
		capity:         capity,
		size:           0,
		expireDuration: expireDuration,
		cache:          map[string]*entry{},
		cancel:         cancel,
		trigger:        trigger,
	}

	if err := c.Init(ctx); err != nil {
		return nil, err
	}

	return c, nil
}

// init
func (c *BaseTableMetaCache) Init(ctx context.Context) error {
	go c.refresh(ctx)
	go c.scanExpire(ctx)

	return nil
}

// refresh
func (c *BaseTableMetaCache) refresh(ctx context.Context) {
	f := func() {
		v, err := c.trigger.LoadAll()
		if err != nil {
		}

		c.lock.Lock()
		defer c.lock.Unlock()

		for i := range v {
			tm := v[i]
			if _, ok := c.cache[tm.Name]; !ok {
				c.cache[tm.Name] = &entry{
					value: tm,
				}
			}
		}
	}

	f()

	ticker := time.NewTicker(time.Duration(1 * time.Minute))
	for range ticker.C {
		f()
	}
}

// scanExpire
func (c *BaseTableMetaCache) scanExpire(ctx context.Context) {
	ticker := time.NewTicker(c.expireDuration)

	for range ticker.C {

		f := func() {
			c.lock.Lock()
			defer c.lock.Unlock()

			cur := time.Now()
			for k := range c.cache {
				entry := c.cache[k]

				if cur.Sub(entry.lastAccess) > c.expireDuration {
					delete(c.cache, k)
				}
			}
		}

		f()
	}
}

// GetTableMeta
func (c *BaseTableMetaCache) GetTableMeta(table string) (types.TableMeta, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	v, ok := c.cache[table]

	if !ok {
		meta, err := c.trigger.LoadOne(table)
		if err != nil {
			return types.TableMeta{}, err
		}

		if !meta.IsEmpty() {
			c.cache[table] = &entry{
				value:      meta,
				lastAccess: time.Now(),
			}

			return meta, nil
		}

		return types.TableMeta{}, errors.New("not found table metadata")
	}

	v.lastAccess = time.Now()
	c.cache[table] = v

	return v.value, nil
}

func (c *BaseTableMetaCache) Destory() error {
	c.cancel()
	return nil
}
