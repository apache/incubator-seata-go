package lock

import (
	"github.com/go-xorm/xorm"
	"xorm.io/builder"
)

import (
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/model"
)

const (
	QueryLockDO               = "select row_key, xid, transaction_id, branch_id, resource_id, table_name, pk, gmt_create, gmt_modified from lock_table"
	BatchDeleteLockByBranchId = `delete from lock_table where xid = ? AND branch_id = ?`
	GetLockDOCount            = "select count(1) as total from lock_table"
)

type LockStore interface {
	AcquireLockByLockDO(lockDO *model.LockDO) bool
	AcquireLock(lockDOs []*model.LockDO) bool
	UnLockByLockDO(lockDO *model.LockDO) bool
	UnLock(lockDOs []*model.LockDO) bool
	UnLockByXidAndBranchId(xid string, branchId int64) bool
	UnLockByXidAndBranchIds(xid string, branchIds []int64) bool
	IsLockable(lockDOs []*model.LockDO) bool
	GetLockCount() int64
}

type LockStoreDataBaseDao struct {
	engine *xorm.Engine
}

func (dao *LockStoreDataBaseDao) AcquireLockByLockDO(lockDO *model.LockDO) bool {
	var lockDOs = []*model.LockDO{lockDO}
	return dao.AcquireLock(lockDOs)
}

func (dao *LockStoreDataBaseDao) AcquireLock(lockDOs []*model.LockDO) bool {
	locks, rowKeys := distinctByKey(lockDOs)
	var existedRowLocks []*model.LockDO
	err := dao.engine.SQL(QueryLockDO).Where(builder.In("row_key", rowKeys)).Find(&existedRowLocks)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	currentXID := locks[0].Xid
	canLock := true
	existedRowKeys := make([]string, 0)
	unrepeatedLockDOs := make([]*model.LockDO, 0)
	for _, rowLock := range existedRowLocks {
		if rowLock.Xid != currentXID {
			logging.Logger.Infof("Global lock on [{%s}:{%s}] is holding by xid {%s} branchId {%d}", "lock_table", rowLock.Pk, rowLock.Xid,
				rowLock.BranchId)
			canLock = false
			break
		}
		existedRowKeys = append(existedRowKeys, rowLock.RowKey)
	}
	if !canLock {
		return false
	}
	if len(existedRowKeys) > 0 {
		for _, lock := range locks {
			if !contains(existedRowKeys, lock.RowKey) {
				unrepeatedLockDOs = append(unrepeatedLockDOs, lock)
			}
		}
	} else {
		unrepeatedLockDOs = locks
	}

	if len(unrepeatedLockDOs) == 0 {
		return true
	}

	_, err = dao.engine.Table("lock_table").Insert(unrepeatedLockDOs)
	if err != nil {
		logging.Logger.Errorf("Global locks batch acquire failed, %v", unrepeatedLockDOs)
		return false
	}
	return true
}

func distinctByKey(lockDOs []*model.LockDO) ([]*model.LockDO, []string) {
	result := make([]*model.LockDO, 0)
	rowKeys := make([]string, 0)
	lockMap := make(map[string]byte)
	for _, lockDO := range lockDOs {
		l := len(lockMap)
		lockMap[lockDO.RowKey] = 0
		if len(lockMap) != l {
			result = append(result, lockDO)
			rowKeys = append(rowKeys, lockDO.RowKey)
		}
	}
	return result, rowKeys
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (dao *LockStoreDataBaseDao) UnLockByLockDO(lockDO *model.LockDO) bool {
	var lockDOs = []*model.LockDO{lockDO}
	return dao.UnLock(lockDOs)
}

func (dao *LockStoreDataBaseDao) UnLock(lockDOs []*model.LockDO) bool {
	if lockDOs != nil && len(lockDOs) == 0 {
		return true
	}
	rowKeys := make([]string, 0)
	for _, lockDO := range lockDOs {
		rowKeys = append(rowKeys, lockDO.RowKey)
	}

	var lock = model.LockDO{}
	_, err := dao.engine.Table("lock_table").
		Where(builder.In("row_key", rowKeys).And(builder.Eq{"xid": lockDOs[0].Xid})).
		Delete(&lock)

	if err != nil {
		logging.Logger.Errorf(err.Error())
		return false
	}
	return true
}

func (dao *LockStoreDataBaseDao) UnLockByXidAndBranchId(xid string, branchId int64) bool {
	_, err := dao.engine.Exec(BatchDeleteLockByBranchId, xid, branchId)

	if err != nil {
		logging.Logger.Errorf(err.Error())
		return false
	}
	return true
}

func (dao *LockStoreDataBaseDao) UnLockByXidAndBranchIds(xid string, branchIds []int64) bool {
	var lock = model.LockDO{}
	_, err := dao.engine.Table("lock_table").
		Where(builder.In("branch_id", branchIds).And(builder.Eq{"xid": xid})).
		Delete(&lock)

	if err != nil {
		logging.Logger.Errorf(err.Error())
		return false
	}
	return true
}

func (dao *LockStoreDataBaseDao) IsLockable(lockDOs []*model.LockDO) bool {
	var existedRowLocks []*model.LockDO
	rowKeys := make([]string, 0)
	for _, lockDO := range lockDOs {
		rowKeys = append(rowKeys, lockDO.RowKey)
	}
	err := dao.engine.SQL(QueryLockDO).Where(builder.In("row_key", rowKeys)).Find(&existedRowLocks)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	currentXID := lockDOs[0].Xid
	for _, rowLock := range existedRowLocks {
		if rowLock.Xid != currentXID {
			return false
		}
	}
	return true
}

func (dao *LockStoreDataBaseDao) GetLockCount() int64 {
	var total int64
	dao.engine.SQL(GetLockDOCount).Cols("total").Get(&total)
	return total
}
