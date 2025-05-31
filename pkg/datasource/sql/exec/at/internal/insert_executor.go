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

package internal

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/arana-db/parser/ast"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

const (
	sqlPlaceholder = "?"
)

type InsertExecutorDialect interface {
	GetPkValues(ctx context.Context, execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error)
	GetPkValuesByColumn(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error)
}

// InsertExecutor execute insert SQL
type InsertExecutor struct {
	BaseExecutor
	InsertExecutorDialect
	IncrementStep int
	// BusinesSQLResult after insert sql
	BusinesSQLResult types.ExecResult
}

// NewInsertExecutor get insert Executor
func NewInsertExecutor(parserCtx *types.ParseContext, execContext *types.ExecContext, hooks []exec.SQLHook, dialect InsertExecutorDialect) *InsertExecutor {
	return &InsertExecutor{
		BaseExecutor:          BaseExecutor{Hooks: hooks, ParserCtx: parserCtx, ExecCtx: execContext},
		InsertExecutorDialect: dialect,
	}
}

func (i *InsertExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	i.beforeHooks(ctx, i.ExecCtx)
	defer func() {
		i.afterHooks(ctx, i.ExecCtx)
	}()

	beforeImage, err := i.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, i.ExecCtx.Query, i.ExecCtx.NamedValues)
	if err != nil {
		return nil, err
	}

	if i.BusinesSQLResult == nil {
		i.BusinesSQLResult = res
	}

	afterImage, err := i.afterImage(ctx)
	if err != nil {
		return nil, err
	}

	i.ExecCtx.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	i.ExecCtx.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

// beforeImage build before image
func (i *InsertExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	metaData, err := i.GetMetaData(ctx)
	if err != nil {
		return nil, err
	}
	return types.NewEmptyRecordImage(metaData, types.SQLTypeInsert), nil
}

// afterImage build after image
func (i *InsertExecutor) afterImage(ctx context.Context) (*types.RecordImage, error) {
	if !i.IsAstStmtValid() {
		return nil, nil
	}

	metaData, err := i.GetMetaData(ctx)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs, err := i.BuildAfterImageSQL(ctx)
	if err != nil {
		return nil, err
	}

	rowsi, err := i.ExecSelectSQL(ctx, selectSQL, selectArgs)
	if err != nil {
		return nil, err
	}

	image, err := i.buildRecordImages(rowsi, metaData, types.SQLTypeInsert)
	if err != nil {
		return nil, err
	}

	lockKey := i.buildLockKey(image, *metaData)
	i.ExecCtx.TxCtx.LockKeys[lockKey] = struct{}{}
	return image, nil
}

// BuildAfterImageSQL build select sql from insert sql
func (i *InsertExecutor) BuildAfterImageSQL(ctx context.Context) (string, []driver.NamedValue, error) {
	// get all pk value
	tableName, _ := i.ParserCtx.GetTableName()

	meta, err := i.GetMetaData(ctx)
	if err != nil {
		return "", nil, err
	}
	pkValuesMap, err := i.GetPkValues(ctx, i.ExecCtx, i.ParserCtx, *meta)
	if err != nil {
		return "", nil, err
	}

	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return "", nil, fmt.Errorf("Pk columnName size is zero")
	}

	dataTypeMap, err := meta.GetPrimaryKeyTypeStrMap()
	if err != nil {
		return "", nil, err
	}
	if len(dataTypeMap) != len(pkColumnNameList) {
		return "", nil, fmt.Errorf("PK columnName size don't equal PK DataType size")
	}
	var pkRowImages []types.RowImage

	rowSize := len(pkValuesMap[pkColumnNameList[0]])
	for i := 0; i < rowSize; i++ {
		for _, name := range pkColumnNameList {
			tmpKey := name
			tmpArray := pkValuesMap[tmpKey]
			pkRowImages = append(pkRowImages, types.RowImage{
				Columns: []types.ColumnImage{{
					KeyType:    types.IndexTypePrimaryKey,
					ColumnName: tmpKey,
					ColumnType: types.MySQLStrToJavaType(dataTypeMap[tmpKey]),
					Value:      tmpArray[i],
				}},
			})
		}
	}
	// build check sql
	sb := strings.Builder{}
	suffix := strings.Builder{}
	var insertColumns []string

	for _, column := range i.ParserCtx.InsertStmt.Columns {
		insertColumns = append(insertColumns, column.Name.O)
	}
	sb.WriteString("SELECT " + strings.Join(i.getNeedColumns(meta, insertColumns, i.DBType()), ", "))
	suffix.WriteString(" FROM " + tableName)
	whereSQL := i.buildWhereConditionByPKs(pkColumnNameList, rowSize, maxInSize)
	suffix.WriteString(" WHERE " + whereSQL + " ")
	sb.WriteString(suffix.String())
	return sb.String(), i.buildPKParams(pkRowImages, pkColumnNameList), nil
}

