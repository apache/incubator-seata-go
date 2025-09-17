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
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/mysql"
	"seata.apache.org/seata-go/pkg/datasource/sql/datasource/postgres"
	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	"seata.apache.org/seata-go/pkg/util/log"
)

func Test_multiDeleteExecutor_buildBeforeImageSQL(t *testing.T) {
	log.Init()
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
			name:            "MySQL: delete without where condition",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "delete from table_update_executor_test where id = ?; delete from table_update_executor_test",
			sourceQueryArgs: []driver.Value{3},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM `table_update_executor_test` FOR UPDATE",
			expectQueryArgs: []driver.Value{},
		},
		{
			name:            "MySQL: delete with same table different conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "delete from table_update_executor_test2 where id = ?; delete from table_update_executor_test2 where id = ?",
			sourceQueryArgs: []driver.Value{3, 2},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM `table_update_executor_test2` WHERE (id=?) OR (id=?) FOR UPDATE",
			expectQueryArgs: []driver.Value{3, 2},
		},
		{
			name:            "MySQL: delete with complex conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "delete from table_update_executor_test2 where id = ?; delete from table_update_executor_test2 where name = ? and age = ?",
			sourceQueryArgs: []driver.Value{3, "seata-go", 4},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM `table_update_executor_test2` WHERE (id=?) OR (name=? AND age=?) FOR UPDATE",
			expectQueryArgs: []driver.Value{3, "seata-go", 4},
		},
		{
			name:              "PostgreSQL: delete without where condition",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "delete from \"table_update_executor_test\" where id = $1; delete from \"table_update_executor_test\"",
			sourceQueryArgs:   []driver.Value{3},
			pgExpectQuery:     "SELECT * FROM \"table_update_executor_test\" FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{},
		},
		{
			name:              "PostgreSQL: delete with same table different conditions",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "delete from \"table_update_executor_test2\" where id = $1; delete from \"table_update_executor_test2\" where id = $2",
			sourceQueryArgs:   []driver.Value{3, 2},
			pgExpectQuery:     "SELECT * FROM \"table_update_executor_test2\" WHERE (id=$1) OR (id=$2) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{3, 2},
		},
		{
			name:              "PostgreSQL: delete with complex conditions",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "delete from \"table_update_executor_test2\" where id = $1; delete from \"table_update_executor_test2\" where name = $2 and age = $3",
			sourceQueryArgs:   []driver.Value{3, "seata-go", 4},
			pgExpectQuery:     "SELECT * FROM \"table_update_executor_test2\" WHERE (id=$1) OR (name=$2 AND age=$3) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{3, "seata-go", 4},
		},
		{
			name:              "PostgreSQL: delete with NULL checks",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "delete from \"users\" where email IS NULL; delete from \"users\" where status = $1",
			sourceQueryArgs:   []driver.Value{"inactive"},
			pgExpectQuery:     "SELECT * FROM \"users\" WHERE (email IS NULL) OR (status=$1) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{"inactive"},
		},
		{
			name:              "PostgreSQL: delete with IN clause",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "delete from \"products\" where category_id IN ($1, $2); delete from \"products\" where price < $3",
			sourceQueryArgs:   []driver.Value{1, 2, 100.0},
			pgExpectQuery:     "SELECT * FROM \"products\" WHERE (category_id IN ($1, $2)) OR (price<$3) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{1, 2, 100.0},
		},
		{
			name:              "PostgreSQL: delete with LIKE operator",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "delete from \"logs\" where message LIKE $1; delete from \"logs\" where created_at < $2",
			sourceQueryArgs:   []driver.Value{"%error%", "2023-01-01"},
			pgExpectQuery:     "SELECT * FROM \"logs\" WHERE (message LIKE $1) OR (created_at<$2) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{"%error%", "2023-01-01"},
		},
		{
			name:              "PostgreSQL: delete with multiple operators",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "delete from \"orders\" where total >= $1 AND status <> $2; delete from \"orders\" where customer_id = $3",
			sourceQueryArgs:   []driver.Value{100.0, "cancelled", 42},
			pgExpectQuery:     "SELECT * FROM \"orders\" WHERE (total>=$1 AND status<>$2) OR (customer_id=$3) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100.0, "cancelled", 42},
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

			queryParser, err := parser.DoParser(tt.sourceQuery, tt.dbType)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: tt.dbType,
				},
			}

			executor := NewMultiDeleteExecutor(queryParser, execCtx, []exec.SQLHook{})
			query, args, err := executor.buildBeforeImageSQL()
			assert.Nil(t, err)

			// Check the appropriate expected query based on database type
			if tt.dbType == types.DBTypePostgreSQL && tt.pgExpectQuery != "" {
				assert.Equal(t, tt.pgExpectQuery, query)
				assert.Equal(t, util.ValueToNamedValue(tt.pgExpectQueryArgs), args)
			} else {
				assert.Equal(t, tt.expectQuery, query)
				assert.Equal(t, util.ValueToNamedValue(tt.expectQueryArgs), args)
			}
		})
	}
}

