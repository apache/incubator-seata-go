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

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/util/bytes"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

const (
	LowerSupportGroupByPksVersion = "5.7.5"
)

// updateJoinExecutor execute update SQL
type updateJoinExecutor struct {
	baseExecutor
	parserCtx                       *types.ParseContext
	execContext                     *types.ExecContext
	isLowerSupportGroupByPksVersion bool
	sqlMode                         string
	tableAliasesMap                 map[string]string
}

// NewUpdateJoinExecutor get executor
func NewUpdateJoinExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	minimumVersion, _ := util.ConvertDbVersion(LowerSupportGroupByPksVersion)
	currentVersion, _ := util.ConvertDbVersion(execContent.DbVersion)
	return &updateJoinExecutor{
		parserCtx:                       parserCtx,
		execContext:                     execContent,
		baseExecutor:                    baseExecutor{hooks: hooks},
		isLowerSupportGroupByPksVersion: currentVersion < minimumVersion,
		tableAliasesMap:                 make(map[string]string, 0),
	}
}

// ExecContext exec SQL, and generate before image and after image
func (u *updateJoinExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	if err := u.beforeHooks(ctx, u.execContext); err != nil {
		return nil, err
	}
	defer func() {
		u.afterHooks(ctx, u.execContext)
	}()

	if u.isAstStmtValid() {
		u.tableAliasesMap = u.parseTableName(u.parserCtx.UpdateStmt.TableRefs.TableRefs)
	}

	beforeImages, err := u.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, u.execContext.Query, u.execContext.NamedValues)
	if err != nil {
		return nil, err
	}
	if len(beforeImages) == 0 {
		var result driver.Result
		if res != nil {
			result = res.GetResult()
		}
		if result == nil {
			err := errors.New("cannot determine affected rows for UPDATE JOIN with empty before images: result is unavailable")
			if res != nil {
				if rows := res.GetRows(); rows != nil {
					err = errors.Join(err, rows.Close())
				}
			}
			return nil, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("cannot determine affected rows for UPDATE JOIN with empty before images: %w", err)
		}
		if rowsAffected != 0 {
			return nil, fmt.Errorf("UPDATE JOIN affected %d rows with empty before images", rowsAffected)
		}
		return res, nil
	}

	afterImages, err := u.afterImage(ctx, beforeImages)
	if err != nil {
		return nil, err
	}

	if len(afterImages) != len(beforeImages) {
		return nil, errors.New("Before image size is not equaled to after image size, probably because you updated the primary keys.")
	}
	for i, beforeImage := range beforeImages {
		if len(beforeImage.Rows) != len(afterImages[i].Rows) {
			return nil, fmt.Errorf("before and after image row count mismatch for table %s: %d != %d", beforeImage.TableName, len(beforeImage.Rows), len(afterImages[i].Rows))
		}
	}

	u.execContext.TxCtx.RoundImages.AppendBeofreImages(beforeImages)
	u.execContext.TxCtx.RoundImages.AppendAfterImages(afterImages)

	return res, nil
}

func (u *updateJoinExecutor) isAstStmtValid() bool {
	return u.parserCtx != nil && u.parserCtx.UpdateStmt != nil && u.parserCtx.UpdateStmt.TableRefs.TableRefs.Right != nil
}

