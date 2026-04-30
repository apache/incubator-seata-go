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
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/model"
	"github.com/arana-db/parser/test_driver"
	gxsort "github.com/dubbogo/gost/sort"
	"github.com/pkg/errors"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/v2/pkg/util/log"
	"seata.apache.org/seata-go/v2/pkg/util/reflectx"
)

type baseExecutor struct {
	hooks []exec.SQLHook
}

var mysqlStringLiteralForPostgresPattern = regexp.MustCompile(`_UTF8MB4([[:alnum:]_]+)`)

func (b *baseExecutor) beforeHooks(ctx context.Context, execCtx *types.ExecContext) error {
	for _, hook := range b.hooks {
		if err := hook.Before(ctx, execCtx); err != nil {
			return err
		}
	}
	return nil
}

func (b *baseExecutor) afterHooks(ctx context.Context, execCtx *types.ExecContext) error {
	for _, hook := range b.hooks {
		if err := hook.After(ctx, execCtx); err != nil {
			log.Errorf("after hook failed: %v", err)
		}
	}
	return nil
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
		case "VARCHAR", "NVARCHAR", "VARCHAR2", "CHAR", "TEXT", "JSON", "JSONB", "TINYTEXT", "UUID", "BPCHAR", "CHARACTER VARYING", "CHARACTER":
			var scanVal sql.NullString
			scanSlice = append(scanSlice, &scanVal)
		case "BIT", "INT", "INTEGER", "INT2", "INT4", "INT8", "LONGBLOB", "SMALLINT", "TINYINT", "BIGINT", "MEDIUMINT", "SERIAL", "BIGSERIAL", "SMALLSERIAL":
			if columnMeta.IsNullable == 0 {
				scanVal := int64(0)
				scanSlice = append(scanSlice, &scanVal)
			} else {
				scanVal := sql.NullInt64{}
				scanSlice = append(scanSlice, &scanVal)
			}
		case "BOOLEAN", "BOOL":
			if columnMeta.IsNullable == 0 {
				scanVal := false
				scanSlice = append(scanSlice, &scanVal)
			} else {
				scanVal := sql.NullBool{}
				scanSlice = append(scanSlice, &scanVal)
			}
		case "DATE", "DATETIME", "TIME", "TIME WITH TIME ZONE", "TIME WITHOUT TIME ZONE", "TIMESTAMP", "TIMESTAMPTZ", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE", "YEAR":
			var scanVal sql.NullTime
			scanSlice = append(scanSlice, &scanVal)
		case "DECIMAL", "DOUBLE", "DOUBLE PRECISION", "FLOAT", "NUMERIC", "REAL":
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

func effectiveDBType(dbType types.DBType) types.DBType {
	if dbType == 0 || dbType == types.DBTypeUnknown {
		return types.DBTypeMySQL
	}
	return dbType
}

func (b *baseExecutor) getTableCache(dbType types.DBType) (datasource.TableMetaCache, error) {
	dbType = effectiveDBType(dbType)
	tableCache := datasource.GetTableCache(dbType)
	if tableCache == nil {
		return nil, fmt.Errorf("table meta cache not registered for dbType %s", dbType.String())
	}
	return tableCache, nil
}

func jdbcTypeForDatabaseType(dbType types.DBType, databaseType string) types.JDBCType {
	dbType = effectiveDBType(dbType)
	if dbType != types.DBTypePostgreSQL {
		return types.MySQLStrToJavaType(databaseType)
	}

	switch strings.ToUpper(databaseType) {
	case "BOOLEAN", "BOOL":
		return types.JDBCTypeBoolean
	case "SMALLINT", "INT2":
		return types.JDBCTypeSmallInt
	case "INTEGER", "INT", "INT4", "SERIAL":
		return types.JDBCTypeInteger
	case "BIGINT", "INT8", "BIGSERIAL":
		return types.JDBCTypeBigInt
	case "REAL":
		return types.JDBCTypeReal
	case "DOUBLE PRECISION", "DOUBLE":
		return types.JDBCTypeDouble
	case "NUMERIC", "DECIMAL":
		return types.JDBCTypeDecimal
	case "CHAR", "BPCHAR":
		return types.JDBCTypeChar
	case "VARCHAR", "TEXT", "UUID":
		return types.JDBCTypeVarchar
	case "DATE":
		return types.JDBCTypeDate
	case "TIME", "TIME WITH TIME ZONE", "TIME WITHOUT TIME ZONE":
		return types.JDBCTypeTime
	case "TIMESTAMP", "TIMESTAMPTZ", "TIMESTAMP WITH TIME ZONE", "TIMESTAMP WITHOUT TIME ZONE":
		return types.JDBCTypeTimestamp
	case "BYTEA":
		return types.JDBCTypeLongVarBinary
	case "JSON", "JSONB":
		return types.JDBCTypeLongVarchar
	default:
		return types.JDBCTypeOther
	}
}

func (b *baseExecutor) normalizeGeneratedSQL(query string, dbType types.DBType) string {
	dbType = effectiveDBType(dbType)
	if dbType == types.DBTypePostgreSQL {
		query = strings.Replace(query, "SELECT SQL_NO_CACHE ", "SELECT ", 1)
		query = mysqlStringLiteralForPostgresPattern.ReplaceAllString(query, "'$1'")
	}
	return util.RewritePlaceholders(query, dbType)
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
	case *ast.UnaryOperationExpr:
		expr := node.(*ast.UnaryOperationExpr)
		b.traversalArgs(expr.V, argsIndex)
		break
	case *ast.FuncCallExpr:
		expr := node.(*ast.FuncCallExpr)
		for _, arg := range expr.Args {
			b.traversalArgs(arg, argsIndex)
		}
		break
	case *ast.SubqueryExpr:
		expr := node.(*ast.SubqueryExpr)
		if expr.Query != nil {
			b.traversalArgs(expr.Query, argsIndex)
		}
		break
	case *ast.ExistsSubqueryExpr:
		expr := node.(*ast.ExistsSubqueryExpr)
		if expr.Sel != nil {
			b.traversalArgs(expr.Sel, argsIndex)
		}
		break
	case *ast.CompareSubqueryExpr:
		expr := node.(*ast.CompareSubqueryExpr)
		b.traversalArgs(expr.L, argsIndex)
		if expr.R != nil {
			b.traversalArgs(expr.R, argsIndex)
		}
		break
	case *ast.PatternLikeExpr:
		expr := node.(*ast.PatternLikeExpr)
		b.traversalArgs(expr.Expr, argsIndex)
		b.traversalArgs(expr.Pattern, argsIndex)
		break
	case *ast.IsNullExpr:
		expr := node.(*ast.IsNullExpr)
		b.traversalArgs(expr.Expr, argsIndex)
		break
	case *ast.CaseExpr:
		expr := node.(*ast.CaseExpr)
		if expr.Value != nil {
			b.traversalArgs(expr.Value, argsIndex)
		}
		for _, whenClause := range expr.WhenClauses {
			b.traversalArgs(whenClause.Expr, argsIndex)
			b.traversalArgs(whenClause.Result, argsIndex)
		}
		if expr.ElseClause != nil {
			b.traversalArgs(expr.ElseClause, argsIndex)
		}
		break
	case *test_driver.ParamMarkerExpr:
		*argsIndex = append(*argsIndex, int32(node.(*test_driver.ParamMarkerExpr).Order))
		break
	}
}

