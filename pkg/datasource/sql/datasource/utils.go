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

package datasource

import (
	"database/sql"
	"reflect"
)

type nullTime = sql.NullTime

var (
	ScanTypeFloat32   = reflect.TypeOf(float32(0))
	ScanTypeFloat64   = reflect.TypeOf(float64(0))
	ScanTypeInt8      = reflect.TypeOf(int8(0))
	ScanTypeInt16     = reflect.TypeOf(int16(0))
	ScanTypeInt32     = reflect.TypeOf(int32(0))
	ScanTypeInt64     = reflect.TypeOf(int64(0))
	ScanTypeNullFloat = reflect.TypeOf(sql.NullFloat64{})
	ScanTypeNullInt   = reflect.TypeOf(sql.NullInt64{})
	ScanTypeNullTime  = reflect.TypeOf(nullTime{})
	ScanTypeUint8     = reflect.TypeOf(uint8(0))
	ScanTypeUint16    = reflect.TypeOf(uint16(0))
	ScanTypeUint32    = reflect.TypeOf(uint32(0))
	ScanTypeUint64    = reflect.TypeOf(uint64(0))
	ScanTypeRawBytes  = reflect.TypeOf(sql.RawBytes{})
	ScanTypeUnknown   = reflect.TypeOf(new(interface{}))
)

func GetScanSlice(types []*sql.ColumnType) []interface{} {
	scanSlice := make([]interface{}, 0, len(types))
	for _, tpy := range types {
		switch tpy.ScanType() {
		case ScanTypeFloat32:
			scanVal := float32(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeFloat64:
			scanVal := float64(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeInt8:
			scanVal := int8(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeInt16:
			scanVal := int16(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeInt32:
			scanVal := int32(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeInt64:
			scanVal := int64(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeNullFloat:
			scanVal := sql.NullFloat64{}
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeNullInt:
			scanVal := sql.NullInt64{}
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeNullTime:
			scanVal := sql.NullTime{}
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeUint8:
			scanVal := uint8(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeUint16:
			scanVal := uint16(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeUint32:
			scanVal := uint32(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeUint64:
			scanVal := uint64(0)
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeRawBytes:
			scanVal := ""
			scanSlice = append(scanSlice, &scanVal)
		case ScanTypeUnknown:
			scanVal := new(interface{})
			scanSlice = append(scanSlice, &scanVal)
		}
	}
	return scanSlice
}

func DeepEqual(x, y interface{}) bool {
	if x == nil || y == nil {
		return x == y
	}

	typx := reflect.ValueOf(x)
	typy := reflect.ValueOf(y)

	switch typx.Kind() {
	case reflect.Ptr:
		typx = typx.Elem()
	}

	switch typy.Kind() {
	case reflect.Ptr:
		typy = typy.Elem()
	}

	flx, okx := parseFloatIfOk(typx)
	fly, oky := parseFloatIfOk(typy)
	if okx && oky {
		return flx == fly
	}

	return reflect.DeepEqual(typx.Interface(), typy.Interface())
}

func parseFloatIfOk(val reflect.Value) (float64, bool) {
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(val.Uint()), true
	case reflect.Float32, reflect.Float64:
		return float64(val.Float()), true
	}
	return 0, false
}
