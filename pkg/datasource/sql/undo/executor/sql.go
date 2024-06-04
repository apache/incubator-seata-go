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

package executor

import (
	"database/sql"
	"strings"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
)

const (
	Dot            = "."
	EscapeStandard = "\""
	EscapeMysql    = "`"
)

// DelEscape del escape by db type
func DelEscape(colName string, dbType types.DBType) string {
	newColName := delEscape(colName, EscapeStandard)
	if dbType == types.DBTypeMySQL {
		newColName = delEscape(newColName, EscapeMysql)
	}
	return newColName
}

// delEscape
func delEscape(colName string, escape string) string {
	if colName == "" {
		return ""
	}

	if string(colName[0]) == escape && string(colName[len(colName)-1]) == escape {
		// like "scheme"."id" `scheme`.`id`
		str := escape + Dot + escape
		index := strings.Index(colName, str)
		if index > -1 {
			return colName[1:index] + Dot + colName[index+len(str):len(colName)-1]
		}

		return colName[1 : len(colName)-1]
	} else {
		// like "scheme".id `scheme`.id
		str := escape + Dot
		index := strings.Index(colName, str)
		if index > -1 && string(colName[0]) == escape {
			return colName[1:index] + Dot + colName[index+len(str):]
		}

		// like scheme."id" scheme.`id`
		str = Dot + escape
		index = strings.Index(colName, str)
		if index > -1 && string(colName[len(colName)-1]) == escape {
			return colName[0:index] + Dot + colName[index+len(str):len(colName)-1]
		}
	}

	return colName
}

// AddEscape if necessary, add escape by db type
func AddEscape(colName string, dbType types.DBType) string {
	if dbType == types.DBTypeMySQL {
		return addEscape(colName, dbType, EscapeMysql)
	}

	return addEscape(colName, dbType, EscapeStandard)
}

func addEscape(colName string, dbType types.DBType, escape string) string {
	if colName == "" {
		return colName
	}

	if string(colName[0]) == escape && string(colName[len(colName)-1]) == escape {
		return colName
	}

	if !checkEscape(colName, dbType) {
		return colName
	}

	if strings.Contains(colName, Dot) {
		// like "scheme".id `scheme`.id
		str := escape + Dot
		dotIndex := strings.Index(colName, str)
		if dotIndex > -1 {
			tempStr := strings.Builder{}
			tempStr.WriteString(colName[0 : dotIndex+len(str)])
			tempStr.WriteString(escape)
			tempStr.WriteString(colName[dotIndex+len(str):])
			tempStr.WriteString(escape)

			return tempStr.String()
		}

		// like scheme."id" scheme.`id`
		str = Dot + escape
		dotIndex = strings.Index(colName, str)
		if dotIndex > -1 {
			tempStr := strings.Builder{}
			tempStr.WriteString(escape)
			tempStr.WriteString(colName[0:dotIndex])
			tempStr.WriteString(escape)
			tempStr.WriteString(colName[dotIndex:])

			return tempStr.String()
		}

		str = Dot
		dotIndex = strings.Index(colName, str)
		if dotIndex > -1 {
			tempStr := strings.Builder{}
			tempStr.WriteString(escape)
			tempStr.WriteString(colName[0:dotIndex])
			tempStr.WriteString(escape)
			tempStr.WriteString(Dot)
			tempStr.WriteString(escape)
			tempStr.WriteString(colName[dotIndex+len(str):])
			tempStr.WriteString(escape)

			return tempStr.String()
		}
	}

	buf := make([]byte, len(colName)+2)
	buf[0], buf[len(buf)-1] = escape[0], escape[0]

	for key, _ := range colName {
		buf[key+1] = colName[key]
	}

	return string(buf)
}

// checkEscape check whether given field or table name use keywords. the method has database special logic.
func checkEscape(colName string, dbType types.DBType) bool {
	switch dbType {
	case types.DBTypeMySQL:
		if _, ok := types.GetMysqlKeyWord()[strings.ToUpper(colName)]; ok {
			return true
		}

		return false
	// TODO impl Oracle PG SQLServer ...
	default:
		return true
	}
}

// BuildWhereConditionByPKs each pk is a condition.the result will like :" id =? and userCode =?"
func BuildWhereConditionByPKs(pkNameList []string, dbType types.DBType) string {
	whereStr := strings.Builder{}
	for i := 0; i < len(pkNameList); i++ {
		if i > 0 {
			whereStr.WriteString(" and ")
		}

		pkName := pkNameList[i]
		whereStr.WriteString(AddEscape(pkName, dbType))
		whereStr.WriteString(" = ? ")
	}

	return whereStr.String()
}

// DataValidationAndGoOn check data valid
// Todo implement dataValidationAndGoOn
func DataValidationAndGoOn(sqlUndoLog undo.SQLUndoLog, conn *sql.Conn) bool {
	return true
}

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
			if strings.Index(col.ColumnName, pkName) > -1 {
				pkFields = append(pkFields, col)
			}
		}
	}

	return pkFields, nil
}
