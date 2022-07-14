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

	insertLocalTccLog = "insert into %s  (xid, branch_id, action_name, status, gmt_create, gmt_modified)  values (?, ?, ?, ?, ?, ?) "

	//The constant QueryByBranchIdAndXid

	queryByBranchIdAndXid = "select xid, branch_id, status, gmt_create, gmt_modified from %s where xid = ? and branch_id = ? for update"

	// The constant UpdateStatusByBranchIdAndXid

	updateStatusByBranchIdAndXid = "update %s set status = ?, gmt_modified = ? where xid = ? and  branch_id = ? and status = ? "

	// The constant DeleteByBranchIdAndXid

	deleteByBranchIdAndXid = "delete from %s where xid = ? and  branch_id = ? "

	// The constant DeleteByDateAndStatus

	deleteByDateAndStatus = "delete from %s where gmt_modified < ?  and status in (" + strconv.Itoa(constant.StatusCommitted) + " , " + strconv.Itoa(constant.StatusRollbacked) + " , " + strconv.Itoa(constant.StatusSuspended) + ")"
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

func GetDeleteSQLByDateAndStatus(localTccTable string) string {
	return fmt.Sprintf(deleteByDateAndStatus, localTccTable)

}
