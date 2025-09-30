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

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/base"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	capacity   int32         = 1024
	expireTime time.Duration = 15 * time.Minute
)

// TableMetaCache
type TableMetaCache struct {
	tableMetaCache *base.BaseTableMetaCache
	db             *sql.DB
	dsn            string
}

// NewTableMetaInstance
func NewTableMetaInstance(db *sql.DB, dsn string) *TableMetaCache {
	return &TableMetaCache{
		tableMetaCache: base.NewBaseCache(capacity, expireTime, NewPostgresqlTrigger(), db, dsn),
		db:             db,
		dsn:            dsn,
	}
}

func (c *TableMetaCache) Init(ctx context.Context, conn *sql.DB) error {
	return nil
}

func (c *TableMetaCache) GetTableMeta(ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
	if tableName == "" {
		return nil, fmt.Errorf("table name is empty")
	}

	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	tableMeta, err := c.tableMetaCache.GetTableMeta(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, err
	}

	return &tableMeta, nil
}

func (c *TableMetaCache) Destroy() error {
	return c.tableMetaCache.Destroy()
}
