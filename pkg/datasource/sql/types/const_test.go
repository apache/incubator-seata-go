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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQLCodeToJava(t *testing.T) {
	tests := []struct {
		name      string
		mysqlType MySQLDefCode
		expected  JDBCType
	}{
		{"FIELD_TYPE_DECIMAL", FIELD_TYPE_DECIMAL, JDBCTypeDecimal},
		{"FIELD_TYPE_NEW_DECIMAL", FIELD_TYPE_NEW_DECIMAL, JDBCTypeDecimal},
		{"FIELD_TYPE_TINY", FIELD_TYPE_TINY, JDBCTypeTinyInt},
		{"FIELD_TYPE_SHORT", FIELD_TYPE_SHORT, JDBCTypeSmallInt},
		{"FIELD_TYPE_LONG", FIELD_TYPE_LONG, JDBCTypeInteger},
		{"FIELD_TYPE_FLOAT", FIELD_TYPE_FLOAT, JDBCTypeReal},
		{"FIELD_TYPE_DOUBLE", FIELD_TYPE_DOUBLE, JDBCTypeDouble},
		{"FIELD_TYPE_NULL", FIELD_TYPE_NULL, JDBCTypeNull},
		{"FIELD_TYPE_TIMESTAMP", FIELD_TYPE_TIMESTAMP, JDBCTypeTimestamp},
		{"FIELD_TYPE_LONGLONG", FIELD_TYPE_LONGLONG, JDBCTypeBigInt},
		{"FIELD_TYPE_INT24", FIELD_TYPE_INT24, JDBCTypeInteger},
		{"FIELD_TYPE_DATE", FIELD_TYPE_DATE, JDBCTypeDate},
		{"FIELD_TYPE_TIME", FIELD_TYPE_TIME, JDBCTypeTime},
		{"FIELD_TYPE_DATETIME", FIELD_TYPE_DATETIME, JDBCTypeTimestamp},
		{"FIELD_TYPE_YEAR", FIELD_TYPE_YEAR, JDBCTypeDate},
		{"FIELD_TYPE_NEWDATE", FIELD_TYPE_NEWDATE, JDBCTypeDate},
		{"FIELD_TYPE_ENUM", FIELD_TYPE_ENUM, JDBCTypeChar},
		{"FIELD_TYPE_SET", FIELD_TYPE_SET, JDBCTypeChar},
		{"FIELD_TYPE_TINY_BLOB", FIELD_TYPE_TINY_BLOB, JDBCTypeVarBinary},
		{"FIELD_TYPE_MEDIUM_BLOB", FIELD_TYPE_MEDIUM_BLOB, JDBCTypeLongVarBinary},
		{"FIELD_TYPE_LONG_BLOB", FIELD_TYPE_LONG_BLOB, JDBCTypeLongVarBinary},
		{"FIELD_TYPE_BLOB", FIELD_TYPE_BLOB, JDBCTypeLongVarBinary},
		{"FIELD_TYPE_VAR_STRING", FIELD_TYPE_VAR_STRING, JDBCTypeVarchar},
		{"FIELD_TYPE_VARCHAR", FIELD_TYPE_VARCHAR, JDBCTypeVarchar},
		{"FIELD_TYPE_STRING", FIELD_TYPE_STRING, JDBCTypeChar},
		{"FIELD_TYPE_JSON", FIELD_TYPE_JSON, JDBCTypeChar},
		{"FIELD_TYPE_GEOMETRY", FIELD_TYPE_GEOMETRY, JDBCTypeBinary},
		{"FIELD_TYPE_BIT", FIELD_TYPE_BIT, JDBCTypeBit},
		{"Unknown type", MySQLDefCode(9999), JDBCTypeVarchar}, // default case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MySQLCodeToJava(tt.mysqlType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMySQLStrToJavaType(t *testing.T) {
	tests := []struct {
		name      string
		mysqlType string
		expected  JDBCType
	}{
		{"BIT", "BIT", JDBCTypeBit},
		{"bit lowercase", "bit", JDBCTypeBit},
		{"TINYINT", "TINYINT", JDBCTypeTinyInt},
		{"SMALLINT", "SMALLINT", JDBCTypeSmallInt},
		{"MEDIUMINT", "MEDIUMINT", JDBCTypeInteger},
		{"INT", "INT", JDBCTypeInteger},
		{"INTEGER", "INTEGER", JDBCTypeInteger},
		{"BIGINT", "BIGINT", JDBCTypeBigInt},
		{"INT24", "INT24", JDBCTypeInteger},
		{"REAL", "REAL", JDBCTypeDouble},
		{"FLOAT", "FLOAT", JDBCTypeReal},
		{"DECIMAL", "DECIMAL", JDBCTypeDecimal},
		{"NUMERIC", "NUMERIC", JDBCTypeDecimal},
		{"DOUBLE", "DOUBLE", JDBCTypeDouble},
		{"CHAR", "CHAR", JDBCTypeChar},
		{"VARCHAR", "VARCHAR", JDBCTypeVarchar},
		{"DATE", "DATE", JDBCTypeDate},
		{"TIME", "TIME", JDBCTypeTime},
		{"YEAR", "YEAR", JDBCTypeDate},
		{"TIMESTAMP", "TIMESTAMP", JDBCTypeTimestamp},
		{"DATETIME", "DATETIME", JDBCTypeTimestamp},
		{"TINYBLOB", "TINYBLOB", JDBCTypeBinary},
		{"BLOB", "BLOB", JDBCTypeLongVarBinary},
		{"MEDIUMBLOB", "MEDIUMBLOB", JDBCTypeLongVarBinary},
		{"LONGBLOB", "LONGBLOB", JDBCTypeLongVarBinary},
		{"TINYTEXT", "TINYTEXT", JDBCTypeVarchar},
		{"TEXT", "TEXT", JDBCTypeLongVarchar},
		{"MEDIUMTEXT", "MEDIUMTEXT", JDBCTypeLongVarchar},
		{"LONGTEXT", "LONGTEXT", JDBCTypeLongVarchar},
		{"ENUM", "ENUM", JDBCTypeChar},
		{"SET", "SET", JDBCTypeChar},
		{"GEOMETRY", "GEOMETRY", JDBCTypeBinary},
		{"BINARY", "BINARY", JDBCTypeBinary},
		{"VARBINARY", "VARBINARY", JDBCTypeVarBinary},
		{"JSON", "JSON", JDBCTypeChar},
		{"Unknown type", "UNKNOWN_TYPE", JDBCTypeOther}, // default case
		{"Mixed case", "VarChar", JDBCTypeVarchar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MySQLStrToJavaType(tt.mysqlType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFieldTypeConstants(t *testing.T) {
	// Test that FieldType constants have expected values
	assert.Equal(t, FieldType(0), FieldTypeDecimal)
	assert.Equal(t, FieldType(1), FieldTypeTiny)
	assert.Equal(t, FieldType(2), FieldTypeShort)
	assert.Equal(t, FieldType(3), FieldTypeLong)
	assert.Equal(t, FieldType(4), FieldTypeFloat)
	assert.Equal(t, FieldType(5), FieldTypeDouble)
	assert.Equal(t, FieldType(6), FieldTypeNULL)
	assert.Equal(t, FieldType(7), FieldTypeTimestamp)
	assert.Equal(t, FieldType(8), FieldTypeLongLong)
	assert.Equal(t, FieldType(9), FieldTypeInt24)
	assert.Equal(t, FieldType(10), FieldTypeDate)
	assert.Equal(t, FieldType(11), FieldTypeTime)
	assert.Equal(t, FieldType(12), FieldTypeDateTime)
	assert.Equal(t, FieldType(13), FieldTypeYear)
	assert.Equal(t, FieldType(14), FieldTypeNewDate)
	assert.Equal(t, FieldType(15), FieldTypeVarChar)
	assert.Equal(t, FieldType(16), FieldTypeBit)
}

func TestJDBCTypeConstants(t *testing.T) {
	// Test some key JDBC type constants
	assert.Equal(t, JDBCType(-7), JDBCTypeBit)
	assert.Equal(t, JDBCType(-6), JDBCTypeTinyInt)
	assert.Equal(t, JDBCType(5), JDBCTypeSmallInt)
	assert.Equal(t, JDBCType(4), JDBCTypeInteger)
	assert.Equal(t, JDBCType(-5), JDBCTypeBigInt)
	assert.Equal(t, JDBCType(6), JDBCTypeFloat)
	assert.Equal(t, JDBCType(7), JDBCTypeReal)
	assert.Equal(t, JDBCType(8), JDBCTypeDouble)
	assert.Equal(t, JDBCType(2), JDBCTypeNumberic)
	assert.Equal(t, JDBCType(3), JDBCTypeDecimal)
	assert.Equal(t, JDBCType(1), JDBCTypeChar)
	assert.Equal(t, JDBCType(12), JDBCTypeVarchar)
	assert.Equal(t, JDBCType(-1), JDBCTypeLongVarchar)
	assert.Equal(t, JDBCType(91), JDBCTypeDate)
	assert.Equal(t, JDBCType(92), JDBCTypeTime)
	assert.Equal(t, JDBCType(93), JDBCTypeTimestamp)
	assert.Equal(t, JDBCType(-2), JDBCTypeBinary)
	assert.Equal(t, JDBCType(-3), JDBCTypeVarBinary)
	assert.Equal(t, JDBCType(-4), JDBCTypeLongVarBinary)
	assert.Equal(t, JDBCType(0), JDBCTypeNull)
	assert.Equal(t, JDBCType(1111), JDBCTypeOther)
}

func TestMySQLDefCodeConstants(t *testing.T) {
	// Test some key MySQL def code constants
	assert.Equal(t, MySQLDefCode(0), FIELD_TYPE_DECIMAL)
	assert.Equal(t, MySQLDefCode(1), FIELD_TYPE_TINY)
	assert.Equal(t, MySQLDefCode(2), FIELD_TYPE_SHORT)
	assert.Equal(t, MySQLDefCode(3), FIELD_TYPE_LONG)
	assert.Equal(t, MySQLDefCode(4), FIELD_TYPE_FLOAT)
	assert.Equal(t, MySQLDefCode(5), FIELD_TYPE_DOUBLE)
	assert.Equal(t, MySQLDefCode(6), FIELD_TYPE_NULL)
	assert.Equal(t, MySQLDefCode(7), FIELD_TYPE_TIMESTAMP)
	assert.Equal(t, MySQLDefCode(8), FIELD_TYPE_LONGLONG)
	assert.Equal(t, MySQLDefCode(9), FIELD_TYPE_INT24)
	assert.Equal(t, MySQLDefCode(10), FIELD_TYPE_DATE)
	assert.Equal(t, MySQLDefCode(11), FIELD_TYPE_TIME)
	assert.Equal(t, MySQLDefCode(12), FIELD_TYPE_DATETIME)
	assert.Equal(t, MySQLDefCode(13), FIELD_TYPE_YEAR)
	assert.Equal(t, MySQLDefCode(14), FIELD_TYPE_NEWDATE)
	assert.Equal(t, MySQLDefCode(15), FIELD_TYPE_VARCHAR)
	assert.Equal(t, MySQLDefCode(16), FIELD_TYPE_BIT)
}

func TestXAErrorCodeConstants(t *testing.T) {
	// Test XA error code constants
	assert.Equal(t, 1399, ErrCodeXAER_RMFAIL_IDLE)
	assert.Equal(t, 1400, ErrCodeXAER_INVAL)
}
