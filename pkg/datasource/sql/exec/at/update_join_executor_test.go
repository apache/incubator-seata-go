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
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
	_ "seata.apache.org/seata-go/pkg/util/log"
)

func getExpectedQueries(mysqlQueries map[string]string, dbType types.DBType) map[string]string {
	if dbType == types.DBTypeMySQL {
		return mysqlQueries
	}

	result := make(map[string]string)
	for table, query := range mysqlQueries {
		pgQuery := strings.ReplaceAll(query, "SQL_NO_CACHE ", "")
		pgQuery = strings.ReplaceAll(pgQuery, "`", "\"")
		result[table] = pgQuery
	}
	return result
}

func TestBuildSelectSQLByUpdateJoin(t *testing.T) {
	MetaDataMap := map[string]*types.TableMeta{
		"table1": {
			TableName: "table1",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"name": {
					ColumnDef:  nil,
					ColumnName: "name",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "name", "age"},
		},
		"table2": {
			TableName: "table2",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"name": {
					ColumnDef:  nil,
					ColumnName: "name",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
				"kk": {
					ColumnDef:  nil,
					ColumnName: "kk",
				},
				"addr": {
					ColumnDef:  nil,
					ColumnName: "addr",
				},
			},
			ColumnNames: []string{"id", "name", "age", "kk", "addr"},
		},
		"table3": {
			TableName: "table3",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "age"},
		},
		"table4": {
			TableName: "table4",
			Indexs: map[string]types.IndexMeta{
				"id": {
					IType: types.IndexTypePrimaryKey,
					Columns: []types.ColumnMeta{
						{ColumnName: "id"},
					},
				},
			},
			Columns: map[string]types.ColumnMeta{
				"id": {
					ColumnDef:  nil,
					ColumnName: "id",
				},
				"age": {
					ColumnDef:  nil,
					ColumnName: "age",
				},
			},
			ColumnNames: []string{"id", "age"},
		},
	}

	undo.InitUndoConfig(undo.Config{OnlyCareUpdateColumns: true})

	tests := []struct {
		name            string
		dbType          types.DBType
		sourceQuery     string
		sourceQueryArgs []driver.Value
		expectQuery     map[string]string
		expectQueryArgs []driver.Value
	}{
		{
			name:            "MySQL LEFT JOIN",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id and t1.age=? set t1.name = 'WILL',t2.name = ?",
			sourceQueryArgs: []driver.Value{18, "Jack"},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id AND t1.age=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id AND t1.age=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{18},
		},
		{
			name:            "PostgreSQL LEFT JOIN",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update table1 set name = 'WILL' from table2 t2 where table1.id = t2.id and table1.age=$1",
			sourceQueryArgs: []driver.Value{18},
			expectQuery: map[string]string{
				"table1": "SELECT \"table1\".\"name\",\"table1\".\"id\" FROM \"table1\",\"table2\" AS \"t2\" WHERE \"table1\".\"id\"=\"t2\".\"id\" AND \"table1\".\"age\"=$1 GROUP BY \"table1\".\"name\",\"table1\".\"id\" FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{18},
		},
		{
			name:            "MySQL INNER JOIN",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update table1 AS t1 inner join table2 AS t2 on t1.id = t2.id set t1.name = 'WILL',t2.name = 'WILL' where t1.id=?",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			name:            "PostgreSQL INNER JOIN",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update table1 set name = 'WILL' from table2 where table1.id = table2.id and table1.id=$1",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table1": "SELECT \"table1\".\"name\",\"table1\".\"id\" FROM \"table1\",\"table2\" WHERE \"table1\".\"id\"=\"table2\".\"id\" AND \"table1\".\"id\"=$1 GROUP BY \"table1\".\"name\",\"table1\".\"id\" FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			name:            "MySQL RIGHT JOIN",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update table1 AS t1 right join table2 AS t2 on t1.id = t2.id set t1.name = 'WILL',t2.name = 'WILL' where t1.id=?",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 RIGHT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM table1 AS t1 RIGHT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			name:            "PostgreSQL RIGHT JOIN",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update table2 set name = 'WILL' from table1 where table1.id = table2.id and table1.id=$1",
			sourceQueryArgs: []driver.Value{1},
			expectQuery: map[string]string{
				"table2": "SELECT \"table2\".\"name\",\"table2\".\"id\" FROM \"table2\",\"table1\" WHERE \"table1\".\"id\"=\"table2\".\"id\" AND \"table1\".\"id\"=$1 GROUP BY \"table2\".\"name\",\"table2\".\"id\" FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1},
		},
		{
			name:            "MySQL complex WHERE",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update table1 t1 inner join table2 t2 on t1.id = t2.id set t1.name = ?, t1.age = ? where t1.id = ? and t1.name = ? and t2.age between ? and ?",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, "Jack", 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.age,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? AND t1.name=? AND t2.age BETWEEN ? AND ? GROUP BY t1.name,t1.age,t1.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, "Jack", 18, 28},
		},
		{
			name:            "PostgreSQL complex WHERE",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update table1 set name = $1, age = $2 from table2 where table1.id = table2.id and table1.id = $3 and table1.name = $4 and table2.age between $5 and $6",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, "Jack", 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT \"table1\".\"name\",\"table1\".\"age\",\"table1\".\"id\" FROM \"table1\",\"table2\" WHERE \"table1\".\"id\"=\"table2\".\"id\" AND \"table1\".\"id\"=$1 AND \"table1\".\"name\"=$2 AND \"table2\".\"age\" BETWEEN $3 AND $4 GROUP BY \"table1\".\"name\",\"table1\".\"age\",\"table1\".\"id\" FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, "Jack", 18, 28},
		},
		{
			name:            "MySQL NULL check and IN",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id set t1.name = ?, t1.age = ? where t1.id=? and t2.id is null and t1.age IN (?,?)",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.age,t1.id FROM table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id WHERE t1.id=? AND t2.id IS NULL AND t1.age IN (?,?) GROUP BY t1.name,t1.age,t1.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 18, 28},
		},
		{
			name:            "PostgreSQL NULL check and IN",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update table1 set name = $1, age = $2 from table2 t2 where table1.id = t2.id and table1.id=$3 and table1.age IN ($4,$5)",
			sourceQueryArgs: []driver.Value{"newJack", 38, 1, 18, 28},
			expectQuery: map[string]string{
				"table1": "SELECT \"table1\".\"name\",\"table1\".\"age\",\"table1\".\"id\" FROM \"table1\",\"table2\" AS \"t2\" WHERE \"table1\".\"id\"=\"t2\".\"id\" AND \"table1\".\"id\"=$1 AND \"table1\".\"age\" IN ($2,$3) GROUP BY \"table1\".\"name\",\"table1\".\"age\",\"table1\".\"id\" FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 18, 28},
		},
		{
			name:            "MySQL with ORDER BY and LIMIT",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update table1 t1 inner join table2 t2 on t1.id = t2.id set t1.name = ?, t2.age = ? where t2.kk between ? and ? and t2.addr in(?,?) and t2.age > ? order by t1.name desc limit ?",
			sourceQueryArgs: []driver.Value{"Jack", 18, 10, 20, "Beijing", "Guangzhou", 18, 2},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t2.kk BETWEEN ? AND ? AND t2.addr IN (?,?) AND t2.age>? GROUP BY t1.name,t1.id ORDER BY t1.name DESC LIMIT ? FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.age,t2.id FROM table1 AS t1 JOIN table2 AS t2 ON t1.id=t2.id WHERE t2.kk BETWEEN ? AND ? AND t2.addr IN (?,?) AND t2.age>? GROUP BY t2.age,t2.id ORDER BY t1.name DESC LIMIT ? FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{10, 20, "Beijing", "Guangzhou", 18, 2},
		},
		{
			name:            "PostgreSQL with ORDER BY and LIMIT",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update table1 set name = $1 from table2 t2 where table1.id = t2.id and t2.kk between $2 and $3 and t2.addr in($4,$5) and t2.age > $6 order by table1.name desc limit $7",
			sourceQueryArgs: []driver.Value{"Jack", 10, 20, "Beijing", "Guangzhou", 18, 2},
			expectQuery: map[string]string{
				"table1": "SELECT \"table1\".\"name\",\"table1\".\"id\" FROM \"table1\",\"table2\" AS \"t2\" WHERE \"table1\".\"id\"=\"t2\".\"id\" AND \"t2\".\"kk\" BETWEEN $1 AND $2 AND \"t2\".\"addr\" IN ($3,$4) AND \"t2\".\"age\">$5 GROUP BY \"table1\".\"name\",\"table1\".\"id\" ORDER BY \"table1\".\"name\" DESC LIMIT $6 FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{10, 20, "Beijing", "Guangzhou", 18, 2},
		},
		{
			name:            "MySQL multiple JOINs",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "update table1 t1 left join table2 t2 on t1.id = t2.id inner join table3 t3 on t3.id = t2.id right join table4 t4 on t4.id = t2.id set t1.name = ?,t2.name = ? where t1.id=? and t3.age=? and t4.age>30",
			sourceQueryArgs: []driver.Value{"Jack", "WILL", 1, 10},
			expectQuery: map[string]string{
				"table1": "SELECT SQL_NO_CACHE t1.name,t1.id FROM ((table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id) JOIN table3 AS t3 ON t3.id=t2.id) RIGHT JOIN table4 AS t4 ON t4.id=t2.id WHERE t1.id=? AND t3.age=? AND t4.age>30 GROUP BY t1.name,t1.id FOR UPDATE",
				"table2": "SELECT SQL_NO_CACHE t2.name,t2.id FROM ((table1 AS t1 LEFT JOIN table2 AS t2 ON t1.id=t2.id) JOIN table3 AS t3 ON t3.id=t2.id) RIGHT JOIN table4 AS t4 ON t4.id=t2.id WHERE t1.id=? AND t3.age=? AND t4.age>30 GROUP BY t2.name,t2.id FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 10},
		},
		{
			name:            "PostgreSQL multiple JOINs",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "update table1 set name = $1 from table2 t2, table3 t3, table4 t4 where table1.id = t2.id and t3.id = t2.id and t4.id = t2.id and table1.id=$2 and t3.age=$3 and t4.age>30",
			sourceQueryArgs: []driver.Value{"Jack", 1, 10},
			expectQuery: map[string]string{
				"table1": "SELECT \"table1\".\"name\",\"table1\".\"id\" FROM \"table1\",\"table2\" AS \"t2\",\"table3\" AS \"t3\",\"table4\" AS \"t4\" WHERE \"table1\".\"id\"=\"t2\".\"id\" AND \"t3\".\"id\"=\"t2\".\"id\" AND \"t4\".\"id\"=\"t2\".\"id\" AND \"table1\".\"id\"=$1 AND \"t3\".\"age\"=$2 AND \"t4\".\"age\">30 GROUP BY \"table1\".\"name\",\"table1\".\"id\" FOR UPDATE",
			},
			expectQueryArgs: []driver.Value{1, 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery, tt.dbType)
			assert.Nil(t, err)
			executor := NewUpdateJoinExecutor(c, &types.ExecContext{
				Values: tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx: &types.TransactionContext{DBType: tt.dbType},
			}, []exec.SQLHook{})

			if !executor.(*updateJoinExecutor).isAstStmtValid() {
				return
			}

			tableNames := executor.(*updateJoinExecutor).parseTableName(c.UpdateStmt.TableRefs.TableRefs)
			for tbName, tableAliases := range tableNames {
				query, args, err := executor.(*updateJoinExecutor).buildBeforeImageSQL(context.Background(), MetaDataMap[tbName], tableAliases, util.ValueToNamedValue(tt.sourceQueryArgs))
				assert.Nil(t, err)
				if query == "" {
					continue
				}
				assert.Equal(t, tt.expectQuery[tbName], query)
				assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
			}
		})
	}
}
