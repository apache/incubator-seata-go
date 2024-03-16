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
	"fmt"
	"strconv"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
)

var (
	// localTccLogPlaced The enum LocalTccLogPlaced
	localTccLogPlaced = " %s "

	// insertLocalTccLog The enum InsertLocalTccLog
	insertLocalTccLog = "insert into " + localTccLogPlaced + " (xid, branch_id, action_name, status, gmt_create, gmt_modified) values ( ?,?,?,?,?,?)"

	// queryByBranchIdAndXid The enum QueryByBranchIdAndXid
	queryByBranchIdAndXid = "select xid, branch_id, action_name, status, gmt_create, gmt_modified from " + localTccLogPlaced + " where xid = ? and branch_id = ? for update"

	// updateStatusByBranchIdAndXid The enum UpdateStatusByBranchIdAndXid
	updateStatusByBranchIdAndXid = "update " + localTccLogPlaced + " set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? "

	// deleteByBranchIdAndXid The enum DeleteByBranchIdAndXid
	deleteByBranchIdAndXid = "delete from " + localTccLogPlaced + " where xid = ? and  branch_id = ? "

	// deleteByDateAndStatus The enum DeleteByDateAndStatus
	deleteByDateAndStatus = "delete from " + localTccLogPlaced + " where gmt_modified < ?  and status in (" + strconv.Itoa(int(enum.StatusCommitted)) + " , " + strconv.Itoa(int(enum.StatusRollbacked)) + " , " + strconv.Itoa(int(enum.StatusSuspended)) + ")"
)

func GetInsertLocalTCCLogSQL(localTccTable string) string {
	return fmt.Sprintf(insertLocalTccLog, localTccTable)
}

func GetQuerySQLByBranchIdAndXid(localTccTable string) string {
	return fmt.Sprintf(queryByBranchIdAndXid, localTccTable)
}

func GetUpdateStatusSQLByBranchIdAndXid(localTccTable string) string {
	return fmt.Sprintf(updateStatusByBranchIdAndXid, localTccTable)
}

func GetDeleteSQLByBranchIdAndXid(localTccTable string) string {
	return fmt.Sprintf(deleteByBranchIdAndXid, localTccTable)
}

func GetDeleteSQLByMdfDateAndStatus(localTccTable string) string {
	return fmt.Sprintf(deleteByDateAndStatus, localTccTable)
}