func (u *updateJoinExecutor) beforeImage(ctx context.Context) ([]*types.RecordImage, error) {
	if !u.isAstStmtValid() {
		return nil, nil
	}

	var recordImages []*types.RecordImage

	for tbName, tableAliases := range u.tableAliasesMap {
		metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, u.execContext.DBName, tbName)
		if err != nil {
			return nil, err
		}
		selectSQL, selectArgs, err := u.buildBeforeImageSQL(ctx, metaData, tableAliases, u.execContext.NamedValues)
		if err != nil {
			return nil, err
		}
		if selectSQL == "" {
			log.Debugf("Skip unused table [{%s}] when build select sql by update sourceQuery", tbName)
			continue
		}

		var image *types.RecordImage
		rowsi, err := u.rowsPrepare(ctx, u.execContext.Conn, selectSQL, selectArgs)
		if err == nil {
			matchedRows := &updateJoinRows{Rows: rowsi}
			primaryKeys := metaData.GetPrimaryKeyMap()
			for i, name := range rowsi.Columns() {
				if _, ok := primaryKeys[util.DelEscape(name, types.DBTypeMySQL)]; ok {
					matchedRows.pkIndexes = append(matchedRows.pkIndexes, i)
				}
			}
			image, err = u.buildRecordImages(matchedRows, metaData, types.SQLTypeUpdate, types.DBTypeMySQL)
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

		if len(image.Rows) == 0 {
			continue
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
		return nil, nil
	}

	var recordImages []*types.RecordImage
	for _, beforeImage := range beforeImages {
		metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, u.execContext.DBName, beforeImage.TableName)
		if err != nil {
			return nil, err
		}

		selectSQL, selectArgs, err := u.buildAfterImageSQL(ctx, *beforeImage, metaData, u.tableAliasesMap[beforeImage.TableName])
		if err != nil {
			return nil, err
		}

		var image *types.RecordImage
		rowsi, err := u.rowsPrepare(ctx, u.execContext.Conn, selectSQL, selectArgs)
		if err == nil {
			image, err = u.buildRecordImages(rowsi, metaData, types.SQLTypeUpdate, types.DBTypeMySQL)
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
func (u *updateJoinExecutor) buildBeforeImageSQL(ctx context.Context, tableMeta *types.TableMeta, tableAliases string, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	updateStmt := u.parserCtx.UpdateStmt
	fields, err := u.buildSelectFields(ctx, tableMeta, tableAliases, updateStmt.List)
	if err != nil {
		return "", nil, err
	}
	if len(fields) == 0 {
		return "", nil, err
	}
	if !undo.UndoConfig.OnlyCareUpdateColumns {
		tableName := tableAliases
		if tableName == "" {
			tableName = tableMeta.TableName
		}
		fields = make([]*ast.SelectField, 0, len(tableMeta.ColumnNames))
		for _, columnName := range tableMeta.ColumnNames {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{Name: &ast.ColumnName{
					Table: model.NewCIStr(tableName),
					Name:  model.NewCIStr(columnName),
				}},
			})
		}
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
			Items: u.buildGroupByClause(ctx, tableMeta.TableName, tableAliases, tableMeta.GetPrimaryKeyOnlyName(), fields),
		},
		LockInfo: &ast.SelectLockInfo{
			LockType: ast.SelectLockForUpdate,
		},
	}

	restoreFlags := format.RestoreKeyWordUppercase
	if !undo.UndoConfig.OnlyCareUpdateColumns {
		restoreFlags |= format.RestoreNameBackQuotes
	}
	b := bytes.NewByteBuffer([]byte{})
	_ = selStmt.Restore(format.NewRestoreCtx(restoreFlags, b))
	sql := string(b.Bytes())
	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	return sql, u.buildSelectArgs(&selStmt, args), nil
}

