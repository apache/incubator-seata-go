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
	"database/sql/driver"
	"seata.apache.org/seata-go/pkg/datasource/sql/mock"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
)

var (
	rowValues = [][]driver.Value{
		{int64(1), "oid11"},
		{int64(2), "oid22"},
		{int64(3), "oid33"},
	}
)

func TestBuildSelectPKSQL(t *testing.T) {
	testCases := []struct {
		name          string
		dbType        types.DBType
		expectedSQL   string
		expectedMySQL string
		expectedPgSQL string
	}{
		{
			name:          "MySQL Database",
			dbType:        types.DBTypeMySQL,
			expectedMySQL: "SELECT SQL_NO_CACHE `id`,`order_id` FROM `t_user` WHERE `age`>? FOR UPDATE",
		},
		{
			name:          "PostgreSQL Database",
			dbType:        types.DBTypePostgreSQL,
			expectedPgSQL: "SELECT \"id\",\"order_id\" FROM \"t_user\" WHERE \"age\">$1 FOR UPDATE",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := selectForUpdateExecutor{dbType: tc.dbType}
			sqlStr := "select name, order_id from t_user where age > ? for update"

			ctx, err := parser.DoParser(sqlStr, types.DBTypeMySQL)

			metaData := types.TableMeta{
				TableName:   "t_user",
				ColumnNames: []string{"id", "order_id", "age"},
				Indexs: map[string]types.IndexMeta{
					"id": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "id",
						Columns: []types.ColumnMeta{
							{ColumnName: "id"},
						},
					},
					"order_id": {
						IType:      types.IndexTypePrimaryKey,
						ColumnName: "order_id",
						Columns: []types.ColumnMeta{
							{ColumnName: "order_id"},
						},
					},
					"age": {
						IType:      types.IndexTypeNull,
						ColumnName: "age",
						Columns: []types.ColumnMeta{
							{ColumnName: "age"},
						},
					},
				},
			}

			assert.Nil(t, err)
			assert.NotNil(t, ctx)
			assert.NotNil(t, ctx.SelectStmt)

			selSQL, err := e.buildSelectPKSQL(ctx.SelectStmt, &metaData)
			assert.Nil(t, err)

			switch tc.dbType {
			case types.DBTypeMySQL:
				assert.Equal(t, tc.expectedMySQL, selSQL)
			case types.DBTypePostgreSQL:
				assert.Equal(t, tc.expectedPgSQL, selSQL)
			}
		})
	}
}

func TestBuildLockKey(t *testing.T) {
	e := selectForUpdateExecutor{}

	metaData := types.TableMeta{
		TableName: "t_user",
		Indexs: map[string]types.IndexMeta{
			"id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "id",
				Columns: []types.ColumnMeta{
					{ColumnName: "id"},
				},
			},
			"order_id": {
				IType:      types.IndexTypePrimaryKey,
				ColumnName: "order_id",
				Columns: []types.ColumnMeta{
					{ColumnName: "order_id"},
				},
			},
			"age": {
				IType:      types.IndexTypeNull,
				ColumnName: "age",
				Columns: []types.ColumnMeta{
					{ColumnName: "age"},
				},
			},
		},
		Columns: map[string]types.ColumnMeta{
			"id": {
				DatabaseTypeString: "INT",
				ColumnName:         "id",
			},
			"order_id": {
				DatabaseTypeString: "VARCHAR",
				ColumnName:         "order_id",
			},
			"age": {
				DatabaseTypeString: "INT",
				ColumnName:         "age",
			},
		},
		ColumnNames: []string{"id", "order_id", "age"},
	}

	rows := mock.NewGenericMockRows(
		[]string{"id", "order_id"},
		rowValues,
	)

	lockKey := e.buildLockKey(rows, &metaData)
	assert.Equal(t, "t_user:1_oid11,2_oid22,3_oid33", lockKey)
}

func TestAdaptSQLSyntax(t *testing.T) {
	testCases := []struct {
		name        string
		dbType      types.DBType
		inputSQL    string
		expectedSQL string
	}{
		{
			name:        "MySQL - Keep original syntax",
			dbType:      types.DBTypeMySQL,
			inputSQL:    "SELECT SQL_NO_CACHE `id` FROM `t_user` WHERE `age`>? FOR UPDATE",
			expectedSQL: "SELECT SQL_NO_CACHE `id` FROM `t_user` WHERE `age`>? FOR UPDATE",
		},
		{
			name:        "PostgreSQL - Remove SQL_NO_CACHE and convert identifiers",
			dbType:      types.DBTypePostgreSQL,
			inputSQL:    "SELECT SQL_NO_CACHE `id` FROM `t_user` WHERE `age`>? FOR UPDATE",
			expectedSQL: "SELECT \"id\" FROM \"t_user\" WHERE \"age\">$1 FOR UPDATE",
		},
		{
			name:        "PostgreSQL - Convert backticks to double quotes",
			dbType:      types.DBTypePostgreSQL,
			inputSQL:    "SELECT `name`, `order_id` FROM `t_user`",
			expectedSQL: "SELECT \"name\", \"order_id\" FROM \"t_user\"",
		},
		{
			name:        "MySQL - Convert double quotes to backticks",
			dbType:      types.DBTypeMySQL,
			inputSQL:    "SELECT \"name\", \"order_id\" FROM \"t_user\"",
			expectedSQL: "SELECT `name`, `order_id` FROM `t_user`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := selectForUpdateExecutor{dbType: tc.dbType}
			result := e.adaptSQLSyntax(tc.inputSQL)
			assert.Equal(t, tc.expectedSQL, result)
		})
	}
}

