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
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/datasource/sql/undo/executor"
	"strings"

	"github.com/arana-db/parser/ast"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
	"github.com/seata/seata-go/pkg/util/log"
)

func init() {
	undo.RegistrUndoLogBuilder(types.InsertExecutor, GetMySQLInsertUndoLogBuilder)
}

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
	selectSQL, selectArgs := u.buildAfterImageSQL(ctx, execCtx)

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

	image, err := u.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}

	return []*types.RecordImage{image}, nil
}

// buildAfterImageSQL build select sql from insert sql
func (u *MySQLInsertUndoLogBuilder) buildAfterImageSQL(ctx context.Context, execCtx *types.ExecContext) (string, []driver.Value) {
	// get all pk value
	if execCtx == nil || execCtx.ParseContext == nil || execCtx.ParseContext.InsertStmt == nil {
		return "", nil
	}
	parseCtx := execCtx.ParseContext
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	if execCtx.MetaDataMap == nil {
		return "", nil
	}
	meta := execCtx.MetaDataMap[tableName]
	pkValuesMap, err := u.getPkValues(execCtx, parseCtx, meta)
	if err != nil {
		return "", nil
	}

	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	// build check sql
	sb := strings.Builder{}
	sb.WriteString("SELECT * FROM " + tableName)
	whereSQL := u.buildWhereConditionByPKs(pkColumnNameList, len(pkColumnNameList), "mysql", maxInSize)
	sb.WriteString(" " + whereSQL + " ")

	dataType, err := meta.GetPrimaryKeyType()
	if err != nil {
		return "", nil
	}
	pkRowImages := make([]types.RowImage, 0)
	for key, array := range pkValuesMap {
		for _, obj := range array {
			pkRowImages = append(pkRowImages, types.RowImage{
				Columns: []types.ColumnImage{{
					KeyType: types.IndexTypePrimaryKey,
					Name:    key,
					Type:    int16(dataType),
					Value:   obj,
				}},
			})
		}
	}
	return sb.String(), u.buildPKParams(pkRowImages, meta.GetPrimaryKeyOnlyName())
}

