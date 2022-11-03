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

import "database/sql"

// ColumnMeta
type ColumnMeta struct {
	// Schema
	Schema string
	// Table
	Table string
	// ColumnDef  the column def
	ColumnDef []byte
	// Info
	Info sql.ColumnType
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
	Name       string
	ColumnName string
	NonUnique  bool
	// IType
	IType IndexType
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

func (m TableMeta) GetPrimaryKeyOnlyName() []string {
	keys := make([]string, 0)
	for _, index := range m.Indexs {
		if index.IType == IndexTypePrimaryKey {
			keys = append(keys, index.ColumnName)
		}
	}
	return keys
}