// ContainsPK the columns contains table meta pk
func (i *InsertExecutor) ContainsPK(meta types.TableMeta, parseCtx *types.ParseContext) bool {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}
	if parseCtx == nil || parseCtx.InsertStmt == nil || parseCtx.InsertStmt.Columns == nil {
		return false
	}
	if len(parseCtx.InsertStmt.Columns) == 0 {
		return false
	}

	matchCounter := 0
	for _, column := range parseCtx.InsertStmt.Columns {
		for _, pkName := range pkColumnNameList {
			if strings.EqualFold(pkName, column.Name.O) ||
				strings.EqualFold(pkName, column.Name.L) {
				matchCounter++
			}
		}
	}

	return matchCounter == len(pkColumnNameList)
}

// ContainPK compare column name and primary key name
func (i *InsertExecutor) ContainPK(columnName string, meta types.TableMeta) bool {
	newColumnName := DelEscape(columnName, i.DBType())
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}
	for _, name := range pkColumnNameList {
		if strings.EqualFold(name, newColumnName) {
			return true
		}
	}
	return false
}

// GetPkIndex get pk index
// return the key is pk column name and the value is index of the pk column
func (i *InsertExecutor) GetPkIndex(InsertStmt *ast.InsertStmt, meta types.TableMeta) map[string]int {
	pkIndexMap := make(map[string]int)
	if InsertStmt == nil {
		return pkIndexMap
	}
	insertColumnsSize := len(InsertStmt.Columns)
	if insertColumnsSize == 0 {
		return pkIndexMap
	}
	if meta.ColumnNames == nil {
		return pkIndexMap
	}
	if len(meta.Columns) > 0 {
		for paramIdx := 0; paramIdx < insertColumnsSize; paramIdx++ {
			sqlColumnName := InsertStmt.Columns[paramIdx].Name.O
			if i.ContainPK(sqlColumnName, meta) {
				pkIndexMap[sqlColumnName] = paramIdx
			}
		}
		return pkIndexMap
	}

	pkIndex := -1
	allColumns := meta.Columns
	for _, columnMeta := range allColumns {
		tmpColumnMeta := columnMeta
		pkIndex++
		if i.ContainPK(tmpColumnMeta.ColumnName, meta) {
			pkIndexMap[DelEscape(tmpColumnMeta.ColumnName, i.DBType())] = pkIndex
		}
	}

	return pkIndexMap
}

