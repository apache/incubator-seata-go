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
	"database/sql/driver"
	"fmt"
	"io"
	"time"

	"github.com/go-sql-driver/mysql"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource/base"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

var (
	capacity    int32 = 1024
	EexpireTime       = 15 * time.Minute
)

type TableMetaCache struct {
	tableMetaCache *base.BaseTableMetaCache
	db             *sql.DB
}

func NewTableMetaInstance(db *sql.DB, cfg *mysql.Config) *TableMetaCache {
	dbName := ""
	if cfg != nil {
		dbName = cfg.DBName
	}

	tableMetaInstance := &TableMetaCache{
		tableMetaCache: base.NewBaseCache(context.Background(), capacity, EexpireTime, NewMysqlTrigger(), db, dbName),
		db:             db,
	}
	return tableMetaInstance
}

func (c *TableMetaCache) ResolveTableMetaKey(ctx context.Context, conn driver.Conn, ref types.TableRef) (types.TableMetaKey, error) {
	if ref.TableName == "" {
		return types.TableMetaKey{}, fmt.Errorf("table name is empty")
	}
	if ref.Qualifier != "" {
		return types.TableMetaKey{DBName: ref.Qualifier, TableName: ref.TableName}, nil
	}

	rows, err := util.CtxDriverQueryWithPrepareFallback(ctx, conn, "SELECT DATABASE()", nil)
	if err != nil {
		return types.TableMetaKey{}, err
	}
	defer rows.Close()
	values := make([]driver.Value, 1)
	if err := rows.Next(values); err != nil {
		if err == io.EOF {
			return types.TableMetaKey{}, fmt.Errorf("current database is not set")
		}
		return types.TableMetaKey{}, err
	}
	var dbName string
	switch value := values[0].(type) {
	case string:
		dbName = value
	case []byte:
		dbName = string(value)
	}
	if dbName == "" {
		return types.TableMetaKey{}, fmt.Errorf("current database is not set")
	}
	return types.TableMetaKey{DBName: dbName, TableName: ref.TableName}, nil
}

// GetTableMeta get table info from cache or information schema
func (c *TableMetaCache) GetTableMeta(ctx context.Context, key types.TableMetaKey) (*types.TableMeta, error) {
	if key.TableName == "" {
		return nil, fmt.Errorf("table name is empty")
	}

	conn, err := c.db.Conn(ctx)
	if err != nil {
		return nil, err
	}

	tableMeta, err := c.tableMetaCache.GetTableMeta(ctx, key, conn)
	if err != nil {
		return nil, err
	}

	return &tableMeta, nil
}

// Destroy
func (c *TableMetaCache) Destroy() error {
	return nil
}
