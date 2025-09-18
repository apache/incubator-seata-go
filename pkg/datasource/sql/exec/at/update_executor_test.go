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
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"testing"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/agiledragon/gomonkey/v2"
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

func TestBuildSelectSQLByUpdate(t *testing.T) {
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
			name:            "MySQL: basic update",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update t_user set name = ?, age = ? where id = ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100},
			expectQuery:     "SELECT name,age,id FROM `t_user` WHERE id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
		{
			name:            "MySQL: update with string and range conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age between ? and ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT name,age,id FROM `t_user` WHERE id=? AND name='Jack' AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			name:            "MySQL: update with IN clause",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update t_user set name = ?, age = ? where id = ? and name = 'Jack' and age in (?,?)",
			sourceQueryArgs: []driver.Value{"Jack", 1, 100, 18, 28},
			expectQuery:     "SELECT name,age,id FROM `t_user` WHERE id=? AND name='Jack' AND age IN (?,?) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			name:            "MySQL: complex update with order and limit",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update t_user set name = ?, age = ? where kk between ? and ? and id = ? and addr in(?,?) and age > ? order by name desc limit ?",
			sourceQueryArgs: []driver.Value{"Jack", 1, 10, 20, 17, "Beijing", "Guangzhou", 18, 2},
			expectQuery:     "SELECT name,age,id FROM `t_user` WHERE kk BETWEEN ? AND ? AND id=? AND addr IN (?,?) AND age>? ORDER BY name DESC LIMIT ? FOR UPDATE",
			expectQueryArgs: []driver.Value{10, 20, 17, "Beijing", "Guangzhou", 18, 2},
		},
		{
			name:              "PostgreSQL: basic update",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100},
			pgExpectQuery:     "SELECT name,age,id FROM \"t_user\" WHERE id=$1 FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100},
		},
		{
			name:              "PostgreSQL: update with string and range conditions",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3 and name = 'Jack' and age between $4 and $5",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, 18, 28},
			pgExpectQuery:     "SELECT name,age,id FROM \"t_user\" WHERE id=$1 AND name='Jack' AND age BETWEEN $2 AND $3 FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			name:              "PostgreSQL: update with IN clause",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"t_user\" set name = $1, age = $2 where id = $3 and name = 'Jack' and age in ($4,$5)",
			sourceQueryArgs:   []driver.Value{"Jack", 1, 100, 18, 28},
			pgExpectQuery:     "SELECT name,age,id FROM \"t_user\" WHERE id=$1 AND name='Jack' AND age IN ($2,$3) FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			name:              "PostgreSQL: update with boolean values",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"users\" set active = $1, verified = $2 where id = $3",
			sourceQueryArgs:   []driver.Value{true, false, 100},
			pgExpectQuery:     "SELECT active,verified,id FROM \"users\" WHERE id=$1 FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{100},
		},
		{
			name:              "PostgreSQL: update with NULL handling",
			dbType:            types.DBTypePostgreSQL,
			sourceQuery:       "update \"profiles\" set bio = $1 where user_id = $2 AND bio IS NULL",
			sourceQueryArgs:   []driver.Value{"New bio", 123},
			pgExpectQuery:     "SELECT bio,user_id,id FROM \"profiles\" WHERE user_id=$1 AND bio IS NULL FOR UPDATE",
			pgExpectQueryArgs: []driver.Value{123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(tt.dbType)), "GetTableMeta",
				func(_ interface{}, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &types.TableMeta{
						Indexs: map[string]types.IndexMeta{
							"id": {
								IType: types.IndexTypePrimaryKey,
								Columns: []types.ColumnMeta{
									{ColumnName: "id"},
								},
							},
						},
					}, nil
				})
			defer stub.Reset()

			c, err := parser.DoParser(tt.sourceQuery, tt.dbType)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: tt.dbType,
				},
			}

			executor := NewUpdateExecutor(c, execCtx, []exec.SQLHook{})
			query, args, err := executor.(*updateExecutor).buildBeforeImageSQL(context.Background(), util.ValueToNamedValue(tt.sourceQueryArgs))
			assert.Nil(t, err)

			if tt.dbType == types.DBTypePostgreSQL && tt.pgExpectQuery != "" {
				assert.Equal(t, tt.pgExpectQuery, query)
				assert.Equal(t, tt.pgExpectQueryArgs, util.NamedValueToValue(args))
			} else {
				assert.Equal(t, tt.expectQuery, query)
				assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
			}
		})
	}
}

