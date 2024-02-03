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

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	sqlPlaceholder = "?"
)

// insertExecutor execute insert SQL
type insertExecutor struct {
	baseExecutor
	parserCtx     *types.ParseContext
	execContext   *types.ExecContext
	incrementStep int
	// businesSQLResult after insert sql
	businesSQLResult types.ExecResult
}

// NewInsertExecutor get insert executor
func NewInsertExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &insertExecutor{parserCtx: parserCtx, execContext: execContent, baseExecutor: baseExecutor{hooks: hooks}}
}

func (i *insertExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	i.beforeHooks(ctx, i.execContext)
	defer func() {
		i.afterHooks(ctx, i.execContext)
	}()

	beforeImage, err := i.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, i.execContext.Query, i.execContext.NamedValues)
	if err != nil {
		return nil, err
	}

	if i.businesSQLResult == nil {
		i.businesSQLResult = res
	}

	afterImage, err := i.afterImage(ctx)
	if err != nil {
		return nil, err
	}

	i.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	i.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

// beforeImage build before image
func (i *insertExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	return types.NewEmptyRecordImage(metaData, types.SQLTypeInsert), nil
}

// afterImage build after image
func (i *insertExecutor) afterImage(ctx context.Context) (*types.RecordImage, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs, err := i.buildAfterImageSQL(ctx)
	if err != nil {
		return nil, err
	}

	var rowsi driver.Rows
	queryerCtx, ok := i.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	if !ok {
		queryer, ok = i.execContext.Conn.(driver.Queryer)
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

	image, err := i.buildRecordImages(rowsi, metaData, types.SQLTypeInsert)
	if err != nil {
		return nil, err
	}

	lockKey := i.buildLockKey(image, *metaData)
	i.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	return image, nil
}

// buildAfterImageSQL build select sql from insert sql
func (i *insertExecutor) buildAfterImageSQL(ctx context.Context) (string, []driver.NamedValue, error) {
	// get all pk value
	tableName, _ := i.parserCtx.GetTableName()

	meta, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return "", nil, err
	}
	pkValuesMap, err := i.getPkValues(ctx, i.execContext, i.parserCtx, *meta)
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
	sb.WriteString("SELECT * FROM " + tableName)
	whereSQL := i.buildWhereConditionByPKs(pkColumnNameList, len(pkValuesMap[pkColumnNameList[0]]), "mysql", maxInSize)
	sb.WriteString(" WHERE " + whereSQL + " ")
	return sb.String(), i.buildPKParams(pkRowImages, pkColumnNameList), nil
}

