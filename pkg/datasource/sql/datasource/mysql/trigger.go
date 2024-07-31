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
	"strings"

	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/executor"
)

type mysqlTrigger struct {
}

func NewMysqlTrigger() *mysqlTrigger {
	return &mysqlTrigger{}
}

// LoadOne get table meta column and index
func (m *mysqlTrigger) LoadOne(ctx context.Context, dbName string, tableName string, conn *sql.Conn) (*types.TableMeta, error) {
	tableMeta := types.TableMeta{
		TableName: tableName,
		Columns:   make(map[string]types.ColumnMeta),
		Indexs:    make(map[string]types.IndexMeta),
	}

	columnMetas, err := m.getColumnMetas(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not found any columnMeta in the table: %s", tableName)
	}

	var columns []string
	for _, columnMeta := range columnMetas {
		tableMeta.Columns[columnMeta.ColumnName] = columnMeta
		columns = append(columns, columnMeta.ColumnName)
	}
	tableMeta.ColumnNames = columns

	indexes, err := m.getIndexes(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not found any index in the table: %s", tableName)
	}
	for _, index := range indexes {
		col := tableMeta.Columns[index.ColumnName]
		idx, ok := tableMeta.Indexs[index.Name]
		if ok {
			idx.Columns = append(idx.Columns, col)
			tableMeta.Indexs[index.Name] = idx
		} else {
			index.Columns = append(index.Columns, col)
			tableMeta.Indexs[index.Name] = index
		}
	}
	if len(tableMeta.Indexs) == 0 {
		return nil, fmt.Errorf("could not found any index in the table: %s", tableName)
	}

	return &tableMeta, nil
}

// LoadAll
func (m *mysqlTrigger) LoadAll() ([]types.TableMeta, error) {
	return []types.TableMeta{}, nil
}

// getColumnMetas get tableMeta column
func (m *mysqlTrigger) getColumnMetas(ctx context.Context, dbName string, table string, conn *sql.Conn) ([]types.ColumnMeta, error) {
	table = executor.DelEscape(table, types.DBTypeMySQL)
	var columnMetas []types.ColumnMeta

	columnMetaSql := "SELECT `TABLE_NAME`, `TABLE_SCHEMA`, `COLUMN_NAME`, `DATA_TYPE`, `COLUMN_TYPE`, `COLUMN_KEY`, `IS_NULLABLE`, `COLUMN_DEFAULT`, `EXTRA` FROM INFORMATION_SCHEMA.COLUMNS WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"
	stmt, err := conn.PrepareContext(ctx, columnMetaSql)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(dbName, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			tableName     string
			tableSchema   string
			columnName    string
			dataType      string
			columnType    string
			columnKey     string
			isNullable    string
			columnDefault []byte
			extra         string
		)

		columnMeta := types.ColumnMeta{}
		if err = rows.Scan(
			&tableName,
			&tableSchema,
			&columnName,
			&dataType,
			&columnType,
			&columnKey,
			&isNullable,
			&columnDefault,
			&extra); err != nil {
			return nil, err
		}

		columnMeta.Schema = tableSchema
		columnMeta.Table = tableName
		columnMeta.ColumnName = strings.Trim(columnName, "` ")
		columnMeta.DatabaseType = types.GetSqlDataType(dataType)
		columnMeta.DatabaseTypeString = dataType
		columnMeta.ColumnType = columnType
		columnMeta.ColumnKey = columnKey
		if strings.ToLower(isNullable) == "yes" {
			columnMeta.IsNullable = 1
		} else {
			columnMeta.IsNullable = 0
		}
		columnMeta.ColumnDef = columnDefault
		columnMeta.Extra = extra
		columnMeta.Autoincrement = strings.Contains(strings.ToLower(extra), "auto_increment")
		columnMetas = append(columnMetas, columnMeta)
	}

	if len(columnMetas) == 0 {
		return nil, fmt.Errorf("can't find column")
	}

	return columnMetas, nil
}

// getIndex get tableMetaIndex
func (m *mysqlTrigger) getIndexes(ctx context.Context, dbName string, tableName string, conn *sql.Conn) ([]types.IndexMeta, error) {
	tableName = executor.DelEscape(tableName, types.DBTypeMySQL)
	result := make([]types.IndexMeta, 0)

	indexMetaSql := "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE` FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"
	stmt, err := conn.PrepareContext(ctx, indexMetaSql)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			indexName  string
			columnName string
			nonUnique  int64
		)

		err = rows.Scan(&indexName, &columnName, &nonUnique)
		if err != nil {
			return nil, err
		}

		index := types.IndexMeta{
			Schema:     dbName,
			Table:      tableName,
			Name:       indexName,
			ColumnName: columnName,
			Columns:    make([]types.ColumnMeta, 0),
		}

		if nonUnique == 1 {
			index.NonUnique = true
		}

		if "primary" == strings.ToLower(indexName) {
			index.IType = types.IndexTypePrimaryKey
		} else if !index.NonUnique {
			index.IType = types.IndexUnique
		} else {
			index.IType = types.IndexNormal
		}

		result = append(result, index)

	}

	return result, nil
}
