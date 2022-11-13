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

import (
	"reflect"
)

// ColumnMeta
type ColumnMeta struct {
	// Schema
	Schema string
	// Table
	Table string
	// ColumnTypeInfo
	ColumnTypeInfo ColumnType
	// Autoincrement
	Autoincrement bool
	ColumnName    string
	ColumnType    string
	DataType      int32
	ColumnKey     string
	IsNullable    int8
	Extra         string
}

type ColumnType struct {
	Name string

	HasNullable       bool
	HasLength         bool
	HasPrecisionScale bool

	Nullable     bool
	Length       int64
	DatabaseType string
	Precision    int64
	Scale        int64
	ScanType     reflect.Type
}

// DatabaseTypeName returns the database system name of the column type. If an empty
// string is returned, then the driver type name is not supported.
// Consult your driver documentation for a list of driver data types. Length specifiers
// are not included.
// Common type names include "VARCHAR", "TEXT", "NVARCHAR", "DECIMAL", "BOOL",
// "INT", and "BIGINT".
func (ci *ColumnType) DatabaseTypeName() string {
	return ci.DatabaseType
}

// IndexMeta
type IndexMeta struct {
	// Schema
	Schema string
	// Table
	Table string
	Name  string
	// todo 待删除
	ColumnName string
	NonUnique  bool
	// IType
	IType IndexType
	// Columns
	Columns []ColumnMeta
}

// TableMeta
type TableMeta struct {
	// TableName
	TableName string
	// Columns
	Columns map[string]ColumnMeta
	// Indexs
	Indexs      map[string]IndexMeta
	ColumnNames []string
}

func (m TableMeta) IsEmpty() bool {
	return m.TableName == ""
}

func (m TableMeta) GetPrimaryKeyOnlyName() []string {
	keys := make([]string, 0)
	for _, index := range m.Indexs {
		if index.IType == IndexTypePrimaryKey {
			for _, column := range index.Columns {
				keys = append(keys, column.ColumnName)
			}
		}
	}

	return keys
}
