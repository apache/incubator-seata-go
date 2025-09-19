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
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"

	seatabytes "seata.apache.org/seata-go/pkg/util/bytes"
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
	SQLType      SQLType
	ExecutorType ExecutorType

	// AST nodes for MySQL statements (based on arana db/parser)
	InsertStmt *ast.InsertStmt
	UpdateStmt *ast.UpdateStmt
	SelectStmt *ast.SelectStmt
	DeleteStmt *ast.DeleteStmt

	// AST nodes for PostgreSQL statements (based on auxextent/postgresql parser)
	AuxtenInsertStmt *tree.Insert
	AuxtenUpdateStmt *tree.Update
	AuxtenSelectStmt *tree.Select
	AuxtenDeleteStmt *tree.Delete

	// Multi statement scenario
	MultiStmt []*ParseContext
}

// HasValidStmt
func (p *ParseContext) HasValidStmt() bool {
	return p.InsertStmt != nil || p.UpdateStmt != nil || p.DeleteStmt != nil ||
		p.AuxtenInsertStmt != nil || p.AuxtenUpdateStmt != nil || p.AuxtenSelectStmt != nil || p.AuxtenDeleteStmt != nil
}

func (p *ParseContext) GetTableName() (string, error) {
	switch {
	case p.InsertStmt != nil:
		b := seatabytes.NewByteBuffer([]byte{})
		p.InsertStmt.Table.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
		return string(b.Bytes()), nil
	case p.SelectStmt != nil && p.SelectStmt.From != nil:
		b := seatabytes.NewByteBuffer([]byte{})
		p.SelectStmt.From.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
		return string(b.Bytes()), nil
	case p.UpdateStmt != nil && p.UpdateStmt.TableRefs != nil:
		b := seatabytes.NewByteBuffer([]byte{})
		p.UpdateStmt.TableRefs.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
		return string(b.Bytes()), nil
	case p.DeleteStmt != nil && p.DeleteStmt.TableRefs != nil:
		b := seatabytes.NewByteBuffer([]byte{})
		p.DeleteStmt.TableRefs.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
		return string(b.Bytes()), nil
	}

	switch {
	case p.AuxtenInsertStmt != nil:
		return getAuxtenTableFromInsert(p.AuxtenInsertStmt)
	case p.AuxtenDeleteStmt != nil:
		return getAuxtenTableFromDelete(p.AuxtenDeleteStmt)
	case p.AuxtenUpdateStmt != nil:
		return getAuxtenTableFromUpdate(p.AuxtenUpdateStmt)
	case p.AuxtenSelectStmt != nil:
		return getAuxtenTableFromSelect(p.AuxtenSelectStmt)
	}

	if len(p.MultiStmt) > 0 {
		for _, stmt := range p.MultiStmt {
			tableName, err := stmt.GetTableName()
			if err == nil && tableName != "" {
				return tableName, nil
			}
		}
	}

	return "", fmt.Errorf("failed to get table name from stmt: %+v", p)
}

func getAuxtenTableFromInsert(stmt *tree.Insert) (string, error) {
	if stmt.Table == nil {
		return "", fmt.Errorf("insert statement has no table")
	}
	return tree.AsString(stmt.Table), nil
}

func getAuxtenTableFromDelete(stmt *tree.Delete) (string, error) {
	if stmt.Table == nil {
		return "", fmt.Errorf("delete statement has no table")
	}
	return tree.AsString(stmt.Table), nil
}

func getAuxtenTableFromUpdate(stmt *tree.Update) (string, error) {
	if stmt.Table == nil {
		return "", fmt.Errorf("update statement has no table")
	}
	return tree.AsString(stmt.Table), nil
}

func getAuxtenTableFromSelect(stmt *tree.Select) (string, error) {
	if stmt.Select == nil {
		return "", fmt.Errorf("select statement has no select clause")
	}

	switch selectStmt := stmt.Select.(type) {
	case *tree.SelectClause:
		if len(selectStmt.From.Tables) == 0 {
			return "", fmt.Errorf("select statement has no from clause")
		}
		firstTable := selectStmt.From.Tables[0]
		return tree.AsString(firstTable), nil
	default:
		return "", fmt.Errorf("unsupported select statement type")
	}
}
