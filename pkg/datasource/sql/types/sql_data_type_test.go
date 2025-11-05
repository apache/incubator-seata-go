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

func TestSqlDataTypes(t *testing.T) {
	// Test that the SqlDataTypes map has expected values
	expectedTypes := map[string]int32{
		"BIT":                     -7,
		"TINYINT":                 -6,
		"SMALLINT":                5,
		"INTEGER":                 4,
		"BIGINT":                  -5,
		"FLOAT":                   6,
		"REAL":                    7,
		"DOUBLE":                  8,
		"NUMERIC":                 2,
		"DECIMAL":                 3,
		"CHAR":                    1,
		"VARCHAR":                 12,
		"LONGVARCHAR":             -1,
		"DATE":                    91,
		"TIME":                    92,
		"TIMESTAMP":               93,
		"BINARY":                  -2,
		"VARBINARY":               -3,
		"LONGVARBINARY":           -4,
		"NULL":                    0,
		"OTHER":                   1111,
		"JAVA_OBJECT":             2000,
		"DISTINCT":                2001,
		"STRUCT":                  2002,
		"ARRAY":                   2003,
		"BLOB":                    2004,
		"CLOB":                    2005,
		"REF":                     2006,
		"DATALINK":                70,
		"BOOLEAN":                 16,
		"ROWID":                   -8,
		"NCHAR":                   -15,
		"NVARCHAR":                -9,
		"LONGNVARCHAR":            -16,
		"NCLOB":                   2011,
		"SQLXML":                  2009,
		"REF_CURSOR":              2012,
		"TIME_WITH_TIMEZONE":      2013,
		"TIMESTAMP_WITH_TIMEZONE": 2014,
	}

	for dataType, expectedValue := range expectedTypes {
		t.Run(dataType, func(t *testing.T) {
			actualValue, exists := SqlDataTypes[dataType]
			assert.True(t, exists, "Data type %s should exist in SqlDataTypes map", dataType)
			assert.Equal(t, expectedValue, actualValue, "Data type %s should have value %d", dataType, expectedValue)
		})
	}
}

func TestGetSqlDataType(t *testing.T) {
	tests := []struct {
		name     string
		dataType string
		expected int32
	}{
		// Test known types
		{"BIT", "BIT", -7},
		{"TINYINT", "TINYINT", -6},
		{"SMALLINT", "SMALLINT", 5},
		{"INTEGER", "INTEGER", 4},
		{"BIGINT", "BIGINT", -5},
		{"FLOAT", "FLOAT", 6},
		{"REAL", "REAL", 7},
		{"DOUBLE", "DOUBLE", 8},
		{"NUMERIC", "NUMERIC", 2},
		{"DECIMAL", "DECIMAL", 3},
		{"CHAR", "CHAR", 1},
		{"VARCHAR", "VARCHAR", 12},
		{"LONGVARCHAR", "LONGVARCHAR", -1},
		{"DATE", "DATE", 91},
		{"TIME", "TIME", 92},
		{"TIMESTAMP", "TIMESTAMP", 93},
		{"BINARY", "BINARY", -2},
		{"VARBINARY", "VARBINARY", -3},
		{"LONGVARBINARY", "LONGVARBINARY", -4},
		{"NULL", "NULL", 0},
		{"OTHER", "OTHER", 1111},
		{"JAVA_OBJECT", "JAVA_OBJECT", 2000},
		{"DISTINCT", "DISTINCT", 2001},
		{"STRUCT", "STRUCT", 2002},
		{"ARRAY", "ARRAY", 2003},
		{"BLOB", "BLOB", 2004},
		{"CLOB", "CLOB", 2005},
		{"REF", "REF", 2006},
		{"DATALINK", "DATALINK", 70},
		{"BOOLEAN", "BOOLEAN", 16},
		{"ROWID", "ROWID", -8},
		{"NCHAR", "NCHAR", -15},
		{"NVARCHAR", "NVARCHAR", -9},
		{"LONGNVARCHAR", "LONGNVARCHAR", -16},
		{"NCLOB", "NCLOB", 2011},
		{"SQLXML", "SQLXML", 2009},
		{"REF_CURSOR", "REF_CURSOR", 2012},
		{"TIME_WITH_TIMEZONE", "TIME_WITH_TIMEZONE", 2013},
		{"TIMESTAMP_WITH_TIMEZONE", "TIMESTAMP_WITH_TIMEZONE", 2014},

		// Test case insensitive
		{"bit lowercase", "bit", -7},
		{"varchar mixed case", "VarChar", 12},
		{"integer lowercase", "integer", 4},

		// Test unknown types (should return 0 - zero value for int32)
		{"unknown type", "UNKNOWN_TYPE", 0},
		{"empty string", "", 0},
		{"partial match", "VAR", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSqlDataType(tt.dataType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSqlDataType_CaseInsensitive(t *testing.T) {
	// Test that the function is case insensitive
	testCases := []string{"varchar", "VARCHAR", "VarChar", "VARCHARACTER"}

	for _, testCase := range testCases {
		if testCase == "VARCHARACTER" {
			// This should return 0 because it's not an exact match
			result := GetSqlDataType(testCase)
			assert.Equal(t, int32(0), result)
		} else {
			// All these should return the same value for VARCHAR
			result := GetSqlDataType(testCase)
			assert.Equal(t, int32(12), result, "Case insensitive test failed for: %s", testCase)
		}
	}
}

func TestSqlDataTypesMapIntegrity(t *testing.T) {
	// Test that the map is not empty
	assert.NotEmpty(t, SqlDataTypes, "SqlDataTypes map should not be empty")

	// Test that all expected data types are present
	expectedCount := 39 // Update this if you add/remove data types
	assert.Equal(t, expectedCount, len(SqlDataTypes), "SqlDataTypes map should have %d entries", expectedCount)

	// Test that no values are duplicated (except for legitimate cases)
	valueCount := make(map[int32][]string)
	for dataType, value := range SqlDataTypes {
		valueCount[value] = append(valueCount[value], dataType)
	}

	// Some values might legitimately be duplicated, but let's check for unexpected duplicates
	for value, dataTypes := range valueCount {
		if len(dataTypes) > 1 {
			t.Logf("Value %d is used by multiple data types: %v", value, dataTypes)
			// This is informational - some SQL types might legitimately map to the same JDBC type
		}
	}
}

func TestSqlDataTypes_SpecificValues(t *testing.T) {
	// Test some specific important mappings
	assert.Equal(t, int32(-7), SqlDataTypes["BIT"])
	assert.Equal(t, int32(4), SqlDataTypes["INTEGER"])
	assert.Equal(t, int32(12), SqlDataTypes["VARCHAR"])
	assert.Equal(t, int32(93), SqlDataTypes["TIMESTAMP"])
	assert.Equal(t, int32(0), SqlDataTypes["NULL"])
	assert.Equal(t, int32(1111), SqlDataTypes["OTHER"])
}
