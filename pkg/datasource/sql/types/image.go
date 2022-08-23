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
	bIndex int32
	before RecordImages
	aIndex int32
	after  RecordImages
}

// AppendBeofreImages
func (r *RoundRecordImage) AppendBeofreImages(images []*RecordImage) {
	for _, image := range images {
		r.AppendBeofreImage(image)
	}
}

// AppendBeofreImage
func (r *RoundRecordImage) AppendBeofreImage(image *RecordImage) {
	r.bIndex++
	image.index = r.bIndex

	r.before = append(r.before, image)
}

// AppendAfterImages
func (r *RoundRecordImage) AppendAfterImages(images []*RecordImage) {
	for _, image := range images {
		r.AppendAfterImage(image)
	}
}

// AppendAfterImage
func (r *RoundRecordImage) AppendAfterImage(image *RecordImage) {
	r.aIndex++
	image.index = r.aIndex

	r.after = append(r.after, image)
}

func (r *RoundRecordImage) BeofreImages() RecordImages {
	return r.before
}

func (r *RoundRecordImage) AfterImages() RecordImages {
	return r.after
}

func (r *RoundRecordImage) IsEmpty() bool {
	return false
}

type RecordImages []*RecordImage

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
	// index
	index int32
	// Table table name
	Table string
	// SQLType sql type
	SQLType SQLType
	// Rows
	Rows []RowImage
}

// RowImage Mirror data information information
type RowImage struct {
	// Columns All columns of image data
	Columns []ColumnImage
}

// ColumnImage The mirror data information of the column
type ColumnImage struct {
	// KeyType index type
	KeyType string
	// Name column name
	Name string
	// Type column type
	Type gosql.ColumnType
	// Value column value
	Value interface{}
}
