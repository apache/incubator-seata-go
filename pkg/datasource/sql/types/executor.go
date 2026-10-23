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

package types

import (
	"fmt"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	seatabytes "seata.apache.org/seata-go/v2/pkg/util/bytes"
)

type ExecutorType int32

const (
	_ ExecutorType = iota
	UnSupportExecutor
	InsertExecutor
	UpdateExecutor
	SelectForUpdateExecutor
	SelectExecutor
	DeleteExecutor
	ReplaceIntoExecutor
	MultiExecutor
	MultiDeleteExecutor
	InsertOnDuplicateExecutor
)

type ParseContext struct {
	DBType       DBType
	SQLType      SQLType
	ExecutorType ExecutorType
	InsertStmt   *ast.InsertStmt
	UpdateStmt   *ast.UpdateStmt
	SelectStmt   *ast.SelectStmt
	DeleteStmt   *ast.DeleteStmt
	MultiStmt    []*ParseContext
	TableRefs    map[*ast.TableName]TableRef
}

// GetTableRef returns the identity of the first table in a single statement.
// Multi-statement execution must use the corresponding child ParseContext.
func (p *ParseContext) GetTableRef() (TableRef, error) {
	if p == nil {
		return TableRef{}, fmt.Errorf("nil parse context")
	}
	var refs *ast.TableRefsClause
	switch {
	case p.InsertStmt != nil:
		refs = p.InsertStmt.Table
	case p.UpdateStmt != nil:
		refs = p.UpdateStmt.TableRefs
	case p.DeleteStmt != nil:
		refs = p.DeleteStmt.TableRefs
	case p.SelectStmt != nil:
		refs = p.SelectStmt.From
	default:
		return TableRef{}, fmt.Errorf("statement has no table reference")
	}
	if refs == nil || refs.TableRefs == nil {
		return TableRef{}, fmt.Errorf("statement has no table reference")
	}
	var source ast.ResultSetNode = refs.TableRefs
	for {
		switch node := source.(type) {
		case *ast.Join:
			source = node.Left
		case *ast.TableSource:
			source = node.Source
		case *ast.TableName:
			if ref, ok := p.TableRefs[node]; ok {
				return ref, nil
			}
			return TableRef{Qualifier: node.Schema.O, TableName: node.Name.O}, nil
		default:
			return TableRef{}, fmt.Errorf("table reference is not a table name")
		}
	}
}

func (p *ParseContext) HasValidStmt() bool {
	return p.InsertStmt != nil || p.UpdateStmt != nil || p.DeleteStmt != nil
}

func (p *ParseContext) GetTableName() (string, error) {
	var table *ast.TableRefsClause

	if p.InsertStmt != nil {
		table = p.InsertStmt.Table
	} else if p.SelectStmt != nil {
		table = p.SelectStmt.From
	} else if p.UpdateStmt != nil {
		table = p.UpdateStmt.TableRefs
	} else if p.DeleteStmt != nil {
		table = p.DeleteStmt.TableRefs
	} else if len(p.MultiStmt) > 0 {
		for _, parser := range p.MultiStmt {
			tableName, err := parser.GetTableName()
			if err != nil {
				return "", err
			}
			if tableName != "" {
				return tableName, nil
			}
		}
	} else {
		return "", fmt.Errorf("invalid stmt %v", p)
	}

	b := seatabytes.NewByteBuffer([]byte{})
	table.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))

	return string(b.Bytes()), nil
}
