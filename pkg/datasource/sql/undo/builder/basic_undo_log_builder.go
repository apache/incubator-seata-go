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
	"database/sql"
	"database/sql/driver"

	"github.com/arana-db/parser/ast"
	"github.com/arana-db/parser/test_driver"
	gxsort "github.com/dubbogo/gost/sort"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

type BasicUndoLogBuilder struct {
}

// getScanSlice get the column type for scann
func (*BasicUndoLogBuilder) getScanSlice(columnNames []string, tableMeta types.TableMeta) []driver.Value {
	scanSlice := make([]driver.Value, 0, len(columnNames))
	for _, columnNmae := range columnNames {
		var (
			scanVal interface{}
			// 从metData获取该列的元信息
			columnMeta = tableMeta.Columns[columnNmae]
		)

		switch columnMeta.Info.ScanType() {
		case types.ScanTypeFloat32:
			scanVal = float32(0)
			break
		case types.ScanTypeFloat64:
			scanVal = float64(0)
			break
		case types.ScanTypeInt8:
			scanVal = int8(0)
			break
		case types.ScanTypeInt16:
			scanVal = int16(0)
			break
		case types.ScanTypeInt32:
			scanVal = int32(0)
			break
		case types.ScanTypeInt64:
			scanVal = int64(0)
			break
		case types.ScanTypeNullFloat:
			scanVal = sql.NullFloat64{}
			break
		case types.ScanTypeNullInt:
			scanVal = sql.NullInt64{}
			break
		case types.ScanTypeNullTime:
			scanVal = sql.NullTime{}
			break
		case types.ScanTypeUint8:
			scanVal = uint8(0)
			break
		case types.ScanTypeUint16:
			scanVal = uint16(0)
			break
		case types.ScanTypeUint32:
			scanVal = uint32(0)
			break
		case types.ScanTypeUint64:
			scanVal = uint64(0)
			break
		case types.ScanTypeRawBytes:
			scanVal = sql.RawBytes{}
			break
		case types.ScanTypeUnknown:
			scanVal = new(interface{})
			break
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
		b.traversalArgs(node.(*ast.BinaryOperationExpr).L, argsIndex)
		b.traversalArgs(node.(*ast.BinaryOperationExpr).R, argsIndex)
		break
	case *test_driver.ParamMarkerExpr:
		*argsIndex = append(*argsIndex, int32(node.(*test_driver.ParamMarkerExpr).Order))
		break
	case *ast.BetweenExpr:
		b.traversalArgs(node.(*ast.BetweenExpr).Left, argsIndex)
		b.traversalArgs(node.(*ast.BetweenExpr).Right, argsIndex)
		break
	case *ast.PatternInExpr:
		for i := 0; i < len(node.(*ast.PatternInExpr).List); i++ {
			b.traversalArgs(node.(*ast.PatternInExpr).List[i], argsIndex)
		}
		break
	}
}
