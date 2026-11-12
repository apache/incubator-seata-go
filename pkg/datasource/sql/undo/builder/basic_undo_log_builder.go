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
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"

	gxsort "github.com/dubbogo/gost/sort"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/parser"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/util"
)

// todo the executor should be stateful
type BasicUndoLogBuilder struct{}

func tableMetaForExec(ctx context.Context, execCtx *types.ExecContext) (*types.TableMeta, error) {
	if execCtx == nil || execCtx.TableMetaReader == nil {
		return nil, fmt.Errorf("table meta reader is nil")
	}
	if execCtx.TableMetaKey == nil {
		if execCtx.ParseContext == nil {
			return nil, fmt.Errorf("parse context is nil")
		}
		ref, err := execCtx.ParseContext.GetTableRef()
		if err != nil {
			return nil, err
		}
		key, err := execCtx.TableMetaReader.ResolveTableMetaKey(ctx, execCtx.Conn, ref)
		if err != nil {
			return nil, err
		}
		execCtx.TableMetaKey = &key
	}
	return execCtx.TableMetaReader.GetTableMeta(ctx, *execCtx.TableMetaKey)
}

func tableNameForExec(execCtx *types.ExecContext, fallback string) string {
	if execCtx == nil || execCtx.TableMetaKey == nil {
		return fallback
	}
	return (undo.SQLUndoLog{TableMetaKey: execCtx.TableMetaKey}).QualifiedTableName(execCtx.DBType)
}

func (b *BasicUndoLogBuilder) buildRecordImagesForExec(rows driver.Rows, meta *types.TableMeta, execCtx *types.ExecContext) (*types.RecordImage, error) {
	image, err := b.buildRecordImages(rows, meta)
	if err == nil {
		image.TableMetaKey = execCtx.TableMetaKey
	}
	return image, err
}

// GetScanSlice get the column type for scann
// todo to use ColumnInfo get slice
func (*BasicUndoLogBuilder) GetScanSlice(columnNames []string, tableMeta *types.TableMeta) []driver.Value {
	scanSlice := make([]driver.Value, 0, len(columnNames))
	for _, columnNmae := range columnNames {
		var (
			scanVal interface{}
			// Gets Meta information about the column from metData
			columnMeta = tableMeta.Columns[columnNmae]
		)
		switch strings.ToUpper(columnMeta.DatabaseTypeString) {
		case "VARCHAR", "NVARCHAR", "VARCHAR2", "CHAR", "TEXT", "JSON", "TINYTEXT":
			scanVal = sql.RawBytes{}
		case "BIT", "INT", "LONGBLOB", "SMALLINT", "TINYINT", "BIGINT", "MEDIUMINT":
			if columnMeta.IsNullable == 0 {
				scanVal = int64(0)
			} else {
				scanVal = sql.NullInt64{}
			}
		case "DATE", "DATETIME", "TIME", "TIMESTAMP", "YEAR":
			scanVal = sql.NullTime{}
		case "DECIMAL", "DOUBLE", "FLOAT":
			if columnMeta.IsNullable == 0 {
				scanVal = float64(0)
			} else {
				scanVal = sql.NullFloat64{}
			}
		default:
			scanVal = sql.RawBytes{}
		}
		scanSlice = append(scanSlice, &scanVal)
	}
	return scanSlice
}

func (b *BasicUndoLogBuilder) buildSelectArgs(stmt *ast.SelectStmt, args []driver.Value) ([]driver.Value, error) {
	var (
		selectArgsIndexs = make([]int32, 0)
		selectArgs       = make([]driver.Value, 0)
	)

	if err := b.traversalArgs(stmt.Where, &selectArgsIndexs); err != nil {
		return nil, err
	}
	if stmt.OrderBy != nil {
		for _, item := range stmt.OrderBy.Items {
			if err := b.traversalArgs(item, &selectArgsIndexs); err != nil {
				return nil, err
			}
		}
	}
	if stmt.Limit != nil {
		if stmt.Limit.Offset != nil {
			if err := b.traversalArgs(stmt.Limit.Offset, &selectArgsIndexs); err != nil {
				return nil, err
			}
		}
		if stmt.Limit.Count != nil {
			if err := b.traversalArgs(stmt.Limit.Count, &selectArgsIndexs); err != nil {
				return nil, err
			}
		}
	}
	// sort selectArgs index array
	gxsort.Int32(selectArgsIndexs)
	for _, index := range selectArgsIndexs {
		if index < 0 || int(index) >= len(args) {
			return nil, fmt.Errorf("parameter index %d out of range (args count %d)", index, len(args))
		}
		selectArgs = append(selectArgs, args[index])
	}

	return selectArgs, nil
}

