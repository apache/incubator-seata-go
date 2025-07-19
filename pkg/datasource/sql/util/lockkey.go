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

// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"strings"
)

func BuildLockKey(records *types.RecordImage, meta types.TableMeta) string {
	var lockKeys strings.Builder
	type ColMapItem struct {
		pkIndex  int
		colIndex int
	}

	lockKeys.WriteString(strings.ToUpper(meta.TableName))
	lockKeys.WriteString(":")

	keys := meta.GetPrimaryKeyOnlyName()
	keyIndexMap := make(map[string]int, len(keys))
	for idx, columnName := range keys {
		keyIndexMap[columnName] = idx
	}

	columns := make([]ColMapItem, 0, len(keys))
	if len(records.Rows) > 0 {
		for colIdx, column := range records.Rows[0].Columns {
			if pkIdx, ok := keyIndexMap[column.ColumnName]; ok {
				columns = append(columns, ColMapItem{pkIndex: pkIdx, colIndex: colIdx})
			}
		}
		for i, row := range records.Rows {
			if i > 0 {
				lockKeys.WriteString(",")
			}
			primaryKeyValues := make([]interface{}, len(keys))
			for _, mp := range columns {
				if mp.colIndex < len(row.Columns) {
					primaryKeyValues[mp.pkIndex] = row.Columns[mp.colIndex].Value
				}
			}
			for j, pkVal := range primaryKeyValues {
				if j > 0 {
					lockKeys.WriteString("_")
				}
				if pkVal == nil {
					continue
				}
				lockKeys.WriteString(fmt.Sprintf("%v", pkVal))
			}
		}
	}
	return lockKeys.String()
}