func TestGenerateSavepointName(t *testing.T) {
	testCases := []struct {
		name              string
		dbType            types.DBType
		timestamp         int64
		expectedSavepoint string
	}{
		{
			name:              "MySQL savepoint name",
			dbType:            types.DBTypeMySQL,
			timestamp:         1234567890,
			expectedSavepoint: "seatago1234567890point",
		},
		{
			name:              "PostgreSQL savepoint name",
			dbType:            types.DBTypePostgreSQL,
			timestamp:         1234567890,
			expectedSavepoint: "seatago_1234567890_point",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := selectForUpdateExecutor{dbType: tc.dbType}
			result := e.generateSavepointName(tc.timestamp)
			assert.Equal(t, tc.expectedSavepoint, result)
		})
	}
}

func TestBuildSavepointSQL(t *testing.T) {
	testCases := []struct {
		name          string
		dbType        types.DBType
		savepointName string
		expectedSQL   string
	}{
		{
			name:          "MySQL savepoint SQL",
			dbType:        types.DBTypeMySQL,
			savepointName: "seatago1234567890point",
			expectedSQL:   "SAVEPOINT seatago1234567890point;",
		},
		{
			name:          "PostgreSQL savepoint SQL",
			dbType:        types.DBTypePostgreSQL,
			savepointName: "seatago_1234567890_point",
			expectedSQL:   "SAVEPOINT \"seatago_1234567890_point\";",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := selectForUpdateExecutor{dbType: tc.dbType}
			result := e.buildSavepointSQL(tc.savepointName)
			assert.Equal(t, tc.expectedSQL, result)
		})
	}
}

func TestBuildRollbackSQL(t *testing.T) {
	testCases := []struct {
		name          string
		dbType        types.DBType
		savepointName string
		expectedSQL   string
	}{
		{
			name:          "MySQL rollback SQL",
			dbType:        types.DBTypeMySQL,
			savepointName: "seatago1234567890point",
			expectedSQL:   "ROLLBACK TO seatago1234567890point;",
		},
		{
			name:          "PostgreSQL rollback SQL",
			dbType:        types.DBTypePostgreSQL,
			savepointName: "seatago_1234567890_point",
			expectedSQL:   "ROLLBACK TO SAVEPOINT \"seatago_1234567890_point\";",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := selectForUpdateExecutor{dbType: tc.dbType}
			result := e.buildRollbackSQL(tc.savepointName)
			assert.Equal(t, tc.expectedSQL, result)
		})
	}
}

func TestEscapeIdentifier(t *testing.T) {
	testCases := []struct {
		name       string
		dbType     types.DBType
		identifier string
		expected   string
	}{
		{
			name:       "MySQL - Escape table name",
			dbType:     types.DBTypeMySQL,
			identifier: "t_user",
			expected:   "`t_user`",
		},
		{
			name:       "MySQL - Already escaped",
			dbType:     types.DBTypeMySQL,
			identifier: "`t_user`",
			expected:   "`t_user`",
		},
		{
			name:       "PostgreSQL - Escape table name",
			dbType:     types.DBTypePostgreSQL,
			identifier: "t_user",
			expected:   "\"t_user\"",
		},
		{
			name:       "PostgreSQL - Already escaped",
			dbType:     types.DBTypePostgreSQL,
			identifier: "\"t_user\"",
			expected:   "\"t_user\"",
		},
		{
			name:       "Empty identifier",
			dbType:     types.DBTypeMySQL,
			identifier: "",
			expected:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			e := selectForUpdateExecutor{dbType: tc.dbType}
			result := e.escapeIdentifier(tc.identifier)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertMySQLIdentifiersToPostgreSQL(t *testing.T) {
	e := selectForUpdateExecutor{dbType: types.DBTypePostgreSQL}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single identifier",
			input:    "`table_name`",
			expected: "\"table_name\"",
		},
		{
			name:     "Multiple identifiers",
			input:    "SELECT `id`, `name` FROM `t_user`",
			expected: "SELECT \"id\", \"name\" FROM \"t_user\"",
		},
		{
			name:     "No identifiers",
			input:    "SELECT * FROM table",
			expected: "SELECT * FROM table",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := e.convertMySQLIdentifiersToPostgreSQL(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvertPostgreSQLIdentifiersToMySQL(t *testing.T) {
	e := selectForUpdateExecutor{dbType: types.DBTypeMySQL}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single identifier",
			input:    "\"table_name\"",
			expected: "`table_name`",
		},
		{
			name:     "Multiple identifiers",
			input:    "SELECT \"id\", \"name\" FROM \"t_user\"",
			expected: "SELECT `id`, `name` FROM `t_user`",
		},
		{
			name:     "No identifiers",
			input:    "SELECT * FROM table",
			expected: "SELECT * FROM table",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := e.convertPostgreSQLIdentifiersToMySQL(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
