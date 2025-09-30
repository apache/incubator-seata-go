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
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
)

func TestInsertOnUpdateBeforeImageSQL(t *testing.T) {
	dbConfigs := []struct {
		name        string
		dbType      types.DBType
		escapeFunc  func(string) string
		conflictTag string
	}{
		{
			name:        "MySQL",
			dbType:      types.DBTypeMySQL,
			escapeFunc:  func(s string) string { return "`" + s + "`" },
			conflictTag: "duplicate_key",
		},
		{
			name:        "PostgreSQL",
			dbType:      types.DBTypePostgreSQL,
			escapeFunc:  func(s string) string { return "\"" + s + "\"" },
			conflictTag: "on_conflict",
		},
	}

	buildTableMeta := func(dbType types.DBType) (types.TableMeta, types.TableMeta) {
		columns := map[string]types.ColumnMeta{
			"id":             {ColumnName: "id", ColumnType: "int"},
			"CAPITALIZED_ID": {ColumnName: "CAPITALIZED_ID", ColumnType: "int"},
			"name":           {ColumnName: "name", ColumnType: "varchar"},
			"age":            {ColumnName: "age", ColumnType: "int"},
		}

		fullIndexes := map[string]types.IndexMeta{
			"PRIMARY": {
				Name:      "PRIMARY",
				IType:     types.IndexTypePrimaryKey,
				Columns:   []types.ColumnMeta{columns["id"]},
				NonUnique: false,
			},
			"name_age_idx": {
				Name:    "name_age_idx",
				IType:   types.IndexUnique,
				Columns: []types.ColumnMeta{columns["name"], columns["age"]},
			},
			"CAPITALIZED_ID_INDEX": {
				Name:    "CAPITALIZED_ID_INDEX",
				IType:   types.IndexUnique,
				Columns: []types.ColumnMeta{columns["CAPITALIZED_ID"]},
			},
		}

		onlyUniqueIndexes := map[string]types.IndexMeta{
			"name_age_idx": {
				Name:    "name_age_idx",
				IType:   types.IndexUnique,
				Columns: []types.ColumnMeta{columns["name"], columns["age"]},
			},
		}

		return types.TableMeta{
				TableName:   "t_user",
				Columns:     columns,
				Indexs:      fullIndexes,
				ColumnNames: []string{"id", "name", "age"},
				DBType:      dbType,
			}, types.TableMeta{
				TableName:   "t_user",
				Columns:     columns,
				Indexs:      onlyUniqueIndexes,
				ColumnNames: []string{"id", "name", "age"},
				DBType:      dbType,
			}
	}

	type TestCase struct {
		name            string
		tags            []string
		queryTemplate   string
		sourceQueryArgs []driver.Value
		expects         []struct {
			queryTemplate string
			args          []driver.Value
		}
		useTableMeta1 bool
	}

	mysqlCases := []TestCase{
		{
			name:            "single_insert_no_conflict",
			tags:            []string{"mysql"},
			queryTemplate:   "insert into {{table}}(id, name, age) values(?,?,?)",
			sourceQueryArgs: []driver.Value{1, "Jack", 20},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = ? ) FOR UPDATE",
					args:          []driver.Value{1},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "multi_insert_no_conflict",
			tags:            []string{"mysql"},
			queryTemplate:   "insert into {{table}}(id, name, age) values(?,?,?),(?,?,?)",
			sourceQueryArgs: []driver.Value{1, "Jack", 20, 2, "Mike", 25},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = ? )  OR ({{escape:id}} = ? ) FOR UPDATE",
					args:          []driver.Value{1, 2},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "case_insensitive_column_insert",
			tags:            []string{"mysql"},
			queryTemplate:   "insert into {{table}}(ID, NAME, AGE) values(?,?,?)",
			sourceQueryArgs: []driver.Value{1, "Jack", 20},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = ? ) FOR UPDATE",
					args:          []driver.Value{1},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "single_insert_duplicate_key",
			tags:            []string{"duplicate_key"},
			queryTemplate:   "insert into {{table}}(id, name, age) values(?,?,?) on duplicate key update name = ?, age = ?",
			sourceQueryArgs: []driver.Value{1, "Jack1", 81, "Link", 18},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? ) FOR UPDATE",
					args:          []driver.Value{1, "Jack1", 81},
				},
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:name}} = ?  and {{escape:age}} = ? )  OR ({{escape:id}} = ? ) FOR UPDATE",
					args:          []driver.Value{"Jack1", 81, 1},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "insert_with_literal_values",
			tags:            []string{"duplicate_key"},
			queryTemplate:   "insert into {{table}}(id, name, age) values(1,'Jack1',?) on duplicate key update name = 'Michael', age = ?",
			sourceQueryArgs: []driver.Value{81, "Link"},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? ) FOR UPDATE",
					args:          []driver.Value{int64(1), "Jack1", 81},
				},
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:name}} = ?  and {{escape:age}} = ? )  OR ({{escape:id}} = ? ) FOR UPDATE",
					args:          []driver.Value{"Jack1", 81, int64(1)},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "capitalized_column_index_duplicate",
			tags:            []string{"duplicate_key"},
			queryTemplate:   "insert into {{table}}(ID, capitalized_id) values(1, 11) on duplicate key update name = 'Michael', age = ?",
			sourceQueryArgs: []driver.Value{"Jack1"},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = ? )  OR ({{escape:CAPITALIZED_ID}} = ? ) FOR UPDATE",
					args:          []driver.Value{int64(1), int64(11)},
				},
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:CAPITALIZED_ID}} = ? )  OR ({{escape:id}} = ? ) FOR UPDATE",
					args:          []driver.Value{int64(11), int64(1)},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "multi_insert_one_index_duplicate",
			tags:            []string{"duplicate_key"},
			queryTemplate:   "insert into {{table}}(id, name, age) values(?,?,?),(?,?,?) on duplicate key update name = ?, age = ?",
			sourceQueryArgs: []driver.Value{1, "Jack1", 81, 2, "Michal", 35, "Link", 18},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:name}} = ?  and {{escape:age}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? ) FOR UPDATE",
					args:          []driver.Value{"Jack1", 81, "Michal", 35},
				},
			},
			useTableMeta1: false,
		},
	}

	pgCases := []TestCase{
		{
			name:            "single_insert_no_conflict",
			tags:            []string{"postgresql"},
			queryTemplate:   "insert into {{table}}(id, name, age) values($1,$2,$3)",
			sourceQueryArgs: []driver.Value{1, "Jack", 20},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = $1 ) FOR UPDATE",
					args:          []driver.Value{1},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "multi_insert_no_conflict",
			tags:            []string{"postgresql"},
			queryTemplate:   "insert into {{table}}(id, name, age) values($1,$2,$3),($4,$5,$6)",
			sourceQueryArgs: []driver.Value{1, "Jack", 20, 2, "Mike", 25},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = $1 )  OR ({{escape:id}} = $2 ) FOR UPDATE",
					args:          []driver.Value{1, 2},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "case_insensitive_column_insert",
			tags:            []string{"postgresql"},
			queryTemplate:   "insert into {{table}}(ID, NAME, AGE) values($1,$2,$3)",
			sourceQueryArgs: []driver.Value{1, "Jack", 20},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = $1 ) FOR UPDATE",
					args:          []driver.Value{1},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "pg_on_conflict_do_update",
			tags:            []string{"on_conflict"},
			queryTemplate:   "insert into {{table}}(id, name, age) values($1,$2,$3) on conflict (id) do update set name = $4, age = $5",
			sourceQueryArgs: []driver.Value{1, "Jack1", 81, "Link", 18},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = $1 )  OR ({{escape:name}} = $2  and {{escape:age}} = $3 ) FOR UPDATE",
					args:          []driver.Value{1, "Jack1", 81},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "pg_multi_insert_on_conflict",
			tags:            []string{"on_conflict"},
			queryTemplate:   "insert into {{table}}(id, name, age) values($1,$2,$3),($4,$5,$6) on conflict (name, age) do update set id = EXCLUDED.id",
			sourceQueryArgs: []driver.Value{1, "Jack", 20, 2, "Mike", 25},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:name}} = $1  and {{escape:age}} = $2 )  OR ({{escape:name}} = $3  and {{escape:age}} = $4 ) FOR UPDATE",
					args:          []driver.Value{"Jack", 20, "Mike", 25},
				},
			},
			useTableMeta1: false,
		},
		{
			name:            "pg_on_conflict_do_nothing",
			tags:            []string{"on_conflict"},
			queryTemplate:   "insert into {{table}}(id, name, age) values($1,$2,$3) on conflict (id) do nothing",
			sourceQueryArgs: []driver.Value{1, "Jack", 20},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = $1 ) FOR UPDATE",
					args:          []driver.Value{1},
				},
			},
			useTableMeta1: true,
		},
		{
			name:            "pg_on_conflict_multi_column",
			tags:            []string{"on_conflict"},
			queryTemplate:   "insert into {{table}}(id, name, age) values($1,$2,$3) on conflict (name, age) do update set name = EXCLUDED.name",
			sourceQueryArgs: []driver.Value{1, "Jack", 20},
			expects: []struct {
				queryTemplate string
				args          []driver.Value
			}{
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:id}} = $1 )  OR ({{escape:name}} = $2  and {{escape:age}} = $3 ) FOR UPDATE",
					args:          []driver.Value{1, "Jack", 20},
				},
				{
					queryTemplate: "SELECT * FROM {{table}}  WHERE ({{escape:name}} = $1  and {{escape:age}} = $2 )  OR ({{escape:id}} = $3 ) FOR UPDATE",
					args:          []driver.Value{"Jack", 20, 1},
				},
			},
			useTableMeta1: true,
		},
	}

	replaceTemplate := func(tpl string, tableName string, escapeFunc func(string) string) string {
		tpl = strings.ReplaceAll(tpl, "{{table}}", tableName)
		re := regexp.MustCompile(`{{escape:(\w+)}}`)
		return re.ReplaceAllStringFunc(tpl, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) < 2 {
				return match
			}
			return escapeFunc(parts[1])
		})
	}

	for _, dbCfg := range dbConfigs {
		t.Run(dbCfg.name, func(t *testing.T) {
			tableMeta1, tableMeta2 := buildTableMeta(dbCfg.dbType)
			escapedTable := dbCfg.escapeFunc("t_user")

			var allCases []TestCase
			if dbCfg.dbType == types.DBTypeMySQL {
				allCases = mysqlCases
			} else if dbCfg.dbType == types.DBTypePostgreSQL {
				allCases = pgCases
			}

			for _, tc := range allCases {

				t.Run(tc.name, func(t *testing.T) {
					actualQuery := replaceTemplate(tc.queryTemplate, "t_user", dbCfg.escapeFunc)
					var expectQueries []string
					var expectArgsList [][]driver.Value
					for _, exp := range tc.expects {
						expectQuery := replaceTemplate(exp.queryTemplate, escapedTable, dbCfg.escapeFunc)
						expectQueries = append(expectQueries, expectQuery)

						expectedArgs := make([]driver.Value, len(exp.args))
						for paramIdx := range exp.args {
							expectedArgs[paramIdx] = getExpectedArgByDBType(
								dbCfg.dbType,
								paramIdx,
								exp.args[paramIdx],
							)
						}
						expectArgsList = append(expectArgsList, expectedArgs)
					}

					execCtx := &types.ExecContext{
						Query:       actualQuery,
						MetaDataMap: map[string]types.TableMeta{"t_user": tableMeta1},
						TxCtx: &types.TransactionContext{
							DBType: dbCfg.dbType,
						},
					}
					if !tc.useTableMeta1 {
						execCtx.MetaDataMap["t_user"] = tableMeta2
					}

					parseCtx, err := parser.DoParser(execCtx.Query, execCtx.TxCtx.DBType)
					assert.Nilf(t, err, "SQL parse failed: %s, query: %s", err, actualQuery)
					if err != nil {
						t.Skipf("skip case '%s' due to SQL parse error", tc.name)
						return
					}
					execCtx.ParseContext = parseCtx

					ioe := insertOnUpdateExecutor{
						beforeImageSqlPrimaryKeys: make(map[string]bool),
						dbType:                    dbCfg.dbType,
						execContext:               execCtx,
					}

					var insertStmt interface{}
					if dbCfg.dbType == types.DBTypeMySQL {
						insertStmt = execCtx.ParseContext.InsertStmt
					} else if dbCfg.dbType == types.DBTypePostgreSQL {
						insertStmt = execCtx.ParseContext.AuxtenInsertStmt
					}

					resultQuery, resultArgs, err := ioe.buildBeforeImageSQL(
						execCtx.MetaDataMap["t_user"],
						util.ValueToNamedValue(tc.sourceQueryArgs),
						insertStmt,
					)
					assert.Nilf(t, err, "build before image SQL failed: %s", err)

					resultArgsFlat := util.NamedValueToValue(resultArgs)
					match := false
					for i := range expectQueries {
						if resultQuery == expectQueries[i] && reflect.DeepEqual(resultArgsFlat, expectArgsList[i]) {
							match = true
							break
						}
					}
					if !match {
						var expectStr string
						for i := range expectQueries {
							expectStr += fmt.Sprintf("Expected SQL: %s\nExpected Args: %v\n", expectQueries[i], expectArgsList[i])
						}
						assert.Failf(t, "result not matched",
							"Actual SQL: %s\nActual Args: %v\n%s",
							resultQuery, resultArgsFlat, expectStr)
					}
				})
			}
		})
	}
}

