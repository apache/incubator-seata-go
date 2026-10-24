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
	"strings"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

type postgresTrigger struct {
	getColumnMetasFn func(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) ([]types.ColumnMeta, error)
	getIndexesFn     func(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) ([]types.IndexMeta, error)
}

func NewPostgresTrigger() *postgresTrigger {
	return &postgresTrigger{}
}

func (p *postgresTrigger) LoadOne(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) (*types.TableMeta, error) {
	tableMeta := types.TableMeta{
		TableName: key.Schema + "." + key.TableName,
		Columns:   make(map[string]types.ColumnMeta),
		Indexs:    make(map[string]types.IndexMeta),
	}

	columnMetas, err := p.getColumnMetas(ctx, key, conn)
	if err != nil {
		return nil, fmt.Errorf("load postgres column metadata for table %s: %w", key.TableName, err)
	}

	columns := make([]string, 0, len(columnMetas))
	for _, columnMeta := range columnMetas {
		tableMeta.Columns[columnMeta.ColumnName] = columnMeta
		columns = append(columns, columnMeta.ColumnName)
	}
	tableMeta.ColumnNames = columns

	indexes, err := p.getIndexes(ctx, key, conn)
	if err != nil {
		return nil, fmt.Errorf("load postgres index metadata for table %s: %w", key.TableName, err)
	}

	for _, index := range indexes {
		col := tableMeta.Columns[index.ColumnName]
		idx, ok := tableMeta.Indexs[index.Name]
		if ok {
			idx.Columns = append(idx.Columns, col)
			tableMeta.Indexs[index.Name] = idx
			continue
		}

		index.Columns = append(index.Columns, col)
		tableMeta.Indexs[index.Name] = index
	}

	if len(tableMeta.Indexs) == 0 {
		return nil, fmt.Errorf("could not find any index in the table: %s", key.TableName)
	}

	return &tableMeta, nil
}

func (p *postgresTrigger) LoadAll(ctx context.Context, conn *sql.Conn, keys ...types.TableMetaKey) (map[types.TableMetaKey]types.TableMeta, error) {
	tableMetas := make(map[types.TableMetaKey]types.TableMeta, len(keys))
	for _, key := range keys {
		tableMeta, err := p.LoadOne(ctx, key, conn)
		if err != nil {
			continue
		}

		tableMetas[key] = *tableMeta
	}

	return tableMetas, nil
}

func (p *postgresTrigger) getColumnMetas(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) ([]types.ColumnMeta, error) {
	if p.getColumnMetasFn != nil {
		return p.getColumnMetasFn(ctx, key, conn)
	}

	query := util.RewritePlaceholders(
		"SELECT table_name, table_schema, column_name, data_type, udt_name, is_nullable, column_default, is_identity "+
			"FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position",
		types.DBTypePostgreSQL,
	)

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(key.Schema, key.TableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnMetas := make([]types.ColumnMeta, 0)
	for rows.Next() {
		var (
			tableName     string
			tableSchema   string
			columnName    string
			dataType      string
			udtName       string
			isNullable    string
			columnDefault sql.NullString
			isIdentity    string
		)

		if err = rows.Scan(
			&tableName,
			&tableSchema,
			&columnName,
			&dataType,
			&udtName,
			&isNullable,
			&columnDefault,
			&isIdentity,
		); err != nil {
			return nil, err
		}

		columnMeta := types.ColumnMeta{
			Schema:             tableSchema,
			Table:              tableName,
			ColumnName:         strings.Trim(columnName, `" `),
			DatabaseType:       types.GetSqlDataType(dataType),
			DatabaseTypeString: dataType,
			ColumnType:         udtName,
			ColumnDef:          []byte(columnDefault.String),
		}
		if strings.EqualFold(isNullable, "yes") {
			columnMeta.IsNullable = 1
		}
		if strings.EqualFold(isIdentity, "yes") {
			columnMeta.Extra = "identity"
			columnMeta.Autoincrement = true
		} else if columnDefault.Valid && strings.Contains(strings.ToLower(columnDefault.String), "nextval(") {
			columnMeta.Autoincrement = true
		}

		columnMetas = append(columnMetas, columnMeta)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(columnMetas) == 0 {
		return nil, fmt.Errorf("can't find column")
	}

	return columnMetas, nil
}

func (p *postgresTrigger) getIndexes(ctx context.Context, key types.TableMetaKey, conn *sql.Conn) ([]types.IndexMeta, error) {
	if p.getIndexesFn != nil {
		return p.getIndexesFn(ctx, key, conn)
	}

	query := util.RewritePlaceholders(
		"SELECT ic.relname AS index_name, a.attname AS column_name, ix.indisprimary, ix.indisunique "+
			"FROM pg_class tc "+
			"JOIN pg_namespace ns ON ns.oid = tc.relnamespace "+
			"JOIN pg_index ix ON tc.oid = ix.indrelid "+
			"JOIN pg_class ic ON ic.oid = ix.indexrelid "+
			"JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS cols(attnum, ordinality) ON TRUE "+
			"JOIN pg_attribute a ON a.attrelid = tc.oid AND a.attnum = cols.attnum "+
			"WHERE ns.nspname = ? AND tc.relname = ? ORDER BY ic.relname, cols.ordinality",
		types.DBTypePostgreSQL,
	)

	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(key.Schema, key.TableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]types.IndexMeta, 0)
	for rows.Next() {
		var (
			indexName  string
			columnName string
			isPrimary  bool
			isUnique   bool
		)

		if err = rows.Scan(&indexName, &columnName, &isPrimary, &isUnique); err != nil {
			return nil, err
		}

		index := types.IndexMeta{
			Schema:     key.Schema,
			Table:      key.TableName,
			Name:       indexName,
			ColumnName: columnName,
			Columns:    make([]types.ColumnMeta, 0),
			NonUnique:  !isUnique,
		}

		switch {
		case isPrimary:
			index.IType = types.IndexTypePrimaryKey
		case isUnique:
			index.IType = types.IndexUnique
		default:
			index.IType = types.IndexNormal
		}

		result = append(result, index)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
