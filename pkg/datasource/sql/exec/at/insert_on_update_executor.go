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

// insertOnUpdateExecutor execute insert on update SQL
type insertOnUpdateExecutor struct {
	baseExecutor
	parserCtx                 *types.ParseContext
	execContext               *types.ExecContext
	beforeImageSqlPrimaryKeys map[string]bool
	beforeSelectSql           string
	beforeSelectArgs          []driver.NamedValue
	isUpdateFlag              bool
}

// NewInsertOnUpdateExecutor get insert on update executor
func NewInsertOnUpdateExecutor(parserCtx *types.ParseContext, execContent *types.ExecContext, hooks []exec.SQLHook) executor {
	return &insertOnUpdateExecutor{
		baseExecutor:              baseExecutor{hooks: hooks},
		parserCtx:                 parserCtx,
		execContext:               execContent,
		beforeImageSqlPrimaryKeys: make(map[string]bool),
	}
}

func (i *insertOnUpdateExecutor) ExecContext(ctx context.Context, f exec.CallbackWithNamedValue) (types.ExecResult, error) {
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

	afterImage, err := i.afterImage(ctx, beforeImage)
	if err != nil {
		return nil, err
	}

	if len(beforeImage.Rows) > 0 {
		beforeImage.SQLType = types.SQLTypeUpdate
		afterImage.SQLType = types.SQLTypeUpdate
	} else {
		beforeImage.SQLType = types.SQLTypeInsert
		afterImage.SQLType = types.SQLTypeInsert
	}

	i.execContext.TxCtx.RoundImages.AppendBeofreImage(beforeImage)
	i.execContext.TxCtx.RoundImages.AppendAfterImage(afterImage)
	return res, nil
}

// beforeImage build before image
func (i *insertOnUpdateExecutor) beforeImage(ctx context.Context) (*types.RecordImage, error) {
	if !i.isAstStmtValid() {
		log.Errorf("invalid insert statement! parser ctx:%+v", i.parserCtx)
		return nil, fmt.Errorf("invalid insert statement! parser ctx:%+v", i.parserCtx)
	}
	tableName, err := i.parserCtx.GetTableName()
	if err != nil {
		return nil, err
	}
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	selectSQL, selectArgs, err := i.buildBeforeImageSQL(i.parserCtx.InsertStmt, *metaData, i.execContext.NamedValues)
	if err != nil {
		return nil, err
	}
	if len(selectArgs) == 0 {
		log.Errorf("the SQL statement has no primary key or unique index value, it will not hit any row data."+
			"recommend to convert to a normal insert statement. db name:%s table name:%s sql:%s", i.execContext.DBName, tableName, i.execContext.Query)
		return nil, fmt.Errorf("invalid insert or update sql")
	}
	i.beforeSelectSql = selectSQL
	i.beforeSelectArgs = selectArgs

	var rowsi driver.Rows
	queryerCtx, queryerCtxExists := i.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	var queryerExists bool

	if !queryerCtxExists {
		queryer, queryerExists = i.execContext.Conn.(driver.Queryer)
	}
	if !queryerExists && !queryerCtxExists {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}
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
	image, err := i.buildRecordImages(rowsi, metaData, types.SQLTypeInsertOnDuplicateUpdate)
	if err != nil {
		return nil, err
	}
	return image, nil
}

