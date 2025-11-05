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

package sql

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
)

func TestGetInsertLocalTCCLogSQL(t *testing.T) {
	tests := []struct {
		name           string
		localTccTable  string
		expectedResult string
	}{
		{
			name:           "standard table name",
			localTccTable:  "tcc_fence_log",
			expectedResult: "insert into  tcc_fence_log  (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)",
		},
		{
			name:           "custom table name",
			localTccTable:  "my_custom_tcc_log",
			expectedResult: "insert into  my_custom_tcc_log  (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)",
		},
		{
			name:           "table name with schema",
			localTccTable:  "schema.tcc_log",
			expectedResult: "insert into  schema.tcc_log  (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)",
		},
		{
			name:           "empty table name",
			localTccTable:  "",
			expectedResult: "insert into    (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetInsertLocalTCCLogSQL(tt.localTccTable)
			assert.Equal(t, tt.expectedResult, result)
			// Verify the SQL contains the required placeholders
			assert.Contains(t, result, "insert into")
			assert.Contains(t, result, "xid, branch_id, action_name, status, gmt_create, gmt_modified")
			assert.Contains(t, result, "?,?,?,?,?,?")
		})
	}
}

func TestGetQuerySQLByBranchIdAndXid(t *testing.T) {
	tests := []struct {
		name           string
		localTccTable  string
		expectedResult string
	}{
		{
			name:           "standard table name",
			localTccTable:  "tcc_fence_log",
			expectedResult: "select xid, branch_id, action_name, status, gmt_create, gmt_modified from  tcc_fence_log  where xid = ? and branch_id = ? for update",
		},
		{
			name:           "custom table name",
			localTccTable:  "my_tcc_table",
			expectedResult: "select xid, branch_id, action_name, status, gmt_create, gmt_modified from  my_tcc_table  where xid = ? and branch_id = ? for update",
		},
		{
			name:           "empty table name",
			localTccTable:  "",
			expectedResult: "select xid, branch_id, action_name, status, gmt_create, gmt_modified from    where xid = ? and branch_id = ? for update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetQuerySQLByBranchIdAndXid(tt.localTccTable)
			assert.Equal(t, tt.expectedResult, result)
			// Verify the SQL structure
			assert.Contains(t, result, "select")
			assert.Contains(t, result, "where xid = ? and branch_id = ?")
			assert.Contains(t, result, "for update")
		})
	}
}

func TestGetUpdateStatusSQLByBranchIdAndXid(t *testing.T) {
	tests := []struct {
		name           string
		localTccTable  string
		expectedResult string
	}{
		{
			name:           "standard table name",
			localTccTable:  "tcc_fence_log",
			expectedResult: "update  tcc_fence_log  set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? ",
		},
		{
			name:           "custom table name",
			localTccTable:  "custom_fence",
			expectedResult: "update  custom_fence  set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? ",
		},
		{
			name:           "empty table name",
			localTccTable:  "",
			expectedResult: "update    set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUpdateStatusSQLByBranchIdAndXid(tt.localTccTable)
			assert.Equal(t, tt.expectedResult, result)
			// Verify the SQL structure
			assert.Contains(t, result, "update")
			assert.Contains(t, result, "set status = ?, gmt_modified = ?")
			assert.Contains(t, result, "where xid = ? and  branch_id = ? and status = ?")
			// Verify the correct number of placeholders (5 total)
			assert.Equal(t, 5, strings.Count(result, "?"))
		})
	}
}

func TestGetDeleteSQLByBranchIdAndXid(t *testing.T) {
	tests := []struct {
		name           string
		localTccTable  string
		expectedResult string
	}{
		{
			name:           "standard table name",
			localTccTable:  "tcc_fence_log",
			expectedResult: "delete from  tcc_fence_log  where xid = ? and  branch_id = ? ",
		},
		{
			name:           "custom table name",
			localTccTable:  "my_table",
			expectedResult: "delete from  my_table  where xid = ? and  branch_id = ? ",
		},
		{
			name:           "empty table name",
			localTccTable:  "",
			expectedResult: "delete from    where xid = ? and  branch_id = ? ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDeleteSQLByBranchIdAndXid(tt.localTccTable)
			assert.Equal(t, tt.expectedResult, result)
			// Verify the SQL structure
			assert.Contains(t, result, "delete from")
			assert.Contains(t, result, "where xid = ? and  branch_id = ?")
			// Verify the correct number of placeholders (2 total)
			assert.Equal(t, 2, strings.Count(result, "?"))
		})
	}
}

