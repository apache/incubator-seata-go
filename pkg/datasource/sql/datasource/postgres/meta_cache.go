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
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"time"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource/base"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

var (
	capacity   int32 = 1024
	expireTime       = 15 * time.Minute
)

type TableMetaCache struct {
	tableMetaCache *base.BaseTableMetaCache
	db             *sql.DB
}

func NewTableMetaInstance(db *sql.DB, dbName string) *TableMetaCache {
	return &TableMetaCache{
		tableMetaCache: base.NewBaseCache(context.Background(), capacity, expireTime, NewPostgresTrigger(), db, dbName),
		db:             db,
	}
}

func (c *TableMetaCache) ResolveTableMetaKey(ctx context.Context, conn driver.Conn, ref types.TableRef) (types.TableMetaKey, error) {
	if ref.TableName == "" {
		return types.TableMetaKey{}, fmt.Errorf("table name is empty")
	}
	name := postgresIdentifier(ref.TableName, ref.TableNameQuoted)
	if ref.Qualifier != "" {
		name = postgresIdentifier(ref.Qualifier, ref.QualifierQuoted) + "." + name
	}
	const query = "SELECT current_database(), ns.nspname, c.relname FROM pg_class c JOIN pg_namespace ns ON ns.oid = c.relnamespace WHERE c.oid = to_regclass($1)"
	rows, err := util.CtxDriverQueryWithPrepareFallback(ctx, conn, query, []driver.NamedValue{{Ordinal: 1, Value: name}})
	if err != nil {
		return types.TableMetaKey{}, err
	}
	defer rows.Close()
	values := make([]driver.Value, 3)
	if err := rows.Next(values); err != nil {
		if err == io.EOF {
			return types.TableMetaKey{}, fmt.Errorf("table %s not found", name)
		}
		return types.TableMetaKey{}, err
	}
	getString := func(value driver.Value) string {
		switch value := value.(type) {
		case string:
			return value
		case []byte:
			return string(value)
		default:
			return ""
		}
	}
	key := types.TableMetaKey{DBName: getString(values[0]), Schema: getString(values[1]), TableName: getString(values[2])}
	if key.DBName == "" || key.Schema == "" || key.TableName == "" {
		return types.TableMetaKey{}, fmt.Errorf("incomplete metadata identity for table %s", name)
	}
	return key, nil
}

func postgresIdentifier(name string, quoted bool) string {
	if quoted {
		return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
	}
	return name
}

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

func (c *TableMetaCache) Destroy() error {
	return nil
}
