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
	var selectArgs []driver.Value
	pkIndexMap := u.getPkIndex(insertStmt, metaData)
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
	paramMap, err := u.buildImageParameters(insertStmt, args, insertRows)
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
						u.BeforeImageSqlPrimaryKeys[columnName] = true
					}
					uniqueList = append(uniqueList, columnName+" = DEFAULT("+columnName+") ")
					columnIsNull = false
					continue
				}
				if strings.EqualFold("PRIMARY", index.Name) {
					u.BeforeImageSqlPrimaryKeys[columnName] = true
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
	log.Infof("build select sql by insert on duplicate sourceQuery, sql {}", sql.String())
	return sql.String(), selectArgs, nil
}

func (u *MySQLInsertOnDuplicateUndoLogBuilder) AfterImage(ctx context.Context, execCtx *types.ExecContext, beforeImages []*types.RecordImage) ([]*types.RecordImage, error) {
	afterSelectSql, selectArgs := u.buildAfterImageSQL(ctx, beforeImages)
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
			if !u.BeforeImageSqlPrimaryKeys[name] {
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

// build sql params
func (u *MySQLInsertOnDuplicateUndoLogBuilder) buildImageParameters(insert *ast.InsertStmt, args []driver.Value, insertRows [][]interface{}) (map[string][]driver.Value, error) {
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
			columnName := executor.DelEscape(col, types.DBTypeMySQL)
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
