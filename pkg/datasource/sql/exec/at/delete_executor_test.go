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
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/exec"
	"seata.apache.org/seata-go/pkg/datasource/sql/parser"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/datasource/sql/util"
)

func TestNewDeleteExecutor(t *testing.T) {
	executor := NewDeleteExecutor(nil, nil, nil)
	_, ok := executor.(*deleteExecutor)
	assert.Equalf(t, true, ok, "should be *deleteExecutor")
}

type testCase struct {
	name            string
	dbType          types.DBType
	sourceQuery     string
	sourceQueryArgs []driver.Value
	expectQuery     string
	expectQueryArgs []driver.Value
}

func Test_deleteExecutor_buildBeforeImageSQL(t *testing.T) {
	tests := []testCase{
		{
			name:            "MySQL: basic delete",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "delete from t_user where id = ?",
			sourceQueryArgs: []driver.Value{100},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
		{
			name:            "MySQL: delete with multiple conditions",
			dbType:          types.DBTypeMySQL,
			sourceQuery:     "delete from t_user where id = ? and name = 'Jack' and age between ? and ?",
			sourceQueryArgs: []driver.Value{100, 18, 28},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM t_user WHERE id=? AND name=_UTF8MB4'Jack' AND age BETWEEN ? AND ? FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			name:            "PostgreSQL: basic delete",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "delete from t_user where id = $1",
			sourceQueryArgs: []driver.Value{100},
			expectQuery:     "SELECT * FROM t_user WHERE id = $1 FOR UPDATE",
			expectQueryArgs: []driver.Value{100},
		},
		{
			name:            "PostgreSQL: delete with multiple conditions",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "delete from t_user where id = $1 and name = 'Jack' and age between $2 and $3",
			sourceQueryArgs: []driver.Value{100, 18, 28},
			expectQuery:     "SELECT * FROM t_user WHERE ((id = $1) AND (name = 'Jack')) AND (age BETWEEN $2 AND $3) FOR UPDATE",
			expectQueryArgs: []driver.Value{100, 18, 28},
		},
		{
			name:            "PostgreSQL: delete with order and limit",
			dbType:          types.DBTypePostgreSQL,
			sourceQuery:     "delete from t_user where kk between $1 and $2 order by name desc limit $3",
			sourceQueryArgs: []driver.Value{10, 20, 2},
			expectQuery:     "SELECT * FROM t_user WHERE kk BETWEEN $1 AND $2 ORDER BY name DESC LIMIT $3 FOR UPDATE",
			expectQueryArgs: []driver.Value{10, 20, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.DoParser(tt.sourceQuery, tt.dbType)
			assert.Nil(t, err)

			execCtx := &types.ExecContext{
				Values:      tt.sourceQueryArgs,
				NamedValues: util.ValueToNamedValue(tt.sourceQueryArgs),
				TxCtx:       &types.TransactionContext{DBType: tt.dbType},
			}

			executor := NewDeleteExecutor(c, execCtx, []exec.SQLHook{})
			deleteExec, ok := executor.(*deleteExecutor)
			assert.True(t, ok, "executor should be *deleteExecutor")

			query, args, err := deleteExec.buildBeforeImageSQL(tt.sourceQuery, execCtx.NamedValues)
			assert.Nil(t, err)

			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, tt.expectQueryArgs, util.NamedValueToValue(args))
		})
	}
}
