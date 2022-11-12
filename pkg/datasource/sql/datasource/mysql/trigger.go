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
	"database/sql/driver"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/executor"
)

const (
	columnMetaSql = "SELECT `TABLE_NAME`, `TABLE_SCHEMA`, `COLUMN_NAME`, `DATA_TYPE`, `COLUMN_TYPE`, `COLUMN_KEY`, `IS_NULLABLE`, `EXTRA` FROM INFORMATION_SCHEMA.COLUMNS WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"
	indexMetaSql  = "SELECT `INDEX_NAME`, `COLUMN_NAME`, `NON_UNIQUE`, `INDEX_TYPE`, `COLLATION`, `CARDINALITY` FROM `INFORMATION_SCHEMA`.`STATISTICS` WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ?"
)

type mysqlTrigger struct {
}

func NewMysqlTrigger() *mysqlTrigger {
	return &mysqlTrigger{}
}

// LoadOne get table meta column and index
func (m *mysqlTrigger) LoadOne(ctx context.Context, dbName string, tableName string, conn driver.Conn) (*types.TableMeta, error) {
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
		idx, ok := tableMeta.Indexs[index.Name]
		if ok {
			idx.Values = append(idx.Values, col)
			tableMeta.Indexs[index.Name] = idx
		} else {
			index.Values = append(index.Values, col)
			tableMeta.Indexs[index.Name] = index
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
func (m *mysqlTrigger) getColumns(ctx context.Context, dbName string, table string, conn driver.Conn) ([]types.ColumnMeta, error) {
	table = executor.DelEscape(table, types.DBTypeMySQL)
	var columnMetas []types.ColumnMeta

	stmt, err := conn.Prepare(columnMetaSql)
	if err != nil {
		return nil, err
	}

	rowsi, err := stmt.Query([]driver.Value{dbName, table})
	if err != nil {
		return nil, err
	}

	for {
		vals := make([]driver.Value, 8)
		err = rowsi.Next(vals)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var (
			tableName   = string(vals[0].([]uint8))
			tableSchema = string(vals[1].([]uint8))
			columnName  = string(vals[2].([]uint8))
			dataType    = string(vals[3].([]uint8))
			columnType  = string(vals[4].([]uint8))
			columnKey   = string(vals[5].([]uint8))
			isNullable  = string(vals[6].([]uint8))
			extra       = string(vals[7].([]uint8))
		)

		col := types.ColumnMeta{}
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
		col.Autoincrement = strings.Contains(strings.ToLower(extra), "auto_increment")

		columnMetas = append(columnMetas, col)
	}

	if len(columnMetas) == 0 {
		return nil, errors.New("can't find column")
	}

	return columnMetas, nil
}

// getIndex get tableMetaIndex
func (m *mysqlTrigger) getIndexes(ctx context.Context, dbName string, tableName string, conn driver.Conn) ([]types.IndexMeta, error) {
	tableName = executor.DelEscape(tableName, types.DBTypeMySQL)
	result := make([]types.IndexMeta, 0)

	stmt, err := conn.Prepare(indexMetaSql)
	if err != nil {
		return nil, err
	}

	rowsi, err := stmt.Query([]driver.Value{dbName, tableName})
	if err != nil {
		return nil, err
	}

	defer rowsi.Close()

	for {
		vals := make([]driver.Value, 6)
		err = rowsi.Next(vals)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		var (
			indexName  = string(vals[0].([]uint8))
			columnName = string(vals[1].([]uint8))
			nonUnique  = vals[2].(int64)
			//indexType   = string(vals[3].([]uint8))
			//collation   = string(vals[4].([]uint8))
			//cardinality = int(vals[6].([]uint8))
		)

		index := types.IndexMeta{
			Schema:     dbName,
			Table:      tableName,
			Name:       indexName,
			ColumnName: columnName,
			Values:     make([]types.ColumnMeta, 0),
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
