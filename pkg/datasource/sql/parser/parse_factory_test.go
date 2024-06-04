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

package parser

import (
	"fmt"
	"testing"

	aparser "github.com/arana-db/parser"
	"github.com/arana-db/parser/format"

	"seata.apache.org/seata-go/pkg/util/bytes"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"

	_ "github.com/arana-db/parser/test_driver"
)

func TestDoParser(t *testing.T) {
	type tt struct {
		sql     string
		sqlType types.SQLType
		types   types.ExecutorType
	}

	for _, t2 := range [...]tt{
		// replace
		{sql: "REPLACE INTO foo VALUES (1 || 2)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES (1 | 2)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES (false || true)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES (bar(5678))", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUES ()", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo (a,b) VALUES (42,314)", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo () VALUES ()", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO foo VALUE ()", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO ta TABLE tb", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		{sql: "REPLACE INTO t.a TABLE t.b", types: types.ReplaceIntoExecutor, sqlType: types.SQLTypeInsert},
		// insert
		{sql: "INSERT INTO foo VALUES (1234)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (1234, 5678)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO t1 (SELECT * FROM t2)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (1 || 2)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (1 | 2)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (false || true)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo VALUES (bar(5678))", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		{sql: "INSERT INTO foo (a) VALUES (42)", types: types.InsertExecutor, sqlType: types.SQLTypeInsert},
		// update
		{sql: "UPDATE LOW_PRIORITY IGNORE t SET id = id + 1 ORDER BY id DESC;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t SET id = id + 1 ORDER BY id DESC;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t SET id = id + 1 ORDER BY id DESC limit 3 ;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t SET id = id + 1, name = 'jojo';", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE items,month SET items.price=month.price WHERE items.id=month.id;", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE user T0 LEFT OUTER JOIN user_profile T1 ON T1.id = T0.profile_id SET T0.profile_id = 1 WHERE T0.profile_id IN (1);", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		{sql: "UPDATE t1, t2 set t1.profile_id = 1, t2.profile_id = 1 where ta.a=t.ba", types: types.UpdateExecutor, sqlType: types.SQLTypeUpdate},
		// delete
		{sql: "DELETE from t1 where a=1 limit 1", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE FROM t1 WHERE t1.a > 0 ORDER BY t1.a LIMIT 1", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE FROM x.y z WHERE z.a > 0", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE FROM t1 AS w WHERE a > 0", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "DELETE from t1 partition (p0,p1)", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "delete low_priority t1, t2 from t1, t2", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "delete quick t1, t2 from t1, t2", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
		{sql: "delete ignore t1, t2 from t1, t2", types: types.DeleteExecutor, sqlType: types.SQLTypeDelete},
	} {
		parser, err := DoParser(t2.sql)
		assert.NoError(t, err)
		assert.Equal(t, parser.ExecutorType, t2.types)
		assert.Equal(t, parser.SQLType, t2.sqlType)
	}
}

func TestK(t *testing.T) {
	sql := "update aa set name = ?, age = ? where id = 123"
	p := aparser.New()
	stmt, _, _ := p.Parse(sql, "", "")

	var bytes = bytes.NewByteBuffer([]byte{})
	var cc = format.NewRestoreCtx(format.RestoreKeyWordUppercase, bytes)
	stmt[0].Restore(cc)

	fmt.Println(stmt)
}
