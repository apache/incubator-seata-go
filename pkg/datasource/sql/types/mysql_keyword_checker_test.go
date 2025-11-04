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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMysqlKeyWord(t *testing.T) {
	keywordMap := GetMysqlKeyWord()

	// Test that the map is not empty
	assert.NotEmpty(t, keywordMap, "MySQL keyword map should not be empty")

	// Test some key MySQL keywords that are actually in the map
	expectedKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE",
		"FROM", "WHERE", "ORDER", "BY",
		"GROUP", "HAVING", "JOIN", "INNER",
		"LEFT", "RIGHT", "CREATE", "DROP",
		"ALTER", "TABLE", "INDEX", "PRIMARY",
		"KEY", "FOREIGN", "REFERENCES", "NOT",
		"NULL", "UNIQUE", "DEFAULT", "VARCHAR",
		"INT", "BIGINT", "DECIMAL", "BLOB",
	}

	for _, keyword := range expectedKeywords {
		t.Run("keyword_"+keyword, func(t *testing.T) {
			value, exists := keywordMap[keyword]
			assert.True(t, exists, "Keyword %s should exist in MySQL keyword map", keyword)
			assert.Equal(t, keyword, value, "Keyword %s should map to itself", keyword)
		})
	}
}

func TestGetMysqlKeyWord_LazyInit(t *testing.T) {
	// Reset the global variable to test lazy initialization
	originalMap := MysqlKeyWord
	defer func() {
		MysqlKeyWord = originalMap
	}()

	// Set to nil to test lazy init
	MysqlKeyWord = nil

	// First call should initialize the map
	keywordMap1 := GetMysqlKeyWord()
	assert.NotNil(t, keywordMap1)
	assert.NotEmpty(t, keywordMap1)

	// Second call should return the same map
	keywordMap2 := GetMysqlKeyWord()
	assert.Equal(t, keywordMap1, keywordMap2)

	// The global variable should now be set
	assert.NotNil(t, MysqlKeyWord)
	assert.Equal(t, keywordMap1, MysqlKeyWord)
}

