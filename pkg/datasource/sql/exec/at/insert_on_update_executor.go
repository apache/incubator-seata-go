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
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/datasource/sql/exec"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/util/log"
)

const (
	SqlPlaceholder = "?"
)

// insertExecutor execute insert SQL
type insertOnUpdateExecutor struct {
	baseExecutor
	parserCtx     *types.ParseContext
	execContent   *types.ExecContext
	incrementStep int
	// businesSQLResult after insert sql
	businesSQLResult types.ExecResult

	BeforeSelectSql           string
	Args                      []driver.Value
	BeforeImageSqlPrimaryKeys map[string]bool
}

// NewInsertOnUpdateExecutor get insert on update executor
func NewInsertOnUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &insertOnUpdateExecutor{
		parserCtx:                 parserCtx,
		execContent:               execContent,
		baseExecutor:              baseExecutor{hooks: hooks},
		Args:                      make([]driver.Value, 0),
		BeforeImageSqlPrimaryKeys: make(map[string]bool),
	}
}

func (iu *insertOnUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
	iu.beforeHooks(ctx, iu.execContent)
	defer func() {
		iu.afterHooks(ctx, iu.execContent)
	}()

	beforeImage, err := iu.beforeImage(ctx)
	if err != nil {
		return nil, err
	}

	res, err := f(ctx, iu.execContent.Query, iu.execContent.NamedValues)
	if err != nil {
		return nil, err
	}

	if iu.businesSQLResult == nil {
		iu.businesSQLResult = res
	}

	afterImage, err := iu.afterImage(ctx, iu.execContent, beforeImage)
	if err != nil {
		return nil, err
	}

	iu.execContent.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	iu.execContent.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

func (iu *insertOnUpdateExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	if iu.parserCtx.InsertStmt == nil {
		log.Errorf("invalid insert stmt")
		return nil, fmt.Errorf("invalid insert stmt")
	}
	vals := iu.execContent.Values
	if vals == nil {
		vals = make([]driver.Value, len(iu.execContent.NamedValues))
		for n, param := range iu.execContent.NamedValues {
			vals[n] = param.Value
		}
	}
	tableName, _ := iu.parserCtx.GteTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, iu.execContent.DBName, tableName)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs, err := iu.buildBeforeImageSQL(iu.execContent.ParseContext.InsertStmt, *metaData, vals)
	if err != nil {
		return nil, err
	}
	if len(selectArgs) == 0 {
		log.Errorf("the SQL statement has no primary key or unique index value, it will not hit any row data.recommend to convert to a normal insert statement")
		return nil, fmt.Errorf("the SQL statement has no primary key or unique index value, it will not hit any row data.recommend to convert to a normal insert statement")
	}
	iu.BeforeSelectSql = selectSQL
	iu.Args = selectArgs
	stmt, err := iu.execContent.Conn.Prepare(selectSQL)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}
	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}
	image, err := iu.buildRecordImages(rows, metaData)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (iu *insertOnUpdateExecutor) buildBeforeImageSQL(insertStmt *ast.InsertStmt, metaData types.TableMeta, args []driver.Value) (string, []driver.Value, error) {
	if err := checkDuplicateKeyUpdate(insertStmt, metaData); err != nil {
		return "", nil, err
	}
	var selectArgs []driver.Value
	pkIndexMap := iu.getPkIndex(insertStmt, metaData)
	var pkIndexArray []int
	for _, val := range pkIndexMap {
		tmpVal := val
		pkIndexArray = append(pkIndexArray, tmpVal)
	}
	insertRows, err := getInsertRows(insertStmt, pkIndexArray)
	if err != nil {
		return "", nil, err
	}
	insertNum := len(insertRows)
	paramMap, err := iu.buildImageParameters(insertStmt, args, insertRows)
	if err != nil {
		return "", nil, err
	}

	sql := strings.Builder{}
	sql.WriteString("SELECT * FROM " + metaData.TableName + " ")
	isContainWhere := false
	for i := 0; i < insertNum; i++ {
		finalI := i
		paramAppenderTempList := make([]driver.Value, 0)
		for _, index := range metaData.Indexs {
			//unique index
			if index.NonUnique || isIndexValueNotNull(index, paramMap, finalI) == false {
				continue
			}
			columnIsNull := true
			uniqueList := make([]string, 0)
			for _, columnMeta := range index.Columns {
				columnName := columnMeta.ColumnName
				imageParameters, ok := paramMap[columnName]
				if !ok && columnMeta.ColumnDef != nil {
					if strings.EqualFold("PRIMARY", index.Name) {
						iu.BeforeImageSqlPrimaryKeys[columnName] = true
					}
					uniqueList = append(uniqueList, columnName+" = DEFAULT("+columnName+") ")
					columnIsNull = false
					continue
				}
				if strings.EqualFold("PRIMARY", index.Name) {
					iu.BeforeImageSqlPrimaryKeys[columnName] = true
				}
				columnIsNull = false
				uniqueList = append(uniqueList, columnName+" = ? ")
				paramAppenderTempList = append(paramAppenderTempList, imageParameters[finalI])
			}

			if !columnIsNull {
				if isContainWhere {
					sql.WriteString(" OR (" + strings.Join(uniqueList, " and ") + ") ")
				} else {
					sql.WriteString(" WHERE (" + strings.Join(uniqueList, " and ") + ") ")
					isContainWhere = true
				}
			}
		}
		selectArgs = append(selectArgs, paramAppenderTempList...)
	}
	log.Infof("build select sql by insert on update sourceQuery, sql {}", sql.String())
	return sql.String(), selectArgs, nil
}