// ParsePkValuesFromStatement parse primary key value from statement.
// return the primary key and values<key:primary key,value:primary key values></key:primary>
func (i *InsertExecutor) ParsePkValuesFromStatement(insertStmt *ast.InsertStmt, meta types.TableMeta, nameValues []driver.NamedValue) (map[string][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	pkIndexMap := i.GetPkIndex(insertStmt, meta)
	if pkIndexMap == nil || len(pkIndexMap) == 0 {
		return nil, fmt.Errorf("pkIndex is not found")
	}
	var pkIndexArray []int
	for _, val := range pkIndexMap {
		tmpVal := val
		pkIndexArray = append(pkIndexArray, tmpVal)
	}

	if insertStmt == nil || len(insertStmt.Lists) == 0 {
		return nil, fmt.Errorf("parCtx is nil, perhaps InsertStmt is empty")
	}

	pkValuesMap := make(map[string][]interface{})

	if nameValues != nil && len(nameValues) > 0 {
		// use prepared statements
		insertRows, err := getInsertRows(insertStmt, pkIndexArray)
		if err != nil {
			return nil, err
		}
		if insertRows == nil || len(insertRows) == 0 {
			return nil, err
		}
		totalPlaceholderNum := -1
		for _, row := range insertRows {
			if len(row) == 0 {
				continue
			}
			currentRowPlaceholderNum := -1
			for _, r := range row {
				rStr, ok := r.(string)
				if ok && strings.EqualFold(rStr, sqlPlaceholder) {
					totalPlaceholderNum += 1
					currentRowPlaceholderNum += 1
				}
			}
			var pkKey string
			var pkIndex int
			var pkValues []interface{}
			for key, index := range pkIndexMap {
				curKey := key
				curIndex := index

				pkKey = curKey
				pkValues = pkValuesMap[pkKey]

				pkIndex = curIndex
				if pkIndex > len(row)-1 {
					continue
				}
				pkValue := row[pkIndex]
				pkValueStr, ok := pkValue.(string)
				if ok && strings.EqualFold(pkValueStr, sqlPlaceholder) {
					currentRowNotPlaceholderNumBeforePkIndex := 0
					for i := range row {
						r := row[i]
						rStr, ok := r.(string)
						if i < pkIndex && ok && !strings.EqualFold(rStr, sqlPlaceholder) {
							currentRowNotPlaceholderNumBeforePkIndex++
						}
					}
					idx := totalPlaceholderNum - currentRowPlaceholderNum + pkIndex - currentRowNotPlaceholderNumBeforePkIndex
					pkValues = append(pkValues, nameValues[idx].Value)
				} else {
					pkValues = append(pkValues, pkValue)
				}
				if _, ok := pkValuesMap[pkKey]; !ok {
					pkValuesMap[pkKey] = pkValues
				}
			}
		}
	} else {
		for _, list := range insertStmt.Lists {
			for pkName, pkIndex := range pkIndexMap {
				tmpPkName := pkName
				tmpPkIndex := pkIndex
				if tmpPkIndex >= len(list) {
					return nil, fmt.Errorf("pkIndex out of range")
				}
				if node, ok := list[tmpPkIndex].(ast.ValueExpr); ok {
					pkValuesMap[tmpPkName] = append(pkValuesMap[tmpPkName], node.GetValue())
				}
			}
		}
	}

	return pkValuesMap, nil
}

func CanAutoIncrement(pkMetaMap map[string]types.ColumnMeta) bool {
	if len(pkMetaMap) != 1 {
		return false
	}
	for _, meta := range pkMetaMap {
		return meta.Autoincrement
	}
	return false
}

func (i *InsertExecutor) IsAstStmtValid() bool {
	return i.ParserCtx != nil && i.ParserCtx.InsertStmt != nil
}

// ContainsColumns judge sql specify column
func ContainsColumns(parseCtx *types.ParseContext) bool {
	if parseCtx == nil || parseCtx.InsertStmt == nil || parseCtx.InsertStmt.Columns == nil {
		return false
	}
	return len(parseCtx.InsertStmt.Columns) > 0
}

func getInsertRows(insertStmt *ast.InsertStmt, pkIndexArray []int) ([][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	if len(insertStmt.Lists) == 0 {
		return nil, nil
	}
	var rows [][]interface{}

	for _, nodes := range insertStmt.Lists {
		var row []interface{}
		for i, node := range nodes {
			if _, ok := node.(ast.ParamMarkerExpr); ok {
				row = append(row, sqlPlaceholder)
			} else if newNode, ok := node.(ast.ValueExpr); ok {
				row = append(row, newNode.GetValue())
			} else if newNode, ok := node.(*ast.VariableExpr); ok {
				row = append(row, newNode.Name)
			} else if _, ok := node.(*ast.FuncCallExpr); ok {
				row = append(row, ast.FuncCallExpr{})
			} else {
				for _, index := range pkIndexArray {
					if index == i {
						return nil, fmt.Errorf("Unknown SQLExpr:%v", node)
					}
				}
				row = append(row, ast.DefaultExpr{})
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}
