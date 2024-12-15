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
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/arana-db/parser/ast"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo/executor"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	SqlPlaceholder = "?"
)

type MySQLInsertUndoLogBuilder struct {
	BasicUndoLogBuilder
	// InsertResult after insert sql
	InsertResult  types.ExecResult
	IncrementStep int
}

func GetMySQLInsertUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLInsertUndoLogBuilder{
		BasicUndoLogBuilder: BasicUndoLogBuilder{},
	}
}

func (u *MySQLInsertUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	return []*types.RecordImage{}, nil
}

func (u *MySQLInsertUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	if execCtx == nil || execCtx.ParseContext == nil || execCtx.ParseContext.InsertStmt == nil {
		return nil, nil
	}

	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
	selectSQL, selectArgs, err := u.buildAfterImageSQL(ctx, execCtx)
	if err != nil {
		return nil, err
	}

	stmt, err := execCtx.Conn.Prepare(selectSQL)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}

	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}

	return []*types.RecordImage{image}, nil
}

// buildAfterImageSQL build select sql from insert sql
func (u *MySQLInsertUndoLogBuilder) buildAfterImageSQL(ctx context.Context, execCtx *types.ExecContext) (string, []driver.Value, error) {
	// get all pk value
	if execCtx == nil || execCtx.ParseContext == nil || execCtx.ParseContext.InsertStmt == nil {
		return "", nil, fmt.Errorf("can't found execCtx or ParseContext or InsertStmt")
	}
	parseCtx := execCtx.ParseContext
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	if execCtx.MetaDataMap == nil {
		return "", nil, fmt.Errorf("can't found  MetaDataMap")
	}
	meta := execCtx.MetaDataMap[tableName]
	pkValuesMap, err := u.getPkValues(execCtx, parseCtx, meta)
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
	whereSQL := u.buildWhereConditionByPKs(pkColumnNameList, len(pkValuesMap[pkColumnNameList[0]]), "mysql", maxInSize)
	sb.WriteString(" WHERE " + whereSQL + " ")
	return sb.String(), u.buildPKParams(pkRowImages, pkColumnNameList), nil
}

func (u *MySQLInsertUndoLogBuilder) getPkValues(execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error) {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	pkValuesMap := make(map[string][]interface{})
	var err error
	//when there is only one pk in the table
	if len(pkColumnNameList) == 1 {
		if u.containsPK(meta, parseCtx) {
			// the insert sql contain pk value
			pkValuesMap, err = u.getPkValuesByColumn(execCtx)
			if err != nil {
				return nil, err
			}
		} else if containsColumns(parseCtx) {
			// the insert table pk auto generated
			pkValuesMap, err = u.getPkValuesByAuto(execCtx)
			if err != nil {
				return nil, err
			}
		} else {
			pkValuesMap, err = u.getPkValuesByColumn(execCtx)
			if err != nil {
				return nil, err
			}
		}
	} else {
		//when there is multiple pk in the table
		//1,all pk columns are filled value.
		//2,the auto increment pk column value is null, and other pk value are not null.
		pkValuesMap, err = u.getPkValuesByColumn(execCtx)
		if err != nil {
			return nil, err
		}
		for _, columnName := range pkColumnNameList {
			if _, ok := pkValuesMap[columnName]; !ok {
				curPkValuesMap, err := u.getPkValuesByAuto(execCtx)
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
func (u *MySQLInsertUndoLogBuilder) containsPK(meta types.TableMeta, parseCtx *types.ParseContext) bool {
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
func (u *MySQLInsertUndoLogBuilder) containPK(columnName string, meta types.TableMeta) bool {
	newColumnName := executor.DelEscape(columnName, types.DBTypeMySQL)
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
func (u *MySQLInsertUndoLogBuilder) getPkIndex(InsertStmt *ast.InsertStmt, meta types.TableMeta) map[string]int {
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
			if u.containPK(sqlColumnName, meta) {
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
		if u.containPK(tmpColumnMeta.ColumnName, meta) {
			pkIndexMap[executor.DelEscape(tmpColumnMeta.ColumnName, types.DBTypeMySQL)] = pkIndex
		}
	}

	return pkIndexMap
}

// parsePkValuesFromStatement parse primary key value from statement.
// return the primary key and values<key:primary key,value:primary key values></key:primary>
func (u *MySQLInsertUndoLogBuilder) parsePkValuesFromStatement(insertStmt *ast.InsertStmt, meta types.TableMeta, nameValues []driver.NamedValue) (map[string][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	pkIndexMap := u.getPkIndex(insertStmt, meta)
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
		//use prepared statements
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
				if ok && strings.EqualFold(rStr, SqlPlaceholder) {
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
				if ok && strings.EqualFold(pkValueStr, SqlPlaceholder) {
					currentRowNotPlaceholderNumBeforePkIndex := 0
					for i := range row {
						r := row[i]
						rStr, ok := r.(string)
						if i < pkIndex && ok && !strings.EqualFold(rStr, SqlPlaceholder) {
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
func (u *MySQLInsertUndoLogBuilder) getPkValuesByColumn(execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if execCtx == nil || execCtx.ParseContext == nil || execCtx.ParseContext.InsertStmt == nil {
		return nil, nil
	}
	parseCtx := execCtx.ParseContext
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	meta := execCtx.MetaDataMap[tableName]
	pkValuesMap, err := u.parsePkValuesFromStatement(parseCtx.InsertStmt, meta, execCtx.NamedValues)
	if err != nil {
		return nil, err
	}

	// generate pkValue by auto increment
	for _, v := range pkValuesMap {
		tmpV := v
		if len(tmpV) == 1 {
			// pk auto generated while single insert primary key is expression
			if _, ok := tmpV[0].(*ast.FuncCallExpr); ok {
				curPkValueMap, err := u.getPkValuesByAuto(execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValueMap)
			}
		} else if len(tmpV) > 0 && tmpV[0] == nil {
			// pk auto generated while column exists and value is null
			curPkValueMap, err := u.getPkValuesByAuto(execCtx)
			if err != nil {
				return nil, err
			}
			pkValuesMapMerge(&pkValuesMap, curPkValueMap)
		}
	}
	return pkValuesMap, nil
}

func (u *MySQLInsertUndoLogBuilder) getPkValuesByAuto(execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if execCtx == nil || execCtx.ParseContext == nil || execCtx.ParseContext.InsertStmt == nil {
		return nil, nil
	}
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
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

	updateCount, err := u.InsertResult.GetResult().RowsAffected()
	if err != nil {
		return nil, err
	}

	lastInsertId, err := u.InsertResult.GetResult().LastInsertId()
	if err != nil {
		return nil, err
	}

	// If there is batch insert
	// do auto increment base LAST_INSERT_ID and variable `auto_increment_increment`
	if lastInsertId > 0 && updateCount > 1 && canAutoIncrement(pkMetaMap) {
		return u.autoGeneratePks(execCtx, autoColumnName, lastInsertId, updateCount)
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

func (u *MySQLInsertUndoLogBuilder) autoGeneratePks(execCtx *types.ExecContext, autoColumnName string, lastInsetId, updateCount int64) (map[string][]interface{}, error) {
	var step int64
	if u.IncrementStep > 0 {
		step = int64(u.IncrementStep)
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
	var i int64
	for i = 0; i < updateCount; i++ {
		pkValues = append(pkValues, lastInsetId+i*step)
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
				row = append(row, SqlPlaceholder)
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

func (u *MySQLInsertUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.InsertExecutor
}
