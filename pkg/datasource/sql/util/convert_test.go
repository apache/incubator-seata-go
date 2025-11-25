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

package util

import (
	"database/sql"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDbVersion(t *testing.T) {
	version1 := "3.1.2"
	v1Int, err1 := ConvertDbVersion(version1)
	assert.NoError(t, err1)

	version2 := "3.1.3"
	v2Int, err2 := ConvertDbVersion(version2)
	assert.NoError(t, err2)

	assert.Less(t, v1Int, v2Int)

	version3 := "3.1.3"
	v3Int, err3 := ConvertDbVersion(version3)
	assert.NoError(t, err3)
	assert.Equal(t, v2Int, v3Int)

	// Test incompatible version format
	_, err4 := ConvertDbVersion("1.2.3.4.5")
	assert.Error(t, err4)

	// Test version with hyphen
	v5Int, err5 := ConvertDbVersion("1.2.3-SNAPSHOT")
	assert.NoError(t, err5)
	assert.Equal(t, 1020300, v5Int)
}

func TestConvertDbVersionWithHyphen(t *testing.T) {
	version := "5.7.30-log"
	v, err := ConvertDbVersion(version)
	assert.NoError(t, err)
	assert.Greater(t, v, 0)

	// Compare with non-hyphenated version
	version2 := "5.7.30"
	v2, err2 := ConvertDbVersion(version2)
	assert.NoError(t, err2)
	assert.Equal(t, v, v2) // Should be equal as we only parse the numeric part
}

func TestConvertDbVersionInvalidFormat(t *testing.T) {
	version := "1.2.3.4.5"
	_, err := ConvertDbVersion(version)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible version format")
}

func TestConvertDbVersionSinglePart(t *testing.T) {
	version := "8"
	v, err := ConvertDbVersion(version)
	assert.NoError(t, err)
	assert.Equal(t, 800, v)
}

func TestConvertDbVersionTwoParts(t *testing.T) {
	version := "8.0"
	v, err := ConvertDbVersion(version)
	assert.NoError(t, err)
	assert.Equal(t, 80000, v)
}

// Test convertAssignRows with string source
func TestConvertAssignRowsStringToString(t *testing.T) {
	src := "test"
	var dest string
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", dest)
}

func TestConvertAssignRowsStringToBytes(t *testing.T) {
	src := "test"
	var dest []byte
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), dest)
}

func TestConvertAssignRowsStringToRawBytes(t *testing.T) {
	src := "test"
	var dest sql.RawBytes
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, sql.RawBytes("test"), dest)
}

func TestConvertAssignRowsStringToNilPointer(t *testing.T) {
	src := "test"
	var dest *string
	err := convertAssignRows(dest, src, nil)
	assert.Error(t, err)
	assert.Equal(t, errNilPtr, err)
}

// Test convertAssignRows with []byte source
func TestConvertAssignRowsBytesToString(t *testing.T) {
	src := []byte("test")
	var dest string
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", dest)
}

func TestConvertAssignRowsBytesToInterface(t *testing.T) {
	src := []byte("test")
	var dest interface{}
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), dest)
}

func TestConvertAssignRowsBytesToBytes(t *testing.T) {
	src := []byte("test")
	var dest []byte
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), dest)
	// Verify it's a clone
	src[0] = 'x'
	assert.NotEqual(t, src, dest)
}

func TestConvertAssignRowsBytesToRawBytes(t *testing.T) {
	src := []byte("test")
	var dest sql.RawBytes
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, sql.RawBytes("test"), dest)
}

// Test convertAssignRows with time.Time source
func TestConvertAssignRowsTimeToTime(t *testing.T) {
	now := time.Now()
	var dest time.Time
	err := convertAssignRows(&dest, now, nil)
	assert.NoError(t, err)
	assert.Equal(t, now, dest)
}

func TestConvertAssignRowsTimeToString(t *testing.T) {
	now := time.Now()
	var dest string
	err := convertAssignRows(&dest, now, nil)
	assert.NoError(t, err)
	assert.Equal(t, now.Format(time.RFC3339Nano), dest)
}

func TestConvertAssignRowsTimeToBytes(t *testing.T) {
	now := time.Now()
	var dest []byte
	err := convertAssignRows(&dest, now, nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte(now.Format(time.RFC3339Nano)), dest)
}

