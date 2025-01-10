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
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"
	"github.com/arana-db/parser/test_driver"
	gxsort "github.com/dubbogo/gost/sort"
	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/reflectx"
)

type baseExecutor struct {
	hooks []exec.SQLHook
}

func (b *baseExecutor) beforeHooks(ctx context.Context, execCtx *types.ExecContext) {
	for _, hook := range b.hooks {
		hook.Before(ctx, execCtx)
	}
}

func (b *baseExecutor) afterHooks(ctx context.Context, execCtx *types.ExecContext) {
	for _, hook := range b.hooks {
		hook.After(ctx, execCtx)
	}
}

// GetScanSlice get the column type for scan
// todo to use ColumnInfo get slice
func (*baseExecutor) GetScanSlice(columnNames []string, tableMeta *types.TableMeta) []interface{} {
	scanSlice := make([]interface{}, 0, len(columnNames))
	for _, columnName := range columnNames {
		var (
			// get from metaData from this column
			columnMeta = tableMeta.Columns[columnName]
		)
		switch strings.ToUpper(columnMeta.DatabaseTypeString) {
		case "VARCHAR", "NVARCHAR", "VARCHAR2", "CHAR", "TEXT", "JSON", "TINYTEXT":
			var scanVal sql.NullString
			scanSlice = append(scanSlice, &scanVal)
		case "BIT", "INT", "LONGBLOB", "SMALLINT", "TINYINT", "BIGINT", "MEDIUMINT":
			if columnMeta.IsNullable == 0 {
				scanVal := int64(0)
				scanSlice = append(scanSlice, &scanVal)
			} else {
				scanVal := sql.NullInt64{}
				scanSlice = append(scanSlice, &scanVal)
			}
		case "DATE", "DATETIME", "TIME", "TIMESTAMP", "YEAR":
			var scanVal sql.NullTime
			scanSlice = append(scanSlice, &scanVal)
		case "DECIMAL", "DOUBLE", "FLOAT":
			if columnMeta.IsNullable == 0 {
				scanVal := float64(0)
				scanSlice = append(scanSlice, &scanVal)
			} else {
				scanVal := sql.NullFloat64{}
				scanSlice = append(scanSlice, &scanVal)
			}
		default:
			scanVal := sql.RawBytes{}
			scanSlice = append(scanSlice, &scanVal)
		}
	}
	return scanSlice
}

func (b *baseExecutor) buildSelectArgs(stmt *ast.SelectStmt, args []driver.NamedValue) []driver.NamedValue {
	var (
		selectArgsIndexs = make([]int32, 0)
		selectArgs       = make([]driver.NamedValue, 0)
	)

	b.traversalArgs(stmt.From.TableRefs, &selectArgsIndexs)
	b.traversalArgs(stmt.Where, &selectArgsIndexs)
	if stmt.GroupBy != nil {
		for _, item := range stmt.GroupBy.Items {
			b.traversalArgs(item, &selectArgsIndexs)
		}
	}
	if stmt.OrderBy != nil {
		for _, item := range stmt.OrderBy.Items {
			b.traversalArgs(item, &selectArgsIndexs)
		}
	}
	if stmt.Limit != nil {
		if stmt.Limit.Offset != nil {
			b.traversalArgs(stmt.Limit.Offset, &selectArgsIndexs)
		}
		if stmt.Limit.Count != nil {
			b.traversalArgs(stmt.Limit.Count, &selectArgsIndexs)
		}
	}
	// sort selectArgs index array
	gxsort.Int32(selectArgsIndexs)
	for _, index := range selectArgsIndexs {
		selectArgs = append(selectArgs, args[index])
	}

	return selectArgs
}

// todo perfect all sql operation
func (b *baseExecutor) traversalArgs(node ast.Node, argsIndex *[]int32) {
	if node == nil {
		return
	}
	switch node.(type) {
	case *ast.BinaryOperationExpr:
		expr := node.(*ast.BinaryOperationExpr)
		b.traversalArgs(expr.L, argsIndex)
		b.traversalArgs(expr.R, argsIndex)
		break
	case *ast.BetweenExpr:
		expr := node.(*ast.BetweenExpr)
		b.traversalArgs(expr.Left, argsIndex)
		b.traversalArgs(expr.Right, argsIndex)
		break
	case *ast.PatternInExpr:
		exprs := node.(*ast.PatternInExpr).List
		for i := 0; i < len(exprs); i++ {
			b.traversalArgs(exprs[i], argsIndex)
		}
		break
	case *ast.Join:
		exprs := node.(*ast.Join)
		b.traversalArgs(exprs.Left, argsIndex)
		if exprs.Right != nil {
			b.traversalArgs(exprs.Right, argsIndex)
		}
		if exprs.On != nil {
			b.traversalArgs(exprs.On.Expr, argsIndex)
		}
		break
	case *test_driver.ParamMarkerExpr:
		*argsIndex = append(*argsIndex, int32(node.(*test_driver.ParamMarkerExpr).Order))
		break
	}
}

