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
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/test_driver"
	gxsort "github.com/dubbogo/gost/sort"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

// todo the executor should be stateful
type BasicUndoLogBuilder struct{}

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

func (b *BasicUndoLogBuilder) buildSelectArgs(stmt *ast.SelectStmt, args []driver.Value) []driver.Value {
	var (
		selectArgsIndexs = make([]int32, 0)
		selectArgs       = make([]driver.Value, 0)
	)

	b.traversalArgs(stmt.Where, &selectArgsIndexs)
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
func (b *BasicUndoLogBuilder) traversalArgs(node ast.Node, argsIndex *[]int32) {
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
	case *ast.ParenthesesExpr:
		expr := node.(*ast.ParenthesesExpr)
		b.traversalArgs(expr.Expr, argsIndex)
		break
	case *test_driver.ParamMarkerExpr:
		*argsIndex = append(*argsIndex, int32(node.(*test_driver.ParamMarkerExpr).Order))
		break
	}
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
func (b *BasicUndoLogBuilder) buildWhereConditionByPKs(pkNameList []string, rowSize int, dbType string, maxInSize int) string {
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

func (b *BasicUndoLogBuilder) buildPKParams(rows []types.RowImage, pkNameList []string) []driver.Value {
	params := make([]driver.Value, 0)
	for _, row := range rows {
		coumnMap := row.GetColumnMap()
		for _, pk := range pkNameList {
			col := coumnMap[pk]
			if col != nil {
				params = append(params, col.Value)
			}
		}
	}
	return params
}

// the string as local key. the local key example(multi pk): "t_user:1_a,2_b"
func (b *BasicUndoLogBuilder) buildLockKey(rows driver.Rows, meta types.TableMeta) string {
	var (
		lockKeys      bytes.Buffer
		filedSequence int
	)
	lockKeys.WriteString(meta.TableName)
	lockKeys.WriteString(":")

	pks := b.GetScanSlice(meta.GetPrimaryKeyOnlyName(), &meta)
	for {
		err := rows.Next(pks)
		if err == io.EOF {
			break
		}

		if filedSequence > 0 {
			lockKeys.WriteString(",")
		}

		pkSplitIndex := 0
		for _, value := range pks {
			if pkSplitIndex > 0 {
				lockKeys.WriteString("_")
			}
			lockKeys.WriteString(fmt.Sprintf("%v", value))
			pkSplitIndex++
		}
		filedSequence++
	}
	return lockKeys.String()
}

// the string as local key. the local key example(multi pk): "t_user:1_a,2_b"
func (b *BasicUndoLogBuilder) buildLockKey2(records *types.RecordImage, meta types.TableMeta) string {
	var lockKeys bytes.Buffer
	lockKeys.WriteString(meta.TableName)
	lockKeys.WriteString(":")

	keys := meta.GetPrimaryKeyOnlyName()
	keyIndexMap := make(map[string]int, len(keys))

	for idx, columnName := range keys {
		keyIndexMap[columnName] = idx
	}

	primaryKeyRows := make([][]interface{}, len(records.Rows))

	for i, row := range records.Rows {
		primaryKeyValues := make([]interface{}, len(keys))
		for _, column := range row.Columns {
			if idx, exist := keyIndexMap[column.ColumnName]; exist {
				primaryKeyValues[idx] = column.Value
			}
		}
		primaryKeyRows[i] = primaryKeyValues
	}

	for i, primaryKeyValues := range primaryKeyRows {
		if i > 0 {
			lockKeys.WriteString(",")
		}
		for j, pkVal := range primaryKeyValues {
			if j > 0 {
				lockKeys.WriteString("_")
			}
			if pkVal == nil {
				continue
			}
			lockKeys.WriteString(fmt.Sprintf("%v", pkVal))
		}
	}

	return lockKeys.String()
}