func TestConvertAssignRowsTimeToRawBytes(t *testing.T) {
	now := time.Now()
	var dest sql.RawBytes
	err := convertAssignRows(&dest, now, nil)
	assert.NoError(t, err)
	expected := now.AppendFormat(nil, time.RFC3339Nano)
	assert.Equal(t, sql.RawBytes(expected), dest)
}

// Test convertAssignRows with nil source
func TestConvertAssignRowsNilToInterface(t *testing.T) {
	var dest interface{}
	err := convertAssignRows(&dest, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, dest)
}

func TestConvertAssignRowsNilToBytes(t *testing.T) {
	var dest []byte
	err := convertAssignRows(&dest, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, dest)
}

func TestConvertAssignRowsNilToRawBytes(t *testing.T) {
	var dest sql.RawBytes
	err := convertAssignRows(&dest, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, dest)
}

// Test convertAssignRows with numeric types
func TestConvertAssignRowsIntToString(t *testing.T) {
	var dest string
	err := convertAssignRows(&dest, int64(123), nil)
	assert.NoError(t, err)
	assert.Equal(t, "123", dest)
}

func TestConvertAssignRowsFloatToString(t *testing.T) {
	var dest string
	err := convertAssignRows(&dest, float64(123.45), nil)
	assert.NoError(t, err)
	assert.Contains(t, dest, "123.45")
}

func TestConvertAssignRowsBoolToString(t *testing.T) {
	var dest string
	err := convertAssignRows(&dest, true, nil)
	assert.NoError(t, err)
	assert.Equal(t, "true", dest)
}

func TestConvertAssignRowsIntToBytes(t *testing.T) {
	var dest []byte
	err := convertAssignRows(&dest, int64(123), nil)
	assert.NoError(t, err)
	assert.Equal(t, []byte("123"), dest)
}

func TestConvertAssignRowsIntToBool(t *testing.T) {
	var dest bool
	err := convertAssignRows(&dest, int64(1), nil)
	assert.NoError(t, err)
	assert.True(t, dest)
}

