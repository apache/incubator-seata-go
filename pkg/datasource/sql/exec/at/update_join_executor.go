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
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/format"
	"github.com/arana-db/parser/model"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/bytes"
	"seata.apache.org/seata-go/pkg/util/log"
)

// updateJoinExecutor execute update SQL
type updateJoinExecutor struct {
	updateExecutor
	parserCtx                       *types.ParseContext
	execContext                     *types.ExecContext
	isLowerSupportGroupByPksVersion bool
	sqlMode                         string
}

// NewUpdateJoinExecutor get executor
func NewUpdateJoinExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	minimumVersion, _ := util.ConvertDbVersion("5.7.5")
	currentVersion, _ := util.ConvertDbVersion(execContent.DbVersion)
	return &updateJoinExecutor{
		parserCtx:                       parserCtx,
		execContext:                     execContent,
		updateExecutor:                  updateExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}},
		isLowerSupportGroupByPksVersion: currentVersion < minimumVersion,
	}
}

// ExecContext exec SQL, and generate before image and after image
func (u *updateJoinExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	u.beforeHooks(ctx, u.execContext)
	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()

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

func (u *updateJoinExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	var recordImages []*types.RecordImage

	// Parsing multiple table name
	updateStmt := u.parserCtx.UpdateStmt
	tableNames := u.parseTableName(updateStmt.TableRefs.TableRefs)

	for _, tbName := range tableNames {
		metaData, err := datasource.GetTableCache(u.execContext.DBType).GetTableMeta(ctx, u.execContext.DBName, tbName)
		if err != nil {
			return nil, err
		}
		selectSQL, selectArgs, err := u.buildBeforeImageSQL(ctx, metaData, u.execContext.NamedValues)
		if err != nil {
			return nil, err
		}
		if selectSQL == "" {
			log.Debugf("Skip unused table [{%s}] when build select sql by update sourceQuery", tbName)
			continue
		}

		var image *types.RecordImage
		rowsi, err := u.rowsPrepare(ctx, selectSQL, selectArgs)
		if err == nil {
			image, err = u.buildRecordImages(rowsi, metaData, types.SQLTypeUpdateJoin)
		}
		if rowsi != nil {
			if rowerr := rows.Close(); rowerr != nil {
				log.Errorf("rows close fail, err:%v", rowerr)
				return nil, rowerr
			}
		}
		if err != nil {
			// If one fail, all fails
			return nil, err
		}

		lockKey := u.buildLockKey(image, *metaData)
		u.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
		image.SQLType = u.parserCtx.SQLType

		recordImages = append(recordImages, image)
	}

	return recordImages, nil
}

func (u *updateJoinExecutor) afterImage(ctx context.Context, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	if len(beforeImages) == 0 {
		return nil, errors.New("empty beforeImages")
	}

	var recordImages []*types.RecordImage
	for _, beforeImage := range beforeImages {
		metaData, err := datasource.GetTableCache(u.execContext.DBType).GetTableMeta(ctx, u.execContext.DBName, beforeImage.TableName)
		if err != nil {
			return nil, err
		}

		selectSQL, selectArgs, err := u.buildAfterImageSQL(ctx, *beforeImage, metaData)
		if err != nil {
			return nil, err
		}

		var image *types.RecordImage
		rowsi, err := u.rowsPrepare(ctx, selectSQL, selectArgs)
		if err == nil {
			image, err = u.buildRecordImages(rowsi, metaData, types.SQLTypeUpdateJoin)
		}
		if rowsi != nil {
			if rowerr := rowsi.Close(); rowerr != nil {
				log.Errorf("rows close fail, err:%v", rowerr)
				return nil, rowerr
			}
		}
		if err != nil {
			// If one fail, all fails
			return nil, err
		}

		image.SQLType = u.parserCtx.SQLType
		recordImages = append(recordImages, image)
	}

	return recordImages, nil
}

// buildAfterImageSQL build the SQL to query before image data
func (u *updateJoinExecutor) buildBeforeImageSQL(ctx context.Context, tableMeta *types.TableMeta, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	if !u.isAstStmtValid() {
		log.Errorf("invalid update join stmt")
		return "", nil, fmt.Errorf("invalid update join stmt")
	}

	updateStmt := u.parserCtx.UpdateStmt
	fields, err := u.buildSelectFields(ctx, tableMeta)
	if err != nil {
		return "", nil, err
	}
	if len(fields) == 0 {
		return "", nil, err
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From:           updateStmt.TableRefs,
		Where:          updateStmt.Where,
		Fields:         &ast.FieldList{Fields: fields},
		OrderBy:        updateStmt.Order,
		Limit:          updateStmt.Limit,
		TableHints:     updateStmt.TableHints,
		// maybe duplicate row for select join sql.remove duplicate row by 'group by' condition
		GroupBy: &ast.GroupByClause{
			Items: u.buildGroupByClause(ctx, tableMeta.TableName, tableMeta.GetPrimaryKeyOnlyName(), fields),
		},
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

func (u *updateJoinExecutor) buildAfterImageSQL(ctx context.Context, beforeImage types.RecordImage, meta *types.TableMeta) (string, []driver.NamedValue, error) {
	selectSQL, selectArgs := u.updateExecutor.buildAfterImageSQL(beforeImage, meta)

	needUpdateColumns, err := u.buildSelectFields(ctx, meta)
	if err != nil {
		return "", nil, err
	}

	// maybe duplicate row for select join sql.remove duplicate row by 'group by' condition
	groupByStr := strings.Builder{}
	groupByItem := u.buildGroupByClause(ctx, meta.TableName, meta.GetPrimaryKeyOnlyName(), needUpdateColumns)

	groupByStr.WriteString(selectSQL)
	groupByStr.WriteString(" GROUP BY ")
	for index, item := range groupByItem {
		if index != 0 {
			groupByStr.WriteString(",")
		}
		groupByStr.WriteString(item.Expr.(*ast.ColumnNameExpr).Name.String())
	}

	groupByStr.WriteString(" ")
	return groupByStr.String(), selectArgs, nil
}

func (u *updateJoinExecutor) parseTableName(joinMate *ast.Join) []string {
	var tableNames []string
	if item, ok := joinMate.Left.(*ast.Join); ok {
		tableNames = u.parseTableName(item)
	} else {
		leftName := joinMate.Left.(*ast.TableSource).Source.(*ast.TableName)
		tableNames = append(tableNames, leftName.Name.O)
	}

	rightName := joinMate.Right.(*ast.TableSource).Source.(*ast.TableName)
	tableNames = append(tableNames, rightName.Name.O)
	return tableNames
}

// build group by condition which used for removing duplicate row in select join sql
func (u *updateJoinExecutor) buildGroupByClause(ctx context.Context, tableName string, pkColumns []string, allSelectColumns []*ast.SelectField) []*ast.ByItem {
	var groupByPks = true
	//only pks group by is valid when db version >= 5.7.5
	if u.isLowerSupportGroupByPksVersion {
		if u.sqlMode == "" {
			rowsi, err := u.rowsPrepare(ctx, "SELECT @@SQL_MODE", nil)
			defer func() {
				if rowsi != nil {
					if rowerr := rowsi.Close(); rowerr != nil {
						log.Errorf("rows close fail, err:%v", rowerr)
					}
				}
			}()
			if err != nil {
				groupByPks = false
				log.Warnf("determine group by pks or all columns error:%s", err)
			} else {
				// getString("@@SQL_MODE")
				mode := make([]driver.Value, 1)
				if err = rowsi.Next(mode); err != nil {
					if err != io.EOF && len(mode) == 1 {
						u.sqlMode = reflect.ValueOf(mode[0]).String()
					}
				}
			}
		}

		if strings.Contains(u.sqlMode, "ONLY_FULL_GROUP_BY") {
			groupByPks = false
		}
	}

	groupByColumns := make([]*ast.ByItem, 0)
	if groupByPks {
		for _, column := range pkColumns {
			groupByColumns = append(groupByColumns, &ast.ByItem{
				Expr: &ast.ColumnNameExpr{
					Name: &ast.ColumnName{
						Table: model.CIStr{
							O: tableName,
							L: strings.ToLower(tableName),
						},
						Name: model.CIStr{
							O: column,
							L: strings.ToLower(column),
						},
					},
				},
			})
		}
	} else {
		for _, column := range allSelectColumns {
			groupByColumns = append(groupByColumns, &ast.ByItem{
				Expr: column.Expr,
			})
		}
	}
	return groupByColumns
}