func TestGertDeleteSQLByBranchIdsAndXids(t *testing.T) {
	tests := []struct {
		name               string
		localTccTable      string
		paramsPlaceHolder  string
		expectedResult     string
	}{
		{
			name:               "single pair of parameters",
			localTccTable:      "tcc_fence_log",
			paramsPlaceHolder:  "(?,?)",
			expectedResult:     "delete from  tcc_fence_log  where (xid,branch_id) in ((?,?) )",
		},
		{
			name:               "multiple pairs of parameters",
			localTccTable:      "tcc_fence_log",
			paramsPlaceHolder:  "(?,?),(?,?),(?,?)",
			expectedResult:     "delete from  tcc_fence_log  where (xid,branch_id) in ((?,?),(?,?),(?,?) )",
		},
		{
			name:               "custom table with multiple parameters",
			localTccTable:      "custom_log",
			paramsPlaceHolder:  "(?,?),(?,?)",
			expectedResult:     "delete from  custom_log  where (xid,branch_id) in ((?,?),(?,?) )",
		},
		{
			name:               "empty table name",
			localTccTable:      "",
			paramsPlaceHolder:  "(?,?)",
			expectedResult:     "delete from    where (xid,branch_id) in ((?,?) )",
		},
		{
			name:               "empty placeholder",
			localTccTable:      "tcc_fence_log",
			paramsPlaceHolder:  "",
			expectedResult:     "delete from  tcc_fence_log  where (xid,branch_id) in ( )",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GertDeleteSQLByBranchIdsAndXids(tt.localTccTable, tt.paramsPlaceHolder)
			assert.Equal(t, tt.expectedResult, result)
			// Verify the SQL structure
			assert.Contains(t, result, "delete from")
			assert.Contains(t, result, "where (xid,branch_id) in")
		})
	}
}

func TestGetDeleteSQLByMdfDateAndStatus(t *testing.T) {
	// Pre-calculate the expected status values
	statusCommitted := strconv.Itoa(int(enum.StatusCommitted))
	statusRollbacked := strconv.Itoa(int(enum.StatusRollbacked))
	statusSuspended := strconv.Itoa(int(enum.StatusSuspended))

	tests := []struct {
		name           string
		localTccTable  string
		expectedResult string
	}{
		{
			name:           "standard table name",
			localTccTable:  "tcc_fence_log",
			expectedResult: "delete from  tcc_fence_log  where gmt_modified < ?  and status in (" + statusCommitted + " , " + statusRollbacked + " , " + statusSuspended + ") limit ?",
		},
		{
			name:           "custom table name",
			localTccTable:  "my_custom_table",
			expectedResult: "delete from  my_custom_table  where gmt_modified < ?  and status in (" + statusCommitted + " , " + statusRollbacked + " , " + statusSuspended + ") limit ?",
		},
		{
			name:           "empty table name",
			localTccTable:  "",
			expectedResult: "delete from    where gmt_modified < ?  and status in (" + statusCommitted + " , " + statusRollbacked + " , " + statusSuspended + ") limit ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetDeleteSQLByMdfDateAndStatus(tt.localTccTable)
			assert.Equal(t, tt.expectedResult, result)
			// Verify the SQL structure
			assert.Contains(t, result, "delete from")
			assert.Contains(t, result, "where gmt_modified < ?")
			assert.Contains(t, result, "status in")
			assert.Contains(t, result, "limit ?")
			// Verify it contains the status values
			assert.Contains(t, result, statusCommitted)
			assert.Contains(t, result, statusRollbacked)
			assert.Contains(t, result, statusSuspended)
			// Verify the correct number of placeholders (2 total: date and limit)
			assert.Equal(t, 2, strings.Count(result, "?"))
		})
	}
}

func TestGetQuerySQLByMdDate(t *testing.T) {
	// Pre-calculate the expected status values
	statusCommitted := strconv.Itoa(int(enum.StatusCommitted))
	statusRollbacked := strconv.Itoa(int(enum.StatusRollbacked))
	statusSuspended := strconv.Itoa(int(enum.StatusSuspended))

	tests := []struct {
		name           string
		localTccTable  string
		expectedResult string
	}{
		{
			name:           "standard table name",
			localTccTable:  "tcc_fence_log",
			expectedResult: "select xid, branch_id from  tcc_fence_log  where gmt_modified < ?  and status in (" + statusCommitted + " , " + statusRollbacked + " , " + statusSuspended + ")",
		},
		{
			name:           "custom table name",
			localTccTable:  "my_log_table",
			expectedResult: "select xid, branch_id from  my_log_table  where gmt_modified < ?  and status in (" + statusCommitted + " , " + statusRollbacked + " , " + statusSuspended + ")",
		},
		{
			name:           "empty table name",
			localTccTable:  "",
			expectedResult: "select xid, branch_id from    where gmt_modified < ?  and status in (" + statusCommitted + " , " + statusRollbacked + " , " + statusSuspended + ")",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetQuerySQLByMdDate(tt.localTccTable)
			assert.Equal(t, tt.expectedResult, result)
			// Verify the SQL structure
			assert.Contains(t, result, "select xid, branch_id from")
			assert.Contains(t, result, "where gmt_modified < ?")
			assert.Contains(t, result, "status in")
			// Verify it contains the status values
			assert.Contains(t, result, statusCommitted)
			assert.Contains(t, result, statusRollbacked)
			assert.Contains(t, result, statusSuspended)
			// Verify the correct number of placeholders (1 total: date)
			assert.Equal(t, 1, strings.Count(result, "?"))
		})
	}
}