// buildBeforeImageSQL build the SQL to query before image data
func (i *insertOnUpdateExecutor) buildBeforeImageSQL(insertStmt *ast.InsertStmt, metaData types.TableMeta, args []driver.NamedValue) (string, []driver.NamedValue, error) {
	if err := checkDuplicateKeyUpdate(insertStmt, metaData); err != nil {
		return "", nil, err
	}

	paramMap, insertNum, err := i.buildBeforeImageSQLParameters(insertStmt, args, metaData)
	if err != nil {
		return "", nil, err
	}
	sql := strings.Builder{}
	sql.WriteString("SELECT * FROM " + metaData.TableName + " ")
	isContainWhere := false
	var selectArgs []driver.NamedValue
	for j := 0; j < insertNum; j++ {
		finalJ := j
		var paramAppenderTempList []driver.NamedValue
		for _, index := range metaData.Indexs {
			// unique index
			if index.NonUnique || isIndexValueNull(index, paramMap, finalJ) {
				continue
			}
			columnIsNull := true
			var uniqueList []string
			for _, columnMeta := range index.Columns {
				columnName := columnMeta.ColumnName
				imageParameters, ok := paramMap[columnName]
				if !ok && columnMeta.ColumnDef != nil {
					if strings.EqualFold("PRIMARY", index.Name) {
						i.beforeImageSqlPrimaryKeys[columnName] = true
					}
					uniqueList = append(uniqueList, columnName+" = DEFAULT("+columnName+") ")
					columnIsNull = false
					continue
				}
				if strings.EqualFold("PRIMARY", index.Name) {
					i.beforeImageSqlPrimaryKeys[columnName] = true
				}
				columnIsNull = false
				uniqueList = append(uniqueList, columnName+" = ? ")
				paramAppenderTempList = append(paramAppenderTempList, imageParameters[finalJ])
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
	log.Infof("build select sql by insert on update sourceQuery, sql %s", sql.String())
	return sql.String(), selectArgs, nil
}

// buildBeforeImageSQLParameters build the SQL parameters to query before image data
func (i *insertOnUpdateExecutor) buildBeforeImageSQLParameters(insertStmt *ast.InsertStmt, args []driver.NamedValue, metaData types.TableMeta) (map[string][]driver.NamedValue, int, error) {
	pkIndexArray := i.getPkIndexArray(insertStmt, metaData)
	insertRows, err := getInsertRows(insertStmt, pkIndexArray)
	if err != nil {
		return nil, 0, err
	}

	parameterMap := make(map[string][]driver.NamedValue)
	insertColumns := getInsertColumns(insertStmt)
	placeHolderIndex := 0
	for _, rowColumns := range insertRows {
		if len(rowColumns) != len(insertColumns) {
			log.Errorf("insert row's column size not equal to insert column size. row columns:%+v insert columns:%+v", rowColumns, insertColumns)
			return nil, 0, fmt.Errorf("invalid insert row's column size")
		}
		for i, col := range insertColumns {
			columnName := DelEscape(col, types.DBTypeMySQL)
			val := rowColumns[i]
			rStr, ok := val.(string)
			if ok && strings.EqualFold(rStr, sqlPlaceholder) {
				objects := args[placeHolderIndex]
				parameterMap[columnName] = append(parameterMap[col], objects)
				placeHolderIndex++
			} else {
				parameterMap[columnName] = append(parameterMap[col], driver.NamedValue{
					Ordinal: i + 1,
					Name:    columnName,
					Value:   val,
				})
			}
		}
	}
	return parameterMap, len(insertRows), nil
}

// afterImage build after image
func (i *insertOnUpdateExecutor) afterImage(ctx context.Context, beforeImages *types.RecordImage) (*types.RecordImage, error) {
	afterSelectSql, selectArgs := i.buildAfterImageSQL(beforeImages)
	var rowsi driver.Rows
	queryerCtx, queryerCtxExists := i.execContext.Conn.(driver.QueryerContext)
	var queryer driver.Queryer
	var queryerExists bool
	if !queryerCtxExists {
		queryer, queryerExists = i.execContext.Conn.(driver.Queryer)
	}
	if !queryerCtxExists && !queryerExists {
		log.Errorf("target conn should been driver.QueryerContext or driver.Queryer")
		return nil, fmt.Errorf("invalid conn")
	}
	rowsi, err := util.CtxDriverQuery(ctx, queryerCtx, queryer, afterSelectSql, selectArgs)
	defer func() {
		if rowsi != nil {
			rowsi.Close()
		}
	}()
	if err != nil {
		log.Errorf("ctx driver query: %+v", err)
		return nil, err
	}
	tableName, err := i.parserCtx.GetTableName()
	if err != nil {
		return nil, err
	}
	metaData, err := datasource.GetTableCache(types.DBTypeMySQL).GetTableMeta(ctx, i.execContext.DBName, tableName)
	if err != nil {
		return nil, err
	}
	afterImage, err := i.buildRecordImages(rowsi, metaData, types.SQLTypeInsertOnDuplicateUpdate)
	if err != nil {
		return nil, err
	}
	lockKey := i.buildLockKey(afterImage, *metaData)
	i.execContext.TxCtx.LockKeys[lockKey] = struct{}{}
	return afterImage, nil
}

// buildAfterImageSQL build the SQL to query after image data
func (i *insertOnUpdateExecutor) buildAfterImageSQL(beforeImage *types.RecordImage) (string, []driver.NamedValue) {
	selectSQL, selectArgs := i.beforeSelectSql, i.beforeSelectArgs
	primaryValueMap := make(map[string][]interface{})

	for _, row := range beforeImage.Rows {
		for _, col := range row.Columns {
			if col.KeyType == types.IndexTypePrimaryKey {
				primaryValueMap[col.ColumnName] = append(primaryValueMap[col.ColumnName], col.Value)
			}
		}
	}

	var afterImageSql strings.Builder
	var primaryValues []driver.NamedValue
	afterImageSql.WriteString(selectSQL)
	for j := 0; j < len(beforeImage.Rows); j++ {
		var wherePrimaryList []string
		for name, value := range primaryValueMap {
			if !i.beforeImageSqlPrimaryKeys[name] {
				wherePrimaryList = append(wherePrimaryList, name+" = ? ")
				primaryValues = append(primaryValues, driver.NamedValue{
					Name:  name,
					Value: value[j],
				})
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

// isPKColumn check the column name to see if it is a primary key column
func (i *insertOnUpdateExecutor) isPKColumn(columnName string, meta types.TableMeta) bool {
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

// getPkIndexArray get index of primary key from insert statement
func (i *insertOnUpdateExecutor) getPkIndexArray(insertStmt *ast.InsertStmt, meta types.TableMeta) []int {
	var pkIndexArray []int
	if insertStmt == nil {
		return pkIndexArray
	}
	insertColumnsSize := len(insertStmt.Columns)
	if insertColumnsSize == 0 {
		return pkIndexArray
	}
	if meta.ColumnNames == nil {
		return pkIndexArray
	}
	if len(meta.Columns) > 0 {
		for paramIdx := 0; paramIdx < insertColumnsSize; paramIdx++ {
			sqlColumnName := insertStmt.Columns[paramIdx].Name.O
			if i.isPKColumn(sqlColumnName, meta) {
				pkIndexArray = append(pkIndexArray, paramIdx)
			}
		}
		return pkIndexArray
	}

	pkIndex := -1
	allColumns := meta.Columns
	for _, columnMeta := range allColumns {
		tmpColumnMeta := columnMeta
		pkIndex++
		if i.isPKColumn(tmpColumnMeta.ColumnName, meta) {
			pkIndexArray = append(pkIndexArray, pkIndex)
		}
	}

	return pkIndexArray
}

func (i *insertOnUpdateExecutor) isAstStmtValid() bool {
	return i.parserCtx != nil && i.parserCtx.InsertStmt != nil
}

// isIndexValueNull check if the index value is null
func isIndexValueNull(indexMeta types.IndexMeta, imageParameterMap map[string][]driver.NamedValue, rowIndex int) bool {
	for _, colMeta := range indexMeta.Columns {
		columnName := colMeta.ColumnName
		imageParameters := imageParameterMap[columnName]
		if imageParameters == nil && colMeta.ColumnDef == nil {
			return true
		} else if imageParameters != nil && (rowIndex >= len(imageParameters) || imageParameters[rowIndex].Value == nil) {
			return true
		}
	}
	return false
}

// getInsertColumns get insert columns from insert statement
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

// checkDuplicateKeyUpdate check whether insert on update sql wants to update the duplicate keys
func checkDuplicateKeyUpdate(insert *ast.InsertStmt, metaData types.TableMeta) error {
	duplicateColsMap := make(map[string]bool)
	for _, v := range insert.OnDuplicate {
		duplicateColsMap[strings.ToLower(v.Column.Name.L)] = true
	}
	if len(duplicateColsMap) == 0 {
		return nil
	}
	for _, index := range metaData.Indexs {
		if types.IndexTypePrimaryKey != index.IType {
			continue
		}
		for _, col := range index.Columns {
			if duplicateColsMap[strings.ToLower(col.ColumnName)] {
				log.Errorf("update pk value is not supported! index name:%s update column name: %s", index.Name, col.ColumnName)
				return fmt.Errorf("update pk value is not supported! index name:%s update column name: %s", index.Name, col.ColumnName)
			}
		}
	}
	return nil
}
