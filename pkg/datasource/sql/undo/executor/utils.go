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
	"fmt"
	"strings"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/util/log"
)

// IsRecordsEquals check before record and after record if equal
func IsRecordsEquals(beforeImage *types.RecordImage, afterImage *types.RecordImage) (bool, error) {
	if beforeImage == nil && afterImage == nil {
		return true, nil
	}
	if beforeImage == nil || afterImage == nil {
		return false, nil
	}

	if !strings.EqualFold(beforeImage.TableName, afterImage.TableName) || len(beforeImage.Rows) != len(afterImage.Rows) {
		return false, nil
	}
	if len(beforeImage.Rows) == 0 {
		return true, nil
	}

	return compareRows(*beforeImage.TableMeta, beforeImage.Rows, afterImage.Rows)
}

func compareRows(tableMeta types.TableMeta, oldRows []types.RowImage, newRows []types.RowImage) (bool, error) {
	oldRowMap := rowListToMap(oldRows, tableMeta.GetPrimaryKeyOnlyName())
	newRowMap := rowListToMap(newRows, tableMeta.GetPrimaryKeyOnlyName())

	for key, oldRow := range oldRowMap {
		newRow := newRowMap[key]
		if newRow == nil {
			log.Infof("compare row failed, rowKey %s, reason new field is null", key)
			return false, nil
		}
		for fieldName, oldValue := range oldRow {
			newValue := newRow[fieldName]
			if !datasource.DeepEqual(newValue, oldValue) {
				log.Infof("compare row failed, rowKey %s, fieldName %s, oldValue %v, newValud %v", key, fieldName, oldValue, newValue)
				return false, nil
			}
		}
	}
	return true, nil
}

func rowListToMap(rows []types.RowImage, primaryKeyList []string) map[string]map[string]interface{} {
	rowMap := make(map[string]map[string]interface{}, 0)
	for _, row := range rows {
		fieldMap := make(map[string]interface{}, 0)
		var rowKey string
		var firstUnderline bool

		for _, column := range row.Columns {
			for i, key := range primaryKeyList {
				if column.ColumnName == key {
					if firstUnderline && i > 0 {
						rowKey += "_##$$_"
					}
					// todo make value more accurate
					rowKey = fmt.Sprintf("%v%v", rowKey, column.GetActualValue())
					firstUnderline = true
				}
			}
			fieldMap[strings.ToUpper(column.ColumnName)] = column.Value
		}
		rowMap[rowKey] = fieldMap
	}
	return rowMap
}

// buildWhereConditionByPKs build where condition by primary keys
// each pk is a condition.the result will like :" (id,userCode) in ((?,?),(?,?)) or (id,userCode) in ((?,?),(?,?) ) or (id,userCode) in ((?,?))"
func buildWhereConditionByPKs(pkNameList []string, rowSize int, maxInSize int) string {
	var (
		whereStr  = &strings.Builder{}
		batchSize = rowSize/maxInSize + 1
	)

	if rowSize%maxInSize == 0 {
		batchSize = rowSize / maxInSize
	}

	for batch := 0; batch < batchSize; batch++ {
		if batch > 0 {
			whereStr.WriteString(" OR ")
		}
		whereStr.WriteString("(")

		for i := 0; i < len(pkNameList); i++ {
			if i > 0 {
				whereStr.WriteString(",")
			}
			// todo add escape
			whereStr.WriteString(fmt.Sprintf("`%s`", pkNameList[i]))
		}
		whereStr.WriteString(") IN (")

		var eachSize int

		if batch == batchSize-1 {
			if rowSize%maxInSize == 0 {
				eachSize = maxInSize
			} else {
				eachSize = rowSize % maxInSize
			}
		} else {
			eachSize = maxInSize
		}

		for i := 0; i < eachSize; i++ {
			if i > 0 {
				whereStr.WriteString(",")
			}
			whereStr.WriteString("(")
			for j := 0; j < len(pkNameList); j++ {
				if j > 0 {
					whereStr.WriteString(",")
				}
				whereStr.WriteString("?")
			}
			whereStr.WriteString(")")
		}
		whereStr.WriteString(")")
	}
	return whereStr.String()
}

func buildPKParams(rows []types.RowImage, pkNameList []string) []interface{} {
	params := make([]interface{}, 0)
	for _, row := range rows {
		coumnMap := row.GetColumnMap()
		for _, pk := range pkNameList {
			col := coumnMap[pk]
			if col != nil {
				params = append(params, col.Value)
			}
		}
	}
	return params
}
