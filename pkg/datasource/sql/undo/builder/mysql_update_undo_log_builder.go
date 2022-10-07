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

	"github.com/seata/seata-go/pkg/datasource/sql/exec"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/seata/seata-go/pkg/datasource/sql/parser"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/util/bytes"
	"github.com/seata/seata-go/pkg/util/log"
)

type MySQLUpdateUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func (u *MySQLUpdateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *exec.ExecContext) (*types.RecordImage, error) {
	vals := execCtx.Values
	if vals == nil {
		for n, param := range execCtx.NamedValues {
			vals[n] = param.Value
		}
	}
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(execCtx.Query, vals)
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

	return u.buildRecordImages(rows, execCtx.MetaData)
}

func (u *MySQLUpdateUndoLogBuilder) AfterImage(types.RecordImages) (*types.RecordImages, error) {
	return nil, nil
}

// buildBeforeImageSQL build select sql from update sql
func (u *MySQLUpdateUndoLogBuilder) buildBeforeImageSQL(query string, args []driver.Value) (string, []driver.Value, error) {
	p, err := parser.DoParser(query)
	if err != nil {
		return "", nil, err
	}

	if p.UpdateStmt == nil {
		log.Errorf("invalid update stmt")
		return "", nil, fmt.Errorf("invalid update stmt")
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
	log.Infof("build select sql by update sourceQuery, sql {}", sql)

	return sql, u.buildSelectArgs(&selStmt, args), nil
}

func (u *MySQLUpdateUndoLogBuilder) GetSQLType() types.SQLType {
	return types.SQLTypeUpdate
}
