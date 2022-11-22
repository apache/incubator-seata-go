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
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
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

//var _ json.Unmarshaler = (*ColumnImage)(nil)
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

//// MarshalJSON returns m as the JSON encoding of m.
//func (m CommonValue) MarshalJSON() ([]byte, error) {
//	if m.Value == nil {
//		return []byte("null"), nil
//	}
//	v := reflect.ValueOf(m.Value)
//	switch val := v.(type) {
//	case int8, int16, int32, int64, bool, uint8, uint16, uint32, uint64, float32, float64, complex64, complex128:
//
//	}
//	return m, nil
//}
//
//// UnmarshalJSON sets *m to a copy of data.
//func (m *CommonValue) UnmarshalJSON(data []byte) error {
//	if m == nil {
//		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
//	}
//	*m = append((*m)[0:0], data...)
//	return nil
//}

func (c *ColumnImage) MarshalJSON() ([]byte, error) {
	return json.Marshal(*c)
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
		valueBytes  []byte
		valueBuffer *bytes.Buffer
		actualValue interface{}
	)
	keyType = tmpImage["keyType"].(string)
	columnType = int16(int64(tmpImage["type"].(float64)))
	columnName = tmpImage["name"].(string)
	actualValue = tmpImage["value"]

	switch JDBCType(columnType) {
	case JDBCTypeReal: // 4 Bytes
		var val float32
		binary.Read(valueBuffer, binary.BigEndian, &val)
		actualValue = val
	case JDBCTypeDecimal, JDBCTypeDouble: // 8 Bytes
		var val float64
		binary.Read(valueBuffer, binary.BigEndian, &val)
		actualValue = val
	case JDBCTypeTinyInt: // 1 Bytes
		var val int8
		binary.Read(valueBuffer, binary.BigEndian, &val)
		actualValue = val
	case JDBCTypeSmallInt: // 2 Bytes
		var val int16
		binary.Read(valueBuffer, binary.BigEndian, &val)
		actualValue = val
	case JDBCTypeInteger: // 4 Bytes
		var val int32
		binary.Read(valueBuffer, binary.BigEndian, &val)
		actualValue = val
	case JDBCTypeBigInt: // 8Bytes
		var val int64
		binary.Read(valueBuffer, binary.BigEndian, &val)
		actualValue = val
	case JDBCTypeTimestamp: // 4 Bytes
		var val = make([]byte, 0)
		// todo
		if val, err = base64.StdEncoding.DecodeString(actualValue.(string)); err != nil {
			return err
		}
		actualValue = string(val)
	case JDBCTypeDate: // 3Bytes
		var val = make([]byte, 0)
		// todo
		if val, err = base64.StdEncoding.DecodeString(actualValue.(string)); err != nil {
			return err
		}
		actualValue = string(val)
	case JDBCTypeTime: // 3Bytes
		var val = make([]byte, 0)
		// todo
		if val, err = base64.StdEncoding.DecodeString(actualValue.(string)); err != nil {
			return err
		}
		actualValue = string(val)
	case JDBCTypeChar, JDBCTypeVarchar:
		var val = make([]byte, 0)
		if val, err = base64.StdEncoding.DecodeString(actualValue.(string)); err != nil {
			return err
		}
		actualValue = string(val)
	case JDBCTypeBinary, JDBCTypeVarBinary, JDBCTypeLongVarBinary, JDBCTypeBit:
		actualValue = valueBytes
	}
	*c = ColumnImage{
		KeyType:    ParseIndexType(keyType),
		ColumnName: columnName,
		ColumnType: JDBCType(columnType),
		Value:      actualValue,
	}
	return nil
}

func getTypeStr(src interface{}) string {
	return fmt.Sprintf("%T", src)
}
