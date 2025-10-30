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
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/mysql"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/postgres"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/log"
)

func TestBuildSelectSQLByMultiUpdate(t *testing.T) {
	log.Init()
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	tests := []struct {
		name              string
		dbType            types.DBType
		sourceQuery       string
		sourceQueryArgs   []driver.Value
		expectQuery       string
		expectQueryArgs   []driver.Value
		pgExpectQuery     string
		pgExpectQueryArgs []driver.Value
	}{
		{
			name:            "MySQL: Basic multi-update with different IDs",
			dbType:          types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where id = ?;" +
				"update t_user set name = ?, age = ? where id = ?;",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, "TOM", 2, 200},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? OR id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 200},
		},
		{
			name:            "MySQL: Multi-update with complex conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age between ? and ?;" +
				"update t_user set name = ?, age = ? where id = ? and name = 'Jack2' and age between ? and ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 28, 38},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND name=_UTF8MB4'Jack' AND age BETWEEN ? AND ? OR id=? AND name=_UTF8MB4'Jack2' AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28, 200, 28, 38},
		},
		{
			name:            "MySQL: Multi-update with IN conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age in (?,?);" +
				"update t_user set name = ?, age = ? where id = ? and name = 'Jack2' and age in (?,?)",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 48, 58},
			expectQuery:     "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND name=_UTF8MB4'Jack' AND age IN (?,?) OR id=? AND name=_UTF8MB4'Jack2' AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28, 200, 48, 58},
		},
		{
			name:              "PostgreSQL: Basic multi-update with different IDs",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3;" +
				"update \"t_user\" set name = $4, age = $5 where id = $6;",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, "TOM", 2, 200},
			pgExpectQuery:     "SELECT \"name\",\"age\",\"id\" FROM \"t_user\" WHERE id=$1 OR id=$2 FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 200},
		},
		{
			name:              "PostgreSQL: Multi-update with complex conditions",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3 and name = 'Jack' and age between $4 and $5;" +
				"update \"t_user\" set name = $6, age = $7 where id = $8 and name = 'Jack2' and age between $9 and $10",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 28, 38},
			pgExpectQuery:     "SELECT \"name\",\"age\",\"id\" FROM \"t_user\" WHERE id=$1 AND name='Jack' AND age BETWEEN $2 AND $3 OR id=$4 AND name='Jack2' AND age BETWEEN $5 AND $6 FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 18, 28, 200, 28, 38},
		},
		{
			name:              "PostgreSQL: Multi-update with IN conditions",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3 and name = 'Jack' and age in ($4,$5);" +
				"update \"t_user\" set name = $6, age = $7 where id = $8 and name = 'Jack2' and age in ($9,$10)",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 48, 58},
			pgExpectQuery:     "SELECT \"name\",\"age\",\"id\" FROM \"t_user\" WHERE id=$1 AND name='Jack' AND age IN ($2,$3) OR id=$4 AND name='Jack2' AND age IN ($5,$6) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 18, 28, 200, 48, 58},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Register appropriate table cache for database type
			switch tt.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					mysqlCfg, ok := cfg.(*mysqlDriver.Config)
					if !ok {
						return mysql.NewTableMetaInstance(db, nil)
					}
					return mysql.NewTableMetaInstance(db, mysqlCfg)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					postgresCfg, ok := cfg.(string)
					if !ok {
						return postgres.NewTableMetaInstance(db, "")
					}
					return postgres.NewTableMetaInstance(db, postgresCfg)
				})
			}

			c, err := parser.DoParser(tt.sourceQuery, tt.dbType)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: tt.dbType,
				},
			}

			executor := NewMultiUpdateExecutor(c, execCtx, []exec.SQLHook{})

			// Create a simple table meta with primary key
			meta := &types.TableMeta{
				TableName:   "t_user",
				ColumnNames: []string{"name", "age", "id"},
				Columns: map[string]types.ColumnMeta{
					"id": {ColumnName: "id", ColumnKey: "PRI"},
					"name": {ColumnName: "name"},
					"age": {ColumnName: "age"},
				},
			}

			query, args, err := executor.buildBeforeImageSQL(util.ValueToNamedValue(tt.sourceQueryArgs), meta)
			assert.NoError(t, err)

			// Check the appropriate expected query based on database type
			if tt.dbType == types.DBTypePostgreSQL && tt.pgExpectQuery != "" {
				assert.Equal(t, tt.pgExpectQuery, query)
				assert.Equal(t, tt.pgExpectQueryArgs, util.NamedValueToValue(args))
			} else {
				assert.Equal(t, tt.expectQuery, query)
				assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
			}
		})
	}

	// Test order by error handling for both databases
	testCases := []struct {
		name        string
		dbType      types.DBType
		sourceQuery string
		args        []driver.Value
	}{
		{
			name:   "MySQL: OrderBy not supported",
			dbType: types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc;" +
				"update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name;",
			args: []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, "Jack2", 1, 10, 20, 17, "Beijing", "Guangzhou", 18},
		},
		{
			name:   "PostgreSQL: OrderBy not supported",
			dbType: types.DBTypePostgreSQL,
			sourceQuery: "update \"t_user\" set name = $1, age = $2 where kk between $3 and $4 and id = $5 and addr in($6,$7) and age > $8 order by name desc;" +
				"update \"t_user\" set name = $9, age = $10 where kk between $11 and $12 and id = $13 and addr in($14,$15) and age > $16 order by name;",
			args: []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, "Jack2", 1, 10, 20, 17, "Beijing", "Guangzhou", 18},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Register appropriate table cache
			switch tc.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					return mysql.NewTableMetaInstance(db, nil)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					return postgres.NewTableMetaInstance(db, "")
				})
			}

			c, err := parser.DoParser(tc.sourceQuery, tc.dbType)
			assert.NoError(t, err)

			execCtx := &types.ExecContext{
				Query:       tc.sourceQuery,
				Values:      tc.args,
				NamedValues: util.ValueToNamedValue(tc.args),
				TxCtx: &types.TransactionContext{
					DBType: tc.dbType,
				},
			}

			executor := NewMultiUpdateExecutor(c, execCtx, []exec.SQLHook{})
			_, _, err = executor.buildBeforeImageSQL(util.ValueToNamedValue(tc.args), &types.TableMeta{})
			assert.Error(t, err)
			assert.Equal(t, "multi update SQL with orderBy condition is not support yet", err.Error())
		})
	}
}