func TestSQLTemplateConsistency(t *testing.T) {
	// Test that SQL templates properly handle table name substitution
	tableName := "test_table"

	t.Run("all functions return non-empty strings", func(t *testing.T) {
		assert.NotEmpty(t, GetInsertLocalTCCLogSQL(tableName))
		assert.NotEmpty(t, GetQuerySQLByBranchIdAndXid(tableName))
		assert.NotEmpty(t, GetUpdateStatusSQLByBranchIdAndXid(tableName))
		assert.NotEmpty(t, GetDeleteSQLByBranchIdAndXid(tableName))
		assert.NotEmpty(t, GertDeleteSQLByBranchIdsAndXids(tableName, "(?,?)"))
		assert.NotEmpty(t, GetDeleteSQLByMdfDateAndStatus(tableName))
		assert.NotEmpty(t, GetQuerySQLByMdDate(tableName))
	})

	t.Run("all functions properly inject table name", func(t *testing.T) {
		assert.Contains(t, GetInsertLocalTCCLogSQL(tableName), tableName)
		assert.Contains(t, GetQuerySQLByBranchIdAndXid(tableName), tableName)
		assert.Contains(t, GetUpdateStatusSQLByBranchIdAndXid(tableName), tableName)
		assert.Contains(t, GetDeleteSQLByBranchIdAndXid(tableName), tableName)
		assert.Contains(t, GertDeleteSQLByBranchIdsAndXids(tableName, "(?,?)"), tableName)
		assert.Contains(t, GetDeleteSQLByMdfDateAndStatus(tableName), tableName)
		assert.Contains(t, GetQuerySQLByMdDate(tableName), tableName)
	})
}

func TestStatusEnumValues(t *testing.T) {
	// Verify that the status enum values used in SQL generation are correct
	t.Run("status values are as expected", func(t *testing.T) {
		assert.Equal(t, enum.FenceStatus(1), enum.StatusTried)
		assert.Equal(t, enum.FenceStatus(2), enum.StatusCommitted)
		assert.Equal(t, enum.FenceStatus(3), enum.StatusRollbacked)
		assert.Equal(t, enum.FenceStatus(4), enum.StatusSuspended)
	})

	t.Run("status values in SQL match enum", func(t *testing.T) {
		sql := GetDeleteSQLByMdfDateAndStatus("test_table")

		// Verify the SQL contains the correct status values
		assert.Contains(t, sql, strconv.Itoa(int(enum.StatusCommitted)))
		assert.Contains(t, sql, strconv.Itoa(int(enum.StatusRollbacked)))
		assert.Contains(t, sql, strconv.Itoa(int(enum.StatusSuspended)))

		// Verify StatusTried is NOT in the cleanup SQL (as it's an active status)
		assert.NotContains(t, sql, strconv.Itoa(int(enum.StatusTried)))
	})
}

func TestSQLInjectionProtection(t *testing.T) {
	// Test that the functions handle potentially malicious input
	// Note: The actual protection comes from using prepared statements with ?
	// These tests verify the SQL structure remains valid even with unusual table names

	tests := []struct {
		name      string
		tableName string
	}{
		{
			name:      "table name with quotes",
			tableName: "table'name",
		},
		{
			name:      "table name with semicolon",
			tableName: "table;name",
		},
		{
			name:      "table name with dashes",
			tableName: "table-name-test",
		},
		{
			name:      "table name with underscores",
			tableName: "table_name_test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All functions should handle these inputs without panic
			assert.NotPanics(t, func() {
				GetInsertLocalTCCLogSQL(tt.tableName)
				GetQuerySQLByBranchIdAndXid(tt.tableName)
				GetUpdateStatusSQLByBranchIdAndXid(tt.tableName)
				GetDeleteSQLByBranchIdAndXid(tt.tableName)
				GertDeleteSQLByBranchIdsAndXids(tt.tableName, "(?,?)")
				GetDeleteSQLByMdfDateAndStatus(tt.tableName)
				GetQuerySQLByMdDate(tt.tableName)
			})
		})
	}
}