func (i *insertExecutor) getPkValues(ctx context.Context, execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error) {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	pkValuesMap := make(map[string][]interface{})
	var err error
	// when there is only one pk in the table
	if len(pkColumnNameList) == 1 {
		if i.containsPK(meta, parseCtx) {
			// the insert sql contain pk value
			pkValuesMap, err = i.getPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else if containsColumns(parseCtx) {
			// the insert table pk auto generated
			pkValuesMap, err = i.getPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else {
			pkValuesMap, err = i.getPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// when there is multiple pk in the table
		// 1,all pk columns are filled value.
		// 2,the auto increment pk column value is null, and other pk value are not null.
		pkValuesMap, err = i.getPkValuesByColumn(ctx, execCtx)
		if err != nil {
			return nil, err
		}
		for _, columnName := range pkColumnNameList {
			if _, ok := pkValuesMap[columnName]; !ok {
				curPkValuesMap, err := i.getPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValuesMap)
			}
		}
	}
	return pkValuesMap, nil
}

// containsPK the columns contains table meta pk
func (i *insertExecutor) containsPK(meta types.TableMeta, parseCtx *types.ParseContext) bool {
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

// containPK compare column name and primary key name
func (i *insertExecutor) containPK(columnName string, meta types.TableMeta) bool {
	newColumnName := DelEscape(columnName, types.DBTypeMySQL)
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

// getPkIndex get pk index
// return the key is pk column name and the value is index of the pk column
func (i *insertExecutor) getPkIndex(InsertStmt *ast.InsertStmt, meta types.TableMeta) map[string]int {
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
			if i.containPK(sqlColumnName, meta) {
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
		if i.containPK(tmpColumnMeta.ColumnName, meta) {
			pkIndexMap[DelEscape(tmpColumnMeta.ColumnName, types.DBTypeMySQL)] = pkIndex
		}
	}

	return pkIndexMap
}

// parsePkValuesFromStatement parse primary key value from statement.
// return the primary key and values<key:primary key,value:primary key values></key:primary>
func (i *insertExecutor) parsePkValuesFromStatement(insertStmt *ast.InsertStmt, meta types.TableMeta, nameValues []driver.NamedValue) (map[string][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	pkIndexMap := i.getPkIndex(insertStmt, meta)
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

// getPkValuesByColumn get pk value by column.
func (i *insertExecutor) getPkValuesByColumn(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	tableName, _ := i.parserCtx.GetTableName()
	meta, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	pkValuesMap, err := i.parsePkValuesFromStatement(i.parserCtx.InsertStmt, *meta, execCtx.NamedValues)
	if err != nil {
		return nil, err
	}

	// generate pkValue by auto increment
	for _, v := range pkValuesMap {
		tmpV := v
		if len(tmpV) == 1 {
			// pk auto generated while single insert primary key is expression
			if _, ok := tmpV[0].(*ast.FuncCallExpr); ok {
				curPkValueMap, err := i.getPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValueMap)
			}
		} else if len(tmpV) > 0 && tmpV[0] == nil {
			// pk auto generated while column exists and value is null
			curPkValueMap, err := i.getPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
			pkValuesMapMerge(&pkValuesMap, curPkValueMap)
		}
	}
	return pkValuesMap, nil
}

func (i *insertExecutor) getPkValuesByAuto(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !i.isAstStmtValid() {
		return nil, nil
	}

	tableName, _ := i.parserCtx.GetTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}

	pkValuesMap := make(map[string][]interface{})
	pkMetaMap := metaData.GetPrimaryKeyMap()
	if len(pkMetaMap) == 0 {
		return nil, fmt.Errorf("pk map is empty")
	}
	var autoColumnName string
	for _, columnMeta := range pkMetaMap {
		tmpColumnMeta := columnMeta
		if tmpColumnMeta.Autoincrement {
			autoColumnName = tmpColumnMeta.ColumnName
			break
		}
	}
	if len(autoColumnName) == 0 {
		return nil, fmt.Errorf("auto increment column not exist")
	}

	updateCount, err := i.businesSQLResult.GetResult().RowsAffected()
	if err != nil {
		return nil, err
	}

	lastInsertId, err := i.businesSQLResult.GetResult().LastInsertId()
	if err != nil {
		return nil, err
	}

	// If there is batch insert
	// do auto increment base LAST_INSERT_ID and variable `auto_increment_increment`
	if lastInsertId > 0 && updateCount > 1 && canAutoIncrement(pkMetaMap) {
		return i.autoGeneratePks(execCtx, autoColumnName, lastInsertId, updateCount)
	}

	if lastInsertId > 0 {
		var pkValues []interface{}
		pkValues = append(pkValues, lastInsertId)
		pkValuesMap[autoColumnName] = pkValues
		return pkValuesMap, nil
	}

	return nil, nil
}

func canAutoIncrement(pkMetaMap map[string]types.ColumnMeta) bool {
	if len(pkMetaMap) != 1 {
		return false
	}
	for _, meta := range pkMetaMap {
		return meta.Autoincrement
	}
	return false
}

func (i *insertExecutor) isAstStmtValid() bool {
	return i.parserCtx != nil && i.parserCtx.InsertStmt != nil
}

func (i *insertExecutor) autoGeneratePks(execCtx *types.ExecContext, autoColumnName string, lastInsetId, updateCount int64) (map[string][]interface{}, error) {
	var step int64
	if i.incrementStep > 0 {
		step = int64(i.incrementStep)
	} else {
		// get step by query sql
		stmt, err := execCtx.Conn.Prepare("SHOW VARIABLES LIKE 'auto_increment_increment'")
		if err != nil {
			log.Errorf("build prepare stmt: %+v", err)
			return nil, err
		}

		rows, err := stmt.Query(nil)
		if err != nil {
			log.Errorf("stmt query: %+v", err)
			return nil, err
		}

		if len(rows.Columns()) > 0 {
			var curStep []driver.Value
			if err := rows.Next(curStep); err != nil {
				return nil, err
			}

			if curStepInt, ok := curStep[0].(int64); ok {
				step = curStepInt
			}
		} else {
			return nil, fmt.Errorf("query is empty")
		}
	}

	if step == 0 {
		return nil, fmt.Errorf("get increment step error")
	}

	var pkValues []interface{}
	for j := int64(0); j < updateCount; j++ {
		pkValues = append(pkValues, lastInsetId+j*step)
	}
	pkValuesMap := make(map[string][]interface{})
	pkValuesMap[autoColumnName] = pkValues
	return pkValuesMap, nil
}

func pkValuesMapMerge(dest *map[string][]interface{}, src map[string][]interface{}) {
	for k, v := range src {
		tmpK := k
		tmpV := v
		(*dest)[tmpK] = append((*dest)[tmpK], tmpV)
	}
}

// containsColumns judge sql specify column
func containsColumns(parseCtx *types.ParseContext) bool {
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
