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
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

// ExecutorType
//go:generate stringer -type=ExecutorType
type ExecutorType int32

const (
	_ ExecutorType = iota
	UnsupportExecutor
	InsertExecutor
	UpdateExecutor
	DeleteExecutor
	ReplaceIntoExecutor
	InsertOnDuplicateExecutor
)

type ParseContext struct {
	// SQLType
	SQLType types.SQLType
	// ExecutorType
	ExecutorType ExecutorType
	// InsertStmt
	InsertStmt *ast.InsertStmt
	// UpdateStmt
	UpdateStmt *ast.UpdateStmt
	// DeleteStmt
	DeleteStmt *ast.DeleteStmt
}

func (p *ParseContext) HasValidStmt() bool {
	return p.InsertStmt != nil || p.UpdateStmt != nil || p.DeleteStmt != nil
}

func DoParser(query string) (*ParseContext, error) {
	p := aparser.New()
	stmtNode, err := p.ParseOneStmt(query, "", "")
	if err != nil {
		return nil, err
	}

	parserCtx := new(ParseContext)

	switch stmt := stmtNode.(type) {
	case *ast.InsertStmt:
		parserCtx.SQLType = types.SQLTypeInsert
		parserCtx.InsertStmt = stmt
		parserCtx.ExecutorType = InsertExecutor

		if stmt.IsReplace {
			parserCtx.ExecutorType = ReplaceIntoExecutor
		}

		if len(stmt.OnDuplicate) != 0 {
			parserCtx.ExecutorType = InsertOnDuplicateExecutor
		}
	case *ast.UpdateStmt:
		parserCtx.SQLType = types.SQLTypeUpdate
		parserCtx.UpdateStmt = stmt
		parserCtx.ExecutorType = UpdateExecutor
	case *ast.DeleteStmt:
		parserCtx.SQLType = types.SQLTypeDelete
		parserCtx.DeleteStmt = stmt
		parserCtx.ExecutorType = DeleteExecutor
	}

	return parserCtx, nil
}
