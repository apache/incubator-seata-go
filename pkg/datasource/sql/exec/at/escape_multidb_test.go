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

package at

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestDelEscapeMariaDB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"backtick escape", "`id`", "id"},
		{"no escape", "id", "id"},
		{"backtick schema.column", "`scheme`.`id`", "scheme.id"},
		{"double quote escape", `"id"`, "id"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DelEscape(tt.input, types.DBTypeMARIADB)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDelEscapeOracle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"double quote escape", `"ID"`, "ID"},
		{"no escape", "ID", "ID"},
		{"double quote schema.column", `"SCHEMA"."ID"`, "SCHEMA.ID"},
		// Oracle does NOT use backtick escaping
		{"backtick preserved", "`ID`", "`ID`"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DelEscape(tt.input, types.DBTypeOracle)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddEscapeMariaDB(t *testing.T) {
	// MariaDB should use backtick escaping (same as MySQL)
	result := AddEscape("select", types.DBTypeMARIADB)
	assert.True(t, result[0] == '`' && result[len(result)-1] == '`', "MariaDB should use backtick escaping")
}

func TestAddEscapeOracle(t *testing.T) {
	// Oracle should use double-quote escaping
	result := AddEscape("ID", types.DBTypeOracle)
	assert.True(t, result[0] == '"' && result[len(result)-1] == '"', "Oracle should use double-quote escaping")
}

func TestCheckEscapeMariaDB(t *testing.T) {
	// MariaDB should use MySQL keyword checker
	assert.True(t, checkEscape("SELECT", types.DBTypeMARIADB))
	assert.True(t, checkEscape("TABLE", types.DBTypeMARIADB))
}

func TestCheckEscapeOracle(t *testing.T) {
	// Oracle currently always escapes (safe default)
	assert.True(t, checkEscape("ID", types.DBTypeOracle))
	assert.True(t, checkEscape("USER_NAME", types.DBTypeOracle))
}

func TestBuildWhereConditionByPKs_MultiDB(t *testing.T) {
	// Use SQL keywords so they get escaped
	pks := []string{"select", "table"}

	mysqlResult := BuildWhereConditionByPKs(pks, types.DBTypeMySQL)
	mariaResult := BuildWhereConditionByPKs(pks, types.DBTypeMARIADB)

	// MySQL and MariaDB should produce same result (both use backtick)
	assert.Equal(t, mysqlResult, mariaResult)
	assert.Contains(t, mysqlResult, "`select`")
	assert.Contains(t, mysqlResult, " and ")

	oracleResult := BuildWhereConditionByPKs(pks, types.DBTypeOracle)
	// Oracle should use double-quote
	assert.Contains(t, oracleResult, `"select"`)
	assert.Contains(t, oracleResult, " and ")
}
