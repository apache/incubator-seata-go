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
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/arana-db/parser/ast"

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

// getPkValues get primary key values for PostgreSQL
func (p *PostgreSQLInsertUndoLogBuilder) getPkValues(execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]driver.Value, error) {
	pkValuesMap := make(map[string][]driver.Value)

	pkColumns := meta.GetPrimaryKeyOnlyName()

	if p.InsertResult != nil && p.InsertResult.GetResult() != nil {
		result := p.InsertResult.GetResult()
		lastInsertId, err := result.LastInsertId()
		if err == nil && lastInsertId != 0 {
			if len(pkColumns) == 1 {
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
	}

	return nil, fmt.Errorf("PostgreSQL insert requires RETURNING clause or sequence information for primary key detection")
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
