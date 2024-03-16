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

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	maxInSize = 1000
)

// updateExecutor execute update SQL
type updateExecutor struct {
	baseExecutor
	parserCtx   *types.ParseContext
	execContext *types.ExecContext
}

// NewUpdateExecutor get update executor
func NewUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &updateExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

// ExecContext exec SQL, and generate before image and after image
func (u *updateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	u.beforeHooks(ctx, u.execContext)
	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()

	beforeImage, err := u.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, u.execContext.Query, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	afterImage, err := u.afterImage(ctx, *beforeImage)
	if err != nil {
		return nil, err
	}

	if len(beforeImage.Rows) != len(afterImage.Rows) {
		return nil, fmt.Errorf("Before image size is not equaled to after image size, probably because you updated the primary keys.")
	}

	u.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	u.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)

	return res, nil
}

// beforeImage build before image
func (u *updateExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	selectSQL, selectArgs, err := u.buildBeforeImageSQL(ctx, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	tableName, _ := u.parserCtx.GetTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}

	var rowsi driver.Rows
	queryerCtx, ok := u.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = u.execContext.Conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	image, err := u.buildRecordImages(rowsi, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}

	lockKey := u.buildLockKey(image, *metaData)
	u.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	image.SQLType = u.parserCtx.SQLType

	return image, nil
}

// afterImage build after image
func (u *updateExecutor) afterImage(ctx context.Context, beforeImage types.RecordImage) (*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}
	if len(beforeImage.Rows) == 0 {
		return &types.RecordImage{}, nil
	}

	tableName, _ := u.parserCtx.GetTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, u.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs := u.buildAfterImageSQL(beforeImage, metaData)

	var rowsi driver.Rows
	queryerCtx, ok := u.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = u.execContext.Conn.(driver.Queryer)
	}
	if ok {
		rowsi, err = util.CtxDriverQuery(ctx, queryerCtx, queryer, selectSQL, selectArgs)
		defer func() {
			if rowsi != nil {
				rowsi.Close()
			}
		}()
		if err != nil {
			log.Errorf("ctx driver query: %+v", err)
			return nil, err
		}
	} else {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}

	afterImage, err := u.buildRecordImages(rowsi, metaData, types.SQLTypeUpdate)
	if err != nil {
		return nil, err
	}
	afterImage.SQLType = u.parserCtx.SQLType

	return afterImage, nil
}

func (u *updateExecutor) isAstStmtValid() bool {
	return u.parserCtx != nil && u.parserCtx.UpdateStmt != nil
}

// buildAfterImageSQL build the SQL to query after image data
func (u *updateExecutor) buildAfterImageSQL(beforeImage types.RecordImage, meta *types.TableMeta) (string, []driver.NamedValue) {
	if len(beforeImage.Rows) == 0 {
		return "", nil
	}
	sb := strings.Builder{}
	// todo: OnlyCareUpdateColumns should load from config first
	var selectFields string
	var separator = ","
	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, row := range beforeImage.Rows {
			for _, column := range row.Columns {
				selectFields += column.ColumnName + separator
			}
		}
		selectFields = strings.TrimSuffix(selectFields, separator)
	} else {
		selectFields = "*"
	}
	sb.WriteString("SELECT " + selectFields + " FROM " + meta.TableName + " WHERE ")
	whereSQL := u.buildWhereConditionByPKs(meta.GetPrimaryKeyOnlyName(), len(beforeImage.Rows), "mysql", maxInSize)
	sb.WriteString(" " + whereSQL + " ")
	return sb.String(), u.buildPKParams(beforeImage.Rows, meta.GetPrimaryKeyOnlyName())
}

// buildAfterImageSQL build the SQL to query before image data
func (u *updateExecutor) buildBeforeImageSQL(ctx context.Context, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	if !u.isAstStmtValid() {
		log.Errorf("invalid update stmt")
		return "", nil, fmt.Errorf("invalid update stmt")
	}

	updateStmt := u.parserCtx.UpdateStmt
	fields := make([]*ast.SelectField, 0, len(updateStmt.List))

	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, column := range updateStmt.List {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: column.Column,
				},
			})
		}

		// select indexes columns
		tableName, _ := u.parserCtx.GetTableName()
		metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, u.execContext.DBName, tableName)
		if err != nil {
			return "", nil, err
		}
		for _, columnName := range metaData.GetPrimaryKeyOnlyName() {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: &ast.ColumnName{
						Name: model.CIStr{
							O: columnName,
							L: columnName,
						},
					},
				},
			})
		}
	} else {
		fields = append(fields, &ast.SelectField{
			Expr: &ast.ColumnNameExpr{
				Name: &ast.ColumnName{
					Name: model.CIStr{
						O: "*",
						L: "*",
					},
				},
			},
		})
	}

	selStmt := ast.SelectStmt{
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

	b := bytes.NewByteBuffer([]byte{})
	_ = selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	return sql, u.buildSelectArgs(&selStmt, args), nil
}
