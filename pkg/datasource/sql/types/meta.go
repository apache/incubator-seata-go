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
	"fmt"
	"reflect"
	"sort"
)

// ColumnMeta
type ColumnMeta struct {
	// Schema
	Schema string
	Table  string
	// ColumnDef  the column default
	ColumnDef []byte
	// Autoincrement
	Autoincrement bool
	// todo get columnType
	//ColumnTypeInfo *sql.ColumnType
	ColumnName         string
	ColumnType         string
	DatabaseType       int32
	DatabaseTypeString string
	ColumnKey          string
	IsNullable         int8
	Extra              string
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

func (m TableMeta) GetPrimaryKeyMap() map[string]ColumnMeta {
	pk := make(map[string]ColumnMeta)
	for _, index := range m.Indexs {
		if index.IType == IndexTypePrimaryKey {
			for _, column := range index.Columns {
				pk[column.ColumnName] = column
			}
		}
	}
	return pk
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

	// need sort again according the m.ColumnNames
	order := make(map[string]int, len(m.ColumnNames))
	for i, name := range m.ColumnNames {
		order[name] = i
	}
	sort.Slice(keys, func(i, j int) bool {
		return order[keys[i]] < order[keys[j]]
	})
	return keys
}

// GetPrimaryKeyType get PK database type
func (m TableMeta) GetPrimaryKeyType() (int32, error) {
	for _, index := range m.Indexs {
		if index.IType == IndexTypePrimaryKey {
			for i := range index.Columns {
				return index.Columns[i].DatabaseType, nil
			}
		}
	}
	return 0, fmt.Errorf("get primary key type error")
}

// GetPrimaryKeyTypeStrMap get all PK type to map
func (m TableMeta) GetPrimaryKeyTypeStrMap() (map[string]string, error) {
	pkMap := make(map[string]string)
	for _, index := range m.Indexs {
		if index.IType == IndexTypePrimaryKey {
			for i := range index.Columns {
				pkMap[index.ColumnName] = index.Columns[i].DatabaseTypeString
			}
		}
	}
	if len(pkMap) == 0 {
		return nil, fmt.Errorf("get primary key type error")
	}
	return pkMap, nil
}
