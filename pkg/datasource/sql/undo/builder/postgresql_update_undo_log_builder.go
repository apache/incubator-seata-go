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
	"regexp"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

type PostgreSQLUpdateUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetPostgreSQLUpdateUndoLogBuilder() undo.UndoLogBuilder {
	return &PostgreSQLUpdateUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
	}
}

func (p *PostgreSQLUpdateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.UpdateExecutor
}

func (p *PostgreSQLUpdateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	if execCtx == nil || execCtx.ParseContext == nil {
		return nil, nil
	}

	if execCtx.ParseContext.UpdateStmt == nil && execCtx.ParseContext.AuxtenUpdateStmt == nil {
		return nil, nil
	}

	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, 0)
		for _, param := range execCtx.NamedValues {
			vals = append(vals, param.Value)
		}
	}

	selectSQL, selectArgs, err := p.buildBeforeImageSQL(ctx, execCtx, vals)
	if err != nil {
		return nil, err
	}

	tableName, err := execCtx.ParseContext.GetTableName()
	if err != nil {
		return nil, err
	}

	metaData, err := datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableName)
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

func (p *PostgreSQLUpdateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if execCtx == nil || execCtx.ParseContext == nil {
		return nil, nil
	}

	if execCtx.ParseContext.UpdateStmt == nil && execCtx.ParseContext.AuxtenUpdateStmt == nil {
		return nil, nil
	}

	if len(beforeImages) == 0 {
		return nil, nil
	}

	tableName, err := execCtx.ParseContext.GetTableName()
	if err != nil {
		return nil, err
	}

	metaData, err := datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableName)
	if err != nil {
		return nil, err
	}

	afterSelectSQL, afterSelectArgs, err := p.buildAfterImageSQL(beforeImages[0], metaData)
	if err != nil {
		return nil, err
	}

	stmt, err := execCtx.Conn.Prepare(afterSelectSQL)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(afterSelectArgs)
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

func (p *PostgreSQLUpdateUndoLogBuilder) buildBeforeImageSQL(ctx context.Context, execCtx *types.ExecContext, args []driver.Value) (string, []driver.Value, error) {
	var selectSQL string
	var selectArgs []driver.Value

	if execCtx.ParseContext.UpdateStmt != nil {
		sql, args, err := p.buildMySQLUpdateBeforeImageSQL(execCtx.ParseContext.UpdateStmt, args)
		if err != nil {
			return "", nil, err
		}
		selectSQL = p.convertToPostgreSQLSyntax(sql)
		selectArgs = args
	} else if execCtx.ParseContext.AuxtenUpdateStmt != nil {
		sql, args, err := p.buildPostgreSQLUpdateBeforeImageSQL(execCtx.ParseContext.AuxtenUpdateStmt, args)
		if err != nil {
			return "", nil, err
		}
		selectSQL = sql
		selectArgs = args
	}

	return selectSQL, selectArgs, nil
}

func (p *PostgreSQLUpdateUndoLogBuilder) buildMySQLUpdateBeforeImageSQL(stmt *ast.UpdateStmt, args []driver.Value) (string, []driver.Value, error) {
	b := bytes.NewByteBuffer([]byte{})
	selectStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From: &ast.TableRefsClause{
			TableRefs: stmt.TableRefs.TableRefs,
		},
		Where:   stmt.Where,
		OrderBy: stmt.Order,
		Limit:   stmt.Limit,
	}

	fields := []*ast.SelectField{{
		WildCard: &ast.WildCardField{},
	}}
	selectStmt.Fields = &ast.FieldList{Fields: fields}

	ctx := format.NewRestoreCtx(format.RestoreKeyWordUppercase, b)
	if err := selectStmt.Restore(ctx); err != nil {
		return "", nil, err
	}

	selectSQL := string(b.Bytes())
	selectArgs := p.buildSelectArgs(&selectStmt, args)

	return selectSQL, selectArgs, nil
}

func (p *PostgreSQLUpdateUndoLogBuilder) buildPostgreSQLUpdateBeforeImageSQL(stmt *tree.Update, args []driver.Value) (string, []driver.Value, error) {
	var sql strings.Builder
	sql.WriteString("SELECT * FROM ")

	tableName := tree.AsString(stmt.Table)
	sql.WriteString(fmt.Sprintf(`"%s"`, tableName))

	if stmt.Where != nil {
		sql.WriteString(" WHERE ")
		whereSQL := tree.AsString(stmt.Where.Expr)
		whereSQL = p.convertToPostgreSQLParams(whereSQL)
		sql.WriteString(whereSQL)
	}

	return sql.String(), args, nil
}

