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
	"reflect"
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

func (rs RecordImages) IsEmptyImage() bool {
	if len(rs) == 0 {
		return true
	}
	for _, r := range rs {
		if r == nil || len(r.Rows) == 0 {
			continue
		}
		return false
	}
	return true
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
	TableMeta *TableMeta `json:"-"`
}

func NewEmptyRecordImage(tableMeta *TableMeta, sqlType SQLType) *RecordImage {
	return &RecordImage{
		TableName: tableMeta.TableName,
		TableMeta: tableMeta,
		SQLType:   sqlType,
	}
}

// RowImage Mirror data information information
type RowImage struct {
	// Columns All columns of image data
	Columns []ColumnImage `json:"fields"`
}

func (r *RowImage) GetColumnMap() map[string]*ColumnImage {
	m := make(map[string]*ColumnImage, 0)
	for _, column := range r.Columns {
		tmpColumn := column
		m[column.ColumnName] = &tmpColumn
	}
	return m
}

// PrimaryKeys Primary keys list.
func (r *RowImage) PrimaryKeys(cols []ColumnImage) []ColumnImage {
	var pkFields []ColumnImage
	for key := range cols {
		if cols[key].KeyType == PrimaryKey.Number() {
			pkFields = append(pkFields, cols[key])
		}
	}

	return pkFields
}

// NonPrimaryKeys get non-primary keys
func (r *RowImage) NonPrimaryKeys(cols []ColumnImage) []ColumnImage {
	var nonPkFields []ColumnImage
	for key := range cols {
		if cols[key].KeyType != PrimaryKey.Number() {
			nonPkFields = append(nonPkFields, cols[key])
		}
	}

	return nonPkFields
}

var _ json.Unmarshaler = (*ColumnImage)(nil)
var _ json.Marshaler = (*ColumnImage)(nil)

type CommonValue struct {
	Value interface{}
}

// ColumnImage The mirror data information of the column
type ColumnImage struct {
	// KeyType index type
	KeyType IndexType `json:"keyType"`
	// ColumnName column name
	ColumnName string `json:"name"`
	// ColumnType column type
	ColumnType JDBCType `json:"type"`
	// Value column value
	Value interface{} `json:"value"`
}

type columnImageAlias ColumnImage

func (c *ColumnImage) MarshalJSON() ([]byte, error) {
	if c == nil || c.Value == nil {
		return json.Marshal(*c)
	}
	value := c.Value
	if t, ok := c.Value.(time.Time); ok {
		value = t.Format(time.RFC3339Nano)
	}
	return json.Marshal(&columnImageAlias{
		KeyType:    c.KeyType,
		ColumnName: c.ColumnName,
		ColumnType: c.ColumnType,
		Value:      value,
	})
}

func (c *ColumnImage) UnmarshalJSON(data []byte) error {
	var err error
	tmpImage := make(map[string]interface{})
	if err := json.Unmarshal(data, &tmpImage); err != nil {
		return err
	}
	var (
		keyType     string
		columnType  int16
		columnName  string
		value       interface{}
		actualValue interface{}
	)
	keyType = tmpImage["keyType"].(string)
	columnType = int16(int64(tmpImage["type"].(float64)))
	columnName = tmpImage["name"].(string)
	value = tmpImage["value"]

	if value != nil {
		switch JDBCType(columnType) {
		case JDBCTypeReal: // 4 Bytes
			actualValue = value.(float32)
		case JDBCTypeDecimal, JDBCTypeDouble: // 8 Bytes
			actualValue = value.(float64)
		case JDBCTypeTinyInt: // 1 Bytes
			actualValue = int8(value.(float64))
		case JDBCTypeSmallInt: // 2 Bytes
			actualValue = int16(value.(float64))
		case JDBCTypeInteger: // 4 Bytes
			actualValue = int32(value.(float64))
		case JDBCTypeBigInt: // 8Bytes
			actualValue = int64(value.(float64))
		case JDBCTypeTimestamp: // 4 Bytes
			actualValue, err = time.Parse(time.RFC3339Nano, value.(string))
			if err != nil {
				return err
			}
		case JDBCTypeDate: // 3Bytes
			actualValue, err = time.Parse(time.RFC3339Nano, value.(string))
			if err != nil {
				return err
			}
		case JDBCTypeTime: // 3Bytes
			actualValue, err = time.Parse(time.RFC3339Nano, value.(string))
			if err != nil {
				return err
			}
		case JDBCTypeChar, JDBCTypeVarchar, JDBCTypeLongVarchar:
			var val []byte
			if val, err = base64.StdEncoding.DecodeString(value.(string)); err != nil {
				val = []byte(value.(string))
			}
			actualValue = string(val)
		case JDBCTypeBinary, JDBCTypeVarBinary, JDBCTypeLongVarBinary, JDBCTypeBit:
			actualValue = value
		}
	}
	*c = ColumnImage{
		KeyType:    ParseIndexType(keyType),
		ColumnName: columnName,
		ColumnType: JDBCType(columnType),
		Value:      actualValue,
	}
	return nil
}

func (c *ColumnImage) GetActualValue() interface{} {
	if c.Value == nil {
		return nil
	}
	value := reflect.ValueOf(c.Value)
	kind := reflect.TypeOf(c.Value).Kind()
	switch kind {
	case reflect.Ptr:
		return value.Elem().Interface()
	}
	return c.Value
}
