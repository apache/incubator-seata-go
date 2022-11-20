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
	return len(r.before) == 0 && len(r.after) == 0
}

func (r *RoundRecordImage) IsBeforeAfterSizeEq() bool {
	return len(r.before) == len(r.after)
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
	index int32 `json:"-"`
	// TableName table name
	TableName string `json:"tableName"`
	// SQLType sql type
	SQLType SQLType `json:"sqlType"`
	// Rows data row
	Rows []RowImage `json:"rows"`
	// TableMeta table information schema
	TableMeta TableMeta `json:"-"`
}

// RowImage Mirror data information information
type RowImage struct {
	// Columns All columns of image data
	Columns []ColumnImage `json:"fields"`
}

func (r *RowImage) GetColumnMap() map[string]*ColumnImage {
	m := make(map[string]*ColumnImage, 0)
	for _, column := range r.Columns {
		m[column.ColumnName] = &column
	}
	return m
}

// PrimaryKeys Primary keys list.
func (r *RowImage) PrimaryKeys(cols []ColumnImage) []ColumnImage {
	var pkFields []ColumnImage
	for key, _ := range cols {
		if cols[key].KeyType == PrimaryKey.Number() {
			pkFields = append(pkFields, cols[key])
		}
	}

	return pkFields
}

// NonPrimaryKeys get non-primary keys
func (r *RowImage) NonPrimaryKeys(cols []ColumnImage) []ColumnImage {
	var nonPkFields []ColumnImage
	for key, _ := range cols {
		if cols[key].KeyType != PrimaryKey.Number() {
			nonPkFields = append(nonPkFields, cols[key])
		}
	}

	return nonPkFields
}

// ColumnImage The mirror data information of the column
type ColumnImage struct {
	// KeyType index type
	KeyType IndexType `json:"keyType"`
	// ColumnName column name
	ColumnName string `json:"name"`
	// Type column type
	Type int16 `json:"type"`
	// Value column value
	Value interface{} `json:"value"`
}
