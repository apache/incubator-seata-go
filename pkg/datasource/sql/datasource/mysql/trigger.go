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
	"strings"

	"github.com/seata/seata-go/pkg/datasource/sql/undo/executor"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

type mysqlTrigger struct {
}

func NewMysqlTrigger() *mysqlTrigger {
	return &mysqlTrigger{}
}

// LoadOne get table meta column and index
func (m *mysqlTrigger) LoadOne(ctx context.Context, dbName string, tableName string, conn *sql.Conn) (*types.TableMeta, error) {
	tableMeta := types.TableMeta{
		Name:    tableName,
		Columns: make(map[string]types.ColumnMeta),
		Indexs:  make(map[string]types.IndexMeta),
	}

	colMetas, err := m.getColumns(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not found any column in the table: %s", tableName)
	}

	var columns []string
	for _, column := range colMetas {
		tableMeta.Columns[column.ColumnName] = column
		columns = append(columns, column.ColumnName)
	}
	tableMeta.ColumnNames = columns

	indexes, err := m.getIndexes(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not found any index in the table: %s", tableName)
	}
	for _, index := range indexes {
		col := tableMeta.Columns[index.ColumnName]
		idx, ok := tableMeta.Indexs[index.IndexName]
		if ok {
			idx.Values = append(idx.Values, col)
		} else {
			index.Values = append(index.Values, col)
			tableMeta.Indexs[index.IndexName] = index
		}
	}
	if len(tableMeta.Indexs) == 0 {
		return nil, errors.Errorf("Could not found any index in the table: %s", tableName)
	}

	return &tableMeta, nil
}

// LoadAll
func (m *mysqlTrigger) LoadAll() ([]types.TableMeta, error) {
	return []types.TableMeta{}, nil
}

// getColumns get tableMeta column
func (m *mysqlTrigger) getColumns(ctx context.Context, dbName string, table string, conn *sql.Conn) ([]types.ColumnMeta, error) {
	table = executor.DelEscape(table, types.DBTypeMySQL)

	var result []types.ColumnMeta

	columnSchemaSql := "select TABLE_CATALOG, TABLE_NAME, TABLE_SCHEMA, COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, COLUMN_KEY, " +
		" IS_NULLABLE, EXTRA from INFORMATION_SCHEMA.COLUMNS where `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	stmt, err := conn.PrepareContext(ctx, columnSchemaSql)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, dbName, table)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var (
			tableCatalog string
			tableName    string
			tableSchema  string
			columnName   string
			dataType     string
			columnType   string
			columnKey    string
			isNullable   string
			extra        string
		)

		col := types.ColumnMeta{}

		if err = rows.Scan(
			&tableCatalog,
			&tableName,
			&tableSchema,
			&columnName,
			&dataType,
			&columnType,
			&columnKey,
			&isNullable,
			&extra); err != nil {
			return nil, err
		}

		col.Schema = tableSchema
		col.Table = tableName
		col.ColumnName = strings.Trim(columnName, "` ")
		col.DataType = types.GetSqlDataType(dataType)
		col.ColumnType = columnType
		col.ColumnKey = columnKey
		if strings.ToLower(isNullable) == "yes" {
			col.IsNullable = 1
		} else {
			col.IsNullable = 0
		}
		col.Extra = extra
		col.Autoincrement = strings.Contains("auto_increment", strings.ToLower(extra))

		result = append(result, col)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if err = rows.Close(); err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("can't find column")
	}

	return result, nil
}

// getIndex get tableMetaIndex
func (m *mysqlTrigger) getIndexes(ctx context.Context, dbName string, tableName string, conn *sql.Conn) ([]types.IndexMeta, error) {
	tableName = executor.DelEscape(tableName, types.DBTypeMySQL)

	result := make([]types.IndexMeta, 0)

	indexSchemaSql := "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE`, `INDEX_TYPE`, `COLLATION`, `CARDINALITY` " +
		"FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"

	stmt, err := conn.PrepareContext(ctx, indexSchemaSql)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.QueryContext(ctx, dbName, tableName)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			indexName, columnName, nonUnique, indexType, collation string
			cardinality                                            int
		)

		if err = rows.Scan(
			&indexName,
			&columnName,
			&nonUnique,
			&indexType,
			&collation,
			&cardinality); err != nil {
			return nil, err
		}

		index := types.IndexMeta{
			Schema:     dbName,
			Table:      tableName,
			IndexName:  indexName,
			ColumnName: columnName,
			Values:     make([]types.ColumnMeta, 0),
		}

		if nonUnique == "1" || "yes" == strings.ToLower(nonUnique) {
			index.NonUnique = true
		}

		if "primary" == strings.ToLower(indexName) {
			index.IndexType = types.IndexPrimary
		} else if !index.NonUnique {
			index.IndexType = types.IndexUnique
		} else {
			index.IndexType = types.IndexNormal
		}

		result = append(result, index)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
