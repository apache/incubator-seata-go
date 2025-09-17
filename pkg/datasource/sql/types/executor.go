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

// ParseContext 解析上下文，存储SQL解析后的关键信息
// 区分MySQL（arana-db/parser）和PostgreSQL（auxten/postgresql-parser）的AST结构
type ParseContext struct {
	SQLType      SQLType
	ExecutorType ExecutorType

	// MySQL语句的AST节点（基于arana-db/parser）
	InsertStmt *ast.InsertStmt
	UpdateStmt *ast.UpdateStmt
	SelectStmt *ast.SelectStmt
	DeleteStmt *ast.DeleteStmt

	// PostgreSQL语句的AST节点（基于auxten/postgresql-parser）
	AuxtenInsertStmt *tree.Insert
	AuxtenUpdateStmt *tree.Update
	AuxtenSelectStmt *tree.Select
	AuxtenDeleteStmt *tree.Delete

	// 多语句场景
	MultiStmt []*ParseContext
}

// HasValidStmt 检查是否包含有效的SQL语句节点
func (p *ParseContext) HasValidStmt() bool {
	return p.InsertStmt != nil || p.UpdateStmt != nil || p.DeleteStmt != nil ||
		p.AuxtenInsertStmt != nil || p.AuxtenUpdateStmt != nil || p.AuxtenSelectStmt != nil || p.AuxtenDeleteStmt != nil
}

func (p *ParseContext) GetTableName() (string, error) {
	// 处理MySQL语句（保持不变）
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

	// 处理PostgreSQL语句（使用auxten/postgresql-parser）
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

	// 处理多语句场景
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

// 辅助函数：从auxten/postgresql-parser的INSERT语句中提取表名
func getAuxtenTableFromInsert(stmt *tree.Insert) (string, error) {
	if stmt.Table == nil {
		return "", fmt.Errorf("insert statement has no table")
	}
	return tree.AsString(stmt.Table), nil
}

// 辅助函数：从auxten/postgresql-parser的DELETE语句中提取表名
func getAuxtenTableFromDelete(stmt *tree.Delete) (string, error) {
	if stmt.Table == nil {
		return "", fmt.Errorf("delete statement has no table")
	}
	return tree.AsString(stmt.Table), nil
}

// 辅助函数：从auxten/postgresql-parser的UPDATE语句中提取表名
func getAuxtenTableFromUpdate(stmt *tree.Update) (string, error) {
	if stmt.Table == nil {
		return "", fmt.Errorf("update statement has no table")
	}
	return tree.AsString(stmt.Table), nil
}

// 辅助函数：从auxten/postgresql-parser的SELECT语句中提取表名
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
