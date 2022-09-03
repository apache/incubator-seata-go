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

package exec

import (
	"context"
	"fmt"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	"github.com/seata/seata-go/pkg/common/bytes"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

type BasicUndoBuilder struct {
}

// buildRowImages build row iamge by exec condition
func (u *BasicUndoBuilder) buildRowImages(ctx context.Context, execCtx *exec.ExecContext) ([]*types.RowImage, error) {
	panic("implement me")
}

// buildRowImages query db table to find data image
func (u *BasicUndoBuilder) buildRecordImage(ctx context.Context, execCtx *exec.ExecContext) ([]*types.RecordImage, error) {
	panic("implement me")
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *BasicUndoBuilder) buildSelectSQLByUpdate(query string) (string, error) {
	p, err := parser.DoParser(query)
	if err != nil {
		return "", err
	}

	if p.UpdateStmt == nil {
		return "", fmt.Errorf("invalid update stmt")
	}

	fields := []*ast.SelectField{}

	for _, column := range p.UpdateStmt.List {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: column.Column,
			},
		})
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           p.UpdateStmt.TableRefs,
		Where:          p.UpdateStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		OrderBy:        p.UpdateStmt.Order,
		Limit:          p.UpdateStmt.Limit,
		TableHints:     p.UpdateStmt.TableHints,
	}

	b := bytes.NewByteBuffer([]byte{})
	selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by update query, sql {}", sql)

	return sql, nil
}