func sliceContains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

func getExpectedArgByDBType(dbType types.DBType, paramIndex int, originalValue driver.Value) driver.Value {
	switch dbType {
	case types.DBTypePostgreSQL:
		return originalValue
	case types.DBTypeMySQL:
		return originalValue
	default:
		return originalValue
	}
}

func TestInsertOnUpdateAfterImageSQL(t *testing.T) {
	dbConfigs := []struct {
		name        string
		dbType      types.DBType
		escapeFunc  func(string) string
		conflictTag string
	}{
		{
			name:        "MySQL",
			dbType:      types.DBTypeMySQL,
			escapeFunc:  func(s string) string { return "`" + s + "`" },
			conflictTag: "duplicate_key",
		},
		{
			name:        "PostgreSQL",
			dbType:      types.DBTypePostgreSQL,
			escapeFunc:  func(s string) string { return "\"" + s + "\"" },
			conflictTag: "on_conflict",
		},
	}

	type TestCase struct {
		name                      string
		tags                      []string
		beforeSelectSqlTemplate   string
		beforeImageSqlPrimaryKeys map[string]bool
		beforeSelectArgs          []driver.Value
		beforeImage               *types.RecordImage
		expectQueryTemplate       string
		expectQueryArgs           []driver.Value
	}

	mysqlCases := []TestCase{
		{
			name:                      "mysql_simple_case",
			tags:                      []string{"mysql"},
			beforeSelectSqlTemplate:   "SELECT * FROM t_user  WHERE ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? ) ",
			beforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 81},
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{
								KeyType:    types.IndexTypePrimaryKey,
								ColumnName: "id",
								Value:      2,
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "name",
								Value:      "Jack",
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "age",
								Value:      18,
							},
						},
					},
				},
			},
			expectQueryTemplate: "SELECT * FROM t_user  WHERE ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? ) ",
			expectQueryArgs:     []driver.Value{1, "Jack1", 81},
		},
		{
			name:                      "mysql_multi_condition_case",
			tags:                      []string{"mysql"},
			beforeSelectSqlTemplate:   "SELECT * FROM t_user  WHERE ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? )  OR ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? ) ",
			beforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 30, 2, "Michael", 18},
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{
								KeyType:    types.IndexTypePrimaryKey,
								ColumnName: "id",
								Value:      1,
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "name",
								Value:      "Jack",
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "age",
								Value:      18,
							},
						},
					},
				},
			},
			expectQueryTemplate: "SELECT * FROM t_user  WHERE ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? )  OR ({{escape:id}} = ? )  OR ({{escape:name}} = ?  and {{escape:age}} = ? ) ",
			expectQueryArgs:     []driver.Value{1, "Jack1", 30, 2, "Michael", 18},
		},
	}

	pgCases := []TestCase{
		{
			name:                      "postgresql_simple_case",
			tags:                      []string{"postgresql"},
			beforeSelectSqlTemplate:   "SELECT * FROM t_user  WHERE ({{escape:id}} = $1 )  OR ({{escape:name}} = $2  and {{escape:age}} = $3 ) ",
			beforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 81},
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{
								KeyType:    types.IndexTypePrimaryKey,
								ColumnName: "id",
								Value:      2,
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "name",
								Value:      "Jack",
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "age",
								Value:      18,
							},
						},
					},
				},
			},
			expectQueryTemplate: "SELECT * FROM t_user  WHERE ({{escape:id}} = $1 )  OR ({{escape:name}} = $2  and {{escape:age}} = $3 ) ",
			expectQueryArgs:     []driver.Value{1, "Jack1", 81},
		},
		{
			name:                      "postgresql_multi_condition_case",
			tags:                      []string{"postgresql"},
			beforeSelectSqlTemplate:   "SELECT * FROM t_user  WHERE ({{escape:id}} = $1 )  OR ({{escape:name}} = $2  and {{escape:age}} = $3 )  OR ({{escape:id}} = $4 )  OR ({{escape:name}} = $5  and {{escape:age}} = $6 ) ",
			beforeImageSqlPrimaryKeys: map[string]bool{"id": true},
			beforeSelectArgs:          []driver.Value{1, "Jack1", 30, 2, "Michael", 18},
			beforeImage: &types.RecordImage{
				TableName: "t_user",
				Rows: []types.RowImage{
					{
						Columns: []types.ColumnImage{
							{
								KeyType:    types.IndexTypePrimaryKey,
								ColumnName: "id",
								Value:      1,
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "name",
								Value:      "Jack",
							},
							{
								KeyType:    types.IndexUnique,
								ColumnName: "age",
								Value:      18,
							},
						},
					},
				},
			},
			expectQueryTemplate: "SELECT * FROM t_user  WHERE ({{escape:id}} = $1 )  OR ({{escape:name}} = $2  and {{escape:age}} = $3 )  OR ({{escape:id}} = $4 )  OR ({{escape:name}} = $5  and {{escape:age}} = $6 ) ",
			expectQueryArgs:     []driver.Value{1, "Jack1", 30, 2, "Michael", 18},
		},
	}

	replaceTemplate := func(tpl string, tableName string, escapeFunc func(string) string) string {
		tpl = strings.ReplaceAll(tpl, "{{table}}", tableName)
		re := regexp.MustCompile(`{{escape:(\w+)}}`)
		return re.ReplaceAllStringFunc(tpl, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) < 2 {
				return match
			}
			return escapeFunc(parts[1])
		})
	}

	for _, dbCfg := range dbConfigs {
		t.Run(dbCfg.name, func(t *testing.T) {
			var allCases []TestCase
			if dbCfg.dbType == types.DBTypeMySQL {
				allCases = mysqlCases
			} else if dbCfg.dbType == types.DBTypePostgreSQL {
				allCases = pgCases
			}

			for _, tc := range allCases {
				t.Run(tc.name, func(t *testing.T) {
					expectQuery := replaceTemplate(tc.expectQueryTemplate, "t_user", dbCfg.escapeFunc)
					beforeSelectSql := replaceTemplate(tc.beforeSelectSqlTemplate, "t_user", dbCfg.escapeFunc)

					ioe := insertOnUpdateExecutor{
						beforeSelectSql:           beforeSelectSql,
						beforeImageSqlPrimaryKeys: tc.beforeImageSqlPrimaryKeys,
						beforeSelectArgs:          util.ValueToNamedValue(tc.beforeSelectArgs),
						dbType:                    dbCfg.dbType,
					}

					query, args := ioe.buildAfterImageSQL(tc.beforeImage)

					assert.Equal(t, expectQuery, query)
					assert.Equal(t, tc.expectQueryArgs, util.NamedValueToValue(args))
				})
			}
		})
	}
}
