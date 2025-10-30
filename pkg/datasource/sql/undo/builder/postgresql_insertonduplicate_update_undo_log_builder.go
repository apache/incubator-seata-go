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
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/log"
)

type PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder struct {
	BasicUndoLogBuilder
	InsertResult  types.ExecResult
	IncrementStep int
}

func GetPostgreSQLInsertOnDuplicateUpdateUndoLogBuilder() undo.UndoLogBuilder {
	return &PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
	}
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.InsertOnDuplicateExecutor
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	if execCtx == nil || execCtx.ParseContext == nil {
		return nil, nil
	}

	var tableName string
	var err error

	if execCtx.ParseContext.InsertStmt != nil {
		tableName = execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	} else if execCtx.ParseContext.AuxtenInsertStmt != nil {
		tableName, err = execCtx.ParseContext.GetTableName()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	metaData, err := datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableName)
	if err != nil {
		return nil, err
	}

	selectSQL, selectArgs, err := p.buildBeforeImageSQL(ctx, execCtx, metaData)
	if err != nil {
		return nil, err
	}

	if selectSQL == "" {
		return []*types.RecordImage{}, nil
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

	image, err := p.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}

	return []*types.RecordImage{image}, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if execCtx == nil || execCtx.ParseContext == nil {
		return nil, nil
	}

	var tableName string
	var err error

	if execCtx.ParseContext.InsertStmt != nil {
		tableName = execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	} else if execCtx.ParseContext.AuxtenInsertStmt != nil {
		tableName, err = execCtx.ParseContext.GetTableName()
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	metaData, err := datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableName)
	if err != nil {
		return nil, err
	}

	selectSQL, selectArgs, err := p.buildAfterImageSQL(ctx, execCtx, metaData)
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

	image, err := p.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}

	return []*types.RecordImage{image}, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) buildBeforeImageSQL(ctx context.Context, execCtx *types.ExecContext, metaData *types.TableMeta) (string, []driver.Value, error) {
	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, 0)
		for _, param := range execCtx.NamedValues {
			vals = append(vals, param.Value)
		}
	}

	paramMap, err := p.buildImageParameters(execCtx, vals, metaData)
	if err != nil {
		return "", nil, err
	}

	var sql strings.Builder
	sql.WriteString("SELECT * FROM ")
	sql.WriteString(fmt.Sprintf(`"%s"`, metaData.TableName))

	var selectArgs []driver.Value
	isContainWhere := false

	for _, index := range metaData.Indexs {
		if index.NonUnique || !p.isIndexValueNotNull(index, paramMap) {
			continue
		}

		columnIsNull := true
		var uniqueList []string
		var paramAppenderTempList []driver.Value

		for _, columnMeta := range index.Columns {
			columnName := strings.ToLower(columnMeta.ColumnName)
			imageParameters, ok := paramMap[columnName]
			if !ok && columnMeta.ColumnDef != nil {
				uniqueList = append(uniqueList, fmt.Sprintf(`"%s" = DEFAULT("%s")`, columnName, columnName))
				columnIsNull = false
				continue
			}

			columnIsNull = false
			uniqueList = append(uniqueList, fmt.Sprintf(`"%s" = $%d`, columnName, len(selectArgs)+1))
			paramAppenderTempList = append(paramAppenderTempList, imageParameters[0])
		}

		if !columnIsNull {
			if isContainWhere {
				sql.WriteString(" OR (" + strings.Join(uniqueList, " AND ") + ")")
			} else {
				sql.WriteString(" WHERE (" + strings.Join(uniqueList, " AND ") + ")")
				isContainWhere = true
			}
			selectArgs = append(selectArgs, paramAppenderTempList...)
		}
	}

	if !isContainWhere {
		return "", nil, fmt.Errorf("no primary key or unique index found for conflict detection")
	}

	return sql.String(), selectArgs, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) buildAfterImageSQL(ctx context.Context, execCtx *types.ExecContext, metaData *types.TableMeta) (string, []driver.Value, error) {
	pkValuesMap, err := p.getPkValues(execCtx, execCtx.ParseContext, *metaData)
	if err != nil {
		return "", nil, err
	}

	return p.buildSelectSQLByPKValues(metaData.TableName, metaData.GetPrimaryKeyOnlyName(), pkValuesMap)
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) extractPrimaryKeyValues(execCtx *types.ExecContext, vals []driver.Value, metaData *types.TableMeta) ([]map[string]driver.Value, error) {
	var result []map[string]driver.Value

	if execCtx.ParseContext.InsertStmt != nil {
		return p.extractPKFromMySQLInsert(execCtx.ParseContext.InsertStmt, vals, metaData)
	} else if execCtx.ParseContext.AuxtenInsertStmt != nil {
		return p.extractPKFromPostgreSQLInsert(execCtx.ParseContext.AuxtenInsertStmt, vals, metaData)
	}

	return result, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) extractPKFromMySQLInsert(stmt *ast.InsertStmt, vals []driver.Value, metaData *types.TableMeta) ([]map[string]driver.Value, error) {
	var result []map[string]driver.Value

	if len(stmt.Columns) == 0 || len(stmt.Lists) == 0 {
		return result, nil
	}

	pkColumns := metaData.GetPrimaryKeyOnlyName()
	pkIndexes := make(map[string]int)

	for i, col := range stmt.Columns {
		columnName := col.Name.O
		for _, pk := range pkColumns {
			if strings.EqualFold(columnName, pk) {
				pkIndexes[pk] = i
			}
		}
	}

	valueIndex := 0
	for _, valueList := range stmt.Lists {
		pkValueMap := make(map[string]driver.Value)
		for pk, colIndex := range pkIndexes {
			if colIndex < len(valueList) && valueIndex < len(vals) {
				pkValueMap[pk] = vals[valueIndex+colIndex]
			}
		}
		if len(pkValueMap) > 0 {
			result = append(result, pkValueMap)
		}
		valueIndex += len(valueList)
	}

	return result, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) extractPKFromPostgreSQLInsert(stmt *tree.Insert, vals []driver.Value, metaData *types.TableMeta) ([]map[string]driver.Value, error) {
	var result []map[string]driver.Value

	if stmt.Columns == nil {
		return result, fmt.Errorf("INSERT statement has no column list")
	}

	pkColumns := metaData.GetPrimaryKeyOnlyName()
	columns := make([]string, 0)

	for _, col := range stmt.Columns {
		columns = append(columns, strings.ToLower(string(col)))
	}

	pkIndexes := make(map[string]int)
	for i, col := range columns {
		for _, pk := range pkColumns {
			if strings.EqualFold(col, pk) {
				pkIndexes[pk] = i
			}
		}
	}

	if len(pkIndexes) == 0 {
		return result, fmt.Errorf("no primary key columns found in INSERT statement")
	}

	pkValueMap := make(map[string]driver.Value)
	for pk, colIndex := range pkIndexes {
		if colIndex < len(vals) {
			pkValueMap[pk] = vals[colIndex]
		}
	}

	if len(pkValueMap) > 0 {
		result = append(result, pkValueMap)
	}

	return result, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) getPkValues(execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]driver.Value, error) {
	pkValuesMap := make(map[string][]driver.Value)
	pkColumns := meta.GetPrimaryKeyOnlyName()

	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, 0)
		for _, param := range execCtx.NamedValues {
			vals = append(vals, param.Value)
		}
	}

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
					values = append(values, lastInsertId+i*int64(p.IncrementStep))
				}
				pkValuesMap[pkColumns[0]] = values
				return pkValuesMap, nil
			}
		}
	}

	pkValues, err := p.extractPrimaryKeyValues(execCtx, vals, &meta)
	if err != nil {
		return nil, err
	}

	if len(pkValues) == 0 {
		return nil, fmt.Errorf("no primary key values found for INSERT ON CONFLICT operation")
	}

	for _, pkValueMap := range pkValues {
		for pk, value := range pkValueMap {
			pkValuesMap[pk] = append(pkValuesMap[pk], value)
		}
	}

	return pkValuesMap, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) buildSelectSQLByPKValues(tableName string, pkNameList []string, pkValuesMap map[string][]driver.Value) (string, []driver.Value, error) {
	if len(pkValuesMap) == 0 {
		return "", nil, fmt.Errorf("no primary key values found")
	}

	var sql strings.Builder
	sql.WriteString("SELECT * FROM ")
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

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) buildPostgreSQLPlaceholders(count int, argIndex *int) string {
	var placeholders []string
	for i := 0; i < count; i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", *argIndex))
		(*argIndex)++
	}
	return strings.Join(placeholders, ", ")
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) buildImageParameters(execCtx *types.ExecContext, vals []driver.Value, metaData *types.TableMeta) (map[string][]driver.Value, error) {
	var parameterMap = make(map[string][]driver.Value)

	if execCtx.ParseContext.InsertStmt != nil {
		return p.buildImageParametersFromMySQL(execCtx.ParseContext.InsertStmt, vals, metaData)
	} else if execCtx.ParseContext.AuxtenInsertStmt != nil {
		return p.buildImageParametersFromPostgreSQL(execCtx.ParseContext.AuxtenInsertStmt, vals, metaData)
	}

	return parameterMap, fmt.Errorf("no valid insert statement found")
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) buildImageParametersFromMySQL(stmt *ast.InsertStmt, vals []driver.Value, metaData *types.TableMeta) (map[string][]driver.Value, error) {
	var parameterMap = make(map[string][]driver.Value)

	if len(stmt.Columns) == 0 || len(stmt.Lists) == 0 {
		return parameterMap, nil
	}

	var placeHolderIndex = 0
	for _, row := range stmt.Lists {
		for i, col := range stmt.Columns {
			columnName := strings.ToLower(col.Name.O)
			val := row[i]
			
			if _, ok := val.(ast.ParamMarkerExpr); ok {
				if placeHolderIndex < len(vals) {
					parameterMap[columnName] = append(parameterMap[columnName], vals[placeHolderIndex])
					placeHolderIndex++
				}
			} else {
				parameterMap[columnName] = append(parameterMap[columnName], val)
			}
		}
	}

	return parameterMap, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) buildImageParametersFromPostgreSQL(stmt *tree.Insert, vals []driver.Value, metaData *types.TableMeta) (map[string][]driver.Value, error) {
	var parameterMap = make(map[string][]driver.Value)

	if stmt.Columns == nil {
		return parameterMap, fmt.Errorf("INSERT statement has no column list")
	}

	columns := make([]string, 0)
	for _, col := range stmt.Columns {
		columns = append(columns, strings.ToLower(string(col)))
	}

	valueIndex := 0
	for i, col := range columns {
		if i < len(columns) && valueIndex < len(vals) {
			parameterMap[col] = append(parameterMap[col], vals[valueIndex])
			valueIndex++
		}
	}

	return parameterMap, nil
}

func (p *PostgreSQLInsertOnDuplicateUpdateUndoLogBuilder) isIndexValueNotNull(index types.IndexMeta, paramMap map[string][]driver.Value) bool {
	for _, colMeta := range index.Columns {
		columnName := strings.ToLower(colMeta.ColumnName)
		imageParameters := paramMap[columnName]
		if imageParameters == nil && colMeta.ColumnDef == nil {
			return false
		} else if imageParameters != nil && (len(imageParameters) == 0 || imageParameters[0] == nil) {
			return false
		}
	}
	return true
}
