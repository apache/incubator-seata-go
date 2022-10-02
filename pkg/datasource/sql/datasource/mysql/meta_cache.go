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
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/seata/seata-go/pkg/datasource/sql/datasource/base"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

var (
	capacity          int32 = 1024
	EexpireTime             = 15 * time.Minute
	tableMetaInstance *tableMetaCache
	tableMetaOnce     sync.Once
	DBName            = "seata"
)

type tableMetaCache struct {
	tableMetaCache *base.BaseTableMetaCache
}

func GetTableMetaInstance() *tableMetaCache {
	// Todo constant.DBName get from config
	tableMetaOnce.Do(func() {
		tableMetaInstance = &tableMetaCache{
			tableMetaCache: base.NewBaseCache(capacity, DBName, EexpireTime, NewMysqlTrigger()),
		}
	})

	return tableMetaInstance
}

// Init
func (c *tableMetaCache) Init(ctx context.Context, conn *sql.DB) error {
	return nil
}

// GetTableMeta get table info from cache or information schema
func (c *tableMetaCache) GetTableMeta(ctx context.Context, tableName string, conn *sql.Conn) (*types.TableMeta, error) {
	if tableName == "" {
		return nil, errors.New("TableMeta cannot be fetched without tableName")
	}

	tableMeta, err := c.tableMetaCache.GetTableMeta(ctx, tableName, conn)
	if err != nil {
		return nil, err
	}

	return &tableMeta, nil
}

// Destroy
func (c *tableMetaCache) Destroy() error {
	return nil
}