func (u *updateJoinExecutor) buildAfterImageSQL(ctx context.Context, beforeImage types.RecordImage, meta *types.TableMeta, tableAliases string) (string, []driver.NamedValue, error) {
	if len(beforeImage.Rows) == 0 {
		return "", nil, nil
	}

	fields, err := u.buildSelectFields(ctx, meta, tableAliases, u.parserCtx.UpdateStmt.List)
	if err != nil {
		return "", nil, err
	}
	if len(fields) == 0 {
		return "", nil, err
	}

	dbType := effectiveDBType(u.execContext.DBType)
	pkNames := meta.GetPrimaryKeyOnlyName()
	if len(pkNames) == 0 {
		return "", nil, fmt.Errorf("primary key metadata is empty for table %s", meta.TableName)
	}
	args := u.buildPKParams(beforeImage.Rows, pkNames, dbType)
	if len(args) != len(beforeImage.Rows)*len(pkNames) {
		return "", nil, fmt.Errorf("incomplete primary keys in before image for table %s", meta.TableName)
	}
	table := findUpdateJoinTable(u.parserCtx.UpdateStmt.TableRefs.TableRefs, meta.TableName, tableAliases)
	if table == nil {
		return "", nil, fmt.Errorf("target table %s not found in update join", meta.TableName)
	}

	selStmt := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{},
		From: &ast.TableRefsClause{TableRefs: &ast.Join{
			Left: &ast.TableSource{
				Source: &ast.TableName{Schema: table.Schema, Name: table.Name},
				AsName: model.NewCIStr(tableAliases),
			},
		}},
		Fields: &ast.FieldList{Fields: fields},
	}

	b := bytes.NewByteBuffer([]byte{})
	if err := selStmt.Restore(format.NewRestoreCtx(format.RestoreKeyWordUppercase, b)); err != nil {
		return "", nil, err
	}
	sql := string(b.Bytes()) + " WHERE " + u.buildWhereConditionByPKs(pkNames, len(beforeImage.Rows), dbType, maxInSize)
	log.Infof("build select sql by update sourceQuery, sql {%s}", sql)

	return sql, args, nil
}

// updateJoinRows skips unmatched outer-join targets before scanning can turn NULL keys into zero values.
type updateJoinRows struct {
	driver.Rows
	pkIndexes []int
}

func (r *updateJoinRows) Next(dest []driver.Value) error {
nextRow:
	for {
		if err := r.Rows.Next(dest); err != nil {
			return err
		}
		for _, i := range r.pkIndexes {
			value := reflect.ValueOf(dest[i])
			if !value.IsValid() || (value.Kind() == reflect.Slice && value.IsNil()) {
				continue nextRow
			}
		}
		return nil
	}
}

func findUpdateJoinTable(node ast.ResultSetNode, tableName, tableAlias string) *ast.TableName {
	switch node := node.(type) {
	case *ast.Join:
		if table := findUpdateJoinTable(node.Left, tableName, tableAlias); table != nil {
			return table
		}
		return findUpdateJoinTable(node.Right, tableName, tableAlias)
	case *ast.TableSource:
		if table, ok := node.Source.(*ast.TableName); ok && table.Name.O == tableName && node.AsName.O == tableAlias {
			return table
		}
	}
	return nil
}

func (u *updateJoinExecutor) parseTableName(joinMate *ast.Join) map[string]string {
	tableNames := make(map[string]string, 0)
	if item, ok := joinMate.Left.(*ast.Join); ok {
		tableNames = u.parseTableName(item)
	} else {
		leftTableSource := joinMate.Left.(*ast.TableSource)
		leftName := leftTableSource.Source.(*ast.TableName)
		tableNames[leftName.Name.O] = leftTableSource.AsName.O
	}

	rightTableSource := joinMate.Right.(*ast.TableSource)
	rightName := rightTableSource.Source.(*ast.TableName)
	tableNames[rightName.Name.O] = rightTableSource.AsName.O
	return tableNames
}

// build group by condition which used for removing duplicate row in select join sql
func (u *updateJoinExecutor) buildGroupByClause(ctx context.Context, tableName string, tableAliases string, pkColumns []string, allSelectColumns []*ast.SelectField) []*ast.ByItem {
	var groupByPks = true
	if tableAliases != "" {
		tableName = tableAliases
	}
	//only pks group by is valid when db version >= 5.7.5
	if u.isLowerSupportGroupByPksVersion {
		if u.sqlMode == "" {
			rowsi, err := u.rowsPrepare(ctx, u.execContext.Conn, "SELECT @@SQL_MODE", nil)
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
