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

package builder

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/log"
)

type PostgreSQLInsertUndoLogBuilder struct {
	BasicUndoLogBuilder
	// InsertResult after insert sql
	InsertResult  types.ExecResult
	IncrementStep int
}

func GetPostgreSQLInsertUndoLogBuilder() undo.UndoLogBuilder {
	return &PostgreSQLInsertUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
	}
}

func (p *PostgreSQLInsertUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.InsertExecutor
}

func (p *PostgreSQLInsertUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	return []*types.RecordImage{}, nil
}

func (p *PostgreSQLInsertUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if execCtx == nil || execCtx.ParseContext == nil {
		return nil, nil
	}

	var tableName string
	var err error

	// Support both MySQL and PostgreSQL AST
	if execCtx.ParseContext.InsertStmt != nil {
		// MySQL AST
		tableName = execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	} else if execCtx.ParseContext.AuxtenInsertStmt != nil {
		// PostgreSQL AST
		tableName, err = execCtx.ParseContext.GetTableName()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	metaData := execCtx.MetaDataMap[tableName]
	selectSQL, selectArgs, err := p.buildAfterImageSQL(ctx, execCtx)
	if err != nil {
		return nil, err
	}

	stmt, err := execCtx.Conn.Prepare(selectSQL)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}

	image, err := p.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}

	return []*types.RecordImage{image}, nil
}

// buildAfterImageSQL build select SQL for PostgreSQL after insert
func (p *PostgreSQLInsertUndoLogBuilder) buildAfterImageSQL(ctx context.Context, execCtx *types.ExecContext) (string, []driver.Value, error) {
	if execCtx == nil || execCtx.ParseContext == nil {
		return "", nil, fmt.Errorf("can't find execCtx or ParseContext")
	}

	var tableName string
	var err error

	// Get table name from appropriate AST
	if execCtx.ParseContext.InsertStmt != nil {
		// MySQL AST
		tableName = execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	} else if execCtx.ParseContext.AuxtenInsertStmt != nil {
		// PostgreSQL AST
		tableName, err = execCtx.ParseContext.GetTableName()
		if err != nil {
			return "", nil, err
		}
	} else {
		return "", nil, fmt.Errorf("can't find valid insert statement")
	}

	if execCtx.MetaDataMap == nil {
		return "", nil, fmt.Errorf("can't find MetaDataMap")
	}

	meta := execCtx.MetaDataMap[tableName]
	pkValuesMap, err := p.getPkValues(execCtx, execCtx.ParseContext, meta)
	if err != nil {
		return "", nil, err
	}

	return p.buildSelectSQLByPKValues(tableName, meta.GetPrimaryKeyOnlyName(), pkValuesMap)
}

func (p *PostgreSQLInsertUndoLogBuilder) getPkValues(execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]driver.Value, error) {
	pkValuesMap := make(map[string][]driver.Value)
	pkColumns := meta.GetPrimaryKeyOnlyName()

	if len(pkColumns) == 1 && p.InsertResult != nil && p.InsertResult.GetResult() != nil {
		result := p.InsertResult.GetResult()
		lastInsertId, err := result.LastInsertId()
		if err == nil && lastInsertId != 0 {
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return nil, err
			}

			values := make([]driver.Value, 0)
			for i := int64(0); i < rowsAffected; i++ {
				values = append(values, lastInsertId+i)
			}
			pkValuesMap[pkColumns[0]] = values
			return pkValuesMap, nil
		}
	}

	pkValuesFromInsert, err := p.extractPkValuesFromInsert(parseCtx, meta)
	if err == nil && len(pkValuesFromInsert) > 0 {
		return pkValuesFromInsert, nil
	}

	seqValues, err := p.getPkValuesFromSequence(execCtx, meta)
	if err == nil && len(seqValues) > 0 {
		return seqValues, nil
	}

	return nil, fmt.Errorf("PostgreSQL insert primary key detection failed: cannot determine primary key values. Consider using RETURNING clause in INSERT statement")
}