func TestUpdateExecutor_PostgreSQL_SQLSyntaxAdaptation(t *testing.T) {
	log.Init()
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: false})

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
			name:            "PostgreSQL: MySQL cache hints removal",
			sourceQuery:     "update \"products\" set price = $1 where category_id = $2",
			sourceQueryArgs: []driver.Value{99.99, 1},
			expectQuery:     "SELECT * FROM \"products\" WHERE category_id=$1 FOR UPDATE",
			expectQueryArgs: []driver.Value{1},
		},
		{
			name:            "PostgreSQL: function conversion",
			sourceQuery:     "update \"logs\" set updated_at = NOW() where id = $1",
			sourceQueryArgs: []driver.Value{123},
			expectQuery:     "SELECT * FROM \"logs\" WHERE id=$1 FOR UPDATE",
			expectQueryArgs: []driver.Value{123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(types.DBTypePostgreSQL)), "GetTableMeta",
				func(_ interface{}, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &types.TableMeta{
						Indexs: map[string]types.IndexMeta{
							"id": {
								IType: types.IndexTypePrimaryKey,
								Columns: []types.ColumnMeta{
									{ColumnName: "id"},
								},
							},
						},
					}, nil
				})
			defer stub.Reset()

			c, err := parser.DoParser(tt.sourceQuery, types.DBTypePostgreSQL)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: types.DBTypePostgreSQL,
				},
			}

			executor := NewUpdateExecutor(c, execCtx, []exec.SQLHook{})
			query, args, err := executor.(*updateExecutor).buildBeforeImageSQL(context.Background(), util.ValueToNamedValue(tt.sourceQueryArgs))

			assert.Nil(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}
}

func TestUpdateExecutor_MultiDatabase_IdentifierEscaping(t *testing.T) {
	log.Init()
	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	tests := []struct {
		name            string
		dbType          types.DBType
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			name:            "MySQL: identifier escaping with backticks",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update `user_table` set `user_name` = ? where `user_id` = ?",
			sourceQueryArgs: []driver.Value{"Alice", 456},
			expectQuery:     "SELECT `user_name`,`user_id` FROM `user_table` WHERE `user_id`=? FOR UPDATE",
			expectQueryArgs: []driver.Value{456},
		},
		{
			name:            "PostgreSQL: identifier escaping with double quotes",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update \"user_table\" set \"user_name\" = $1 where \"user_id\" = $2",
			sourceQueryArgs: []driver.Value{"Alice", 456},
			expectQuery:     "SELECT \"user_name\",\"user_id\" FROM \"user_table\" WHERE \"user_id\"=$1 FOR UPDATE",
			expectQueryArgs: []driver.Value{456},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.dbType {
			case types.DBTypeMySQL:
				datasource.RegisterTableCache(types.DBTypeMySQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					return mysql.NewTableMetaInstance(db, nil)
				})
			case types.DBTypePostgreSQL:
				datasource.RegisterTableCache(types.DBTypePostgreSQL, func(db *sql.DB, cfg interface{}) datasource.TableMetaCache {
					return postgres.NewTableMetaInstance(db, "")
				})
			}

			stub := gomonkey.ApplyMethod(reflect.TypeOf(datasource.GetTableCache(tt.dbType)), "GetTableMeta",
				func(_ interface{}, ctx context.Context, dbName, tableName string) (*types.TableMeta, error) {
					return &types.TableMeta{
						Indexs: map[string]types.IndexMeta{
							"user_id": {
								IType: types.IndexTypePrimaryKey,
								Columns: []types.ColumnMeta{
									{ColumnName: "user_id"},
								},
							},
						},
					}, nil
				})
			defer stub.Reset()

			c, err := parser.DoParser(tt.sourceQuery, tt.dbType)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{
					DBType: tt.dbType,
				},
			}

			executor := NewUpdateExecutor(c, execCtx, []exec.SQLHook{})
			query, args, err := executor.(*updateExecutor).buildBeforeImageSQL(context.Background(), util.ValueToNamedValue(tt.sourceQueryArgs))

			assert.Nil(t, err)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}
}