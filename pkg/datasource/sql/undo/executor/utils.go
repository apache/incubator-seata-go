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
	"reflect"
	"strings"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
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

	return compareRows(beforeImage.TableMeta, beforeImage.Rows, afterImage.Rows)
}

func compareRows(tableMeta types.TableMeta, oldRows []types.RowImage, newRows []types.RowImage) (bool, error) {
	oldRowMap := rowListToMap(oldRows, tableMeta.GetPrimaryKeyOnlyName())
	newRowMap := rowListToMap(newRows, tableMeta.GetPrimaryKeyOnlyName())

	for key, oldRow := range oldRowMap {
		newRow := newRowMap[key]
		if newRow == nil {
			return false, fmt.Errorf("compare row failed, rowKey %s, reason [newField is null]", key)
		}
		for fieldName, oldValue := range oldRow {
			newValue := newRow[fieldName]
			if newValue == nil {
				return false, fmt.Errorf("compare row failed, rowKey %s, fieldName %s, reason [newField is null]", key, fieldName)
			}
			if !reflect.DeepEqual(newValue, oldValue) {
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
					rowKey = fmt.Sprintf("%v%v", rowKey, column.Value)
					firstUnderline = true
				}
			}
			fieldMap[strings.ToUpper(column.ColumnName)] = column.Value
		}
		rowMap[rowKey] = fieldMap
	}
	return rowMap
}
