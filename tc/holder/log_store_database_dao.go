package holder

import (
	"github.com/go-xorm/xorm"
	"xorm.io/builder"
)

import (
	"github.com/dk-lockdown/seata-golang/pkg/logging"
	"github.com/dk-lockdown/seata-golang/tc/model"
)

const (
	QueryGlobalTransactionDOByXid = `select xid, transaction_id, status, application_id, transaction_service_group, transaction_name,
		timeout, begin_time, application_data, gmt_create, gmt_modified from global_table where xid = ?`
	QueryGlobalTransactionDOByTransactionId = `select xid, transaction_id, status, application_id, transaction_service_group, transaction_name,
		timeout, begin_time, application_data, gmt_create, gmt_modified from global_table where transaction_id = ?`
	InsertGlobalTransactionDO = `insert into global_table (xid, transaction_id, status, application_id, transaction_service_group,
        transaction_name, timeout, begin_time, application_data, gmt_create, gmt_modified) values(?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now())`
	UpdateGlobalTransactionDO     = "update global_table set status = ?, gmt_modified = now() where xid = ?"
	DeleteGlobalTransactionDO     = "delete from global_table where xid = ?"
	QueryBranchTransactionDOByXid = `select xid, branch_id, transaction_id, resource_group_id, resource_id, branch_type, status, client_id,
	    application_data, gmt_create, gmt_modified from branch_table where xid = ? order by gmt_create asc`
	InsertBranchTransactionDO = `insert into branch_table (xid, branch_id, transaction_id, resource_group_id, resource_id, branch_type,
        status, client_id, application_data, gmt_create, gmt_modified) values(?, ?, ?, ?, ?, ?, ?, ?, ?, now(6), now(6))`
	UpdateBranchTransactionDO = "update branch_table set status = ?, gmt_modified = now(6) where xid = ? and branch_id = ?"
	DeleteBranchTransactionDO = "delete from branch_table where xid = ? and branch_id = ?"
	QueryMaxTransactionId     = "select max(transaction_id) as maxTransactioId from global_table where transaction_id < ? and transaction_id > ?"
	QueryMaxBranchId          = "select max(branch_id) as maxBranchId from branch_table where branch_id < ? and branch_id > ?"
)

type ILogStore interface {
	QueryGlobalTransactionDOByXid(xid string) *model.GlobalTransactionDO
	QueryGlobalTransactionDOByTransactionId(transactionId int64) *model.GlobalTransactionDO
	QueryGlobalTransactionDOByStatuses(statuses []int, limit int) []*model.GlobalTransactionDO
	InsertGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool
	UpdateGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool
	DeleteGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool
	QueryBranchTransactionDOByXid(xid string) []*model.BranchTransactionDO
	QueryBranchTransactionDOByXids(xids []string) []*model.BranchTransactionDO
	InsertBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool
	UpdateBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool
	DeleteBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool
	GetCurrentMaxSessionId(high int64, low int64) int64
}

type LogStoreDataBaseDAO struct {
	engine *xorm.Engine
}

func (dao *LogStoreDataBaseDAO) QueryGlobalTransactionDOByXid(xid string) *model.GlobalTransactionDO {
	var globalTransactionDO model.GlobalTransactionDO
	has, err := dao.engine.SQL(QueryGlobalTransactionDOByXid, xid).
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
	has, err := dao.engine.SQL(QueryGlobalTransactionDOByTransactionId, transactionId).
		Get(&globalTransactionDO)
	if has {
		return &globalTransactionDO
	}
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return nil
}

func (dao *LogStoreDataBaseDAO) QueryGlobalTransactionDOByStatuses(statuses []int, limit int) []*model.GlobalTransactionDO {
	var globalTransactionDOs []*model.GlobalTransactionDO
	err := dao.engine.Table("global_table").
		Where(builder.In("status", statuses)).
		OrderBy("gmt_modified").
		Limit(limit).
		Find(&globalTransactionDOs)

	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return globalTransactionDOs
}

func (dao *LogStoreDataBaseDAO) InsertGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool {
	_, err := dao.engine.Exec(InsertGlobalTransactionDO,
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
	_, err := dao.engine.Exec(UpdateGlobalTransactionDO, globalTransaction.Status, globalTransaction.Xid)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) DeleteGlobalTransactionDO(globalTransaction model.GlobalTransactionDO) bool {
	_, err := dao.engine.Exec(DeleteGlobalTransactionDO, globalTransaction.Xid)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) QueryBranchTransactionDOByXid(xid string) []*model.BranchTransactionDO {
	var branchTransactionDos []*model.BranchTransactionDO
	err := dao.engine.SQL(QueryBranchTransactionDOByXid, xid).Find(&branchTransactionDos)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return branchTransactionDos
}

func (dao *LogStoreDataBaseDAO) QueryBranchTransactionDOByXids(xids []string) []*model.BranchTransactionDO {
	var branchTransactionDos []*model.BranchTransactionDO
	err := dao.engine.Table("branch_table").
		Where(builder.In("xid", xids)).
		OrderBy("gmt_create asc").
		Find(&branchTransactionDos)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	return branchTransactionDos
}

func (dao *LogStoreDataBaseDAO) InsertBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool {
	_, err := dao.engine.Exec(InsertBranchTransactionDO,
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
	_, err := dao.engine.Exec(UpdateBranchTransactionDO,
		branchTransaction.Status,
		branchTransaction.Xid,
		branchTransaction.BranchId)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) DeleteBranchTransactionDO(branchTransaction model.BranchTransactionDO) bool {
	_, err := dao.engine.Exec(DeleteBranchTransactionDO,
		branchTransaction.Xid,
		branchTransaction.BranchId)
	if err == nil {
		return true
	}
	return false
}

func (dao *LogStoreDataBaseDAO) GetCurrentMaxSessionId(high int64, low int64) int64 {
	var maxTransactionId, maxBranchId int64
	_, err := dao.engine.SQL(QueryMaxTransactionId, high, low).
		Cols("maxTransactionId").
		Get(&maxTransactionId)
	if err != nil {
		logging.Logger.Errorf(err.Error())
	}
	_, err = dao.engine.SQL(QueryMaxBranchId, high, low).
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
