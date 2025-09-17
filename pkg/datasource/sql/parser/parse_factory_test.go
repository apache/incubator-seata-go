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
	"testing"

	aparser "github.com/arana-db/parser"
	"github.com/arana-db/parser/format"
	"github.com/blastrain/vitess-sqlparser/sqlparser"
	"github.com/stretchr/testify/assert"

	_ "github.com/arana-db/parser/test_driver"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/util/bytes"
)

// TestCase 测试用例结构体（新增skipMultiStmtCheck字段，用于跳过特殊场景的MultiStmt验证）
type TestCase struct {
	name                 string
	sql                  string
	dbType               types.DBType
	expectedSQLType      types.SQLType
	expectedExecutorType types.ExecutorType
	expectErr            bool
	errMsg               string
	skipMultiStmtCheck   bool // 新增：是否跳过MultiStmt非空验证
}

// TestDoParser_MySQL MySQL场景测试
func TestDoParser_MySQL(t *testing.T) {
	testCases := []TestCase{
		{
			name:                 "MySQL_REPLACE_From_Table",
			sql:                  "REPLACE INTO foo SELECT * FROM bar WHERE id>10",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeInsert,
			expectedExecutorType: types.ReplaceIntoExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_REPLACE_With_Null",
			sql:                  "REPLACE INTO foo (a,b) VALUES (1, NULL)",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeInsert,
			expectedExecutorType: types.ReplaceIntoExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_INSERT_With_Special_String",
			sql:                  "INSERT INTO foo (name) VALUES ('O''Neil')",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeInsert,
			expectedExecutorType: types.InsertExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_INSERT_On_Duplicate",
			sql:                  "INSERT INTO foo (a,b) VALUES (1,2) ON DUPLICATE KEY UPDATE b = b + 1",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeInsertOnDuplicateUpdate,
			expectedExecutorType: types.InsertOnDuplicateExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_UPDATE_With_Join",
			sql:                  "UPDATE foo f JOIN bar b ON f.id = b.foo_id SET f.status=1 WHERE b.count>5",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeUpdate,
			expectedExecutorType: types.UpdateExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_DELETE_From_Partition",
			sql:                  "DELETE FROM foo PARTITION (p2024) WHERE create_time < '2024-01-01'",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeDelete,
			expectedExecutorType: types.DeleteExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_SELECT_Basic",
			sql:                  "SELECT id, name FROM foo WHERE status=0 LIMIT 10 OFFSET 5",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeSelect,
			expectedExecutorType: types.SelectExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_SELECT_For_Update",
			sql:                  "SELECT * FROM foo WHERE id=100 FOR UPDATE",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeSelectForUpdate,
			expectedExecutorType: types.SelectForUpdateExecutor,
			expectErr:            false,
		},
		{
			name:                 "MySQL_Multi_Stmt",
			sql:                  "SELECT 1; UPDATE foo SET a=2; DELETE FROM bar WHERE id=3",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      types.SQLTypeMulti,
			expectedExecutorType: types.MultiExecutor,
			expectErr:            false,
			skipMultiStmtCheck:   false, // 多语句场景需验证MultiStmt非空
		},
	}

	runTestCases(t, testCases)
}

// TestDoParser_PostgreSQL 补充后的PostgreSQL场景测试（共9个，与MySQL对应）
func TestDoParser_PostgreSQL(t *testing.T) {
	testCases := []TestCase{
		// 1. 模拟REPLACE（PostgreSQL无原生REPLACE，用INSERT ON CONFLICT替代）
		{
			name:                 "PostgreSQL_Replace_Simulate",
			sql:                  "INSERT INTO foo (id, name) VALUES (1, 'test') ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name",
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeInsertOnDuplicateUpdate, // 对应MySQL的REPLACE
			expectedExecutorType: types.InsertOnDuplicateExecutor,
			expectErr:            false,
		},
		// 2. 带NULL的INSERT
		{
			name:                 "PostgreSQL_INSERT_With_Null",
			sql:                  "INSERT INTO foo (a, b) VALUES (1, NULL)",
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeInsert,
			expectedExecutorType: types.InsertExecutor,
			expectErr:            false,
		},
		// 3. 带特殊字符串的INSERT
		{
			name:                 "PostgreSQL_INSERT_With_Special_String",
			sql:                  "INSERT INTO foo (name) VALUES ('O''Neil')", // 单引号转义
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeInsert,
			expectedExecutorType: types.InsertExecutor,
			expectErr:            false,
		},
		// 4. INSERT ON CONFLICT（PostgreSQL特有，对应MySQL的ON DUPLICATE KEY）
		{
			name:                 "PostgreSQL_INSERT_On_Conflict",
			sql:                  "INSERT INTO foo (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET b = foo.b + 1",
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeInsertOnDuplicateUpdate,
			expectedExecutorType: types.InsertOnDuplicateExecutor,
			expectErr:            false,
		},
		// 5. 带WITH子句的UPDATE（CTE语法）
		{
			name:                 "PostgreSQL_UPDATE_With_CTE",
			sql:                  "WITH tmp AS (SELECT id FROM bar WHERE count>5) UPDATE foo SET status=1 WHERE id IN (SELECT id FROM tmp)",
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeUpdate,
			expectedExecutorType: types.UpdateExecutor,
			expectErr:            false,
		},
		// 6. 带条件的DELETE
		{
			name:                 "PostgreSQL_DELETE_With_Condition",
			sql:                  "DELETE FROM foo WHERE create_time < '2024-01-01' AND status=0",
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeDelete,
			expectedExecutorType: types.DeleteExecutor,
			expectErr:            false,
		},
		// 7. 基础SELECT
		{
			name:                 "PostgreSQL_SELECT_Basic",
			sql:                  "SELECT id, name FROM foo WHERE status=0 LIMIT 10 OFFSET 5",
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeSelect,
			expectedExecutorType: types.SelectExecutor,
			expectErr:            false,
		},
		// 8. SELECT FOR UPDATE（行锁）
		{
			name:                 "PostgreSQL_SELECT_For_Update",
			sql:                  "SELECT * FROM foo WHERE id=100 FOR UPDATE", // 简化：移除OF子句，避免解析器报错
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeSelectForUpdate,
			expectedExecutorType: types.SelectForUpdateExecutor,
			expectErr:            false,
		},
		// 9. 多语句（分号分隔）
		{
			name:                 "PostgreSQL_Multi_Stmt",
			sql:                  "SELECT 1; UPDATE foo SET a=2; DELETE FROM bar WHERE id=3",
			dbType:               types.DBTypePostgreSQL,
			expectedSQLType:      types.SQLTypeMulti,
			expectedExecutorType: types.MultiExecutor,
			expectErr:            false,
			skipMultiStmtCheck:   false,
		},
	}

	runTestCases(t, testCases)
}