func TestBuildSelectSQLByMultiUpdateAllColumns(t *testing.T) {
	log.Init()
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})

	tests := []struct {
		name              string
		dbType            types.DBType
		sourceQuery       string
		sourceQueryArgs   []driver.Value
		expectQuery       string
		expectQueryArgs   []driver.Value
		pgExpectQuery     string
		pgExpectQueryArgs []driver.Value
	}{
		{
			name:            "MySQL: Basic all columns multi-update",
			dbType:          types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where id = ?;" +
				"update t_user set name = ?, age = ? where id = ?;",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, "TOM", 2, 200},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,sex,birthdate FROM t_user WHERE id=? OR id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 200},
		},
		{
			name:            "MySQL: All columns with complex conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age between ? and ?;" +
				"update t_user set name = ?, age = ? where id = ? and name = 'Jack2' and age between ? and ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 28, 38},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,sex,birthdate FROM t_user WHERE id=? AND name=_UTF8MB4'Jack' AND age BETWEEN ? AND ? OR id=? AND name=_UTF8MB4'Jack2' AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28, 200, 28, 38},
		},
		{
			name:            "MySQL: All columns with IN conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age in (?,?);" +
				"update t_user set name = ?, age = ? where id = ? and name = 'Jack2' and age in (?,?)",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 48, 58},
			expectQuery:     "SELECT SQL_NO_CACHE name,age,sex,birthdate FROM t_user WHERE id=? AND name=_UTF8MB4'Jack' AND age IN (?,?) OR id=? AND name=_UTF8MB4'Jack2' AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28, 200, 48, 58},
		},
		{
			name:              "PostgreSQL: Basic all columns multi-update",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3;" +
				"update \"t_user\" set name = $4, age = $5 where id = $6;",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, "TOM", 2, 200},
			pgExpectQuery:     "SELECT \"name\",\"age\",\"sex\",\"birthdate\" FROM \"t_user\" WHERE id=$1 OR id=$2 FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 200},
		},
		{
			name:              "PostgreSQL: All columns with complex conditions",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3 and name = 'Jack' and age between $4 and $5;" +
				"update \"t_user\" set name = $6, age = $7 where id = $8 and name = 'Jack2' and age between $9 and $10",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 28, 38},
			pgExpectQuery:     "SELECT \"name\",\"age\",\"sex\",\"birthdate\" FROM \"t_user\" WHERE id=$1 AND name='Jack' AND age BETWEEN $2 AND $3 OR id=$4 AND name='Jack2' AND age BETWEEN $5 AND $6 FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 18, 28, 200, 28, 38},
		},
		{
			name:              "PostgreSQL: All columns with IN conditions",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3 and name = 'Jack' and age in ($4,$5);" +
				"update \"t_user\" set name = $6, age = $7 where id = $8 and name = 'Jack2' and age in ($9,$10)",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, 18, 28, "Jack2", 2, 200, 48, 58},
			pgExpectQuery:     "SELECT \"name\",\"age\",\"sex\",\"birthdate\" FROM \"t_user\" WHERE id=$1 AND name='Jack' AND age IN ($2,$3) OR id=$4 AND name='Jack2' AND age IN ($5,$6) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 18, 28, 200, 48, 58},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Register appropriate table cache for database type
			switch tt.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					mysqlCfg, ok := cfg.(*mysqlDriver.Config)
					if !ok {
						return mysql.NewTableMetaInstance(db, nil)
					}
					return mysql.NewTableMetaInstance(db, mysqlCfg)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					postgresCfg, ok := cfg.(string)
					if !ok {
						return postgres.NewTableMetaInstance(db, "")
					}
					return postgres.NewTableMetaInstance(db, postgresCfg)
				})
			}

			c, err := parser.DoParser(tt.sourceQuery, tt.dbType)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: tt.dbType,
				},
			}

			executor := NewMultiUpdateExecutor(c, execCtx, []exec.SQLHook{})

			meta := &types.TableMeta{
				TableName:   "t_user",
				ColumnNames: []string{"name", "age", "sex", "birthdate"},
				Columns: map[string]types.ColumnMeta{
					"id": {ColumnName: "id", ColumnKey: "PRI"},
					"name": {ColumnName: "name"},
					"age": {ColumnName: "age"},
					"sex": {ColumnName: "sex"},
					"birthdate": {ColumnName: "birthdate"},
				},
			}

			query, args, err := executor.buildBeforeImageSQL(util.ValueToNamedValue(tt.sourceQueryArgs), meta)
			assert.NoError(t, err)

			// Check the appropriate expected query based on database type
			if tt.dbType == types.DBTypePostgreSQL && tt.pgExpectQuery != "" {
				assert.Equal(t, tt.pgExpectQuery, query)
				assert.Equal(t, tt.pgExpectQueryArgs, util.NamedValueToValue(args))
			} else {
				assert.Equal(t, tt.expectQuery, query)
				assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
			}
		})
	}

	// Test order by error handling for both databases in all columns mode
	testCases := []struct {
		name        string
		dbType      types.DBType
		sourceQuery string
		args        []driver.Value
	}{
		{
			name:   "MySQL: OrderBy not supported (all columns)",
			dbType: types.DBTypeMySQL,
			sourceQuery: "update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc;" +
				"update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name;",
			args: []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, "Jack2", 1, 10, 20, 17, "Beijing", "Guangzhou", 18},
		},
		{
			name:   "PostgreSQL: OrderBy not supported (all columns)",
			dbType: types.DBTypePostgreSQL,
			sourceQuery: "update \"t_user\" set name = $1, age = $2 where kk between $3 and $4 and id = $5 and addr in($6,$7) and age > $8 order by name desc;" +
				"update \"t_user\" set name = $9, age = $10 where kk between $11 and $12 and id = $13 and addr in($14,$15) and age > $16 order by name;",
			args: []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, "Jack2", 1, 10, 20, 17, "Beijing", "Guangzhou", 18},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Register appropriate table cache
			switch tc.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					return mysql.NewTableMetaInstance(db, nil)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					return postgres.NewTableMetaInstance(db, "")
				})
			}

			c, err := parser.DoParser(tc.sourceQuery, tc.dbType)
			assert.NoError(t, err)

			execCtx := &types.ExecContext{
				Query:       tc.sourceQuery,
				Values:      tc.args,
				NamedValues: util.ValueToNamedValue(tc.args),
				TxCtx: &types.TransactionContext{
					DBType: tc.dbType,
				},
			}

			executor := NewMultiUpdateExecutor(c, execCtx, []exec.SQLHook{})

			meta := &types.TableMeta{
				TableName:   "t_user",
				ColumnNames: []string{"name", "age", "sex", "birthdate"},
				Columns: map[string]types.ColumnMeta{
					"id": {ColumnName: "id", ColumnKey: "PRI"},
					"name": {ColumnName: "name"},
					"age": {ColumnName: "age"},
					"sex": {ColumnName: "sex"},
					"birthdate": {ColumnName: "birthdate"},
				},
			}

			_, _, err = executor.buildBeforeImageSQL(util.ValueToNamedValue(tc.args), meta)
			assert.Error(t, err)
			assert.Equal(t, "multi update SQL with orderBy condition is not support yet", err.Error())
		})
	}
}

