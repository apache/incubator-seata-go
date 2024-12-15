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
	"strings"

	"github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

//func init() {
//	undo.RegisterUndoLogBuilder(types.MultiExecutor, GetMySQLMultiUpdateUndoLogBuilder)
//}

type updateVisitor struct {
	stmt *ast.UpdateStmt
}

func (m *updateVisitor) Enter(n ast.Node) (node ast.Node, skipChildren bool) {
	return n, true
}

func (m *updateVisitor) Leave(n ast.Node) (node ast.Node, ok bool) {
	node = n
	return node, true
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
	vals := execCtx.Values
	if vals == nil {
		for n, param := range execCtx.NamedValues {
			vals[n] = param.Value
		}
	}

	var updateStmts []*ast.UpdateStmt
	for _, v := range execCtx.ParseContext.MultiStmt {
		updateStmts = append(updateStmts, v.UpdateStmt)
	}

	// use
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(updateStmts, vals)
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

	tableName := execCtx.ParseContext.UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]

	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}

	image.SQLType = execCtx.ParseContext.SQLType

	return []*types.RecordImage{image}, nil
}

func (u *MySQLMultiUpdateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
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

	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}

	image.SQLType = execCtx.ParseContext.SQLType
	return []*types.RecordImage{image}, nil
}

func (u *MySQLMultiUpdateUndoLogBuilder) buildAfterImageSQL(beforeImage *types.RecordImage, meta types.TableMeta) (string, []driver.Value) {
	sb := strings.Builder{}
	// todo use ONLY_CARE_UPDATE_COLUMNS to judge select all columns or not
	sb.WriteString("SELECT * FROM " + meta.TableName + " ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), "mysql", maxInSize)
	sb.WriteString(" " + whereSQL + " ")
	return sb.String(), u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName())
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *MySQLMultiUpdateUndoLogBuilder) buildBeforeImageSQL(updateStmts []*ast.UpdateStmt, args []driver.Value) (string, []driver.Value, error) {
	if len(updateStmts) == 0 {
		log.Errorf("invalid multi update stmt")
		return "", nil, fmt.Errorf("invalid muliti update stmt")
	}

	var newArgs []driver.Value
	var fields []*ast.SelectField
	fieldsExits := make(map[string]struct{})
	var whereCondition strings.Builder
	for _, updateStmt := range updateStmts {
		if updateStmt.Limit != nil {
			return "", nil, fmt.Errorf("multi update SQL with limit condition is not support yet")
		}
		if updateStmt.Order != nil {
			return "", nil, fmt.Errorf("multi update SQL with orderBy condition is not support yet")
		}

		// todo use ONLY_CARE_UPDATE_COLUMNS to judge select all columns or not
		for _, column := range updateStmt.List {
			if _, exist := fieldsExits[column.Column.String()]; exist {
				continue
			}
			fieldsExits[column.Column.String()] = struct{}{}
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: column.Column,
				},
			})
		}

		tmpSelectStmt := ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{},
			From:           updateStmt.TableRefs,
			Where:          updateStmt.Where,
			Fields:         &ast.FieldList{Fields: fields},
			OrderBy:        updateStmt.Order,
			Limit:          updateStmt.Limit,
			TableHints:     updateStmt.TableHints,
			LockInfo: &ast.SelectLockInfo{
				LockType: ast.SelectLockForUpdate,
			},
		}
		newArgs = append(newArgs, u.buildSelectArgs(&tmpSelectStmt, args)...)

		in := bytes.NewByteBuffer([]byte{})
		updateStmt.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, in))
		whereConditionStr := string(in.Bytes())

		if whereCondition.Len() > 0 {
			whereCondition.Write([]byte(" OR "))
		}
		whereCondition.Write([]byte(whereConditionStr))
	}

	// only just get the where condition
	fakeSql := "select * from t where " + whereCondition.String()
	fakeStmt, err := parser.New().ParseOneStmt(fakeSql, "", "")
	if err != nil {
		return "", nil, errors.Wrap(err, "multi update parse fake sql error")
	}
	fakeNode, ok := fakeStmt.Accept(&updateVisitor{})
	if !ok {
		return "", nil, errors.Wrap(err, "multi update accept update visitor error")
	}
	fakeSelectStmt, ok := fakeNode.(*ast.SelectStmt)
	if !ok {
		return "", nil, fmt.Errorf("multi update fake node is not select stmt")
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           updateStmts[0].TableRefs,
		Where:          fakeSelectStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		TableHints:     updateStmts[0].TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := bytes.NewByteBuffer([]byte{})
	selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by update sourceQuery, sql {}", sql)

	return sql, newArgs, nil
}

func (u *MySQLMultiUpdateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.UpdateExecutor
}
