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

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/base"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	capacity      int32 = 1024
	EexpireTime         = 15 * time.Minute
	tableMetaOnce sync.Once
)

type TableMetaCache struct {
	tableMetaCache *base.BaseTableMetaCache
	db             *sql.DB
}

func NewTableMetaInstance(db *sql.DB) *TableMetaCache {
	tableMetaInstance := &TableMetaCache{
		tableMetaCache: base.NewBaseCache(capacity, EexpireTime, NewMysqlTrigger()),
		db:             db,
	}
	return tableMetaInstance
}

// Init
func (c *TableMetaCache) Init(ctx context.Context, conn *sql.DB) error {
	return nil
}

// GetTableMeta get table info from cache or information schema
func (c *TableMetaCache) GetTableMeta(ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
	if tableName == "" {
		return nil, fmt.Errorf("table name is empty")
	}

	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	tableMeta, err := c.tableMetaCache.GetTableMeta(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, err
	}

	return &tableMeta, nil
}

// Destroy
func (c *TableMetaCache) Destroy() error {
	return nil
}
