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

package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSelectSQLByUpdate(t *testing.T) {
	builder := BasicUndoBuilder{}

	tests := []struct {
		name   string
		query  string
		expect string
	}{
		{
			query:  "update t_user set name = ?, age = ? where id = ?",
			expect: "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=?",
		},
		{
			query:  "update t_user set name = ?, age = ? where id = ? and addr = ? order by id, name desc",
			expect: "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND addr=? ORDER BY id,name DESC",
		},
		{
			query:  "update t_user set name = ?, age = ? where id = ? and addr = ? order by id, name desc limit 99",
			expect: "SELECT SQL_NO_CACHE name,age FROM t_user WHERE id=? AND addr=? ORDER BY id,name DESC LIMIT 99",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, err := builder.buildSelectSQLByUpdate(tt.query)
			assert.Nil(t, err)
			assert.Equal(t, tt.expect, sql)
		})
	}
}
