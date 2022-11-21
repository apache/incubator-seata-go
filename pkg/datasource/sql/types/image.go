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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
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

var _ json.Unmarshaler = (*ColumnImage)(nil)
var _ json.Marshaler = (*ColumnImage)(nil)

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

// columnImageAlias support ColumnImage to json serialization
type columnImageAlias ColumnImage

// columnImageSerialization support ColumnImage to json serialization
type columnImageSerialization struct {
	ValueType string `json:"valueType"`
	*columnImageAlias
}

func (c *ColumnImage) MarshalJSON() ([]byte, error) {
	tmp := columnImageSerialization{
		columnImageAlias: (*columnImageAlias)(c),
	}
	tmp.ValueType = fmt.Sprintf("%T", tmp.Value)
	data, err := json.Marshal(&tmp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *ColumnImage) UnmarshalJSON(data []byte) error {
	tmp := columnImageSerialization{
		columnImageAlias: (*columnImageAlias)(c),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	switch tmp.ValueType {
	case getTypeStr([]uint8{}):
		if str, ok := tmp.Value.(string); ok {
			// base64 decode
			if strB, err := base64.StdEncoding.DecodeString(str); err == nil {
				tmp.columnImageAlias.Value = strB
			}
		}

	case getTypeStr(int32(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = int32(float64Tmp)

	case getTypeStr(uint8(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = uint8(float64Tmp)

	case getTypeStr(int32(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = int32(float64Tmp)

	case getTypeStr(0):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = int(float64Tmp)
	case getTypeStr(uint(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = uint(float64Tmp)
	case getTypeStr(int8(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = int8(float64Tmp)

	case getTypeStr(uint8(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = uint8(float64Tmp)

	case getTypeStr(int16(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = int16(float64Tmp)

	case getTypeStr(uint16(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = uint16(float64Tmp)

	case getTypeStr(int32(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = int32(float64Tmp)

	case getTypeStr(uint32(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = uint32(float64Tmp)

	case getTypeStr(int64(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = int64(float64Tmp)

	case getTypeStr(uint64(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = uint64(float64Tmp)

	case getTypeStr(float32(0)):
		float64Tmp, _ := tmp.Value.(float64)
		tmp.columnImageAlias.Value = float32(float64Tmp)

	case getTypeStr(float64(0)):
		tmp.Value, _ = tmp.Value.(float64)

	case getTypeStr(time.Now()):
		strTmp, _ := tmp.Value.(string)
		tmp.columnImageAlias.Value, _ = time.ParseInLocation("2006-01-02T15:04:05.999999Z07:00", strTmp, time.Local)
	}
	return nil
}

func getTypeStr(src interface{}) string {
	return fmt.Sprintf("%T", src)
}
