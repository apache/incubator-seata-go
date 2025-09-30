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
	"github.com/arana-db/parser/format"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/bytes"
)

type PostgreSQLMultiUpdateUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetPostgreSQLMultiUpdateUndoLogBuilder() undo.UndoLogBuilder {
	return &PostgreSQLMultiUpdateUndoLogBuilder{}
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.UpdateExecutor
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, 0)
		for _, param := range execCtx.NamedValues {
			vals = append(vals, param.Value)
		}
	}

	updateStatements, err := p.splitStatementsAdvanced(execCtx.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to split statements: %v", err)
	}
	if len(updateStatements) == 0 {
		return []*types.RecordImage{}, nil
	}

	var allImages []*types.RecordImage
	argOffset := 0

	for i, updateSQL := range updateStatements {
		updateSQL = strings.TrimSpace(updateSQL)
		if updateSQL == "" {
			return nil, fmt.Errorf("update statement %d is empty", i)
		}

		stmtArgs, newOffset, err := p.extractArgsForStatementAdvanced(updateSQL, vals, argOffset)
		if err != nil {
			return nil, fmt.Errorf("failed to extract args for statement %d: %v", i, err)
		}
		argOffset = newOffset

		selectSQL, selectArgs, tableName, err := p.buildSingleBeforeImageSQL(updateSQL, stmtArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to build before image SQL for statement %d: %v", i, err)
		}

		if tableName == "" {
			return nil, fmt.Errorf("table name is empty for statement %d", i)
		}

		schemaName, tableNameOnly := p.parseTableName(tableName)
		
		var metaData *types.TableMeta
		if schemaName != "" {
			metaData, err = datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, schemaName, tableNameOnly)
		} else {
			metaData, err = datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableNameOnly)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to get table meta for %s: %v", tableName, err)
		}

		stmt, err := execCtx.Conn.Prepare(selectSQL)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare statement %d: %v", i, err)
		}
		defer stmt.Close()

		rows, err := stmt.Query(selectArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to query statement %d: %v", i, err)
		}
		defer rows.Close()

		image, err := p.buildRecordImages(rows, metaData)
		if err != nil {
			return nil, fmt.Errorf("failed to build record images for statement %d: %v", i, err)
		}

		lockKey := p.buildLockKey2(image, *metaData)
		execCtx.TxCtx.LockKeys[lockKey] = struct{}{}

		allImages = append(allImages, image)
	}

	return allImages, nil
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if len(beforeImages) == 0 {
		return []*types.RecordImage{}, nil
	}

	updateStatements := p.splitStatements(execCtx.Query)
	if len(updateStatements) == 0 {
		return []*types.RecordImage{}, nil
	}

	if len(beforeImages) != len(updateStatements) {
		return nil, fmt.Errorf("mismatch between before images count (%d) and update statements count (%d)", len(beforeImages), len(updateStatements))
	}

	var afterImages []*types.RecordImage

	for i, beforeImage := range beforeImages {
		if beforeImage == nil {
			return nil, fmt.Errorf("before image %d is nil", i)
		}

		updateSQL := strings.TrimSpace(updateStatements[i])
		if updateSQL == "" {
			return nil, fmt.Errorf("update statement %d is empty", i)
		}

		_, _, tableName, err := p.buildSingleBeforeImageSQL(updateSQL, []driver.Value{})
		if err != nil {
			return nil, fmt.Errorf("failed to parse table name from statement %d: %v", i, err)
		}

		schemaName, tableNameOnly := p.parseTableName(tableName)
		
		var metaData *types.TableMeta
		if schemaName != "" {
			metaData, err = datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, schemaName, tableNameOnly)
		} else {
			metaData, err = datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableNameOnly)
		}
		
		if err != nil {
			return nil, fmt.Errorf("failed to get table meta for %s: %v", tableName, err)
		}

		selectSQL, selectArgs := p.buildAfterImageSQL(beforeImage, *metaData)

		stmt, err := execCtx.Conn.Prepare(selectSQL)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare after image SQL: %v", err)
		}
		defer stmt.Close()

		rows, err := stmt.Query(selectArgs)
		if err != nil {
			return nil, fmt.Errorf("failed to query after image: %v", err)
		}
		defer rows.Close()

		image, err := p.buildRecordImages(rows, metaData)
		if err != nil {
			return nil, fmt.Errorf("failed to build record images: %v", err)
		}

		afterImages = append(afterImages, image)
	}

	return afterImages, nil
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) splitStatements(query string) []string {
	statements := strings.Split(query, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" && len(stmt) >= 6 && strings.ToUpper(stmt[:6]) == "UPDATE" {
			result = append(result, stmt)
		}
	}
	return result
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) splitStatementsAdvanced(query string) ([]string, error) {
	var statements []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	inComment := false
	i := 0

	for i < len(query) {
		ch := query[i]
		
		if !inComment && !inSingleQuote && !inDoubleQuote {
			if ch == '-' && i+1 < len(query) && query[i+1] == '-' {
				inComment = true
				current.WriteByte(ch)
				i++
				continue
			}
			if ch == '/' && i+1 < len(query) && query[i+1] == '*' {
				inComment = true
				current.WriteByte(ch)
				i++
				continue
			}
		}
		
		if inComment {
			current.WriteByte(ch)
			if ch == '\n' || (ch == '*' && i+1 < len(query) && query[i+1] == '/') {
				inComment = false
				if ch == '*' {
					i++
					current.WriteByte('/')
				}
			}
			i++
			continue
		}
		
		if ch == '\'' && !inDoubleQuote {
			if i+1 < len(query) && query[i+1] == '\'' {
				current.WriteByte(ch)
				i++
				current.WriteByte(ch)
			} else {
				inSingleQuote = !inSingleQuote
				current.WriteByte(ch)
			}
		} else if ch == '"' && !inSingleQuote {
			if i+1 < len(query) && query[i+1] == '"' {
				current.WriteByte(ch)
				i++
				current.WriteByte(ch)
			} else {
				inDoubleQuote = !inDoubleQuote
				current.WriteByte(ch)
			}
		} else if ch == ';' && !inSingleQuote && !inDoubleQuote {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" && len(stmt) >= 6 && strings.ToUpper(stmt[:6]) == "UPDATE" {
				statements = append(statements, stmt)
			}
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
		i++
	}
	
	finalStmt := strings.TrimSpace(current.String())
	if finalStmt != "" && len(finalStmt) >= 6 && strings.ToUpper(finalStmt[:6]) == "UPDATE" {
		statements = append(statements, finalStmt)
	}
	
	return statements, nil
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) extractArgsForStatement(stmt string, allArgs []driver.Value, offset int) ([]driver.Value, int) {
	paramCount := strings.Count(stmt, "?") + strings.Count(stmt, "$")
	if offset+paramCount > len(allArgs) {
		paramCount = len(allArgs) - offset
	}
	if paramCount <= 0 {
		return []driver.Value{}, offset
	}
	return allArgs[offset:offset+paramCount], offset + paramCount
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) extractArgsForStatementAdvanced(stmt string, allArgs []driver.Value, offset int) ([]driver.Value, int, error) {
	paramCount := 0
	inSingleQuote := false
	inDoubleQuote := false
	inComment := false
	i := 0
	
	for i < len(stmt) {
		ch := stmt[i]
		
		if !inComment && !inSingleQuote && !inDoubleQuote {
			if ch == '-' && i+1 < len(stmt) && stmt[i+1] == '-' {
				inComment = true
				i++
				continue
			}
			if ch == '/' && i+1 < len(stmt) && stmt[i+1] == '*' {
				inComment = true
				i++
				continue
			}
		}
		
		if inComment {
			if ch == '\n' || (ch == '*' && i+1 < len(stmt) && stmt[i+1] == '/') {
				inComment = false
				if ch == '*' {
					i++
				}
			}
			i++
			continue
		}
		
		if ch == '\'' && !inDoubleQuote {
			if i+1 < len(stmt) && stmt[i+1] == '\'' {
				i++
			} else {
				inSingleQuote = !inSingleQuote
			}
		} else if ch == '"' && !inSingleQuote {
			if i+1 < len(stmt) && stmt[i+1] == '"' {
				i++
			} else {
				inDoubleQuote = !inDoubleQuote
			}
		} else if !inSingleQuote && !inDoubleQuote {
			if ch == '?' {
				paramCount++
			} else if ch == '$' && i+1 < len(stmt) {
				j := i + 1
				for j < len(stmt) && stmt[j] >= '0' && stmt[j] <= '9' {
					j++
				}
				if j > i+1 {
					paramCount++
					i = j - 1
				}
			}
		}
		i++
	}
	
	if offset+paramCount > len(allArgs) {
		paramCount = len(allArgs) - offset
	}
	if paramCount <= 0 {
		return []driver.Value{}, offset, nil
	}
	return allArgs[offset:offset+paramCount], offset + paramCount, nil
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) parseTableName(tableName string) (string, string) {
	parts := strings.Split(tableName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", tableName
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) buildSingleBeforeImageSQL(query string, args []driver.Value) (string, []driver.Value, string, error) {
	parseCtx, err := parser.DoParser(query, types.DBTypePostgreSQL)
	if err != nil {
		return "", nil, "", err
	}

	if parseCtx.UpdateStmt != nil {
		return p.buildMySQLSingleUpdateBeforeImageSQL(parseCtx.UpdateStmt, args)
	} else if parseCtx.AuxtenUpdateStmt != nil {
		return p.buildPostgreSQLSingleUpdateBeforeImageSQL(parseCtx.AuxtenUpdateStmt, args)
	}

	return "", nil, "", fmt.Errorf("invalid update statement")
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) buildMySQLSingleUpdateBeforeImageSQL(stmt *ast.UpdateStmt, args []driver.Value) (string, []driver.Value, string, error) {
	selStmt := ast.SelectStmt{
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
	selStmt.Fields = &ast.FieldList{Fields: fields}

	b := bytes.NewByteBuffer([]byte{})
	restoreCtx := format.NewRestoreCtx(format.RestoreKeyWordUppercase, b)
	if err := selStmt.Restore(restoreCtx); err != nil {
		return "", nil, "", err
	}

	selectSQL := string(b.Bytes())
	selectSQL = p.convertToPostgreSQLSyntax(selectSQL)
	
	selectArgs := p.buildSelectArgs(&selStmt, args)
	
	tableName := p.extractSingleTableName(stmt.TableRefs.TableRefs)

	return selectSQL, selectArgs, tableName, nil
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) buildPostgreSQLSingleUpdateBeforeImageSQL(stmt *tree.Update, args []driver.Value) (string, []driver.Value, string, error) {
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

	if stmt.OrderBy != nil && len(stmt.OrderBy) > 0 {
		sql.WriteString(" ORDER BY ")
		var orderBys []string
		for _, order := range stmt.OrderBy {
			orderBys = append(orderBys, tree.AsString(order))
		}
		sql.WriteString(strings.Join(orderBys, ", "))
	}

	if stmt.Limit != nil {
		sql.WriteString(" LIMIT ")
		sql.WriteString(tree.AsString(stmt.Limit.Count))
		if stmt.Limit.Offset != nil {
			sql.WriteString(" OFFSET ")
			sql.WriteString(tree.AsString(stmt.Limit.Offset))
		}
	}

	return sql.String(), args, tableName, nil
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) convertToPostgreSQLSyntax(mysqlSQL string) string {
	result := mysqlSQL
	result = strings.ReplaceAll(result, "`", `"`)
	return result
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) convertToPostgreSQLParams(whereClause string) string {
	result := whereClause
	paramIndex := 1
	for strings.Contains(result, "?") {
		result = strings.Replace(result, "?", fmt.Sprintf("$%d", paramIndex), 1)
		paramIndex++
	}
	return result
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) extractSingleTableName(tableRefs *ast.Join) string {
	if tableRefs.Left != nil {
		if tableSource, ok := tableRefs.Left.(*ast.TableSource); ok {
			if tableName, ok := tableSource.Source.(*ast.TableName); ok {
				return tableName.Name.O
			}
		}
	}
	return ""
}

func (p *PostgreSQLMultiUpdateUndoLogBuilder) buildAfterImageSQL(beforeImage *types.RecordImage, meta types.TableMeta) (string, []driver.Value) {
	var sql strings.Builder
	sql.WriteString("SELECT * FROM ")
	sql.WriteString(fmt.Sprintf(`"%s"`, meta.TableName))
	sql.WriteString(" WHERE ")

	pkNames := meta.GetPrimaryKeyOnlyName()
	if len(pkNames) == 0 {
		return "", []driver.Value{}
	}

	var conditions []string
	var args []driver.Value
	argIndex := 1

	for rowIdx, row := range beforeImage.Rows {
		if rowIdx > 0 {
			conditions = append(conditions, " OR ")
		}
		
		var rowConditions []string
		for _, pkName := range pkNames {
			var pkValue driver.Value
			for _, col := range row.Columns {
				if col.ColumnName == pkName {
					pkValue = col.Value
					break
				}
			}
			if pkValue != nil {
				rowConditions = append(rowConditions, fmt.Sprintf(`"%s" = $%d`, pkName, argIndex))
				args = append(args, pkValue)
				argIndex++
			}
		}
		
		if len(rowConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(rowConditions, " AND ")+")")
		}
	}

	sql.WriteString(strings.Join(conditions, ""))
	return sql.String(), args
}