func (iu *insertOnUpdateExecutor) afterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages *types.RecordImage) (*types.RecordImage, error) {
	afterSelectSql, selectArgs := iu.buildAfterImageSQL(ctx, beforeImages)
	stmt, err := execCtx.Conn.Prepare(afterSelectSql)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}

	rows, err := stmt.Query(selectArgs)
	if err != nil {
		log.Errorf("stmt query: %+v", err)
		return nil, err
	}
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
	image, err := iu.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (iu *insertOnUpdateExecutor) buildAfterImageSQL(ctx context.Context, beforeImage *types.RecordImage) (string, []driver.Value) {
	selectSQL, selectArgs := iu.BeforeSelectSql, iu.Args
	primaryValueMap := make(map[string][]interface{})
	for _, row := range beforeImage.Rows {
		for _, col := range row.Columns {
			if col.KeyType == types.IndexTypePrimaryKey {
				primaryValueMap[col.ColumnName] = append(primaryValueMap[col.ColumnName], col.Value)
			}
		}
	}

	var afterImageSql strings.Builder
	var primaryValues []driver.Value
	afterImageSql.WriteString(selectSQL)
	for i := 0; i < len(beforeImage.Rows); i++ {
		wherePrimaryList := make([]string, 0)
		for name, value := range primaryValueMap {
			if !iu.BeforeImageSqlPrimaryKeys[name] {
				wherePrimaryList = append(wherePrimaryList, name+" = ? ")
				primaryValues = append(primaryValues, value[i])
			}
		}
		if len(wherePrimaryList) != 0 {
			afterImageSql.WriteString(" OR (" + strings.Join(wherePrimaryList, " and ") + ") ")
		}
	}
	selectArgs = append(selectArgs, primaryValues...)
	log.Infof("build after select sql by insert on duplicate sourceQuery, sql {}", afterImageSql.String())
	return afterImageSql.String(), selectArgs
}

