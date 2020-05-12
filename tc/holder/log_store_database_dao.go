package holder

import (
	"github.com/Davmuz/gqt"
	"github.com/go-xorm/xorm"
	"xorm.io/builder"
)

import (
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/model"
)

type ILogStore interface {
	QueryGlobalTransactionDOByXid(xid string) *model.GlobalTransactionDO
	QueryGlobalTransactionDOByTransactionId(transactionId int64) *model.GlobalTransactionDO
	QueryGlobalTransactionDOByStatuses(statuses []int,limit int) []*model.GlobalTransactionDO
	InsertGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool
	UpdateGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool
	DeleteGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool
	QueryBranchTransactionDOByXid(xid string) []*model.BranchTransactionDO
	QueryBranchTransactionDOByXids(xids []string) []*model.BranchTransactionDO
	InsertBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool
	UpdateBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool
	DeleteBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool
	GetCurrentMaxSessionId(high int64,low int64) int64
}

type LogStoreDataBaseDAO struct {
	engine *xorm.Engine
}


func (dao *LogStoreDataBaseDAO) QueryGlobalTransactionDOByXid(xid string) *model.GlobalTransactionDO {
	var globalTransactionDO model.GlobalTransactionDO
	has, err := dao.engine.SQL(gqt.Get("_queryGlobalTransactionDOByXid"),xid).
		Get(&globalTransactionDO)
	if has {
		return &globalTransactionDO
	}
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return nil
}

func (dao *LogStoreDataBaseDAO) QueryGlobalTransactionDOByTransactionId(transactionId int64) *model.GlobalTransactionDO {
	var globalTransactionDO model.GlobalTransactionDO
	has, err := dao.engine.SQL(gqt.Get("_queryGlobalTransactionDOByTransactionId"),transactionId).
		Get(&globalTransactionDO)
	if has {
		return &globalTransactionDO
	}
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return nil
}

func (dao *LogStoreDataBaseDAO) QueryGlobalTransactionDOByStatuses(statuses []int,limit int) []*model.GlobalTransactionDO {
	var globalTransactionDOs []*model.GlobalTransactionDO
	err := dao.engine.Table("global_table").
		Where(builder.In("status",statuses)).
		OrderBy("gmt_modified").
		Limit(limit).
		Find(&globalTransactionDOs)

	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return globalTransactionDOs
}

func (dao *LogStoreDataBaseDAO) InsertGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool {
	_, err := dao.engine.Exec(gqt.Get("_insertGlobalTransactionDO"),
		globalTransaction.Xid,
		globalTransaction.TransactionId,
		globalTransaction.Status,
		globalTransaction.ApplicationId,
		globalTransaction.TransactionServiceGroup,
		globalTransaction.TransactionName,
		globalTransaction.Timeout,
		globalTransaction.BeginTime,
		globalTransaction.ApplicationData)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) UpdateGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool {
	_, err := dao.engine.Exec(gqt.Get("_updateGlobalTransactionDO"),
		globalTransaction.Status,
		globalTransaction.Xid)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) DeleteGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool {
	_, err := dao.engine.Exec(gqt.Get("_deleteGlobalTransactionDO"),
		globalTransaction.Xid)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) QueryBranchTransactionDOByXid(xid string) []*model.BranchTransactionDO {
	var branchTransactionDos []*model.BranchTransactionDO
	err :=dao.engine.SQL(gqt.Get("_queryBranchTransactionDOByXid"),xid).
		Find(&branchTransactionDos)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return branchTransactionDos
}

func (dao *LogStoreDataBaseDAO) QueryBranchTransactionDOByXids(xids []string) []*model.BranchTransactionDO {
	var branchTransactionDos []*model.BranchTransactionDO
	err :=dao.engine.Table("branch_table").
		Where(builder.In("xid",xids)).
		OrderBy("gmt_create asc").
		Find(&branchTransactionDos)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return branchTransactionDos
}

func (dao *LogStoreDataBaseDAO) InsertBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool {
	_, err := dao.engine.Exec(gqt.Get("_insertBranchTransactionDO"),
		branchTransaction.Xid,
		branchTransaction.BranchId,
		branchTransaction.TransactionId,
		branchTransaction.ResourceGroupId,
		branchTransaction.ResourceId,
		branchTransaction.BranchType,
		branchTransaction.Status,
		branchTransaction.ClientId,
		branchTransaction.ApplicationData)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) UpdateBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool {
	_, err := dao.engine.Exec(gqt.Get("_updateBranchTransactionDO"),
		branchTransaction.Status,
		branchTransaction.Xid,
		branchTransaction.BranchId)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) DeleteBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool {
	_, err := dao.engine.Exec(gqt.Get("_deleteBranchTransactionDO"),
		branchTransaction.Xid,
		branchTransaction.BranchId)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) GetCurrentMaxSessionId(high int64,low int64) int64 {
	var maxTransactionId,maxBranchId int64
	_, err := dao.engine.SQL(gqt.Get("_queryMaxTransactionId"),high,low).
		Cols("maxTransactionId").
		Get(&maxTransactionId)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	_, err = dao.engine.SQL(gqt.Get("_queryMaxBranchId"),high,low).
		Cols("maxBranchId").
		Get(&maxBranchId)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	if maxTransactionId > maxBranchId {
		return maxTransactionId
	}
	return maxBranchId
}