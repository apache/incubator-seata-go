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
	aparser "github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

func DoParser(query string) (*types.ParseContext, error) {
	p := aparser.New()
	stmtNodes, _, err := p.Parse(query, "", "")
	if err != nil {
		return nil, err
	}

	if len(stmtNodes) == 1 {
		return parseParseContext(stmtNodes[0]), err
	}

	parserCtx := types.ParseContext{
		SQLType:      types.SQLTypeMulti,
		ExecutorType: types.MultiExecutor,
		MultiStmt:    make([]*types.ParseContext, 0, len(stmtNodes)),
	}

	for _, node := range stmtNodes {
		parserCtx.MultiStmt = append(parserCtx.MultiStmt, parseParseContext(node))
	}

	return &parserCtx, nil
}

func parseParseContext(stmtNode ast.StmtNode) *types.ParseContext {
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