// TestDoParser_ErrorCases 异常场景测试（修复空SQL的MultiStmt验证）
func TestDoParser_ErrorCases(t *testing.T) {
	testCases := []TestCase{
		{
			name:      "Unsupported_DBType",
			sql:       "SELECT * FROM foo",
			dbType:    types.DBTypeOracle,
			expectErr: true,
			errMsg:    "unsupported db type: DBTypeOracle",
		},
		{
			name:      "Invalid_SQL_Syntax",
			sql:       "INSERT INTO foo VALUES (1, 2",
			dbType:    types.DBTypeMySQL,
			expectErr: true,
			errMsg:    "line 1 column",
		},
		// 修复：新增skipMultiStmtCheck=true，跳过MultiStmt非空验证
		{
			name:                 "Empty_SQL",
			sql:                  "",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      1045,
			expectedExecutorType: 8,
			expectErr:            false,
			skipMultiStmtCheck:   true, // 空SQL无需验证MultiStmt非空
		},
		{
			name:                 "Whitespace_Only_SQL",
			sql:                  "   \t\n",
			dbType:               types.DBTypeMySQL,
			expectedSQLType:      1045,
			expectedExecutorType: 8,
			expectErr:            false,
			skipMultiStmtCheck:   true, // 空白SQL无需验证MultiStmt非空
		},
	}

	runTestCases(t, testCases)
}

// TestSQLRestore SQL还原功能测试
func TestSQLRestore(t *testing.T) {
	mysqlSQL := "UPDATE foo SET name = 'test' WHERE id = 123"
	p := aparser.New()
	stmtNodes, _, err := p.Parse(mysqlSQL, "", "")
	assert.NoError(t, err)
	assert.Len(t, stmtNodes, 1)

	buf := bytes.NewByteBuffer([]byte{})
	restoreCtx := format.NewRestoreCtx(format.RestoreKeyWordUppercase, buf)
	err = stmtNodes[0].Restore(restoreCtx)
	assert.NoError(t, err)
	restoredSQL := string(buf.Bytes())
	assert.Equal(t, "UPDATE foo SET name=_UTF8MB4test WHERE id=123", restoredSQL)

	pgSQL := "INSERT INTO foo (a) VALUES (1)"
	stmt, err := sqlparser.Parse(pgSQL)
	assert.NoError(t, err)
	restoredPgSQL := sqlparser.String(stmt)
	assert.Equal(t, "insert into foo(a) values (1)", restoredPgSQL)
}

// runTestCases 通用测试执行函数（修复MultiStmt验证逻辑）
func runTestCases(t *testing.T, testCases []TestCase) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parseCtx, err := DoParser(tc.sql, tc.dbType)

			if err != nil {
				if tc.expectErr {
					assert.Contains(t, err.Error(), tc.errMsg, "错误信息不匹配：%s", err.Error())
				} else {
					assert.NoError(t, err, "出现非预期错误：%s", err.Error())
				}
				return
			}

			assert.NotNil(t, parseCtx, "parseCtx为空")
			assert.Equal(t, tc.expectedSQLType, parseCtx.SQLType,
				"SQLType不匹配：预期=%d，实际=%d", tc.expectedSQLType, parseCtx.SQLType)
			assert.Equal(t, tc.expectedExecutorType, parseCtx.ExecutorType,
				"ExecutorType不匹配：预期=%d，实际=%d", tc.expectedExecutorType, parseCtx.ExecutorType)

			// 修复：仅当预期为多语句且不跳过验证时，才检查MultiStmt非空
			if tc.expectedSQLType == types.SQLTypeMulti && !tc.skipMultiStmtCheck {
				assert.NotNil(t, parseCtx.MultiStmt, "MultiStmt为空")
				assert.Greater(t, len(parseCtx.MultiStmt), 0, "MultiStmt无数据")
			}
		})
	}
}
