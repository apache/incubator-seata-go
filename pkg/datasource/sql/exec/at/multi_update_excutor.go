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

package at

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/arana-db/parser"
	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"
	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

// multiUpdateExecutor execute multiple update SQL
type multiUpdateExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

var rows driver.Rows
var comma = ","

// NewMultiUpdateExecutor get new multi update executor
func NewMultiUpdateExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook) *multiUpdateExecutor {
	return &multiUpdateExecutor{parserCtx: parserCtx, execContext: execContext, baseExecutor: baseExecutor{hooks: hooks}}
}

// ExecContext exec SQL, and generate before image and after image
func (u *multiUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	u.beforeHooks(ctx, u.execContext)
	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()

	//single update sql handler
	if len(u.parserCtx.MultiStmt) == 1 {
		u.parserCtx.UpdateStmt = u.parserCtx.MultiStmt[0].UpdateStmt
		return NewUpdateExecutor(u.parserCtx, u.execContext, u.hooks).ExecContext(ctx, f)
	}
	beforeImages, err := u.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, u.execContext.Query, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImages, err := u.afterImage(ctx, beforeImages)
	if err != nil {
		return nil, err
	}

	if len(afterImages) != len(beforeImages) {
		return nil, errors.New("Before image size is not equaled to after image size, probably because you updated the primary keys.")
	}

	for i, afterImage := range afterImages {
		beforeImage := afterImages[i]
		if len(beforeImage.Rows) != len(afterImage.Rows) {
			return nil, errors.New("Before image size is not equaled to after image size, probably because you updated the primary keys.")
		}

		u.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
		u.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	}

	return res, nil
}

func (u *multiUpdateExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	tableName := u.parserCtx.MultiStmt[0].UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}

	// use
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(u.execContext.NamedValues, metaData)
	if err != nil {
		return nil, err
	}

	rows, err := u.rowsPrepare(ctx, selectSQL, selectArgs)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("rows close fail, err:%v", err)
			return
		}
	}()
	if err != nil {
		return nil, err
	}

	image, err := u.buildRecordImages(rows, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey(image, *metaData)
	u.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	image.SQLType = u.parserCtx.SQLType

	return []*types.RecordImage{image}, nil
}

func (u *multiUpdateExecutor) afterImage(ctx context.Context, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	if len(beforeImages) == 0 {
		return nil, errors.New("empty beforeImages")
	}
	beforeImage := beforeImages[0]

	tableName := u.parserCtx.MultiStmt[0].UpdateStmt.TableRefs.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}

	// use
	selectSQL, selectArgs := u.buildAfterImageSQL(beforeImage, *metaData)

	rows, err = u.rowsPrepare(ctx, selectSQL, selectArgs)
	defer func() {
		if err := rows.Close(); err != nil {
			log.Errorf("rows close fail, err:%v", err)
			return
		}
	}()
	if err != nil {
		return nil, err
	}

	image, err := u.buildRecordImages(rows, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}

	image.SQLType = u.parserCtx.SQLType
	return []*types.RecordImage{image}, nil
}

func (u *multiUpdateExecutor) rowsPrepare(ctx context.Context, selectSQL string, selectArgs []driver.NamedValue) (driver.Rows, error) {
	var queryer driver.Queryer

	queryerContext, ok := u.execContext.Conn.(driver.QueryerContext)
	if !ok {
		queryer, ok = u.execContext.Conn.(driver.Queryer)
	}
	if ok {
		var err error
		rows, err = util.CtxDriverQuery(ctx, queryerContext, queryer, selectSQL, selectArgs)

		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, errors.New("invalid conn")
	}
	return rows, nil
}

// buildAfterImageSQL build the SQL to query after image data
func (u *multiUpdateExecutor) buildAfterImageSQL(beforeImage *types.RecordImage, meta types.TableMeta) (string, []driver.NamedValue) {
	if !u.isAstStmtValid() {
		return "", nil
	}

	selectSql := strings.Builder{}
	selectFields := make([]string, 0, len(meta.ColumnNames))
	var selectFieldsStr string
	var fieldsExits = make(map[string]struct{})
	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, row := range beforeImage.Rows {
			for _, column := range row.Columns {
				if _, exist := fieldsExits[column.ColumnName]; exist {
					continue
				}

				fieldsExits[column.ColumnName] = struct{}{}
				selectFields = append(selectFields, column.ColumnName)
			}
		}
		selectFieldsStr = strings.Join(selectFields, comma)
	} else {
		selectFieldsStr = strings.Join(meta.ColumnNames, comma)
	}
	selectSql.WriteString("SELECT " + selectFieldsStr + " FROM " + meta.TableName + " WHERE ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), "mysql", maxInSize)
	selectSql.WriteString(" " + whereSQL + " ")
	return selectSql.String(), u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName())
}

