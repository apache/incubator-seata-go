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

package types

import "errors"

// ColumnMeta
type ColumnMeta struct {
	TableCat string
	// Schema
	Schema string
	// Table
	Table string
	// Autoincrement
	Autoincrement bool
	ColumnName    string
	ColumnType    string
	DataType      int32
	ColumnKey     string
	IsNullable    int8
	Extra         string
}

// IndexMeta
type IndexMeta struct {
	// Schema
	Schema string
	// Table
	Table      string
	IndexName  string
	ColumnName string
	NonUnique  bool
	// IType
	IndexType IndexType
	// Values
	Values []ColumnMeta
}

// TableMeta
type TableMeta struct {
	// Schema
	Schema string
	// Name
	Name string
	// Columns
	Columns map[string]ColumnMeta
	// Indexs
	Indexs      map[string]IndexMeta
	ColumnNames []string
}

func (m TableMeta) IsEmpty() bool {
	return m.Name == ""
}

// GetPrimaryKeyOnlyName
func (m TableMeta) GetPrimaryKeyOnlyName() ([]string, error) {
	pkName := make([]string, 0)

	for _, index := range m.Indexs {
		if index.IndexType == IndexPrimary {
			for _, col := range index.Values {
				pkName = append(pkName, col.ColumnName)
			}
		}
	}

	if len(pkName) == 0 {
		return nil, errors.New("needs to contain the primary key")
	}

	return pkName, nil
}