func (iu *insertOnUpdateExecutor) buildImageParameters(insert *ast.InsertStmt, args []driver.Value, insertRows [][]interface{}) (map[string][]driver.Value, error) {
	var (
		parameterMap = make(map[string][]driver.Value)
	)
	insertColumns := getInsertColumns(insert)
	var placeHolderIndex = 0
	for _, row := range insertRows {
		if len(row) != len(insertColumns) {
			log.Errorf("insert row's column size not equal to insert column size")
			return nil, fmt.Errorf("insert row's column size not equal to insert column size")
		}
		for i, col := range insertColumns {
			columnName := DelEscape(col, types.DBTypeMySQL)
			val := row[i]
			rStr, ok := val.(string)
			if ok && strings.EqualFold(rStr, SqlPlaceholder) {
				objects := args[placeHolderIndex]
				parameterMap[columnName] = append(parameterMap[col], objects)
				placeHolderIndex++
			} else {
				parameterMap[columnName] = append(parameterMap[col], val)
			}
		}
	}
	return parameterMap, nil
}

func (iu *insertOnUpdateExecutor) getPkValues(ctx context.Context, execCtx *types.ExecContext, parseCtx *types.ParseContext, meta types.TableMeta) (map[string][]interface{}, error) {
	pkColumnNameList := meta.GetPrimaryKeyOnlyName()
	pkValuesMap := make(map[string][]interface{})
	var err error
	//when there is only one pk in the table
	if len(pkColumnNameList) == 1 {
		if iu.containsPK(meta, parseCtx) {
			// the insert sql contain pk value
			pkValuesMap, err = iu.getPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else if containsColumns(parseCtx) {
			// the insert table pk auto generated
			pkValuesMap, err = iu.getPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		} else {
			pkValuesMap, err = iu.getPkValuesByColumn(ctx, execCtx)
			if err != nil {
				return nil, err
			}
		}
	} else {
		//when there is multiple pk in the table
		//1,all pk columns are filled value.
		//2,the auto increment pk column value is null, and other pk value are not null.
		pkValuesMap, err = iu.getPkValuesByColumn(ctx, execCtx)
		if err != nil {
			return nil, err
		}
		for _, columnName := range pkColumnNameList {
			if _, ok := pkValuesMap[columnName]; !ok {
				curPkValuesMap, err := iu.getPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValuesMap)
			}
		}
	}
	return pkValuesMap, nil
}