func (b *baseExecutor) buildRecordImages(rowsi driver.Rows, tableMetaData *types.TableMeta, sqlType types.SQLType) (*types.RecordImage, error) {
	// select column names
	columnNames := rowsi.Columns()
	rowImages := make([]types.RowImage, 0)

	sqlRows := util.NewScanRows(rowsi)

	for sqlRows.Next() {
		ss := b.GetScanSlice(columnNames, tableMetaData)

		err := sqlRows.Scan(ss...)
		if err != nil {
			return nil, err
		}

		columns := make([]types.ColumnImage, 0)
		// build record image
		for i, name := range columnNames {
			columnMeta := tableMetaData.Columns[name]

			keyType := types.IndexTypeNull
			if _, ok := tableMetaData.GetPrimaryKeyMap()[name]; ok {
				keyType = types.IndexTypePrimaryKey
			}
			jdbcType := types.MySQLStrToJavaType(columnMeta.DatabaseTypeString)

			columns = append(columns, types.ColumnImage{
				KeyType:    keyType,
				ColumnName: name,
				ColumnType: jdbcType,
				Value:      getSqlNullValue(reflectx.GetElemDataValue(ss[i])),
			})
		}
		rowImages = append(rowImages, types.RowImage{Columns: columns})
	}

	return &types.RecordImage{TableName: tableMetaData.TableName, Rows: rowImages, SQLType: sqlType}, nil
}

func (b *baseExecutor) getNeedColumns(meta *types.TableMeta, columns []string, dbType types.DBType) []string {
	var needUpdateColumns []string
	if undo.UndoConfig.OnlyCareUpdateColumns && columns != nil && len(columns) > 0 {
		needUpdateColumns = columns
		if !b.containsPKByName(meta, columns) {
			pkNames := meta.GetPrimaryKeyOnlyName()
			if pkNames != nil && len(pkNames) > 0 {
				for _, name := range pkNames {
					needUpdateColumns = append(needUpdateColumns, name)
				}
			}
		}
		// todo If it contains onUpdate columns, add onUpdate columns
	} else {
		needUpdateColumns = meta.ColumnNames
	}

	for i := range needUpdateColumns {
		needUpdateColumns[i] = AddEscape(needUpdateColumns[i], dbType)
	}
	return needUpdateColumns
}

func (b *baseExecutor) containsPKByName(meta *types.TableMeta, columns []string) bool {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}

	matchCounter := 0
	for _, column := range columns {
		for _, pkName := range pkColumnNameList {
			if strings.EqualFold(pkName, column) ||
				strings.EqualFold(pkName, strings.ToLower(column)) {
				matchCounter++
			}
		}
	}

	return matchCounter == len(pkColumnNameList)
}