// TestMultiUpdateExecutor_PostgreSQL_SQLAdaptation tests PostgreSQL-specific SQL adaptations
func TestMultiUpdateExecutor_PostgreSQL_SQLAdaptation(t *testing.T) {
	log.Init()
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	// Register PostgreSQL table cache
	datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
		return postgres.NewTableMetaInstance(db, "")
	})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			name: "PostgreSQL: UPDATE with boolean values",
			sourceQuery: "update \"users\" set active = $1 where id = $2;" +
				"update \"users\" set active = $3 where id = $4;",
			sourceQueryArgs: []driver.Value{true, 1, false, 2},
			expectQuery:     "SELECT \"active\",\"id\" FROM \"users\" WHERE id=$1 OR id=$2 FOR UPDATE",
			expectQueryArgs: []driver.Value{1, 2},
		},
		{
			name: "PostgreSQL: UPDATE with NULL values",
			sourceQuery: "update \"products\" set description = $1 where id = $2;" +
				"update \"products\" set description = $3 where category IS NULL;",
			sourceQueryArgs: []driver.Value{"new description", 1, nil},
			expectQuery:     "SELECT \"description\",\"id\" FROM \"products\" WHERE id=$1 OR category IS NULL FOR UPDATE",
			expectQueryArgs: []driver.Value{1},
		},
		{
			name: "PostgreSQL: UPDATE with timestamp",
			sourceQuery: "update \"logs\" set updated_at = $1 where id = $2;" +
				"update \"logs\" set updated_at = CURRENT_TIMESTAMP where id = $3;",
			sourceQueryArgs: []driver.Value{"2023-12-01 10:00:00", 1, 2},
			expectQuery:     "SELECT \"updated_at\",\"id\" FROM \"logs\" WHERE id=$1 OR id=$2 FOR UPDATE",
			expectQueryArgs: []driver.Value{1, 2},
		},
		{
			name: "PostgreSQL: UPDATE with LIKE pattern",
			sourceQuery: "update \"users\" set status = $1 where email LIKE $2;" +
				"update \"users\" set status = $3 where name ILIKE $4;",
			sourceQueryArgs: []driver.Value{"active", "%@example.com", "inactive", "%test%"},
			expectQuery:     "SELECT \"status\",\"id\" FROM \"users\" WHERE email LIKE $1 OR name ILIKE $2 FOR UPDATE",
			expectQueryArgs: []driver.Value{"%@example.com", "%test%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery, types.DBTypePostgreSQL)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: types.DBTypePostgreSQL,
				},
			}

			executor := NewMultiUpdateExecutor(c, execCtx, []exec.SQLHook{})

			// Create table meta based on test context
			meta := &types.TableMeta{
				TableName:   getTableNameFromQuery(tt.sourceQuery),
				ColumnNames: getColumnsFromTest(tt.name),
				Columns:     getColumnMetaFromTest(tt.name),
			}

			query, args, err := executor.buildBeforeImageSQL(util.ValueToNamedValue(tt.sourceQueryArgs), meta)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}
}