func (iu *insertOnUpdateExecutor) containsPK(meta types.TableMeta, parseCtx *types.ParseContext) bool {
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

func (iu *insertOnUpdateExecutor) containPK(columnName string, meta types.TableMeta) bool {
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

func (iu *insertOnUpdateExecutor) getPkIndex(InsertStmt *ast.InsertStmt, meta types.TableMeta) map[string]int {
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
			if iu.containPK(sqlColumnName, meta) {
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
		if iu.containPK(tmpColumnMeta.ColumnName, meta) {
			pkIndexMap[DelEscape(tmpColumnMeta.ColumnName, types.DBTypeMySQL)] = pkIndex
		}
	}

	return pkIndexMap
}

func (iu *insertOnUpdateExecutor) parsePkValuesFromStatement(insertStmt *ast.InsertStmt, meta types.TableMeta, nameValues []driver.NamedValue) (map[string][]interface{}, error) {
	if insertStmt == nil {
		return nil, nil
	}
	pkIndexMap := iu.getPkIndex(insertStmt, meta)
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

func (iu *insertOnUpdateExecutor) getPkValuesByColumn(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !iu.isAstStmtValid() {
		return nil, nil
	}
	tableName, _ := iu.parserCtx.GteTableName()
	meta, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, iu.execContent.DBName, tableName)
	if err != nil {
		return nil, err
	}
	pkValuesMap, err := iu.parsePkValuesFromStatement(iu.parserCtx.InsertStmt, *meta, execCtx.NamedValues)
	if err != nil {
		return nil, err
	}

	// generate pkValue by auto increment
	for _, v := range pkValuesMap {
		tmpV := v
		if len(tmpV) == 1 {
			// pk auto generated while single insert primary key is expression
			if _, ok := tmpV[0].(*ast.FuncCallExpr); ok {
				curPkValueMap, err := iu.getPkValuesByAuto(ctx, execCtx)
				if err != nil {
					return nil, err
				}
				pkValuesMapMerge(&pkValuesMap, curPkValueMap)
			}
		} else if len(tmpV) > 0 && tmpV[0] == nil {
			// pk auto generated while column exists and value is null
			curPkValueMap, err := iu.getPkValuesByAuto(ctx, execCtx)
			if err != nil {
				return nil, err
			}
			pkValuesMapMerge(&pkValuesMap, curPkValueMap)
		}
	}
	return pkValuesMap, nil
}

func (iu *insertOnUpdateExecutor) getPkValuesByAuto(ctx context.Context, execCtx *types.ExecContext) (map[string][]interface{}, error) {
	if !iu.isAstStmtValid() {
		return nil, nil
	}
	tableName, _ := iu.parserCtx.GteTableName()
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, iu.execContent.DBName, tableName)
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

	updateCount, err := iu.businesSQLResult.GetResult().RowsAffected()
	if err != nil {
		return nil, err
	}

	lastInsertId, err := iu.businesSQLResult.GetResult().LastInsertId()
	if err != nil {
		return nil, err
	}

	// If there is batch insert
	// do auto increment base LAST_INSERT_ID and variable `auto_increment_increment`
	if lastInsertId > 0 && updateCount > 1 && canAutoIncrement(pkMetaMap) {
		return iu.autoGeneratePks(execCtx, autoColumnName, lastInsertId, updateCount)
	}

	if lastInsertId > 0 {
		var pkValues []interface{}
		pkValues = append(pkValues, lastInsertId)
		pkValuesMap[autoColumnName] = pkValues
		return pkValuesMap, nil
	}

	return nil, nil
}

func (iu *insertOnUpdateExecutor) isAstStmtValid() bool {
	return iu.parserCtx != nil && iu.parserCtx.InsertStmt != nil
}

func (iu *insertOnUpdateExecutor) autoGeneratePks(execCtx *types.ExecContext, autoColumnName string, lastInsetId, updateCount int64) (map[string][]interface{}, error) {
	var step int64
	if iu.incrementStep > 0 {
		step = int64(iu.incrementStep)
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

func isIndexValueNotNull(indexMeta types.IndexMeta, imageParameterMap map[string][]driver.Value, rowIndex int) bool {
	for _, colMeta := range indexMeta.Columns {
		columnName := colMeta.ColumnName
		imageParameters := imageParameterMap[columnName]
		if imageParameters == nil && colMeta.ColumnDef == nil {
			return false
		} else if imageParameters != nil && (rowIndex >= len(imageParameters) || imageParameters[rowIndex] == nil) {
			return false
		}
	}
	return true
}

func getInsertColumns(insertStmt *ast.InsertStmt) []string {
	if insertStmt == nil {
		return nil
	}
	colList := insertStmt.Columns
	if len(colList) == 0 {
		return nil
	}
	var list []string
	for _, col := range colList {
		list = append(list, col.Name.L)
	}
	return list
}

func checkDuplicateKeyUpdate(insert *ast.InsertStmt, metaData types.TableMeta) error {
	duplicateColsMap := make(map[string]bool)
	for _, v := range insert.OnDuplicate {
		duplicateColsMap[v.Column.Name.L] = true
	}
	if len(duplicateColsMap) == 0 {
		return nil
	}
	for _, index := range metaData.Indexs {
		if types.IndexTypePrimaryKey != index.IType {
			continue
		}
		for name, col := range index.Columns {
			if duplicateColsMap[strings.ToLower(col.ColumnName)] {
				log.Errorf("update pk value is not supported! index name:%s update column name: %s", name, col.ColumnName)
				return fmt.Errorf("update pk value is not supported! ")
			}
		}
	}
	return nil
}