func (p *PostgreSQLInsertUndoLogBuilder) extractPkValuesFromInsert(parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]driver.Value, error) {
	pkColumns := meta.GetPrimaryKeyOnlyName()
	pkValuesMap := make(map[string][]driver.Value)

	if parseCtx.AuxtenInsertStmt != nil {
		insertStmt := parseCtx.AuxtenInsertStmt
		if insertStmt.Columns != nil && insertStmt.Rows != nil {
			colIndexMap := make(map[string]int)
			for i, col := range insertStmt.Columns {
				colName := col.String()
				colName = strings.Trim(colName, `" `)
				colIndexMap[strings.ToLower(colName)] = i
			}

			for _, pkCol := range pkColumns {
				pkColLower := strings.ToLower(pkCol)
				if colIdx, exists := colIndexMap[pkColLower]; exists {
					values := make([]driver.Value, 0)
					if selectStmt, ok := insertStmt.Rows.Select.(*tree.ValuesClause); ok {
						for _, row := range selectStmt.Rows {
							if colIdx < len(row) {
								if datum, ok := row[colIdx].(*tree.StrVal); ok {
									values = append(values, datum.RawString())
								} else if datum, ok := row[colIdx].(*tree.NumVal); ok {
									values = append(values, datum.String())
								}
							}
						}
					}
					if len(values) > 0 {
						pkValuesMap[pkCol] = values
					}
				}
			}
		}
	} else if parseCtx.InsertStmt != nil {
		insertStmt := parseCtx.InsertStmt
		if insertStmt.Columns != nil && insertStmt.Lists != nil {
			colIndexMap := make(map[string]int)
			for i, col := range insertStmt.Columns {
				colName := col.Name.O
				colIndexMap[strings.ToLower(colName)] = i
			}

			for _, pkCol := range pkColumns {
				pkColLower := strings.ToLower(pkCol)
				if colIdx, exists := colIndexMap[pkColLower]; exists {
					values := make([]driver.Value, 0)
					for _, row := range insertStmt.Lists {
						if colIdx < len(row) {
						}
					}
					if len(values) > 0 {
						pkValuesMap[pkCol] = values
					}
				}
			}
		}
	}

	return pkValuesMap, nil
}

func (p *PostgreSQLInsertUndoLogBuilder) getPkValuesFromSequence(execCtx *types.ExecContext, meta types.TableMeta) (map[string][]driver.Value, error) {
	pkValuesMap := make(map[string][]driver.Value)
	pkColumns := meta.GetPrimaryKeyOnlyName()

	if p.InsertResult == nil || p.InsertResult.GetResult() == nil {
		return pkValuesMap, fmt.Errorf("no insert result available")
	}

	rowsAffected, err := p.InsertResult.GetResult().RowsAffected()
	if err != nil || rowsAffected == 0 {
		return pkValuesMap, fmt.Errorf("no rows affected")
	}

	for _, pkCol := range pkColumns {
		colMeta, exists := meta.Columns[pkCol]
		if !exists {
			continue
		}

		if colMeta.Autoincrement || strings.Contains(strings.ToLower(colMeta.Extra), "serial") {
			seqValues, err := p.querySequenceValues(execCtx, meta.TableName, pkCol, int(rowsAffected))
			if err == nil && len(seqValues) > 0 {
				pkValuesMap[pkCol] = seqValues
			}
		}
	}

	return pkValuesMap, nil
}

