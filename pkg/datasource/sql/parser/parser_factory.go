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

package parser

import (
	"fmt"

	aparser "github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	pgparser "github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func DoParser(query string, dbType types.DBType) (*types.ParseContext, error) {
	switch dbType {
	case types.DBTypeMySQL:
		return parseMySQL(query)
	case types.DBTypePostgreSQL:
		return parsePostgreSQL(query)
	default:
		return nil, fmt.Errorf("unsupported db type: %v", dbType)
	}
}

func parseMySQL(query string) (*types.ParseContext, error) {
	p := aparser.New()
	stmtNodes, _, err := p.Parse(query, "", "")
	if err != nil {
		return nil, err
	}

	if len(stmtNodes) == 1 {
		return parseMySQLParseContext(stmtNodes[0]), nil
	}

	parserCtx := types.ParseContext{
		SQLType:      types.SQLTypeMulti,
		ExecutorType: types.MultiExecutor,
		MultiStmt:    make([]*types.ParseContext, 0, len(stmtNodes)),
	}

	for _, node := range stmtNodes {
		parserCtx.MultiStmt = append(parserCtx.MultiStmt, parseMySQLParseContext(node))
	}

	return &parserCtx, nil
}

// 使用auxten/postgresql-parser解析PostgreSQL
func parsePostgreSQL(query string) (*types.ParseContext, error) {
	stmts, err := pgparser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("auxten parse error: %w", err)
	}

	if len(stmts) == 0 {
		return &types.ParseContext{
			SQLType:      types.SQLTypeMulti,
			ExecutorType: types.MultiExecutor,
			MultiStmt:    make([]*types.ParseContext, 0),
		}, nil
	}

	if len(stmts) == 1 {
		return parseAuxtenParseContext(stmts[0].AST), nil
	}

	parserCtx := types.ParseContext{
		SQLType:      types.SQLTypeMulti,
		ExecutorType: types.MultiExecutor,
		MultiStmt:    make([]*types.ParseContext, 0, len(stmts)),
	}

	for _, stmt := range stmts {
		parserCtx.MultiStmt = append(parserCtx.MultiStmt, parseAuxtenParseContext(stmt.AST))
	}

	return &parserCtx, nil
}

func parseMySQLParseContext(stmtNode ast.StmtNode) *types.ParseContext {
	parserCtx := new(types.ParseContext)

	switch stmt := stmtNode.(type) {
	case *ast.InsertStmt:
		parserCtx.SQLType = types.SQLTypeInsert
		parserCtx.InsertStmt = stmt
		parserCtx.ExecutorType = types.InsertExecutor

		if stmt.IsReplace {
			parserCtx.ExecutorType = types.ReplaceIntoExecutor
		}
		if len(stmt.OnDuplicate) != 0 {
			parserCtx.SQLType = types.SQLTypeInsertOnDuplicateUpdate
			parserCtx.ExecutorType = types.InsertOnDuplicateExecutor
		}
	case *ast.UpdateStmt:
		parserCtx.SQLType = types.SQLTypeUpdate
		parserCtx.UpdateStmt = stmt
		parserCtx.ExecutorType = types.UpdateExecutor
	case *ast.SelectStmt:
		if stmt.LockInfo != nil && stmt.LockInfo.LockType == ast.SelectLockForUpdate {
			parserCtx.SQLType = types.SQLTypeSelectForUpdate
			parserCtx.SelectStmt = stmt
			parserCtx.ExecutorType = types.SelectForUpdateExecutor
		} else {
			parserCtx.SQLType = types.SQLTypeSelect
			parserCtx.SelectStmt = stmt
			parserCtx.ExecutorType = types.SelectExecutor
		}
	case *ast.DeleteStmt:
		parserCtx.SQLType = types.SQLTypeDelete
		parserCtx.DeleteStmt = stmt
		parserCtx.ExecutorType = types.DeleteExecutor
	}
	return parserCtx
}

// parseAuxtenParseContext 解析auxten/postgresql-parser返回的AST
func parseAuxtenParseContext(stmt tree.Statement) *types.ParseContext {
	parserCtx := new(types.ParseContext)

	switch s := stmt.(type) {
	case *tree.Insert:
		parserCtx.SQLType = types.SQLTypeInsert
		parserCtx.ExecutorType = types.InsertExecutor
		parserCtx.AuxtenInsertStmt = s

		// 检查是否有ON CONFLICT子句（PostgreSQL的upsert语法）
		if s.OnConflict != nil {
			parserCtx.SQLType = types.SQLTypeInsertOnDuplicateUpdate
			parserCtx.ExecutorType = types.InsertOnDuplicateExecutor
		}
	case *tree.Update:
		parserCtx.SQLType = types.SQLTypeUpdate
		parserCtx.ExecutorType = types.UpdateExecutor
		parserCtx.AuxtenUpdateStmt = s
	case *tree.Select:
		// 检查是否有FOR UPDATE锁定子句
		if hasForUpdate(s) {
			parserCtx.SQLType = types.SQLTypeSelectForUpdate
			parserCtx.ExecutorType = types.SelectForUpdateExecutor
		} else {
			parserCtx.SQLType = types.SQLTypeSelect
			parserCtx.ExecutorType = types.SelectExecutor
		}
		parserCtx.AuxtenSelectStmt = s
	case *tree.Delete:
		parserCtx.SQLType = types.SQLTypeDelete
		parserCtx.ExecutorType = types.DeleteExecutor
		parserCtx.AuxtenDeleteStmt = s
	}
	return parserCtx
}

// hasForUpdate 检查SELECT语句是否包含FOR UPDATE子句
func hasForUpdate(stmt *tree.Select) bool {
	if stmt.Select == nil {
		return false
	}

	// 检查顶层SELECT的Locking子句
	if stmt.Locking != nil && len(stmt.Locking) > 0 {
		return true
	}

	// 对于更复杂的嵌套查询，暂时返回false
	return false
}
