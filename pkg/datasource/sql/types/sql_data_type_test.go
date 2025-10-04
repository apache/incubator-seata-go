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

	"github.com/arana-db/parser/mysql"
	"github.com/stretchr/testify/assert"
)

func TestConvertJdbcTypeToMySQLType(t *testing.T) {
	tests := []struct {
		name      string
		jdbcType  int32
		wantMysql byte
	}{
		{"VARCHAR", 12, mysql.TypeVarchar},
		{"INTEGER", 4, mysql.TypeLong},
		{"BIGINT", -5, mysql.TypeLonglong},
		{"DOUBLE", 8, mysql.TypeDouble},
		{"DECIMAL", 3, mysql.TypeNewDecimal},
		{"TIMESTAMP", 93, mysql.TypeTimestamp},
		{"DATE", 91, mysql.TypeDate},
		{"CHAR", 1, mysql.TypeString},
		{"BLOB", 2004, mysql.TypeBlob},
		{"TINYINT", -6, mysql.TypeTiny},
		{"SMALLINT", 5, mysql.TypeShort},
		{"FLOAT", 6, mysql.TypeFloat},
		{"Unknown", 9999, mysql.TypeVarchar}, // fallback to VARCHAR
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertJdbcTypeToMySQLType(tt.jdbcType)
			assert.Equal(t, tt.wantMysql, got, "JDBC type %d should map to MySQL type %d", tt.jdbcType, tt.wantMysql)
		})
	}
}

func TestGetOrBuildFieldType_WithJdbcType(t *testing.T) {
	tests := []struct {
		name            string
		columnMeta      ColumnMeta
		expectedTp      byte
		expectedNotNull bool
	}{
		{
			name: "VARCHAR from JDBC type",
			columnMeta: ColumnMeta{
				DatabaseType: 12, // JDBC VARCHAR
				IsNullable:   1,
				FieldType:    nil,
			},
			expectedTp:      mysql.TypeVarchar,
			expectedNotNull: false,
		},
		{
			name: "NOT NULL INTEGER from JDBC type",
			columnMeta: ColumnMeta{
				DatabaseType: 4, // JDBC INTEGER
				IsNullable:   0,
				FieldType:    nil,
			},
			expectedTp:      mysql.TypeLong,
			expectedNotNull: true,
		},
		{
			name: "BIGINT from JDBC type",
			columnMeta: ColumnMeta{
				DatabaseType: -5, // JDBC BIGINT
				IsNullable:   1,
				FieldType:    nil,
			},
			expectedTp:      mysql.TypeLonglong,
			expectedNotNull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ft := tt.columnMeta.GetOrBuildFieldType()
			assert.NotNil(t, ft)
			assert.Equal(t, tt.expectedTp, ft.Tp, "Expected MySQL type %d, got %d", tt.expectedTp, ft.Tp)

			hasNotNull := mysql.HasNotNullFlag(ft.Flag)
			assert.Equal(t, tt.expectedNotNull, hasNotNull, "NOT NULL flag mismatch")
		})
	}
}