// todo perfect all sql operation
func (b *BasicUndoLogBuilder) traversalArgs(node ast.Node, argsIndex *[]int32) error {
	if node == nil {
		return nil
	}
	switch expr := node.(type) {
	case *ast.BinaryOperationExpr:
		if err := b.traversalArgs(expr.L, argsIndex); err != nil {
			return err
		}
		return b.traversalArgs(expr.R, argsIndex)
	case *ast.BetweenExpr:
		if err := b.traversalArgs(expr.Left, argsIndex); err != nil {
			return err
		}
		return b.traversalArgs(expr.Right, argsIndex)
	case *ast.PatternInExpr:
		for i := 0; i < len(expr.List); i++ {
			if err := b.traversalArgs(expr.List[i], argsIndex); err != nil {
				return err
			}
		}
	case *ast.ParenthesesExpr:
		return b.traversalArgs(expr.Expr, argsIndex)
	case *ast.Join:
		if err := b.traversalArgs(expr.Left, argsIndex); err != nil {
			return err
		}
		if expr.Right != nil {
			if err := b.traversalArgs(expr.Right, argsIndex); err != nil {
				return err
			}
		}
		if expr.On != nil {
			return b.traversalArgs(expr.On.Expr, argsIndex)
		}
	case *ast.UnaryOperationExpr:
		return b.traversalArgs(expr.V, argsIndex)
	case *ast.FuncCallExpr:
		for _, arg := range expr.Args {
			if err := b.traversalArgs(arg, argsIndex); err != nil {
				return err
			}
		}
	case *ast.SubqueryExpr:
		if expr.Query != nil {
			return b.traversalArgs(expr.Query, argsIndex)
		}
	case *ast.ExistsSubqueryExpr:
		if expr.Sel != nil {
			return b.traversalArgs(expr.Sel, argsIndex)
		}
	case *ast.CompareSubqueryExpr:
		if err := b.traversalArgs(expr.L, argsIndex); err != nil {
			return err
		}
		if expr.R != nil {
			return b.traversalArgs(expr.R, argsIndex)
		}
	case *ast.PatternLikeOrIlikeExpr:
		if err := b.traversalArgs(expr.Expr, argsIndex); err != nil {
			return err
		}
		return b.traversalArgs(expr.Pattern, argsIndex)
	case *ast.IsNullExpr:
		return b.traversalArgs(expr.Expr, argsIndex)
	case *ast.CaseExpr:
		if expr.Value != nil {
			if err := b.traversalArgs(expr.Value, argsIndex); err != nil {
				return err
			}
		}
		for _, whenClause := range expr.WhenClauses {
			if err := b.traversalArgs(whenClause.Expr, argsIndex); err != nil {
				return err
			}
			if err := b.traversalArgs(whenClause.Result, argsIndex); err != nil {
				return err
			}
		}
		if expr.ElseClause != nil {
			return b.traversalArgs(expr.ElseClause, argsIndex)
		}
	case ast.ParamMarkerExpr:
		order, ok := parser.GetParamMarkerOrder(expr)
		if !ok {
			return fmt.Errorf("param marker order missing for expression %v", expr)
		}
		*argsIndex = append(*argsIndex, int32(order))
	}
	return nil
}

func (b *BasicUndoLogBuilder) buildRecordImages(rowsi driver.Rows, tableMetaData *types.TableMeta) (*types.RecordImage, error) {
	// select column names
	columnNames := rowsi.Columns()
	rowImages := make([]types.RowImage, 0)
	ss := b.GetScanSlice(columnNames, tableMetaData)

	for {
		err := rowsi.Next(ss)
		if err == io.EOF {
			break
		}
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
				Value:      ss[i],
			})
		}
		rowImages = append(rowImages, types.RowImage{Columns: columns})
	}

	return &types.RecordImage{TableName: tableMetaData.TableName, Rows: rowImages}, nil
}

// buildWhereConditionByPKs build where condition by primary keys
// each pk is a condition.the result will like :" (id,userCode) in ((?,?),(?,?)) or (id,userCode) in ((?,?),(?,?) ) or (id,userCode) in ((?,?))"
func (b *BasicUndoLogBuilder) buildWhereConditionByPKs(pkNameList []string, rowSize int, _ string, maxInSize int) string {
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
			fmt.Fprintf(whereStr, "`%s`", pkNameList[i])
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

func (b *BasicUndoLogBuilder) buildPKParams(rows []types.RowImage, pkNameList []string) []driver.Value {
	params := make([]driver.Value, 0)
	for _, row := range rows {
		coumnMap := row.GetColumnMap()
		// Build a normalized map with escaped characters removed
		normalizedMap := make(map[string]*types.ColumnImage, len(coumnMap))
		for k, v := range coumnMap {
			normalizedMap[util.DelEscape(k, types.DBTypeMySQL)] = v
		}
		for _, pk := range pkNameList {
			cleanPK := util.DelEscape(pk, types.DBTypeMySQL)
			col := normalizedMap[cleanPK]
			if col != nil {
				params = append(params, col.Value)
			}
		}
	}
	return params
}

// the string as local key. the local key example(multi pk): "t_user:1_a,2_b"
func (b *BasicUndoLogBuilder) buildLockKey(records *types.RecordImage, meta types.TableMeta) string {
	return util.BuildLockKey(records, meta)
}