func (p *PostgreSQLInsertUndoLogBuilder) querySequenceValues(execCtx *types.ExecContext, tableName, columnName string, rowCount int) ([]driver.Value, error) {
	seqQuery := `SELECT pg_get_serial_sequence($1, $2) AS sequence_name`
	
	stmt, err := execCtx.Conn.Prepare(seqQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare sequence query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query([]driver.Value{tableName, columnName})
	if err != nil {
		return nil, fmt.Errorf("failed to query sequence: %w", err)
	}
	defer rows.Close()

	columns := rows.Columns()
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns returned from sequence query")
	}

	values := make([]driver.Value, len(columns))
	err = rows.Next(values)
	if err != nil {
		return nil, fmt.Errorf("no sequence found for column %s.%s", tableName, columnName)
	}

	var sequenceName sql.NullString
	if values[0] != nil {
		if str, ok := values[0].(string); ok {
			sequenceName.String = str
			sequenceName.Valid = true
		}
	}
	
	if !sequenceName.Valid {
		return nil, fmt.Errorf("no sequence found for column %s.%s", tableName, columnName)
	}

	currValQuery := fmt.Sprintf("SELECT currval('%s')", sequenceName.String)
	stmt2, err := execCtx.Conn.Prepare(currValQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare currval query: %w", err)
	}
	defer stmt2.Close()

	rows2, err := stmt2.Query([]driver.Value{})
	if err != nil {
		return nil, fmt.Errorf("failed to query currval: %w", err)
	}
	defer rows2.Close()

	columns2 := rows2.Columns()
	if len(columns2) == 0 {
		return nil, fmt.Errorf("no columns returned from currval query")
	}

	values2 := make([]driver.Value, len(columns2))
	err = rows2.Next(values2)
	if err != nil {
		return nil, fmt.Errorf("failed to get current sequence value")
	}

	var currentVal int64
	if values2[0] != nil {
		if val, ok := values2[0].(int64); ok {
			currentVal = val
		} else if val, ok := values2[0].(string); ok {
			if parsed, err := fmt.Sscanf(val, "%d", &currentVal); err != nil || parsed != 1 {
				return nil, fmt.Errorf("failed to parse current sequence value: %s", val)
			}
		}
	}

	result := make([]driver.Value, 0, rowCount)
	startVal := currentVal - int64(rowCount) + 1
	for i := 0; i < rowCount; i++ {
		result = append(result, startVal+int64(i))
	}

	return result, nil
}

// buildSelectSQLByPKValues build select SQL using primary key values for PostgreSQL
func (p *PostgreSQLInsertUndoLogBuilder) buildSelectSQLByPKValues(tableName string, pkNameList []string, pkValuesMap map[string][]driver.Value) (string, []driver.Value, error) {
	if len(pkValuesMap) == 0 {
		return "", nil, fmt.Errorf("no primary key values found")
	}

	var sql strings.Builder
	sql.WriteString("SELECT ")

	sql.WriteString("*")

	sql.WriteString(" FROM ")
	sql.WriteString(fmt.Sprintf(`"%s"`, tableName))
	sql.WriteString(" WHERE ")

	var conditions []string
	var args []driver.Value
	argIndex := 1

	if len(pkNameList) == 1 {
		pkName := pkNameList[0]
		values := pkValuesMap[pkName]
		if len(values) == 0 {
			return "", nil, fmt.Errorf("no values for primary key %s", pkName)
		}

		conditions = append(conditions, fmt.Sprintf(`"%s" IN (%s)`, pkName, p.buildPostgreSQLPlaceholders(len(values), &argIndex)))
		args = append(args, values...)
	} else {
		maxRows := 0
		for _, values := range pkValuesMap {
			if len(values) > maxRows {
				maxRows = len(values)
			}
		}

		for i := 0; i < maxRows; i++ {
			var rowConditions []string
			for _, pkName := range pkNameList {
				values := pkValuesMap[pkName]
				if i < len(values) {
					rowConditions = append(rowConditions, fmt.Sprintf(`"%s" = $%d`, pkName, argIndex))
					args = append(args, values[i])
					argIndex++
				}
			}
			if len(rowConditions) > 0 {
				conditions = append(conditions, "("+strings.Join(rowConditions, " AND ")+")")
			}
		}
	}

	sql.WriteString(strings.Join(conditions, " OR "))
	return sql.String(), args, nil
}

// buildPostgreSQLPlaceholders build PostgreSQL style placeholders ($1, $2, ...)
func (p *PostgreSQLInsertUndoLogBuilder) buildPostgreSQLPlaceholders(count int, argIndex *int) string {
	var placeholders []string
	for i := 0; i < count; i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", *argIndex))
		(*argIndex)++
	}
	return strings.Join(placeholders, ", ")
}
