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

func TestPostgreSQLPlaceholdersLeaveQuotedTokensUnchanged(t *testing.T) {
	for _, quoted := range []string{
		`$$literal ? $1$$`,
		`$Tag$literal ' " ? $9 $tag$ $Tag$`,
		`'literal '' ? $1'`,
		`"column "" ? $1"`,
		`E'it\'s ? $9'`,
		`e'it\'s ? $9'`,
		`E'it''s ? $9'`,
		`E'backslash\\'`,
		`E'it\\\'s ? $9'`,
		`'backslash\'`,
		`"column\"`,
	} {
		t.Run(quoted, func(t *testing.T) {
			require.Equal(t, "SELECT "+quoted+", $1", RewritePlaceholders("SELECT "+quoted+", ?", types.DBTypePostgreSQL))
			query, args, err := CompactPostgreSQLPlaceholders("SELECT "+quoted+", $2", []driver.NamedValue{
				{Ordinal: 1, Value: "unused"}, {Ordinal: 2, Value: "selected"},
			})
			require.NoError(t, err)
			require.Equal(t, "SELECT "+quoted+", $1", query)
			require.Equal(t, []driver.NamedValue{{Ordinal: 1, Value: "selected"}}, args)
		})
	}
}

func TestPostgreSQLQuotedTokenEndEscapeStringPrefix(t *testing.T) {
	for _, tt := range []struct {
		prefix string
		escape bool
	}{
		{prefix: "E", escape: true},
		{prefix: "e", escape: true},
		{prefix: " E", escape: true},
		{prefix: "(e", escape: true},
		{prefix: "nameE"},
		{prefix: "1e"},
		{prefix: "_E"},
		{prefix: "$e"},
		{prefix: "\u540dE"},
		{prefix: "E "},
	} {
		t.Run(tt.prefix, func(t *testing.T) {
			query := tt.prefix + `'it\'s ? $1'`
			want := len(tt.prefix) + len(`'it\'`)
			if tt.escape {
				want = len(query)
			}
			require.Equal(t, want, PostgreSQLQuotedTokenEnd(query, len(tt.prefix)))
		})
	}
}