func TestConvertAssignRowsToInterface(t *testing.T) {
	var dest interface{}
	src := "test"
	err := convertAssignRows(&dest, src, nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", dest)
}

// Test convertAssignRows with reflection-based conversions
func TestConvertAssignRowsStringToInt(t *testing.T) {
	var dest int
	err := convertAssignRows(&dest, "123", nil)
	assert.NoError(t, err)
	assert.Equal(t, 123, dest)
}

func TestConvertAssignRowsStringToInt64(t *testing.T) {
	var dest int64
	err := convertAssignRows(&dest, "123", nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(123), dest)
}

func TestConvertAssignRowsStringToUint(t *testing.T) {
	var dest uint
	err := convertAssignRows(&dest, "123", nil)
	assert.NoError(t, err)
	assert.Equal(t, uint(123), dest)
}

func TestConvertAssignRowsStringToFloat64(t *testing.T) {
	var dest float64
	err := convertAssignRows(&dest, "123.45", nil)
	assert.NoError(t, err)
	assert.InDelta(t, 123.45, dest, 0.0001)
}

func TestConvertAssignRowsInvalidIntConversion(t *testing.T) {
	var dest int
	err := convertAssignRows(&dest, "invalid", nil)
	assert.Error(t, err)
}

func TestConvertAssignRowsIntOverflow(t *testing.T) {
	var dest int8
	err := convertAssignRows(&dest, "1000", nil)
	assert.Error(t, err)
}

func TestConvertAssignRowsNullToInt(t *testing.T) {
	var dest int
	err := convertAssignRows(&dest, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "converting NULL")
}

func TestConvertAssignRowsNullToFloat(t *testing.T) {
	var dest float64
	err := convertAssignRows(&dest, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "converting NULL")
}

func TestConvertAssignRowsNullToString(t *testing.T) {
	var dest string
	err := convertAssignRows(&dest, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "converting NULL")
}

// Test pointer types
func TestConvertAssignRowsNilToPointer(t *testing.T) {
	var dest *string
	err := convertAssignRows(&dest, nil, nil)
	assert.NoError(t, err)
	assert.Nil(t, dest)
}

func TestConvertAssignRowsValueToPointer(t *testing.T) {
	var dest *string
	err := convertAssignRows(&dest, "test", nil)
	assert.NoError(t, err)
	require.NotNil(t, dest)
	assert.Equal(t, "test", *dest)
}

// Test type conversion and assignability
func TestConvertAssignRowsAssignableTypes(t *testing.T) {
	type MyString string
	var dest MyString
	err := convertAssignRows(&dest, "test", nil)
	assert.NoError(t, err)
	assert.Equal(t, MyString("test"), dest)
}

func TestConvertAssignRowsConvertibleTypes(t *testing.T) {
	type MyInt int
	var dest MyInt
	err := convertAssignRows(&dest, "123", nil)
	assert.NoError(t, err)
	assert.Equal(t, MyInt(123), dest)
}

func TestConvertAssignRowsNotPointer(t *testing.T) {
	var dest string
	err := convertAssignRows(dest, "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "destination not a pointer")
}

func TestConvertAssignRowsUnsupportedConversion(t *testing.T) {
	type MyStruct struct{}
	var dest MyStruct
	err := convertAssignRows(&dest, "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported Scan")
}

// Test custom Scanner implementation
type customScanner struct {
	value string
}

func (cs *customScanner) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	cs.value = asString(src)
	return nil
}

func TestConvertAssignRowsCustomScanner(t *testing.T) {
	var dest customScanner
	err := convertAssignRows(&dest, "test", nil)
	assert.NoError(t, err)
	assert.Equal(t, "test", dest.value)
}

// Test helper functions
func TestCloneBytes(t *testing.T) {
	src := []byte("test")
	dest := cloneBytes(src)
	assert.Equal(t, src, dest)
	src[0] = 'x'
	assert.NotEqual(t, src, dest)
}

func TestCloneBytesNil(t *testing.T) {
	var src []byte
	dest := cloneBytes(src)
	assert.Nil(t, dest)
}

func TestAsStringVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"[]byte", []byte("test"), "test"},
		{"int", int(123), "123"},
		{"int64", int64(123), "123"},
		{"uint", uint(123), "123"},
		{"float64", float64(123.45), "123.45"},
		{"float32", float32(123.45), "123.45"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := asString(tt.input)
			if tt.name == "float32" || tt.name == "float64" {
				assert.Contains(t, result, "123.45")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestAsBytesVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
		ok       bool
	}{
		{"int", int(123), "123", true},
		{"int64", int64(123), "123", true},
		{"uint", uint(123), "123", true},
		{"float64", float64(123.45), "123.45", true},
		{"float32", float32(123.45), "123.45", true},
		{"bool", true, "true", true},
		{"string", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := reflect.ValueOf(tt.input)
			result, ok := asBytes(nil, rv)
			assert.Equal(t, tt.ok, ok)
			if ok {
				if tt.name == "float32" || tt.name == "float64" {
					assert.Contains(t, string(result), "123.45")
				} else {
					assert.Equal(t, tt.expected, string(result))
				}
			}
		})
	}
}

func TestAsBytesUnsupportedType(t *testing.T) {
	type MyStruct struct{}
	rv := reflect.ValueOf(MyStruct{})
	_, ok := asBytes(nil, rv)
	assert.False(t, ok)
}

func TestStrconvErr(t *testing.T) {
	// Test with NumError
	numErr := &strconv.NumError{
		Func: "ParseInt",
		Num:  "abc",
		Err:  strconv.ErrSyntax,
	}
	result := strconvErr(numErr)
	assert.Equal(t, strconv.ErrSyntax, result)

	// Test with regular error
	regularErr := errors.New("regular error")
	result = strconvErr(regularErr)
	assert.Equal(t, regularErr, result)
}

func TestCalculatePartValue(t *testing.T) {
	assert.Equal(t, 3000000, calculatePartValue(3, 3, 0))
	assert.Equal(t, 10000, calculatePartValue(1, 3, 1))
	assert.Equal(t, 200, calculatePartValue(2, 3, 2))
}

// Test edge cases
func TestConvertAssignRowsEmptyString(t *testing.T) {
	var dest string
	err := convertAssignRows(&dest, "", nil)
	assert.NoError(t, err)
	assert.Equal(t, "", dest)
}

func TestConvertAssignRowsZeroInt(t *testing.T) {
	var dest int
	err := convertAssignRows(&dest, "0", nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, dest)
}

func TestConvertAssignRowsNegativeInt(t *testing.T) {
	var dest int
	err := convertAssignRows(&dest, "-123", nil)
	assert.NoError(t, err)
	assert.Equal(t, -123, dest)
}

func TestConvertAssignRowsScientificNotation(t *testing.T) {
	var dest float64
	err := convertAssignRows(&dest, "1.23e2", nil)
	assert.NoError(t, err)
	assert.InDelta(t, 123.0, dest, 0.0001)
}
