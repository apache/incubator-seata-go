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

type MySQLInsertOnDuplicateUndoLogBuilder struct {
	MySQLInsertUndoLogBuilder
	BeforeSelectSql           string
	Args                      []driver.Value
	BeforeImageSqlPrimaryKeys map[string]bool
}

func GetMySQLInsertOnDuplicateUndoLogBuilder() undo.UndoLogBuilder {
	return &MySQLInsertOnDuplicateUndoLogBuilder{
		MySQLInsertUndoLogBuilder: MySQLInsertUndoLogBuilder{},
		Args:                      make([]driver.Value, 0),
		BeforeImageSqlPrimaryKeys: make(map[string]bool),
	}
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) GetExecutorType() types.ExecutorType {
	return types.InsertOnDuplicateExecutor
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) BeforeImage(ctx context.Context, execCtx *types.ExecContext) ([]*types.RecordImage, error) {
	if execCtx.ParseContext.InsertStmt == nil {
		log.Errorf("invalid insert stmt")
		return nil, fmt.Errorf("invalid insert stmt")
	}
	vals := execCtx.Values
	if vals == nil {
		vals = make([]driver.Value, len(execCtx.NamedValues))
		for n, param := range execCtx.NamedValues {
			vals[n] = param.Value
		}
	}
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
	selectSQL, selectArgs, err := u.buildBeforeImageSQL(execCtx.ParseContext.InsertStmt, metaData, vals)
	if err != nil {
		return nil, err
	}
	if len(selectArgs) == 0 {
		log.Errorf("the SQL statement has no primary key or unique index value, it will not hit any row data.recommend to convert to a normal insert statement")
		return nil, fmt.Errorf("the SQL statement has no primary key or unique index value, it will not hit any row data.recommend to convert to a normal insert statement")
	}
	u.BeforeSelectSql = selectSQL
	u.Args = selectArgs
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

// buildBeforeImageSQL build select sql from insert on duplicate update sql
func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildBeforeImageSQL(insertStmt *ast.InsertStmt, metaData types.TableMeta, args []driver.Value) (string, []driver.Value, error) {
	if err := checkDuplicateKeyUpdate(insertStmt, metaData); err != nil {
		return "", nil, err
	}
	u.BeforeImageSqlPrimaryKeys = make(map[string]bool, len(metaData.Indexs))
	pkIndexMap := u.getPkIndex(insertStmt, metaData)
	var pkIndexArray []int
	for _, val := range pkIndexMap {
		pkIndexArray = append(pkIndexArray, val)
	}
	insertRows, err := getInsertRows(insertStmt, pkIndexArray)
	if err != nil {
		return "", nil, err
	}
	paramMap, err := u.buildImageParameters(insertStmt, args, insertRows)
	if err != nil {
		return "", nil, err
	}
	if len(paramMap) == 0 || len(metaData.Indexs) == 0 {
		return "", nil, nil
	}
	hasPK := false
	for _, index := range metaData.Indexs {
		if strings.EqualFold("PRIMARY", index.Name) {
			allPKColumnsHaveValue := true
			for _, col := range index.Columns {
				if params, ok := paramMap[col.ColumnName]; !ok || len(params) == 0 || params[0] == nil {
					allPKColumnsHaveValue = false
					break
				}
			}
			hasPK = allPKColumnsHaveValue
			break
		}
	}
	if !hasPK {
		hasValidUniqueIndex := false
		for _, index := range metaData.Indexs {
			if !index.NonUnique && !strings.EqualFold("PRIMARY", index.Name) {
				if _, _, valid := validateIndexPrefix(index, paramMap, 0); valid {
					hasValidUniqueIndex = true
					break
				}
			}
		}
		if !hasValidUniqueIndex {
			return "", nil, nil
		}
	}
	var sql strings.Builder
	sql.WriteString("SELECT * FROM " + metaData.TableName + "  ")
	var selectArgs []driver.Value
	isContainWhere := false
	hasConditions := false
	for i := 0; i < len(insertRows); i++ {
		var rowConditions []string
		var rowArgs []driver.Value
		usedParams := make(map[string]bool)

		// First try unique indexes
		for _, index := range metaData.Indexs {
			if index.NonUnique || strings.EqualFold("PRIMARY", index.Name) {
				continue
			}
			if conditions, args, valid := validateIndexPrefix(index, paramMap, i); valid {
				rowConditions = append(rowConditions, "("+strings.Join(conditions, " and ")+")")
				rowArgs = append(rowArgs, args...)
				hasConditions = true
				for _, colMeta := range index.Columns {
					usedParams[colMeta.ColumnName] = true
				}
			}
		}

		// Then try primary key
		for _, index := range metaData.Indexs {
			if !strings.EqualFold("PRIMARY", index.Name) {
				continue
			}
			if conditions, args, valid := validateIndexPrefix(index, paramMap, i); valid {
				rowConditions = append(rowConditions, "("+strings.Join(conditions, " and ")+")")
				rowArgs = append(rowArgs, args...)
				hasConditions = true
				for _, colMeta := range index.Columns {
					usedParams[colMeta.ColumnName] = true
				}
			}
		}

		if len(rowConditions) > 0 {
			if !isContainWhere {
				sql.WriteString("WHERE ")
				isContainWhere = true
			} else {
				sql.WriteString(" OR ")
			}
			sql.WriteString(strings.Join(rowConditions, "  OR ") + " ")
			selectArgs = append(selectArgs, rowArgs...)
		}
	}
	if !hasConditions {
		return "", nil, nil
	}
	sqlStr := sql.String()
	log.Infof("build select sql by insert on duplicate sourceQuery, sql: %s", sqlStr)
	return sqlStr, selectArgs, nil
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	afterSelectSql, selectArgs := u.buildAfterImageSQL(ctx, beforeImages)
	stmt, err := execCtx.Conn.Prepare(afterSelectSql)
	if err != nil {
		log.Errorf("build prepare stmt: %+v", err)
		return nil, err
	}
	defer stmt.Close()
	tableName := execCtx.ParseContext.InsertStmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O
	metaData := execCtx.MetaDataMap[tableName]
	rows, err := stmt.Query(selectArgs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	image, err := u.buildRecordImages(rows, &metaData)
	if err != nil {
		return nil, err
	}
	return []*types.RecordImage{image}, nil
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildAfterImageSQL(ctx context.Context, beforeImages []*types.RecordImage) (string, []driver.Value) {
	selectSQL, selectArgs := u.BeforeSelectSql, u.Args
	var beforeImage *types.RecordImage
	if len(beforeImages) > 0 {
		beforeImage = beforeImages[0]
	}
	if beforeImage == nil || len(beforeImage.Rows) == 0 {
		return selectSQL, selectArgs
	}
	primaryValueMap := make(map[string][]interface{})
	for _, row := range beforeImage.Rows {
		for _, col := range row.Columns {
			if col.KeyType == types.IndexTypePrimaryKey {
				primaryValueMap[col.ColumnName] = append(primaryValueMap[col.ColumnName], col.Value)
			}
		}
	}
	var afterImageSql strings.Builder
	afterImageSql.WriteString(selectSQL)
	if len(primaryValueMap) == 0 || len(selectArgs) == len(beforeImage.Rows)*len(primaryValueMap) {
		return selectSQL, selectArgs
	}
	var primaryValues []driver.Value
	usedPrimaryKeys := make(map[string]bool)
	for name := range primaryValueMap {
		if !u.BeforeImageSqlPrimaryKeys[name] {
			usedPrimaryKeys[name] = true
			for i := 0; i < len(beforeImage.Rows); i++ {
				if value := primaryValueMap[name][i]; value != nil {
					if dv, ok := value.(driver.Value); ok {
						primaryValues = append(primaryValues, dv)
					} else {
						primaryValues = append(primaryValues, value)
					}
				}
			}
		}
	}
	if len(primaryValues) > 0 {
		afterImageSql.WriteString(" OR (" + strings.Join(u.buildPrimaryKeyConditions(primaryValueMap, usedPrimaryKeys), " and  ") + ") ")
	}
	finalArgs := make([]driver.Value, len(selectArgs)+len(primaryValues))
	copy(finalArgs, selectArgs)
	copy(finalArgs[len(selectArgs):], primaryValues)
	sqlStr := afterImageSql.String()
	log.Infof("build after select sql by insert on duplicate sourceQuery, sql %s", sqlStr)
	return sqlStr, finalArgs
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildPrimaryKeyConditions(primaryValueMap map[string][]interface{}, usedPrimaryKeys map[string]bool) []string {
	var conditions []string
	for name := range primaryValueMap {
		if !usedPrimaryKeys[name] {
			conditions = append(conditions, name+" = ? ")
		}
	}
	return conditions
}

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
		for name, col := range index.Columns {
			if duplicateColsMap[strings.ToLower(col.ColumnName)] {
				log.Errorf("update pk value is not supported! index name:%s update column name: %s", name, col.ColumnName)
				return fmt.Errorf("update pk value is not supported! ")
			}
		}
	}
	return nil
}

// build sql params
func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildImageParameters(insert *ast.InsertStmt, args []driver.Value, insertRows [][]interface{}) (map[string][]driver.Value, error) {
	parameterMap := make(map[string][]driver.Value)
	insertColumns := getInsertColumns(insert)
	placeHolderIndex := 0

	for _, row := range insertRows {
		if len(row) != len(insertColumns) {
			log.Errorf("insert row's column size not equal to insert column size")
			return nil, fmt.Errorf("insert row's column size not equal to insert column size")
		}
		for i, col := range insertColumns {
			columnName := strings.ToLower(executor.DelEscape(col, types.DBTypeMySQL))
			val := row[i]
			if str, ok := val.(string); ok && strings.EqualFold(str, SqlPlaceholder) {
				if placeHolderIndex >= len(args) {
					return nil, fmt.Errorf("not enough parameters for placeholders")
				}
				parameterMap[columnName] = append(parameterMap[columnName], args[placeHolderIndex])
				placeHolderIndex++
			} else {
				parameterMap[columnName] = append(parameterMap[columnName], val)
			}
		}
	}
	return parameterMap, nil
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
		list = append(list, strings.ToLower(col.Name.L))
	}
	return list
}

func isIndexValueNotNull(indexMeta types.IndexMeta, imageParameterMap map[string][]driver.Value, rowIndex int) bool {
	for _, colMeta := range indexMeta.Columns {
		columnName := strings.ToLower(colMeta.ColumnName)
		imageParameters := imageParameterMap[columnName]
		if imageParameters == nil && colMeta.ColumnDef == nil {
			return false
		} else if imageParameters != nil && (rowIndex >= len(imageParameters) || imageParameters[rowIndex] == nil) {
			return false
		}
	}
	return true
}

func validateIndexPrefix(index types.IndexMeta, paramMap map[string][]driver.Value, rowIndex int) ([]string, []driver.Value, bool) {
	var indexConditions []string
	var indexArgs []driver.Value
	if len(index.Columns) > 1 {
		for _, colMeta := range index.Columns {
			params, ok := paramMap[colMeta.ColumnName]
			if !ok || len(params) <= rowIndex || params[rowIndex] == nil {
				return nil, nil, false
			}
		}
	}
	for _, colMeta := range index.Columns {
		columnName := colMeta.ColumnName
		params, ok := paramMap[columnName]
		if ok && len(params) > rowIndex && params[rowIndex] != nil {
			indexConditions = append(indexConditions, columnName+" = ? ")
			indexArgs = append(indexArgs, params[rowIndex])
		}
	}
	if len(indexConditions) != len(index.Columns) {
		return nil, nil, false
	}
	return indexConditions, indexArgs, true
}