func (u *baseExecutor) buildSelectFields(ctx context.Context, tableMeta *types.TableMeta, tableAliases string, inUseFields []*ast.Assignment) ([]*ast.SelectField, error) {
	fields := make([]*ast.SelectField, 0, len(inUseFields))

	tableName := tableAliases
	if tableAliases == "" {
		tableName = tableMeta.TableName
	}
	if undo.UndoConfig.OnlyCareUpdateColumns {
		for _, column := range inUseFields {
			tn := column.Column.Table.O
			if tn != "" && tn != tableName {
				continue
			}

			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: column.Column,
				},
			})
		}

		if len(fields) == 0 {
			return fields, nil
		}

		// select indexes columns
		for _, columnName := range tableMeta.GetPrimaryKeyOnlyName() {
			fields = append(fields, &ast.SelectField{
				Expr: &ast.ColumnNameExpr{
					Name: &ast.ColumnName{
						Table: model.CIStr{
							O: tableName,
							L: tableName,
						},
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

	return fields, nil
}

func getSqlNullValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	if v, ok := value.(sql.NullString); ok {
		if v.Valid {
			return v.String
		}
		return nil
	}
	if v, ok := value.(sql.NullFloat64); ok {
		if v.Valid {
			return v.Float64
		}
		return nil
	}
	if v, ok := value.(sql.NullBool); ok {
		if v.Valid {
			return v.Bool
		}
		return nil
	}
	if v, ok := value.(sql.NullTime); ok {
		if v.Valid {
			return v.Time
		}
		return nil
	}
	if v, ok := value.(sql.NullByte); ok {
		if v.Valid {
			return v.Byte
		}
		return nil
	}
	if v, ok := value.(sql.NullInt16); ok {
		if v.Valid {
			return v.Int16
		}
		return nil
	}
	if v, ok := value.(sql.NullInt32); ok {
		if v.Valid {
			return v.Int32
		}
		return nil
	}
	if v, ok := value.(sql.NullInt64); ok {
		if v.Valid {
			return v.Int64
		}
		return nil
	}
	return value
}

// buildWhereConditionByPKs build where condition by primary keys
// each pk is a condition.the result will like :" (id,userCode) in ((?,?),(?,?)) or (id,userCode) in ((?,?),(?,?) ) or (id,userCode) in ((?,?))"
func (b *baseExecutor) buildWhereConditionByPKs(pkNameList []string, rowSize int, dbType string, maxInSize int) string {
	var (
		whereStr  = &strings.Builder{}
		batchSize = rowSize/maxInSize + 1
	)

	if rowSize%maxInSize == 0 {
		batchSize = rowSize / maxInSize
	}

	for batch := 0; batch < batchSize; batch++ {
		if batch > 0 {
			whereStr.WriteString(" OR ")
		}
		whereStr.WriteString("(")

		for i := 0; i < len(pkNameList); i++ {
			if i > 0 {
				whereStr.WriteString(",")
			}
			// todo add escape
			whereStr.WriteString(fmt.Sprintf("`%s`", pkNameList[i]))
		}
		whereStr.WriteString(") IN (")

		var eachSize int

		if batch == batchSize-1 {
			if rowSize%maxInSize == 0 {
				eachSize = maxInSize
			} else {
				eachSize = rowSize % maxInSize
			}
		} else {
			eachSize = maxInSize
		}

		for i := 0; i < eachSize; i++ {
			if i > 0 {
				whereStr.WriteString(",")
			}
			whereStr.WriteString("(")
			for j := 0; j < len(pkNameList); j++ {
				if j > 0 {
					whereStr.WriteString(",")
				}
				whereStr.WriteString("?")
			}
			whereStr.WriteString(")")
		}
		whereStr.WriteString(")")
	}
	return whereStr.String()
}

func (b *baseExecutor) buildPKParams(rows []types.RowImage, pkNameList []string) []driver.NamedValue {
	params := make([]driver.NamedValue, 0)
	for _, row := range rows {
		coumnMap := row.GetColumnMap()
		for i, pk := range pkNameList {
			if col, ok := coumnMap[pk]; ok {
				params = append(params, driver.NamedValue{
					Ordinal: i, Value: col.Value,
				})
			}
		}
	}
	return params
}

// the string as local key. the local key example(multi pk): "t_user:1_a,2_b"
func (b *baseExecutor) buildLockKey(records *types.RecordImage, meta types.TableMeta) string {
	var (
		lockKeys      bytes.Buffer
		filedSequence int
	)
	lockKeys.WriteString(meta.TableName)
	lockKeys.WriteString(":")

	keys := meta.GetPrimaryKeyOnlyName()

	for _, row := range records.Rows {
		if filedSequence > 0 {
			lockKeys.WriteString(",")
		}
		pkSplitIndex := 0
		for _, column := range row.Columns {
			var hasKeyColumn bool
			for _, key := range keys {
				if column.ColumnName == key {
					hasKeyColumn = true
					if pkSplitIndex > 0 {
						lockKeys.WriteString("_")
					}
					lockKeys.WriteString(fmt.Sprintf("%v", column.Value))
					pkSplitIndex++
				}
			}
			if hasKeyColumn {
				filedSequence++
			}
		}
	}

	return lockKeys.String()
}

func (b *baseExecutor) rowsPrepare(ctx context.Context, conn driver.Conn, selectSQL string, selectArgs []driver.NamedValue) (driver.Rows, error) {
	var queryer driver.Queryer

	queryerContext, ok := conn.(driver.QueryerContext)
	if !ok {
		queryer, ok = conn.(driver.Queryer)
	}
	if ok {
		var err error
		rows, err = util.CtxDriverQuery(ctx, queryerContext, queryer, selectSQL, selectArgs)

		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("target conn should been driver.QueryerContext or driver.Queryer")
	}
	return rows, nil
}
