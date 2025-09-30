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
	"seata.apache.org/seata-go/pkg/util/log"
)

type PostgreSQLMultiDeleteUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetPostgreSQLMultiDeleteUndoLogBuilder() undo.UndoLogBuilder {
	return &PostgreSQLMultiDeleteUndoLogBuilder{}
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.MultiDeleteExecutor
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, 0)
		for _, param := range execCtx.NamedValues {
			vals = append(vals, param.Value)
		}
	}

	deleteStatements := p.splitStatements(execCtx.Query)
	if len(deleteStatements) == 0 {
		return []*types.RecordImage{}, nil
	}

	var allImages []*types.RecordImage
	argOffset := 0

	for _, deleteSQL := range deleteStatements {
		deleteSQL = strings.TrimSpace(deleteSQL)
		if deleteSQL == "" {
			continue
		}

		stmtArgs, newOffset := p.extractArgsForStatement(deleteSQL, vals, argOffset)
		argOffset = newOffset

		selectSQL, selectArgs, tableName, err := p.buildSingleBeforeImageSQL(deleteSQL, stmtArgs)
		if err != nil {
			log.Errorf("failed to build before image SQL for statement %s: %v", deleteSQL, err)
			continue
		}

		if tableName == "" {
			continue
		}

		schemaName, tableNameOnly := p.parseTableName(tableName)
		
		var metaData *types.TableMeta
		if schemaName != "" {
			metaData, err = datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, schemaName, tableNameOnly)
		} else {
			metaData, err = datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableNameOnly)
		}
		
		if err != nil {
			log.Errorf("failed to get table meta for %s: %v", tableName, err)
			continue
		}

		stmt, err := execCtx.Conn.Prepare(selectSQL)
		if err != nil {
			log.Errorf("build prepare stmt: %+v", err)
			continue
		}

		rows, err := stmt.Query(selectArgs)
		if err != nil {
			log.Errorf("stmt query: %+v", err)
			stmt.Close()
			continue
		}

		image, err := p.buildRecordImages(rows, metaData)
		if err != nil {
			log.Errorf("failed to build record images: %v", err)
			rows.Close()
			stmt.Close()
			continue
		}

		lockKey := p.buildLockKey2(image, *metaData)
		execCtx.TxCtx.LockKeys[lockKey] = struct{}{}

		allImages = append(allImages, image)
		
		rows.Close()
		stmt.Close()
	}

	return allImages, nil
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	return []*types.RecordImage{}, nil
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) splitStatements(query string) []string {
	statements := strings.Split(query, ";")
	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" && len(stmt) >= 6 && strings.ToUpper(stmt[:6]) == "DELETE" {
			result = append(result, stmt)
		}
	}
	return result
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) extractArgsForStatement(stmt string, allArgs []driver.Value, offset int) ([]driver.Value, int) {
	paramCount := strings.Count(stmt, "?") + strings.Count(stmt, "$")
	if offset+paramCount > len(allArgs) {
		paramCount = len(allArgs) - offset
	}
	if paramCount <= 0 {
		return []driver.Value{}, offset
	}
	return allArgs[offset:offset+paramCount], offset + paramCount
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) parseTableName(tableName string) (string, string) {
	parts := strings.Split(tableName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", tableName
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) buildSingleBeforeImageSQL(query string, args []driver.Value) (string, []driver.Value, string, error) {
	parseCtx, err := parser.DoParser(query, types.DBTypePostgreSQL)
	if err != nil {
		return "", nil, "", err
	}

	if parseCtx.DeleteStmt != nil {
		return p.buildMySQLSingleDeleteBeforeImageSQL(parseCtx.DeleteStmt, args)
	} else if parseCtx.AuxtenDeleteStmt != nil {
		return p.buildPostgreSQLSingleDeleteBeforeImageSQL(parseCtx.AuxtenDeleteStmt, args)
	}

	return "", nil, "", fmt.Errorf("invalid delete statement")
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) buildMySQLSingleDeleteBeforeImageSQL(stmt *ast.DeleteStmt, args []driver.Value) (string, []driver.Value, string, error) {
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

func (p *PostgreSQLMultiDeleteUndoLogBuilder) buildPostgreSQLSingleDeleteBeforeImageSQL(stmt *tree.Delete, args []driver.Value) (string, []driver.Value, string, error) {
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

func (p *PostgreSQLMultiDeleteUndoLogBuilder) convertToPostgreSQLSyntax(mysqlSQL string) string {
	result := mysqlSQL
	result = strings.ReplaceAll(result, "`", `"`)
	return result
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) convertToPostgreSQLParams(whereClause string) string {
	result := whereClause
	paramIndex := 1
	for strings.Contains(result, "?") {
		result = strings.Replace(result, "?", fmt.Sprintf("$%d", paramIndex), 1)
		paramIndex++
	}
	return result
}

func (p *PostgreSQLMultiDeleteUndoLogBuilder) extractSingleTableName(tableRefs *ast.Join) string {
	if tableRefs.Left != nil {
		if tableSource, ok := tableRefs.Left.(*ast.TableSource); ok {
			if tableName, ok := tableSource.Source.(*ast.TableName); ok {
				return tableName.Name.O
			}
		}
	}
	return ""
}