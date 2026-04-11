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

func TestDBType_String(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DBType
		expected string
	}{
		{"DBTypeUnknown", DBTypeUnknown, "DBTypeUnknown"},
		{"DBTypeMySQL", DBTypeMySQL, "DBTypeMySQL"},
		{"DBTypePostgreSQL", DBTypePostgreSQL, "DBTypePostgreSQL"},
		{"DBTypeSQLServer", DBTypeSQLServer, "DBTypeSQLServer"},
		{"DBTypeOracle", DBTypeOracle, "DBTypeOracle"},
		{"DBTypeMARIADB", DBTypeMARIADB, "DBType(6)"},
		{"Invalid negative", DBType(-1), "DBType(-1)"},
		{"Invalid large", DBType(100), "DBType(100)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dbType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDBType_StringValidation(t *testing.T) {
	// Test that all valid constants return proper string representations
	assert.Contains(t, DBTypeUnknown.String(), "Unknown")
	assert.Contains(t, DBTypeMySQL.String(), "MySQL")
	assert.Contains(t, DBTypePostgreSQL.String(), "PostgreSQL")
	assert.Contains(t, DBTypeSQLServer.String(), "SQLServer")
	assert.Contains(t, DBTypeOracle.String(), "Oracle")
}
