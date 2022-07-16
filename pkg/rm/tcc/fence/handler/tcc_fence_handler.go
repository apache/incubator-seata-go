package handler

import (
	"container/list"
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"

	"github.com/seata/seata-go/pkg/tm"

	"github.com/seata/seata-go/pkg/rm/tcc"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"

	"github.com/seata/seata-go/pkg/common/log"

	"go.uber.org/atomic"

	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/dao"
)

type TCCFenceHandler struct {
	tccFenceDao         dao.TCCFenceStore
	datasource          driver.Connector
	logQueue            list.List
	transactionTemplate interface{}
	done                atomic.Bool
}

type FenceLogIdentity struct {
	xid      string
	branchId int64
}

const (
	MaxTreadClean = 1
	MaxQueueSize  = 500
)

func init() {

}

func (handler *TCCFenceHandler) PrepareFence(xid string, branchId int64, actionName string, callback func()) interface{} {
	return nil
}

func (handler *TCCFenceHandler) CommitFence(proxy tcc.TCCServiceProxy, xid string, branchId int64, args ...interface{}) bool {
	// todo try catch and set rollback only.
	if conn, err := handler.datasource.Connect(context.Background()); err == nil {
		fenceDo := handler.tccFenceDao.QueryTCCFenceDO(conn, xid, branchId)
		if fenceDo == nil {
			panic(fmt.Sprintf("TCC fence record not exists, commit fence method failed. xid= %s, branchId= %d ", xid, branchId))
		}
		if fenceDo.Status == constant.StatusCommitted {
			log.Infof("Branch transaction has already committed before. idempotency rejected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
			return true
		}
		if fenceDo.Status == constant.StatusRollbacked || fenceDo.Status == constant.StatusSuspended {
			// enable warn level
			log.Warnf("Branch transaction status is unexpected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
			return false
		}
		return handler.updateStatusAndInvokeTargetMethod(conn, proxy, xid, branchId, constant.StatusCommitted, "transaction status obj")
	} else {
		panic("obtain connection failed ")
	}
	return false
}

func (handler *TCCFenceHandler) RollbackFence(proxy tcc.TCCServiceProxy, xid string, branchId int64, args ...interface{}) bool {
	// todo try catch and set rollback only.
	if conn, err := handler.datasource.Connect(context.Background()); err == nil {
		fenceDo := handler.tccFenceDao.QueryTCCFenceDO(conn, xid, branchId)
		if fenceDo == nil {
			result := handler.InsertTCCFenceLog(conn, xid, branchId, proxy.GetActionName(), constant.StatusSuspended)
			log.Infof("Insert tcc fence record result: %v. xid: %s, branchId: %d", result, xid, branchId)
			if !result {
				panic(fmt.Sprintf("Insert tcc fence record error, rollback fence method failed. xid= %s, branchId= %d", xid, branchId))
			}
			panic(fmt.Sprintf("TCC fence record not exists, commit fence method failed. xid= %s, branchId= %d ", xid, branchId))
			return true
		}
		if fenceDo.Status == constant.StatusRollbacked || fenceDo.Status == constant.StatusSuspended {
			// enable warn level
			log.Infof("Branch transaction had already rollbacked before, idempotency rejected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
			return true
		}
		if fenceDo.Status == constant.StatusCommitted {
			log.Warnf("Branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
			return false
		}
		return handler.updateStatusAndInvokeTargetMethod(conn, proxy, xid, branchId, constant.StatusRollbacked, "transaction status obj")
	} else {
		panic("obtain connection failed ")
	}
	return false
	return false
}

func (handler *TCCFenceHandler) InsertTCCFenceLog(conn driver.Conn, xid string, branchId int64, actionName string, status int32) bool {
	tccFenceDo := model.TCCFenceDO{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
		Status:     status,
	}
	return handler.tccFenceDao.InsertTCCFenceDO(conn, tccFenceDo)
}

func (handler *TCCFenceHandler) updateStatusAndInvokeTargetMethod(conn driver.Conn, proxy tcc.TCCServiceProxy, xid string, branchId int64, status int32, transactionStatus interface{}, args ...interface{}) bool {
	result := handler.tccFenceDao.UpdateTCCFenceDO(conn, xid, branchId, status, constant.StatusTried)
	if result {
		if status == constant.StatusCommitted {
			// todo implement invoke, two phase need return bool value.
			err := proxy.Commit(context.Background(), tm.BusinessActionContext{})
			if err != nil {
				// todo set rollback only
				result = false
			}
		}
	}
	return result
}

func (handler *TCCFenceHandler) DeleteFence(xid string, id int64) error {
	return nil
}

func (handler *TCCFenceHandler) InitLogCleanExecutor() {
	go handler.FenceLogCleanRunnable()
}

func (handler *TCCFenceHandler) DeleteFenceByDate(datetime time.Time) int32 {
	return 0
}

func (handler *TCCFenceHandler) AddToLogCleanQueue(xid string, branchId int64) {

}

func (handler *TCCFenceHandler) SetDatasource(connector driver.Connector) {
	handler.datasource = connector
}

func (handler *TCCFenceHandler) SetTransactionManager(transactionManager interface{}) {
	// todo
}

func (handler *TCCFenceHandler) FenceLogCleanRunnable() {
	for {
		logIdentity := handler.logQueue.Front().Value.(FenceLogIdentity)
		if err := handler.DeleteFence(logIdentity.xid, logIdentity.branchId); err != nil {
			log.Errorf("delete fence log failed, xid: %s, branchId: &s", logIdentity.xid, logIdentity.branchId)
		}
		if handler.done.String() == "true" {
			log.Errorf("take fence log from queue for clean be interrupted")
		}
		<-time.Tick(time.Duration(5))
	}
}