func Test_multiDeleteExecutor_PostgreSQL_ErrorHandling(t *testing.T) {
	log.Init()

	// Register PostgreSQL table cache
	datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
		postgresCfg, ok := cfg.(string)
		if !ok {
			return postgres.NewTableMetaInstance(db, "")
		}
		return postgres.NewTableMetaInstance(db, postgresCfg)
	})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectError     bool
		errorContains   string
	}{
		{
			name:            "PostgreSQL: delete with ORDER BY (should fail)",
			sourceQuery:     "delete from \"users\" where id = $1 ORDER BY name",
			sourceQueryArgs: []driver.Value{1},
			expectError:     true,
			errorContains:   "ORDER BY is not supported",
		},
		{
			name:            "PostgreSQL: delete with LIMIT (should fail)",
			sourceQuery:     "delete from \"users\" where id = $1 LIMIT 10",
			sourceQueryArgs: []driver.Value{1},
			expectError:     true,
			errorContains:   "LIMIT is not supported",
		},
		{
			name:            "PostgreSQL: delete without table (should fail)",
			sourceQuery:     "delete where id = $1",
			sourceQueryArgs: []driver.Value{1},
			expectError:     true,
			errorContains:   "syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParser, err := parser.DoParser(tt.sourceQuery, types.DBTypePostgreSQL)
			// If parsing fails, that's expected for malformed queries
			if err != nil && tt.expectError {
				assert.Contains(t, err.Error(), tt.errorContains, "Error message should contain expected text")
				return
			}
			assert.Nil(t, err, "Parsing should succeed for valid SQL")

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: types.DBTypePostgreSQL,
				},
			}

			executor := NewMultiDeleteExecutor(queryParser, execCtx, []exec.SQLHook{})
			_, _, err = executor.buildBeforeImageSQL()

			if tt.expectError {
				assert.NotNil(t, err, "Should return error for: %s", tt.name)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "Error message should contain expected text")
				}
			} else {
				assert.Nil(t, err, "Should not return error for: %s", tt.name)
			}
		})
	}
}

func Test_multiDeleteExecutor_PostgreSQL_AST_Processing(t *testing.T) {
	log.Init()

	// Register PostgreSQL table cache
	datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
		postgresCfg, ok := cfg.(string)
		if !ok {
			return postgres.NewTableMetaInstance(db, "")
		}
		return postgres.NewTableMetaInstance(db, postgresCfg)
	})

	tests := []struct {
		name            string
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			name:            "PostgreSQL: DELETE with boolean conditions",
			sourceQuery:     "delete from \"flags\" where enabled = $1 AND visible = $2",
			sourceQueryArgs: []driver.Value{true, false},
			expectQuery:     "SELECT * FROM \"flags\" WHERE (enabled=$1 AND visible=$2) FOR UPDATE",
			expectQueryArgs: []driver.Value{true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParser, err := parser.DoParser(tt.sourceQuery, types.DBTypePostgreSQL)
			assert.Nil(t, err, "Parsing should succeed")

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: types.DBTypePostgreSQL,
				},
			}

			executor := NewMultiDeleteExecutor(queryParser, execCtx, []exec.SQLHook{})
			query, args, err := executor.buildBeforeImageSQL()

			assert.Nil(t, err, "buildBeforeImageSQL should succeed")
			assert.Equal(t, tt.expectQuery, query, "Generated query should match expected")
			assert.Equal(t, util.ValueToNamedValue(tt.expectQueryArgs), args, "Generated args should match expected")
		})
	}
}

func Test_multiDeleteExecutor_PostgreSQL_Parameter_Extraction(t *testing.T) {
	log.Init()

	// Register PostgreSQL table cache
	datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
		return postgres.NewTableMetaInstance(db, "")
	})

	// Test parameter extraction with complex placeholder patterns
	tests := []struct {
		name             string
		sourceQuery      string
		sourceQueryArgs  []driver.Value
		expectParamCount int
	}{
		{
			name:             "PostgreSQL: Simple single parameter",
			sourceQuery:      "delete from \"users\" where id = $1",
			sourceQueryArgs:  []driver.Value{123},
			expectParamCount: 1,
		},
		{
			name:             "PostgreSQL: Multiple parameters in order",
			sourceQuery:      "delete from \"users\" where id = $1 AND name = $2 AND age > $3",
			sourceQueryArgs:  []driver.Value{123, "john", 25},
			expectParamCount: 3,
		},
		{
			name:             "PostgreSQL: Parameters with IN clause",
			sourceQuery:      "delete from \"products\" where category IN ($1, $2, $3) AND price > $4",
			sourceQueryArgs:  []driver.Value{"electronics", "books", "toys", 10.0},
			expectParamCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParser, err := parser.DoParser(tt.sourceQuery, types.DBTypePostgreSQL)
			assert.Nil(t, err, "Parsing should succeed")

			execCtx := &types.ExecContext{
				Query:       tt.sourceQuery,
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: types.DBTypePostgreSQL,
				},
			}

			executor := NewMultiDeleteExecutor(queryParser, execCtx, []exec.SQLHook{})
			_, args, err := executor.buildBeforeImageSQL()

			assert.Nil(t, err, "buildBeforeImageSQL should succeed")
			assert.Equal(t, tt.expectParamCount, len(args), "Parameter count should match expected")

			// Verify parameter values match
			for i, expectedValue := range tt.sourceQueryArgs {
				if i < len(args) {
					assert.Equal(t, expectedValue, args[i].Value, "Parameter value %d should match", i)
				}
			}
		})
	}
}