func (u *MySQLInsertUndoLogBuilder) getPkValues(execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error) {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	pkValuesMap := make(map[string][]interface{})
	var err error
	//when there is only one pk in the table
	if len(pkColumnNameList) == 1 {
		if u.containsPK(meta, parseCtx) {
			// the insert sql contain pk value
			pkValuesMap, err = u.getPkValuesByColumn(execCtx, pkColumnNameList[0])
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
			pkValuesMap, err = u.getPkValuesByColumn(execCtx, pkColumnNameList[0])
			if err != nil {
				return nil, err
			}
		}
	} else {
		//when there is multiple pk in the table
		//1,all pk columns are filled value.
		//2,the auto increment pk column value is null, and other pk value are not null.
		pkValuesMap, err = u.getPkValuesByColumn(execCtx, pkColumnNameList[0])
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

	pkColumnNameListStr := strings.Join(pkColumnNameList, ",") + ","

	//todo 这个方式并不快 改进一下
	for _, column := range parseCtx.InsertStmt.Columns {
		if strings.Contains(pkColumnNameListStr, column.Name.O) ||
			strings.Contains(pkColumnNameListStr, column.Name.O) {
			return true
		}
	}

	return false
}

// containPK compare column name and primary key name
func (u *MySQLInsertUndoLogBuilder) containPK(columnName string, meta types.TableMeta) bool {
	newColumnName := executor.DelEscape(columnName, types.DBTypeMySQL)
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	if len(pkColumnNameList) == 0 {
		return false
	}
	for _, name := range pkColumnNameList {
		if strings.ToUpper(name) == strings.ToUpper(newColumnName) {
			return true
		}
	}
	return false
}

// getPkIndex get pk index
// return the key is pk column name and the value is index of the pk column
func (u *MySQLInsertUndoLogBuilder) getPkIndex(columnName string, parseCtx *types.ParseContext, meta types.TableMeta) map[string]int {
	pkIndexMap := make(map[string]int)
	if parseCtx == nil || parseCtx.InsertStmt == nil || parseCtx.InsertStmt.Columns == nil {
		return pkIndexMap
	}
	insertColumnsSize := len(parseCtx.InsertStmt.Columns)
	if insertColumnsSize == 0 {
		return pkIndexMap
	}
	if meta.ColumnNames == nil {
		return pkIndexMap
	}
	if len(meta.Columns) > 0 {
		for paramIdx := 0; paramIdx < insertColumnsSize; paramIdx++ {
			sqlColumnName := parseCtx.InsertStmt.Columns[paramIdx].Name.O
			if u.containPK(sqlColumnName, meta) {
				pkIndexMap[sqlColumnName] = paramIdx
			}
		}
		return pkIndexMap
	}

	pkIndex := -1
	allColumns := meta.Columns
	for _, columnMeta := range allColumns {
		pkIndex++
		if u.containPK(columnMeta.ColumnName, meta) {
			pkIndexMap[executor.DelEscape(columnMeta.ColumnName, types.DBTypeMySQL)] = pkIndex
		}
	}

	return pkIndexMap
}

// parsePkValuesFromStatement parse primary key value from statement.
// return the primary key and values<key:primary key,value:primary key values></key:primary>
func (u *MySQLInsertUndoLogBuilder) parsePkValuesFromStatement(columnName string, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error) {
	pkIndexMap := u.getPkIndex(columnName, parseCtx, meta)
	if pkIndexMap == nil || len(pkIndexMap) == 0 {
		return nil, errors.New("pkIndex is not found")
	}
	var pkIndexArray []int
	for _, val := range pkIndexMap {
		pkIndexArray = append(pkIndexArray, val)
	}

	if parseCtx == nil || parseCtx.InsertStmt == nil || len(parseCtx.InsertStmt.Lists) == 0 {
		return nil, errors.New("parCtx is nil, perhaps InsertStmt is empty")
	}

	pkValuesMap := make(map[string][]interface{})
	for pkName, pkIndex := range pkIndexMap {
		if pkIndex >= len(parseCtx.InsertStmt.Lists) {
			return nil, errors.New("pkIndex out of range")
		}
		for i := range parseCtx.InsertStmt.Lists[pkIndex] {
			if node, ok := parseCtx.InsertStmt.Lists[pkIndex][i].(ast.ValueExpr); ok {
				pkValuesMap[pkName] = append(pkValuesMap[pkName], node.GetValue())
			}
		}
	}

	return pkValuesMap, nil
}

// getPkValuesByColumn get pk value by column.
func (u *MySQLInsertUndoLogBuilder) getPkValuesByColumn(execCtx *types.ExecContext, columnName string) (map[string][]interface{}, error) {
	if execCtx == nil || execCtx.ParseContext == nil || execCtx.ParseContext.InsertStmt == nil {
		return nil, nil
	}
	parseCtx := execCtx.ParseContext
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	meta := execCtx.MetaDataMap[tableName]
	pkValuesMap, err := u.parsePkValuesFromStatement(columnName, parseCtx, meta)
	if err != nil {
		return nil, err
	}

	// generate pkValue by auto increment
	for _, v := range pkValuesMap {
		if len(v) == 1 {
			// pk auto generated while single insert primary key is expression
			if _, ok := v[0].(*ast.FuncCallExpr); ok {
				curPkValueMap, err := u.getPkValuesByAuto(execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValueMap)
			}
		} else if len(v) > 0 && v[0] == nil {
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
	if len(pkValuesMap) == 0 {
		return nil, errors.New("pk map is empty")
	}
	var autoColumnName string
	for _, columnMeta := range pkMetaMap {
		if columnMeta.Autoincrement {
			autoColumnName = columnMeta.ColumnName
			break
		}
	}
	if len(autoColumnName) == 0 {
		return nil, errors.New("auto increment column not exist")
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
		return u.autoGeneratePks(execCtx, autoColumnName, int(lastInsertId), int(updateCount))
	}

	if lastInsertId > 0 {
		pkValues := make([]interface{}, 0)
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

func (u *MySQLInsertUndoLogBuilder) autoGeneratePks(execCtx *types.ExecContext, autoColumnName string, lastInsetId, updateCount int) (map[string][]interface{}, error) {
	var step int
	if u.IncrementStep > 0 {
		step = u.IncrementStep
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
			curStep := make([]driver.Value, 0)
			if err := rows.Next(curStep); err != nil {
				return nil, err
			}

			if curStepInt, ok := curStep[0].(int64); ok {
				step = int(curStepInt)
			}
		} else {
			return nil, errors.New("query is empty")
		}
	}

	if step == 0 {
		return nil, errors.New("get increment step error")
	}

	pkValues := make([]interface{}, 0)
	for i := 0; i < updateCount; i++ {
		pkValues = append(pkValues, lastInsetId+updateCount*step)
	}
	pkValuesMap := make(map[string][]interface{})
	pkValuesMap[autoColumnName] = pkValues
	return pkValuesMap, nil
}

func pkValuesMapMerge(dest *map[string][]interface{}, src map[string][]interface{}) {
	for k, v := range src {
		(*dest)[k] = append((*dest)[k], v)
	}
}

// containsColumns judge sql specify column
func containsColumns(parseCtx *types.ParseContext) bool {
	if parseCtx == nil || parseCtx.InsertStmt == nil || parseCtx.InsertStmt.Lists == nil {
		return false
	}
	return len(parseCtx.InsertStmt.Lists) > 0
}

func getInsertRows(parseCtx *types.ParseContext, pkIndexArray []int) ([][]interface{}, error) {
	if parseCtx == nil || parseCtx.InsertStmt == nil {
		return nil, nil
	}
	if len(parseCtx.InsertStmt.Lists) == 0 {
		return nil, nil
	}
	rows := make([][]interface{}, 0)

	for _, nodes := range parseCtx.InsertStmt.Lists {
		row := make([]interface{}, 0)
		for i, node := range nodes {
			if _, ok := node.(*ast.IsNullExpr); ok {
				//todo: row.add(Null.get());
				row = append(row, nil)
			} else if newNode, ok := node.(ast.ValueExpr); ok {
				row = append(row, newNode.GetValue())
			} else if newNode, ok := node.(*ast.VariableExpr); ok {
				row = append(row, newNode.Name)
			} else if _, ok := node.(*ast.FuncCallExpr); ok {
				row = append(row, ast.FuncCallExpr{})
			} else {
				for _, index := range pkIndexArray {
					if index == i {
						return nil, errors.Errorf("Unknown SQLExpr:%v", node)
					}
				}
				row = append(row, ast.DefaultExpr{})
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (u *MySQLInsertUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.InsertExecutor
}
