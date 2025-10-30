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

	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/executor"
)

// postgresqlTrigger implement trigger interface for PostgreSQL
type postgresqlTrigger struct{}

// NewPostgresqlTrigger create PostgreSQL trigger instance
func NewPostgresqlTrigger() *postgresqlTrigger {
	return &postgresqlTrigger{}
}

// LoadOne load single table meta
func (p *postgresqlTrigger) LoadOne(ctx context.Context, dbName string, tableName string, conn *sql.Conn) (*types.TableMeta, error) {
	tableMeta := types.TableMeta{
		TableName: tableName,
		Columns:   make(map[string]types.ColumnMeta),
		Indexs:    make(map[string]types.IndexMeta),
	}

	columnMetas, err := p.getColumnMetas(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not find any columnMeta in the table: %s", tableName)
	}

	var columns []string
	for _, columnMeta := range columnMetas {
		tableMeta.Columns[columnMeta.ColumnName] = columnMeta
		columns = append(columns, columnMeta.ColumnName)
	}
	tableMeta.ColumnNames = columns

	indexes, err := p.getIndexes(ctx, dbName, tableName, conn)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not find any index in the table: %s", tableName)
	}
	for _, index := range indexes {
		col, ok := tableMeta.Columns[index.ColumnName]
		if !ok {
			continue
		}

		if existingIndex, exists := tableMeta.Indexs[index.Name]; exists {
			existingIndex.Columns = append(existingIndex.Columns, col)
			tableMeta.Indexs[index.Name] = existingIndex
		} else {
			index.Columns = append(index.Columns, col)
			tableMeta.Indexs[index.Name] = index
		}
	}

	if len(tableMeta.Indexs) == 0 {
		return nil, fmt.Errorf("could not find any index in the table: %s", tableName)
	}

	return &tableMeta, nil
}

// LoadAll load multiple tables meta
func (p *postgresqlTrigger) LoadAll(ctx context.Context, dbName string, conn *sql.Conn, tables ...string) ([]types.TableMeta, error) {
	var tableMetas []types.TableMeta
	for _, tableName := range tables {
		tableMeta, err := p.LoadOne(ctx, dbName, tableName, conn)
		if err != nil {
			continue
		}
		tableMetas = append(tableMetas, *tableMeta)
	}
	return tableMetas, nil
}

// getColumnMetas get column metadata from information_schema.columns
func (p *postgresqlTrigger) getColumnMetas(ctx context.Context, dbName string, tableName string, conn *sql.Conn) ([]types.ColumnMeta, error) {
	tableName = executor.DelEscape(tableName, types.DBTypePostgreSQL)

	columnSQL := `
		SELECT 
			table_name, 
			table_catalog, 
			column_name,    
			data_type,    
			data_type AS column_type,  
			is_nullable,    
			column_default,
			CASE 
				WHEN column_default LIKE 'nextval(%' THEN 'sequence'
				WHEN is_identity = 'YES' THEN 'identity'
				ELSE '' 
			END AS extra,
			COALESCE(is_identity, 'NO') AS is_identity
		FROM 
			information_schema.columns  
		WHERE 
			table_catalog = $1  
			AND table_name = $2 
			AND table_schema = COALESCE(NULLIF($3, ''), 'public')
	`
	stmt, err := conn.PrepareContext(ctx, columnSQL)
	if err != nil {
		return nil, errors.Wrap(err, "prepare column SQL failed")
	}
	defer stmt.Close()

	schema := p.extractSchemaFromDBName(dbName)
	rows, err := stmt.QueryContext(ctx, dbName, tableName, schema)
	if err != nil {
		return nil, errors.Wrapf(err, "query columns failed for table: %s", tableName)
	}
	defer rows.Close()

	columns := make([]types.ColumnMeta, 0)
	for rows.Next() {
		var (
			tableName     string
			tableCatalog  string
			columnName    string
			dataType      string
			columnType    string
			isNullable    string
			columnDefault sql.NullString
			extra         string
			isIdentity    string
		)

		if err := rows.Scan(
			&tableName,
			&tableCatalog,
			&columnName,
			&dataType,
			&columnType,
			&isNullable,
			&columnDefault,
			&extra,
			&isIdentity,
		); err != nil {
			return nil, errors.Wrap(err, "scan column row failed")
		}

		cleanColumnName := strings.Trim(columnName, `" `)

		isAutoIncrement := strings.Contains(strings.ToLower(columnDefault.String), "nextval(") ||
			strings.ToLower(isIdentity) == "yes" ||
			strings.Contains(strings.ToLower(extra), "sequence") ||
			strings.Contains(strings.ToLower(extra), "identity")

		colMeta := types.ColumnMeta{
			Schema:             tableCatalog,
			Table:              tableName,
			ColumnName:         cleanColumnName,
			DatabaseType:       types.GetSqlDataType(dataType),
			DatabaseTypeString: dataType,
			ColumnType:         columnType,
			IsNullable:         0,
			ColumnDef:          []byte(columnDefault.String),
			Extra:              extra,
			Autoincrement:      isAutoIncrement,
		}

		if strings.ToLower(isNullable) == "yes" {
			colMeta.IsNullable = 1
		}
		columns = append(columns, colMeta)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows iteration error")
	}

	pkSQL := `
		SELECT 
			kcu.column_name 
		FROM 
			information_schema.table_constraints tco
		INNER JOIN 
			information_schema.key_column_usage kcu
			ON tco.constraint_name = kcu.constraint_name
			AND tco.table_catalog = kcu.table_catalog
			AND tco.table_schema = kcu.table_schema
			AND tco.table_name = kcu.table_name
		WHERE 
			tco.table_catalog = $1
			AND tco.table_name = $2
			AND tco.table_schema = COALESCE(NULLIF($3, ''), 'public')
			AND tco.constraint_type = 'PRIMARY KEY'
	`
	pkStmt, err := conn.PrepareContext(ctx, pkSQL)
	if err != nil {
		return nil, errors.Wrap(err, "prepare primary key SQL failed")
	}
	defer pkStmt.Close()

	pkRows, err := pkStmt.QueryContext(ctx, dbName, tableName, schema)
	if err != nil {
		return nil, errors.Wrapf(err, "query primary keys failed for table: %s", tableName)
	}
	defer pkRows.Close()

	pkColumns := make(map[string]bool)
	for pkRows.Next() {
		var colName string
		if err := pkRows.Scan(&colName); err != nil {
			return nil, errors.Wrap(err, "scan primary key row failed")
		}
		pkColumns[strings.Trim(colName, `" `)] = true
	}

	if err := pkRows.Err(); err != nil {
		return nil, errors.Wrap(err, "primary key rows iteration error")
	}

	for i := range columns {
		if pkColumns[columns[i].ColumnName] {
			columns[i].ColumnKey = "PRI"
		}
	}

	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns found for table: %s (check if table exists in schema 'public')", tableName)
	}

	return columns, nil
}