func (b *baseExecutor) buildRecordImages(rowsi driver.Rows, tableMetaData *types.TableMeta, sqlType types.SQLType, dbType types.DBType) (*types.RecordImage, error) {
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
			cleanName := util.DelEscape(name, effectiveDBType(dbType))
			columnMeta := tableMetaData.Columns[cleanName]

			keyType := types.IndexTypeNull
			if _, ok := tableMetaData.GetPrimaryKeyMap()[cleanName]; ok {
				keyType = types.IndexTypePrimaryKey
			}
			jdbcType := jdbcTypeForDatabaseType(dbType, columnMeta.DatabaseTypeString)

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
		needUpdateColumns[i] = util.AddEscape(needUpdateColumns[i], dbType)
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
		cleanColumn := util.DelEscape(column, types.DBTypeMySQL)
		for _, pkName := range pkColumnNameList {
			if strings.EqualFold(pkName, cleanColumn) {
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
func (b *baseExecutor) buildWhereConditionByPKs(pkNameList []string, rowSize int, dbType types.DBType, maxInSize int) string {
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
			if effectiveDBType(dbType) == types.DBTypeMySQL {
				whereStr.WriteString(fmt.Sprintf("`%s`", pkNameList[i]))
				continue
			}
			whereStr.WriteString(util.AddEscape(pkNameList[i], dbType))
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
	return b.normalizeGeneratedSQL(whereStr.String(), dbType)
}

func (b *baseExecutor) buildPKParams(rows []types.RowImage, pkNameList []string, dbType types.DBType) []driver.NamedValue {
	dbType = effectiveDBType(dbType)
	params := make([]driver.NamedValue, 0)
	for _, row := range rows {
		coumnMap := row.GetColumnMap()
		// Build a normalized map with escaped characters removed
		normalizedMap := make(map[string]*types.ColumnImage, len(coumnMap))
		for k, v := range coumnMap {
			normalizedMap[util.DelEscape(k, dbType)] = v
		}
		for _, pk := range pkNameList {
			cleanPK := util.DelEscape(pk, dbType)
			if col, ok := normalizedMap[cleanPK]; ok {
				params = append(params, driver.NamedValue{
					Ordinal: len(params) + 1,
					Value:   col.Value,
				})
			}
		}
	}
	return params
}

// the string as local key. the local key example(multi pk): "t_user:1_a,2_b"
func (b *baseExecutor) buildLockKey(records *types.RecordImage, meta types.TableMeta) string {
	return util.BuildLockKey(records, meta)
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