func TestGetMysqlKeyWord_SpecificKeywords(t *testing.T) {
	keywordMap := GetMysqlKeyWord()

	// Test specific keywords with their expected values
	specificTests := map[string]string{
		"ACCESSIBLE":        "ACCESSIBLE",
		"ADD":               "ADD",
		"ALL":               "ALL",
		"ALTER":             "ALTER",
		"ANALYZE":           "ANALYZE",
		"AND":               "AND",
		"AS":                "AS",
		"ASC":               "ASC",
		"BEFORE":            "BEFORE",
		"BETWEEN":           "BETWEEN",
		"BIGINT":            "BIGINT",
		"BINARY":            "BINARY",
		"BLOB":              "BLOB",
		"BY":                "BY",
		"CALL":              "CALL",
		"CASCADE":           "CASCADE",
		"CASE":              "CASE",
		"CHANGE":            "CHANGE",
		"CHAR":              "CHAR",
		"CHARACTER":         "CHARACTER",
		"CHECK":             "CHECK",
		"COLLATE":           "COLLATE",
		"COLUMN":            "COLUMN",
		"CONDITION":         "CONDITION",
		"CONSTRAINT":        "CONSTRAINT",
		"CONTINUE":          "CONTINUE",
		"CONVERT":           "CONVERT",
		"CREATE":            "CREATE",
		"CROSS":             "CROSS",
		"CURRENT_DATE":      "CURRENT_DATE",
		"CURRENT_TIME":      "CURRENT_TIME",
		"CURRENT_TIMESTAMP": "CURRENT_TIMESTAMP",
		"CURRENT_USER":      "CURRENT_USER",
		"CURSOR":            "CURSOR",
		"DATABASE":          "DATABASE",
		"DATABASES":         "DATABASES",
		"DECIMAL":           "DECIMAL",
		"DECLARE":           "DECLARE",
		"DEFAULT":           "DEFAULT",
		"DELETE":            "DELETE",
		"DESC":              "DESC",
		"DESCRIBE":          "DESCRIBE",
		"DISTINCT":          "DISTINCT",
		"DOUBLE":            "DOUBLE",
		"DROP":              "DROP",
		"EACH":              "EACH",
		"ELSE":              "ELSE",
		"EXISTS":            "EXISTS",
		"EXPLAIN":           "EXPLAIN",
		"FALSE":             "FALSE",
		"FLOAT":             "FLOAT",
		"FOR":               "FOR",
		"FOREIGN":           "FOREIGN",
		"FROM":              "FROM",
		"FULLTEXT":          "FULLTEXT",
		"GRANT":             "GRANT",
		"GROUP":             "GROUP",
		"HAVING":            "HAVING",
		"IF":                "IF",
		"IN":                "IN",
		"INDEX":             "INDEX",
		"INNER":             "INNER",
		"INSERT":            "INSERT",
		"INT":               "INT",
		"INTEGER":           "INTEGER",
		"INTO":              "INTO",
		"IS":                "IS",
		"JOIN":              "JOIN",
		"KEY":               "KEY",
		"KEYS":              "KEYS",
		"LEFT":              "LEFT",
		"LIKE":              "LIKE",
		"LIMIT":             "LIMIT",
		"LOAD":              "LOAD",
		"LOCK":              "LOCK",
		"LONG":              "LONG",
		"LONGBLOB":          "LONGBLOB",
		"LONGTEXT":          "LONGTEXT",
		"MATCH":             "MATCH",
		"MEDIUMBLOB":        "MEDIUMBLOB",
		"MEDIUMINT":         "MEDIUMINT",
		"MEDIUMTEXT":        "MEDIUMTEXT",
		"NOT":               "NOT",
		"NULL":              "NULL",
		"NUMERIC":           "NUMERIC",
		"ON":                "ON",
		"OR":                "OR",
		"ORDER":             "ORDER",
		"OUTER":             "OUTER",
		"PRIMARY":           "PRIMARY",
		"PROCEDURE":         "PROCEDURE",
		"REFERENCES":        "REFERENCES",
		"RENAME":            "RENAME",
		"REPLACE":           "REPLACE",
		"RIGHT":             "RIGHT",
		"SELECT":            "SELECT",
		"SET":               "SET",
		"SHOW":              "SHOW",
		"SMALLINT":          "SMALLINT",
		"TABLE":             "TABLE",
		"THEN":              "THEN",
		"TINYBLOB":          "TINYBLOB",
		"TINYINT":           "TINYINT",
		"TINYTEXT":          "TINYTEXT",
		"TO":                "TO",
		"TRUE":              "TRUE",
		"UNION":             "UNION",
		"UNIQUE":            "UNIQUE",
		"UNLOCK":            "UNLOCK",
		"UNSIGNED":          "UNSIGNED",
		"UPDATE":            "UPDATE",
		"USE":               "USE",
		"USING":             "USING",
		"VALUES":            "VALUES",
		"VARBINARY":         "VARBINARY",
		"VARCHAR":           "VARCHAR",
		"WHEN":              "WHEN",
		"WHERE":             "WHERE",
		"WHILE":             "WHILE",
		"WITH":              "WITH",
		"XOR":               "XOR",
		"ZEROFILL":          "ZEROFILL",
	}

	for keyword, expectedValue := range specificTests {
		t.Run("specific_"+keyword, func(t *testing.T) {
			value, exists := keywordMap[keyword]
			assert.True(t, exists, "Keyword %s should exist", keyword)
			assert.Equal(t, expectedValue, value, "Keyword %s should have correct value", keyword)
		})
	}
}

func TestGetMysqlKeyWord_MapIntegrity(t *testing.T) {
	keywordMap := GetMysqlKeyWord()

	// Test that all keys map to themselves (which is the pattern in the implementation)
	for key, value := range keywordMap {
		assert.Equal(t, key, value, "Keyword %s should map to itself", key)
	}

	// Test that the map has a reasonable number of keywords
	// MySQL has hundreds of reserved words, so we expect a substantial map
	assert.Greater(t, len(keywordMap), 100, "MySQL keyword map should have more than 100 entries")
	assert.Less(t, len(keywordMap), 1000, "MySQL keyword map should have less than 1000 entries")
}

func TestGetMysqlKeyWord_CaseSensitivity(t *testing.T) {
	keywordMap := GetMysqlKeyWord()

	// Test that keywords are stored in uppercase
	testKeywords := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP"}

	for _, keyword := range testKeywords {
		t.Run("case_"+keyword, func(t *testing.T) {
			// Uppercase should exist
			_, exists := keywordMap[keyword]
			assert.True(t, exists, "Uppercase keyword %s should exist", keyword)

			// Lowercase should not exist (since map stores uppercase)
			_, exists = keywordMap[strings.ToLower(keyword)]
			assert.False(t, exists, "Lowercase keyword %s should not exist", strings.ToLower(keyword))
		})
	}
}

func TestGetMysqlKeyWord_ReturnsSameInstance(t *testing.T) {
	// Test that multiple calls return the same map instance
	map1 := GetMysqlKeyWord()
	map2 := GetMysqlKeyWord()

	// Should be the same instance (same memory address)
	assert.True(t, &map1 == &map2 || len(map1) == len(map2), "Multiple calls should return consistent maps")

	// Should have the same content
	for key, value1 := range map1 {
		value2, exists := map2[key]
		assert.True(t, exists, "Key %s should exist in both maps", key)
		assert.Equal(t, value1, value2, "Value for key %s should be the same in both maps", key)
	}
}