// buildSelectSQLByUpdate build select sql from update sql
func (u *multiUpdateExecutor) buildBeforeImageSQL(args []driver.NamedValue, meta *types.TableMeta) (string, []driver.NamedValue, error) {
	if !u.isAstStmtValid() {
		log.Errorf("invalid multi update stmt")
		return "", nil, errors.New("invalid muliti update stmt")
	}

	var (
		whereCondition strings.Builder
		multiStmts     = u.parserCtx.MultiStmt
		newArgs        = make([]driver.NamedValue, 0, len(u.parserCtx.MultiStmt))
		fields         = make([]*ast.SelectField, 0, len(meta.ColumnNames))
		fieldsExits    = make(map[string]struct{}, len(meta.ColumnNames))
	)

	for _, multiStmt := range u.parserCtx.MultiStmt {
		updateStmt := multiStmt.UpdateStmt
		if updateStmt.Limit != nil {
			return "", nil, fmt.Errorf("multi update SQL with limit condition is not support yet")
		}
		if updateStmt.Order != nil {
			return "", nil, fmt.Errorf("multi update SQL with orderBy condition is not support yet")
		}

		if undo.UndoConfig.OnlyCareUpdateColumns {
			//select update columns
			for _, column := range updateStmt.List {
				if _, exist := fieldsExits[column.Column.String()]; exist {
					continue
				}

				fieldsExits[column.Column.String()] = struct{}{}
				fields = append(fields, &ast.SelectField{Expr: &ast.ColumnNameExpr{Name: column.Column}})
			}

			for _, columnName := range meta.GetPrimaryKeyOnlyName() {
				if _, exist := fieldsExits[columnName]; exist {
					continue
				}

				//select index columns
				fieldsExits[columnName] = struct{}{}
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.CIStr{O: columnName, L: columnName}}},
				})
			}
		} else {
			fields = make([]*ast.SelectField, 0, len(meta.ColumnNames))
			for _, column := range meta.ColumnNames {
				fields = append(fields, &ast.SelectField{
					Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{Name: model.CIStr{O: column}}}})
			}
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
		_ = updateStmt.Where.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, in))

		if whereCondition.Len() > 0 {
			whereCondition.Write([]byte(" OR "))
		}
		whereCondition.Write(in.Bytes())
	}

	// only just get the where condition
	fakeSql := "select * from t where " + whereCondition.String()
	fakeStmt, err := parser.New().ParseOneStmt(fakeSql, "", "")
	if err != nil {
		log.Errorf("multi update parse fake sql error")
		return "", nil, err
	}
	fakeNode, ok := fakeStmt.Accept(&updateVisitor{})
	if !ok {
		log.Errorf("multi update accept update visitor error")
		return "", nil, err
	}
	fakeSelectStmt, ok := fakeNode.(*ast.SelectStmt)
	if !ok {
		log.Errorf("multi update fake node is not select stmt")
		return "", nil, err
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           multiStmts[0].UpdateStmt.TableRefs,
		Where:          fakeSelectStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		TableHints:     multiStmts[0].UpdateStmt.TableHints,
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	b := bytes.NewByteBuffer([]byte{})
	selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	log.Infof("build select sql by update sourceQuery, sql {}", string(b.Bytes()))

	return string(b.Bytes()), newArgs, nil
}

func (u *multiUpdateExecutor) isAstStmtValid() bool {
	return u.parserCtx != nil && u.parserCtx.MultiStmt != nil && len(u.parserCtx.MultiStmt) > 0
}

type updateVisitor struct {
	stmt *ast.UpdateStmt
}

func (m *updateVisitor) Enter(n ast.Node) (ast.Node, bool) {
	return n, true
}

func (m *updateVisitor) Leave(n ast.Node) (ast.Node, bool) {
	node := n
	return node, true
}
