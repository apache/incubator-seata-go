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
	gosql "database/sql"
)

// RoundRecordImage Front and rear mirror data
type RoundRecordImage struct {
	BeforeImages RecordImages
	AfterImages  RecordImages
}

func (r *RoundRecordImage) IsEmpty() bool {
	return false
}

type RecordImages []RecordImage

// Reserve The order of reverse mirrors, when executing undo, needs to be executed in reverse
func (rs RecordImages) Reserve() {
	l := 0
	r := len(rs) - 1

	for l <= r {
		rs[l], rs[r] = rs[r], rs[l]

		l++
		r--
	}
}

// RecordImage
type RecordImage struct {
	// Index
	Index int32
	// Table
	Table string
	// SQLType
	SQLType string
	// Rows
	Rows []RowImage
}

// RowImage Mirror data information information
type RowImage struct {
	Columns []ColumnImage
}

// ColumnImage The mirror data information of the column
type ColumnImage struct {
	// Name column name
	Name string
	// Type column type
	Type gosql.ColumnType
	// Value column value
	Value interface{}
}
