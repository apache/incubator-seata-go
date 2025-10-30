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

type PostgreSQLDeleteUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetPostgreSQLDeleteUndoLogBuilder() undo.UndoLogBuilder {
	return &PostgreSQLDeleteUndoLogBuilder{}
}

func (p *PostgreSQLDeleteUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.DeleteExecutor
}

func (p *PostgreSQLDeleteUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, 0)
		for _, param := range execCtx.NamedValues {
			vals = append(vals, param.Value)
		}
	}

	selectSQL, selectArgs, err := p.buildBeforeImageSQL(execCtx.Query, vals)
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

	tableName, err := execCtx.ParseContext.GetTableName()
	if err != nil {
		return nil, err
	}

	metaData, err := datasource.GetTableCache(types.DBTypePostgreSQL).GetTableMeta(ctx, execCtx.DBName, tableName)
	if err != nil {
		return nil, err
	}

	image, err := p.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}

	lockKey := p.buildLockKey2(image, *metaData)
	execCtx.TxCtx.LockKeys[lockKey] = struct{}{}

	return []*types.RecordImage{image}, nil
}

func (p *PostgreSQLDeleteUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	return []*types.RecordImage{}, nil
}

func (p *PostgreSQLDeleteUndoLogBuilder) buildBeforeImageSQL(query string, args []driver.Value) (string, []driver.Value, error) {
	parseCtx, err := parser.DoParser(query, types.DBTypePostgreSQL)
	if err != nil {
		return "", nil, err
	}

	if parseCtx.DeleteStmt != nil {
		return p.buildMySQLDeleteBeforeImageSQL(parseCtx.DeleteStmt, args)
	} else if parseCtx.AuxtenDeleteStmt != nil {
		return p.buildPostgreSQLDeleteBeforeImageSQL(parseCtx.AuxtenDeleteStmt, args)
	}

	return "", nil, fmt.Errorf("invalid delete statement")
}

func (p *PostgreSQLDeleteUndoLogBuilder) buildMySQLDeleteBeforeImageSQL(stmt *ast.DeleteStmt, args []driver.Value) (string, []driver.Value, error) {
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
		return "", nil, err
	}

	selectSQL := string(b.Bytes())
	selectSQL = p.convertToPostgreSQLSyntax(selectSQL)
	
	selectArgs := p.buildSelectArgs(&selStmt, args)

	return selectSQL, selectArgs, nil
}

func (p *PostgreSQLDeleteUndoLogBuilder) buildPostgreSQLDeleteBeforeImageSQL(stmt *tree.Delete, args []driver.Value) (string, []driver.Value, error) {
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

	return sql.String(), args, nil
}

func (p *PostgreSQLDeleteUndoLogBuilder) convertToPostgreSQLSyntax(mysqlSQL string) string {
	result := mysqlSQL
	result = strings.ReplaceAll(result, "`", `"`)
	return result
}

func (p *PostgreSQLDeleteUndoLogBuilder) convertToPostgreSQLParams(whereClause string) string {
	result := whereClause
	paramIndex := 1
	for strings.Contains(result, "?") {
		result = strings.Replace(result, "?", fmt.Sprintf("$%d", paramIndex), 1)
		paramIndex++
	}
	return result
}