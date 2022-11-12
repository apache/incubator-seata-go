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

package builder

import (
	"testing"

	"database/sql/driver"

	"github.com/stretchr/testify/assert"
)

func TestBuildSelectSQLByMultiDelete(t *testing.T) {
	tests := []struct {
		name            string
		sourceQuery     []string
		sourceQueryArgs []driver.Value
		expectQuery     string
		expectQueryArgs []driver.Value
	}{
		{
			sourceQuery:     []string{"delete from table_update_executor_test where id = ?", "delete from table_update_executor_test"},
			sourceQueryArgs: []driver.Value{3},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test FOR UPDATE",
			expectQueryArgs: []driver.Value{},
		},
		{
			sourceQuery:     []string{"delete from table_update_executor_test2 where id = ?", "delete from table_update_executor_test2 where id = ?"},
			sourceQueryArgs: []driver.Value{3, 2},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test2 WHERE id=? OR id=? FOR UPDATE",
			expectQueryArgs: []driver.Value{3, 2},
		},
		{
			sourceQuery:     []string{"delete from table_update_executor_test2 where id = ?", "delete from table_update_executor_test2 where name = ? and age = ?"},
			sourceQueryArgs: []driver.Value{3, "seata-go", 4},
			expectQuery:     "SELECT SQL_NO_CACHE * FROM table_update_executor_test2 WHERE id=? OR name=? AND age=? FOR UPDATE",
			expectQueryArgs: []driver.Value{3, "seata-go", 4},
		},
	}

	var builder = MySQLMultiDeleteUndoLogBuilder{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, args, err := builder.buildBeforeImageSQL(tt.sourceQuery, tt.sourceQueryArgs)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(items))
			assert.Equal(t, items[0], tt.expectQuery)
			assert.Equal(t, tt.expectQueryArgs, args)
		})
	}
}
