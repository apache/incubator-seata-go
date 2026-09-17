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
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
)

func TestRewritePlaceholders(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		dbType types.DBType
		want   string
	}{
		{
			name:   "postgres placeholders are rewritten",
			query:  "SELECT * FROM t WHERE id = ? AND name = ?",
			dbType: types.DBTypePostgreSQL,
			want:   "SELECT * FROM t WHERE id = $1 AND name = $2",
		},
		{
			name:   "mysql placeholders are unchanged",
			query:  "SELECT * FROM t WHERE id = ?",
			dbType: types.DBTypeMySQL,
			want:   "SELECT * FROM t WHERE id = ?",
		},
		{
			name:   "query without placeholders is unchanged",
			query:  "SELECT * FROM t",
			dbType: types.DBTypePostgreSQL,
			want:   "SELECT * FROM t",
		},
		{
			name:   "postgres placeholders in quoted and commented text are unchanged",
			query:  "SELECT '?' AS literal, \"?\" FROM t WHERE id = ? -- ?\n/* ? */",
			dbType: types.DBTypePostgreSQL,
			want:   "SELECT '?' AS literal, \"?\" FROM t WHERE id = $1 -- ?\n/* ? */",
		},
		{
			name:   "postgres placeholders in dollar quoted strings are unchanged",
			query:  "SELECT $$?$$, $tag$?$tag$ FROM t WHERE id = ?",
			dbType: types.DBTypePostgreSQL,
			want:   "SELECT $$?$$, $tag$?$tag$ FROM t WHERE id = $1",
		},
		{
			name:   "postgres placeholders after escaped quotes and nested comments are unchanged",
			query:  "SELECT 'it''s ?' AS literal, \"column\"\"?\" FROM t /* outer ? /* inner ? */ ? */ WHERE id = ?",
			dbType: types.DBTypePostgreSQL,
			want:   "SELECT 'it''s ?' AS literal, \"column\"\"?\" FROM t /* outer ? /* inner ? */ ? */ WHERE id = $1",
		},
		{
			name:   "postgres standard strings do not treat backslashes as quote escapes",
			query:  "SELECT 'ends with backslash\\' AS literal FROM t WHERE id = ?",
			dbType: types.DBTypePostgreSQL,
			want:   "SELECT 'ends with backslash\\' AS literal FROM t WHERE id = $1",
		},
		{
			name:   "postgres escape strings preserve escaped quotes and placeholders",
			query:  "SELECT E'it\\'s ?' AS literal FROM t WHERE id = ?",
			dbType: types.DBTypePostgreSQL,
			want:   "SELECT E'it\\'s ?' AS literal FROM t WHERE id = $1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RewritePlaceholders(tt.query, tt.dbType))
		})
	}
}

func TestCompactPostgreSQLPlaceholders(t *testing.T) {
	query, args, err := CompactPostgreSQLPlaceholders(
		"SELECT * FROM t WHERE id = $3 AND age BETWEEN $4 AND $5 AND id = $3",
		[]driver.NamedValue{
			{Ordinal: 1, Value: "name"},
			{Ordinal: 2, Value: 18},
			{Ordinal: 3, Value: 100},
			{Ordinal: 4, Value: 20},
			{Ordinal: 5, Value: 30},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM t WHERE id = $1 AND age BETWEEN $2 AND $3 AND id = $1", query)
	assert.Equal(t, []driver.Value{100, 20, 30}, NamedValueToValue(args))
	assert.Equal(t, 1, args[0].Ordinal)
	assert.Equal(t, 2, args[1].Ordinal)
	assert.Equal(t, 3, args[2].Ordinal)
}

func TestCompactPostgreSQLPlaceholdersIgnoresQuotedAndCommentedText(t *testing.T) {
	query, args, err := CompactPostgreSQLPlaceholders(
		"SELECT '$3', \"column$4\", $$ $5 $$, id FROM t WHERE id = $3 -- $4\n/* $5 */",
		[]driver.NamedValue{
			{Ordinal: 1, Value: "name"},
			{Ordinal: 2, Value: 18},
			{Ordinal: 3, Value: 100},
			{Ordinal: 4, Value: 20},
			{Ordinal: 5, Value: 30},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "SELECT '$3', \"column$4\", $$ $5 $$, id FROM t WHERE id = $1 -- $4\n/* $5 */", query)
	assert.Equal(t, []driver.Value{100}, NamedValueToValue(args))
}

func TestCompactPostgreSQLPlaceholdersIgnoresDollarSignsInIdentifiers(t *testing.T) {
	query, args, err := CompactPostgreSQLPlaceholders(
		"SELECT column$3 FROM t WHERE id = $3",
		[]driver.NamedValue{
			{Ordinal: 1, Value: "name"},
			{Ordinal: 2, Value: 18},
			{Ordinal: 3, Value: 100},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "SELECT column$3 FROM t WHERE id = $1", query)
	assert.Equal(t, []driver.Value{100}, NamedValueToValue(args))
}

func TestStripPostgreSQLStringCharset(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "simple literal",
			query: "SELECT * FROM t WHERE name=_UTF8MB4'Jack'",
			want:  "SELECT * FROM t WHERE name='Jack'",
		},
		{
			name:  "literal containing placeholder text",
			query: "SELECT * FROM t WHERE note=_UTF8MB4'$2'",
			want:  "SELECT * FROM t WHERE note='$2'",
		},
		{
			name:  "literal with escaped quote",
			query: "SELECT * FROM t WHERE note=_UTF8MB4'O''Reilly $2'",
			want:  "SELECT * FROM t WHERE note='O''Reilly $2'",
		},
		{
			name:  "quoted and commented text is unchanged",
			query: "SELECT '_UTF8MB4''$2''' AS literal -- _UTF8MB4'$3'\n/* _UTF8MB4'$4' */",
			want:  "SELECT '_UTF8MB4''$2''' AS literal -- _UTF8MB4'$3'\n/* _UTF8MB4'$4' */",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StripPostgreSQLStringCharset(tt.query))
		})
	}
}