// TestMultiUpdateExecutor_PostgreSQL_ErrorHandling tests PostgreSQL-specific error handling
func TestMultiUpdateExecutor_PostgreSQL_ErrorHandling(t *testing.T) {
	log.Init()
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	// Register PostgreSQL table cache
	datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
		return postgres.NewTableMetaInstance(db, "")
	})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectError     bool
		errorContains   string
	}{
		{
			name: "PostgreSQL: LIMIT not supported",
			sourceQuery: "update \"users\" set name = $1 where id = $2 LIMIT 1;" +
				"update \"users\" set name = $3 where id = $4;",
			sourceQueryArgs: []driver.Value{"John", 1, "Jane", 2},
			expectError:     true,
			errorContains:   "multi update SQL with limit condition is not support yet",
		},
		{
			name: "PostgreSQL: ORDER BY not supported",
			sourceQuery: "update \"users\" set name = $1 where id = $2 ORDER BY created_at;" +
				"update \"users\" set name = $3 where id = $4;",
			sourceQueryArgs: []driver.Value{"John", 1, "Jane", 2},
			expectError:     true,
			errorContains:   "multi update SQL with orderBy condition is not support yet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery, types.DBTypePostgreSQL)
			// If parsing itself fails, that might be expected
			if err != nil && tt.expectError {
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				return
			}
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: types.DBTypePostgreSQL,
				},
			}

			executor := NewMultiUpdateExecutor(c, execCtx, []exec.SQLHook{})

			meta := &types.TableMeta{
				TableName:   "users",
				ColumnNames: []string{"id", "name"},
				Columns: map[string]types.ColumnMeta{
					"id":   {ColumnName: "id", ColumnKey: "PRI"},
					"name": {ColumnName: "name"},
				},
			}

			_, _, err = executor.buildBeforeImageSQL(util.ValueToNamedValue(tt.sourceQueryArgs), meta)

			if tt.expectError {
				assert.NotNil(t, err)
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// Helper functions for test setup
func getTableNameFromQuery(query string) string {
	if strings.Contains(query, "users") {
		return "users"
	}
	if strings.Contains(query, "products") {
		return "products"
	}
	if strings.Contains(query, "logs") {
		return "logs"
	}
	return "users" // default
}

func getColumnsFromTest(testName string) []string {
	if strings.Contains(testName, "boolean") {
		return []string{"active", "id"}
	}
	if strings.Contains(testName, "NULL") {
		return []string{"description", "id"}
	}
	if strings.Contains(testName, "timestamp") {
		return []string{"updated_at", "id"}
	}
	if strings.Contains(testName, "LIKE") {
		return []string{"status", "id"}
	}
	return []string{"id", "name"} // default
}

func getColumnMetaFromTest(testName string) map[string]types.ColumnMeta {
	if strings.Contains(testName, "boolean") {
		return map[string]types.ColumnMeta{
			"id":     {ColumnName: "id", ColumnKey: "PRI"},
			"active": {ColumnName: "active"},
		}
	}
	if strings.Contains(testName, "NULL") {
		return map[string]types.ColumnMeta{
			"id":          {ColumnName: "id", ColumnKey: "PRI"},
			"description": {ColumnName: "description"},
		}
	}
	if strings.Contains(testName, "timestamp") {
		return map[string]types.ColumnMeta{
			"id":         {ColumnName: "id", ColumnKey: "PRI"},
			"updated_at": {ColumnName: "updated_at"},
		}
	}
	if strings.Contains(testName, "LIKE") {
		return map[string]types.ColumnMeta{
			"id":     {ColumnName: "id", ColumnKey: "PRI"},
			"status": {ColumnName: "status"},
		}
	}
	return map[string]types.ColumnMeta{
		"id":   {ColumnName: "id", ColumnKey: "PRI"},
		"name": {ColumnName: "name"},
	} // default
}
