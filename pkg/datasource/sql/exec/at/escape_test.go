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
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

// TestDelEscape tests removing escape characters for different databases
func TestDelEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dbType   types.DBType
		expected string
	}{
		{
			name:     "MySQL: remove backtick escape from table and column",
			input:    "`scheme`.`id`",
			dbType:   types.DBTypeMySQL,
			expected: "scheme.id",
		},
		{
			name:     "MySQL: remove backtick escape from table only",
			input:    "`scheme`.id",
			dbType:   types.DBTypeMySQL,
			expected: "scheme.id",
		},
		{
			name:     "MySQL: remove backtick escape from column only",
			input:    "scheme.`id`",
			dbType:   types.DBTypeMySQL,
			expected: "scheme.id",
		},
		{
			name:     "MySQL: mixed quotes (prioritize backtick handling)",
			input:    "`scheme`.'id'",
			dbType:   types.DBTypeMySQL,
			expected: "scheme.'id'",
		},

		{
			name:     "PostgreSQL: remove double quote escape from table and column",
			input:    `"scheme"."id"`,
			dbType:   types.DBTypePostgreSQL,
			expected: "scheme.id",
		},
		{
			name:     "PostgreSQL: remove double quote escape from table only",
			input:    `"scheme".id`,
			dbType:   types.DBTypePostgreSQL,
			expected: "scheme.id",
		},
		{
			name:     "PostgreSQL: remove double quote escape from column only",
			input:    "scheme.\"id\"",
			dbType:   types.DBTypePostgreSQL,
			expected: "scheme.id",
		},
		{
			name:     "PostgreSQL: no escape characters remain unchanged",
			input:    "scheme.id",
			dbType:   types.DBTypePostgreSQL,
			expected: "scheme.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DelEscape(tt.input, tt.dbType)
			log.Printf("Input: %s, DBType: %s, Result: %s", tt.input, tt.dbType, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAddEscape tests adding escape characters for different databases
func TestAddEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		dbType   types.DBType
		expected string
	}{
		{
			name:     "MySQL: already escaped table and column remain unchanged",
			input:    "`scheme`.`id`",
			dbType:   types.DBTypeMySQL,
			expected: "`scheme`.`id`",
		},
		{
			name:     "MySQL: keyword column gets backtick escape",
			input:    "ALTER",
			dbType:   types.DBTypeMySQL,
			expected: "`ALTER`",
		},
		{
			name:     "MySQL: non-keyword remains unescaped",
			input:    "normal_column",
			dbType:   types.DBTypeMySQL,
			expected: "normal_column",
		},
		{
			name:     "MySQL: dotted identifier with keyword table",
			input:    "ALTER.normal_column",
			dbType:   types.DBTypeMySQL,
			expected: "`ALTER`.normal_column",
		},
		{
			name:     "MySQL: both table and column are keywords",
			input:    "ALTER.ANALYZE",
			dbType:   types.DBTypeMySQL,
			expected: "`ALTER`.`ANALYZE`",
		},
		
		{
			name:     "PostgreSQL: already escaped table and column remain unchanged",
			input:    `"scheme"."id"`,
			dbType:   types.DBTypePostgreSQL,
			expected: `"scheme"."id"`,
		},
		{
			name:     "PostgreSQL: keyword column gets double quote escape",
			input:    "USER",
			dbType:   types.DBTypePostgreSQL,
			expected: `"USER"`,
		},
		{
			name:     "PostgreSQL: non-keyword remains unescaped",
			input:    "normal_column",
			dbType:   types.DBTypePostgreSQL,
			expected: "normal_column",
		},
		{
			name:     "PostgreSQL: dotted identifier with keyword table",
			input:    "ORDER.normal_column",
			dbType:   types.DBTypePostgreSQL,
			expected: `"ORDER".normal_column`,
		},
		{
			name:     "PostgreSQL: both table and column are keywords",
			input:    "ORDER.USER",
			dbType:   types.DBTypePostgreSQL,
			expected: `"ORDER"."USER"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddEscape(tt.input, tt.dbType)
			log.Printf("Input: %s, DBType: %s, Result: %s", tt.input, tt.dbType, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}
