package dao

import (
	"database/sql/driver"
	"sync"
	"time"

	sql2 "github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/sql"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"
)

type TCCFenceStore interface {
	QueryTCCFenceDO(conn driver.Conn, xid string, branchId int64) *model.TCCFenceDO

	InsertTCCFenceDO(conn driver.Conn, tccFenceDo model.TCCFenceDO) bool

	UpdateTCCFenceDO(conn driver.Conn, xid string, branchId int64, newStatus int32, oldStatus int32) bool

	DeleteTCCFenceDO(conn driver.Conn, xid string, branchId int64) bool

	DeleteTCCFenceDOByDate(conn driver.Conn, datetime time.Time) bool

	SetLogTableName(logTable string)
}

var (
	once                                                     = sync.Once{}
	tccFenceStoreDatabaseMapper *TccFenceStoreDatabaseMapper = nil
	logTableName                                             = "tcc_fence_log"
)

func GetTccFenceStoreDatabaseMapperSingleton() *TccFenceStoreDatabaseMapper {
	if tccFenceStoreDatabaseMapper == nil {
		once.Do(func() {
			tccFenceStoreDatabaseMapper = &TccFenceStoreDatabaseMapper{}
		})
	}
	return tccFenceStoreDatabaseMapper

}

type TccFenceStoreDatabaseMapper struct {
}

func (tcs *TccFenceStoreDatabaseMapper) QueryTCCFenceDO(conn driver.Conn, xid string, branchId int64) *model.TCCFenceDO {
	return &model.TCCFenceDO{}
}

func (tcs *TccFenceStoreDatabaseMapper) InsertTCCFenceDO(conn driver.Conn, tccFenceDo model.TCCFenceDO) bool {
	var prepareStmt driver.Stmt = nil
	timeNow := time.Now()
	sql := sql2.GetInsertLocalTCCLogSQL(logTableName)
	prepareStmt, err := conn.Prepare(sql)
	defer prepareStmt.Close()
	if err == nil {
		result, err := prepareStmt.Exec([]driver.Value{tccFenceDo.Xid, tccFenceDo.BranchId, tccFenceDo.ActionName, tccFenceDo.Status, timeNow, timeNow})
		if err == nil {
			if affected, err := result.RowsAffected(); err == nil {
				return affected > 0
			} else {
				panic("insert tcc fence get rows affected failed")
			}
		} else {
			panic("insert tcc fence execute sql failed")
		}
	} else {
		panic("insert tcc fence prepare sql failed")
	}
	return true
}

func (tcs *TccFenceStoreDatabaseMapper) UpdateTCCFenceDO(conn driver.Conn, xid string, branchId int64, newStatus int32, oldStatus int32) bool {
	return true
}

func (tcs *TccFenceStoreDatabaseMapper) DeleteTCCFenceDO(conn driver.Conn, xid string, branchId int64) bool {
	return true
}

func (tcs *TccFenceStoreDatabaseMapper) DeleteTCCFenceDOByDate(conn driver.Conn, datetime time.Time) bool {
	return true
}

func (tcs *TccFenceStoreDatabaseMapper) SetLogTableName(logTable string) {
	logTableName = logTable
}
