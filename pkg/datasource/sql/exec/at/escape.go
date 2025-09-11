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
	"database/sql"
	"strings"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

const (
	dot            = "."
	escapeStandard = "\""
	escapeMysql    = "`"
)

// DelEscape del escape by db type
func DelEscape(colName string, dbType types.DBType) string {
	newColName := delEscape(colName, escapeStandard)
	if dbType == types.DBTypeMySQL {
		newColName = delEscape(newColName, escapeMysql)
	}
	return newColName
}

// delEscape
func delEscape(colName string, escape string) string {
	if colName == "" {
		return ""
	}

	if len(colName) >= 2 &&
		string(colName[0]) == escape &&
		string(colName[len(colName)-1]) == escape {

		escapedDot := escape + dot + escape
		index := strings.Index(colName, escapedDot)
		if index > -1 {
			return colName[1:index] + dot + colName[index+len(escapedDot):len(colName)-1]
		}
		return colName[1 : len(colName)-1]
	}

	escapePrefixDot := escape + dot
	index := strings.Index(colName, escapePrefixDot)
	if index > -1 && string(colName[0]) == escape {
		return colName[1:index] + dot + colName[index+len(escapePrefixDot):]
	}

	dotEscapeSuffix := dot + escape
	index = strings.Index(colName, dotEscapeSuffix)
	if index > -1 && string(colName[len(colName)-1]) == escape {
		return colName[0:index] + dot + colName[index+len(dotEscapeSuffix):len(colName)-1]
	}

	return colName
}

// AddEscape if necessary, add escape by db type
func AddEscape(colName string, dbType types.DBType) string {
	if dbType == types.DBTypeMySQL {
		return addEscape(colName, dbType, escapeMysql)
	}
	return addEscape(colName, dbType, escapeStandard)
}

func addEscape(colName string, dbType types.DBType, escape string) string {
	if colName == "" {
		return colName
	}

	if !strings.Contains(colName, dot) &&
		len(colName) >= 2 &&
		string(colName[0]) == escape &&
		string(colName[len(colName)-1]) == escape {
		return colName
	}

	if strings.Contains(colName, dot) {
		parts := strings.SplitN(colName, dot, 2)
		if len(parts) != 2 {
			return colName
		}

		part1 := parts[0]
		part2 := parts[1]

		escapedPart1 := part1
		if checkEscape(strings.ToUpper(part1), dbType) && !isEscaped(part1, escape) {
			escapedPart1 = escape + part1 + escape
		}

		escapedPart2 := part2
		if checkEscape(strings.ToUpper(part2), dbType) && !isEscaped(part2, escape) {
			escapedPart2 = escape + part2 + escape
		}

		return escapedPart1 + dot + escapedPart2
	}

	if checkEscape(strings.ToUpper(colName), dbType) {
		return escape + colName + escape
	}

	return colName
}

// isEscaped checks if a string is already escaped with the given escape character
func isEscaped(str, escape string) bool {
	return len(str) >= 2 && string(str[0]) == escape && string(str[len(str)-1]) == escape
}

// checkEscape check whether given field or table name use keywords. the method has database special logic.
func checkEscape(upperColName string, dbType types.DBType) bool {
	switch dbType {
	case types.DBTypeMySQL:
		_, isKeyword := types.GetMysqlKeyWord()[upperColName]
		return isKeyword
	case types.DBTypePostgreSQL:
		_, isKeyword := types.GetPostgresKeyWord()[upperColName]
		return isKeyword
	default:
		return false
	}
}

// BuildWhereConditionByPKs each pk is a condition.the result will like :" id =? and userCode =?"
func BuildWhereConditionByPKs(pkNameList []string, dbType types.DBType) string {
	whereStr := strings.Builder{}
	for i, pkName := range pkNameList {
		if i > 0 {
			whereStr.WriteString(" AND ")
		}
		whereStr.WriteString(AddEscape(pkName, dbType))
		whereStr.WriteString(" = ?")
	}
	return whereStr.String()
}

// DataValidationAndGoOn check data valid
// Todo implement dataValidationAndGoOn
func DataValidationAndGoOn(sqlUndoLog undo.SQLUndoLog, conn *sql.Conn) bool {
	return true
}

// GetOrderedPkList gets ordered primary key list
func GetOrderedPkList(image *types.RecordImage, row types.RowImage, dbType types.DBType) ([]types.ColumnImage, error) {
	pkColumnNameListByOrder := image.TableMeta.GetPrimaryKeyOnlyName()
	pkColumnNameListNoOrder := make([]types.ColumnImage, 0)
	pkFields := make([]types.ColumnImage, 0)

	for _, column := range row.PrimaryKeys(row.Columns) {
		column.ColumnName = DelEscape(column.ColumnName, dbType)
		pkColumnNameListNoOrder = append(pkColumnNameListNoOrder, column)
	}

	for _, pkName := range pkColumnNameListByOrder {
		for _, col := range pkColumnNameListNoOrder {
			if strings.EqualFold(col.ColumnName, pkName) {
				pkFields = append(pkFields, col)
				break
			}
		}
	}

	return pkFields, nil
}
