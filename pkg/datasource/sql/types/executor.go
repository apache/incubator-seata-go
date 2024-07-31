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
	InsertStmt   *ast.InsertStmt
	UpdateStmt   *ast.UpdateStmt
	SelectStmt   *ast.SelectStmt
	DeleteStmt   *ast.DeleteStmt
	MultiStmt    []*ParseContext
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
