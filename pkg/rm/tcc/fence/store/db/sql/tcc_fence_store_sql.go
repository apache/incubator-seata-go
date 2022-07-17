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

	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"
)

var (
	// The constant LocalTccLogPlaced

	localTccLogPlaced = " %s "

	// The constant InsertLocalTccLog

	insertLocalTccLog = "insert into " + localTccLogPlaced + "  (xid, branch_id, action_name, status, gmt_create, gmt_modified)  values (?, ?, ?, ?, ?, ?) "

	//The constant QueryByBranchIdAndXid

	queryByBranchIdAndXid = "select xid, branch_id, action_name, status, gmt_create, gmt_modified from " + localTccLogPlaced + " where xid = ? and branch_id = ? for update"

	// The constant UpdateStatusByBranchIdAndXid

	updateStatusByBranchIdAndXid = "update " + localTccLogPlaced + " set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? "

	// The constant DeleteByBranchIdAndXid

	deleteByBranchIdAndXid = "delete from " + localTccLogPlaced + " where xid = ? and  branch_id = ? "

	// The constant DeleteByDateAndStatus

	deleteByDateAndStatus = "delete from " + localTccLogPlaced + " where gmt_modified < ?  and status in (" + strconv.Itoa(int(constant.StatusCommitted)) + " , " + strconv.Itoa(int(constant.StatusRollbacked)) + " , " + strconv.Itoa(int(constant.StatusSuspended)) + ")"
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
