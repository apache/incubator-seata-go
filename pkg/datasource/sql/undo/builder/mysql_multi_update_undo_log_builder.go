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
	"github.com/pkg/errors"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/util/bytes"
	"github.com/seata/seata-go/pkg/util/log"
)

func init() {
	undo.RegistrUndoLogBuilder(types.UpdateExecutor, GetMySQLMultiUpdateUndoLogBuilder)
}

type MySQLMultiUpdateUndoLogBuilder struct {
	BasicUndoLogBuilder
}

func GetMySQLMultiUpdateUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLMultiUpdateUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
	}
}

func (u *MySQLMultiUpdateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	//if execCtx.ParseContext.UpdateStmt == nil {
	//	return nil, nil
	//}
	//
	//vals := execCtx.Values
	//if vals == nil {
	//	for n, param := range execCtx.NamedValues {
	//		vals[n] = param.Value
	//	}
	//}
	//// use
	//selectSQL, selectArgs, err := u.buildBeforeImageSQL(execCtx.ParseContext.UpdateStmt, vals)
	//if err != nil {
	//	return nil, err
	//}
	//
	//stmt, err := execCtx.Conn.Prepare(selectSQL)
	//if err != nil {
	//	log.Errorf("build prepare stmt: %+v", err)
	//	return nil, err
	//}
	//
	//rows, err := stmt.Query(selectArgs)
	//if err != nil {
	//	log.Errorf("stmt query: %+v", err)
	//	return nil, err
	//}
	//
	//tableName := execCtx.ParseContext.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	//metaData := execCtx.MetaDataMap[tableName]
	//
	//image, err := u.buildRecordImages(rows, metaData)
	//if err != nil {
	//	return nil, err
	//}
	//
	//return []*types.RecordImage{image}, nil
	return nil, nil
}

func (u *MySQLMultiUpdateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if execCtx.ParseContext.UpdateStmt == nil {
		return nil, nil
	}

	var beforeImage *types.RecordImage
	if len(beforeImages) > 0 {
		beforeImage = beforeImages[0]
	}

	tableName := execCtx.ParseContext.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
	selectSQL, selectArgs := u.buildAfterImageSQL(beforeImage, metaData)

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

	image, err := u.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}

	return []*types.RecordImage{image}, nil
}

func (u *MySQLMultiUpdateUndoLogBuilder) buildAfterImageSQL(beforeImage *types.RecordImage, meta types.TableMeta) (string, []driver.Value) {
	sb := strings.Builder{}
	// todo use ONLY_CARE_UPDATE_COLUMNS to judge select all columns or not
	sb.WriteString("SELECT * FROM " + meta.Name + " ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), "mysql", maxInSize)
	sb.WriteString(" " + whereSQL + " ")
	return sb.String(), u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName())
}

type Myvistor struct {
	stmt *ast.SelectStmt
}

func (m *Myvistor) Enter(n ast.Node) (node ast.Node, skipChildren bool) {
	if m.stmt != nil && m.stmt.Where != nil && m.stmt.Where == n {
		var younewNode ast.Node
		node = younewNode
		skipChildren = true
		return
	}
	switch val := n.(type) {
	case *ast.SelectStmt:
		m.stmt = val
	default:
		node = n
	}
	return
}

func (m *Myvistor) Leave(n ast.Node) (node ast.Node, ok bool) {
	return node, true
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *MySQLMultiUpdateUndoLogBuilder) buildBeforeImageSQL(updateStmts []*ast.UpdateStmt, args []driver.Value) (string, []driver.Value, error) {
	if len(updateStmts) == 0 {
		log.Errorf("invalid multi update stmt")
		return "", nil, fmt.Errorf("invalid muliti update stmt")
	}

	fields := []*ast.SelectField{}
	var whereCondition strings.Builder
	for _, updateStmt := range updateStmts {
		if updateStmt.Limit != nil {
			return "", nil, errors.New("Multi update SQL with limit condition is not support yet !")
		}
		if updateStmt.Order != nil {
			return "", nil, errors.New("Multi update SQL with orderBy condition is not support yet !")
		}

		// todo use ONLY_CARE_UPDATE_COLUMNS to judge select all columns or not
		for _, column := range updateStmt.List {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: column.Column,
				},
			})
		}

		in := bytes.NewByteBuffer([]byte{})
		updateStmt.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, in))
		whereConditionStr := string(in.Bytes())

		if whereCondition.Len() > 0 {
			whereCondition.Write([]byte(" OR "))
		}
		whereCondition.Write([]byte(whereConditionStr))
	}

	newNode, ok := updateStmts[0].Accept(&Myvistor{})
	if !ok {
		fmt.Println(newNode)
	}

	return "", nil, nil

	//return sql, u.buildSelectArgs(&selStmt, args), nil
}

func (u *MySQLMultiUpdateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.UpdateExecutor
}