func (p *PostgreSQLUpdateUndoLogBuilder) buildAfterImageSQL(beforeImage *types.RecordImage, metaData *types.TableMeta) (string, []driver.Value, error) {
	if beforeImage == nil || len(beforeImage.Rows) == 0 {
		return "", nil, nil
	}

	tableName := beforeImage.TableName
	pkNameList := metaData.GetPrimaryKeyOnlyName()

	var sql strings.Builder
	sql.WriteString("SELECT * FROM ")
	sql.WriteString(fmt.Sprintf(`"%s"`, tableName))
	sql.WriteString(" WHERE ")

	var conditions []string
	var args []driver.Value
	argIndex := 1

	for _, row := range beforeImage.Rows {
		var rowConditions []string
		for _, pk := range pkNameList {
			for _, column := range row.Columns {
				if strings.EqualFold(column.ColumnName, pk) {
					rowConditions = append(rowConditions, fmt.Sprintf(`"%s" = $%d`, pk, argIndex))
					args = append(args, column.Value)
					argIndex++
					break
				}
			}
		}
		if len(rowConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(rowConditions, " AND ")+")")
		}
	}

	sql.WriteString(strings.Join(conditions, " OR "))
	return sql.String(), args, nil
}

func (p *PostgreSQLUpdateUndoLogBuilder) convertToPostgreSQLSyntax(mysqlSQL string) string {
	result := mysqlSQL
	
	result = strings.ReplaceAll(result, "`", `"`)
	
	result = p.convertMySQLFunctionsToPostgreSQL(result)
	
	result = p.convertMySQLDataTypesToPostgreSQL(result)
	
	result = p.convertMySQLLimitToPostgreSQL(result)
	
	return result
}

func (p *PostgreSQLUpdateUndoLogBuilder) convertMySQLFunctionsToPostgreSQL(sql string) string {
	functionMappings := map[string]string{
		"NOW()":           "CURRENT_TIMESTAMP",
		"CURDATE()":       "CURRENT_DATE",
		"CURTIME()":       "CURRENT_TIME",
		"UNIX_TIMESTAMP(": "EXTRACT(EPOCH FROM ",
		"FROM_UNIXTIME(":  "TO_TIMESTAMP(",
		"DATE_FORMAT(":    "TO_CHAR(",
		"STR_TO_DATE(":    "TO_DATE(",
		"IFNULL(":         "COALESCE(",
		"ISNULL(":         "COALESCE(",
		"CONCAT(":         "CONCAT(",
		"SUBSTRING(":      "SUBSTR(",
		"LENGTH(":         "LENGTH(",
		"LOCATE(":         "POSITION(",
		"REPLACE(":        "REPLACE(",
		"LOWER(":          "LOWER(",
		"UPPER(":          "UPPER(",
		"TRIM(":           "TRIM(",
		"LTRIM(":          "LTRIM(",
		"RTRIM(":          "RTRIM(",
		"ABS(":            "ABS(",
		"ROUND(":          "ROUND(",
		"FLOOR(":          "FLOOR(",
		"CEIL(":           "CEIL(",
		"MOD(":            "MOD(",
		"GREATEST(":       "GREATEST(",
		"LEAST(":          "LEAST(",
	}
	
	result := sql
	for mysqlFunc, pgFunc := range functionMappings {
		result = strings.ReplaceAll(result, mysqlFunc, pgFunc)
	}
	
	return result
}

func (p *PostgreSQLUpdateUndoLogBuilder) convertMySQLDataTypesToPostgreSQL(sql string) string {
	typeMapping := map[string]string{
		"TINYINT":   "SMALLINT",
		"MEDIUMINT": "INTEGER",
		"LONGTEXT":  "TEXT",
		"TINYTEXT":  "TEXT",
		"MEDIUMTEXT": "TEXT",
		"DATETIME":  "TIMESTAMP",
		"YEAR":      "INTEGER",
		"ENUM":      "VARCHAR",
		"SET":       "VARCHAR",
		"BIT":       "BOOLEAN",
	}
	
	result := sql
	for mysqlType, pgType := range typeMapping {
		result = strings.ReplaceAll(result, mysqlType, pgType)
	}
	
	return result
}

func (p *PostgreSQLUpdateUndoLogBuilder) convertMySQLLimitToPostgreSQL(sql string) string {
	limitRegex := `LIMIT\s+(\d+)\s*,\s*(\d+)`
	re := regexp.MustCompile(limitRegex)
	
	result := re.ReplaceAllStringFunc(sql, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) == 3 {
			offset := parts[1]
			limit := parts[2]
			return fmt.Sprintf("LIMIT %s OFFSET %s", limit, offset)
		}
		return match
	})
	
	return result
}

func (p *PostgreSQLUpdateUndoLogBuilder) convertToPostgreSQLParams(whereClause string) string {
	result := whereClause
	paramIndex := 1
	for strings.Contains(result, "?") {
		result = strings.Replace(result, "?", fmt.Sprintf("$%d", paramIndex), 1)
		paramIndex++
	}
	return result
}