// getIndexes
func (p *postgresqlTrigger) getIndexes(ctx context.Context, dbName string, tableName string, conn *sql.Conn) ([]types.IndexMeta, error) {
	tableName = executor.DelEscape(tableName, types.DBTypePostgreSQL)

	indexSQL := `
		SELECT 
        	c.relname AS index_name,
        	a.attname AS column_name,
        	CASE WHEN idx.indisunique THEN 0 ELSE 1 END AS non_unique,
        	idx.indisprimary AS is_primary,
        	u.ord AS column_position
    	FROM 
        	pg_catalog.pg_index idx
    	JOIN 
        	pg_catalog.pg_class c ON c.oid = idx.indexrelid
    	JOIN 
        	pg_catalog.pg_class t ON t.oid = idx.indrelid
    	JOIN 
        	pg_catalog.pg_namespace n ON n.oid = t.relnamespace
    	JOIN 
        	pg_catalog.pg_attribute a ON a.attrelid = idx.indrelid
    	JOIN 
        	unnest(idx.indkey) WITH ORDINALITY AS u(attnum, ord) ON a.attnum = u.attnum
    	WHERE 
        	n.nspname = COALESCE(NULLIF($2, ''), 'public')
        	AND t.relname = $1
    	ORDER BY c.relname, u.ord
	`
	stmt, err := conn.PrepareContext(ctx, indexSQL)
	if err != nil {
		return nil, errors.Wrap(err, "prepare index SQL failed")
	}
	defer stmt.Close()

	schema := p.extractSchemaFromDBName(dbName)
	rows, err := stmt.QueryContext(ctx, tableName, schema)
	if err != nil {
		return nil, errors.Wrapf(err, "query indexes failed for table: %s", tableName)
	}
	defer rows.Close()

	indexes := make([]types.IndexMeta, 0)
	for rows.Next() {
		var (
			indexName      string
			columnName     string
			nonUnique      int64
			isPrimary      bool
			columnPosition int64
		)

		if err := rows.Scan(&indexName, &columnName, &nonUnique, &isPrimary, &columnPosition); err != nil {
			return nil, errors.Wrap(err, "scan index row failed")
		}

		cleanColumnName := strings.Trim(columnName, `" `)
		indexMeta := types.IndexMeta{
			Schema:     dbName,
			Table:      tableName,
			Name:       indexName,
			ColumnName: cleanColumnName,
			Columns:    make([]types.ColumnMeta, 0),
			NonUnique:  nonUnique == 1,
		}

		if isPrimary {
			indexMeta.IType = types.IndexTypePrimaryKey
		} else if !indexMeta.NonUnique {
			indexMeta.IType = types.IndexUnique
		} else {
			indexMeta.IType = types.IndexNormal
		}

		indexes = append(indexes, indexMeta)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "rows iteration error for indexes")
	}

	return indexes, nil
}

func (p *postgresqlTrigger) extractSchemaFromDBName(dbName string) string {
	if !strings.Contains(dbName, ".") {
		return ""
	}

	parts := strings.Split(dbName, ".")
	if len(parts) == 2 {
		if strings.Contains(parts[0], "localhost") ||
			strings.Contains(parts[0], "127.0.0.1") ||
			strings.Contains(parts[0], "::1") ||
			strings.HasPrefix(parts[0], "postgres") {
			return ""
		}
		return parts[1]
	}

	if len(parts) >= 3 {
		return ""
	}

	return ""
}
