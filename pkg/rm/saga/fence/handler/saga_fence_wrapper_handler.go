package handler

import (
	"container/list"
	"context"
	"database/sql"
	"fmt"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/enum"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/dao"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	"sync"
	"time"
)

var (
	fenceHandler *sagaFenceWrapperHandler
	fenceOnce    sync.Once
)

const (
	maxQueueSize = 500
)

type sagaFenceWrapperHandler struct {
	tccFenceDao       dao.TCCFenceStore
	logQueue          chan *FenceLogIdentity
	logCache          list.List
	logQueueOnce      sync.Once
	logQueueCloseOnce sync.Once
}

type FenceLogIdentity struct {
	xid      string
	branchId int64
}

func GetSagaFenceHandler() *sagaFenceWrapperHandler {
	if fenceHandler == nil {
		fenceOnce.Do(func() {
			fenceHandler = &sagaFenceWrapperHandler{
				tccFenceDao: dao.GetTccFenceStoreDatabaseMapper(),
			}
		})
	}
	return fenceHandler
}

func (handler *sagaFenceWrapperHandler) ActionFence(ctx context.Context, tx *sql.Tx) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId

	fenceDo, err := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	if err != nil {
		return fmt.Errorf(" commit fence method failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
	}
	if fenceDo == nil {
		return fmt.Errorf("tcc fence record not exists, commit fence method failed. xid= %s, branchId= %d", xid, branchId)
	}

	if fenceDo.Status == enum.StatusCommitted {
		log.Infof("branch transaction has already committed before. idempotency rejected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == enum.StatusRollbacked || fenceDo.Status == enum.StatusSuspended {
		// enable warn level
		log.Warnf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return fmt.Errorf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
	}

	return handler.updateFenceStatus(tx, xid, branchId, enum.StatusCommitted)
}

func (handler *sagaFenceWrapperHandler) CompensationFence(ctx context.Context, tx *sql.Tx) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName
	fenceDo, err := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	if err != nil {
		return fmt.Errorf("rollback fence method failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
	}

	// record is null, mean the need suspend
	if fenceDo == nil {
		err = handler.insertSagaFenceLog(tx, xid, branchId, actionName, enum.StatusSuspended)
		if err != nil {
			return fmt.Errorf("insert tcc fence record errors, rollback fence failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
		}
		log.Infof("Insert tcc fence suspend record xid: %s, branchId: %d", xid, branchId)
		return nil
	}

	// have rollbacked or suspended
	if fenceDo.Status == enum.StatusRollbacked || fenceDo.Status == enum.StatusSuspended {
		// enable warn level
		log.Infof("Branch transaction had already rollbacked before, idempotency rejected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == enum.StatusCommitted {
		log.Warnf("Branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return fmt.Errorf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
	}

	return handler.updateFenceStatus(tx, xid, branchId, enum.StatusRollbacked)
}

func (handler *sagaFenceWrapperHandler) insertSagaFenceLog(tx *sql.Tx, xid string, branchId int64, actionName string, status enum.FenceStatus) error {
	tccFenceDo := model.TCCFenceDO{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
		Status:     status,
	}
	return handler.tccFenceDao.InsertTCCFenceDO(tx, &tccFenceDo)
}

func (handler *sagaFenceWrapperHandler) updateFenceStatus(tx *sql.Tx, xid string, branchId int64, status enum.FenceStatus) error {
	return handler.tccFenceDao.UpdateTCCFenceDO(tx, xid, branchId, enum.StatusTried, status)
}

func (handler *sagaFenceWrapperHandler) InitLogCleanChannel() {
	handler.logQueueOnce.Do(func() {
		go handler.traversalCleanChannel()
	})
}

func (handler *sagaFenceWrapperHandler) DestroyLogCleanChannel() {
	handler.logQueueCloseOnce.Do(func() {
		close(handler.logQueue)
	})
}

func (handler *sagaFenceWrapperHandler) deleteFence(xid string, id int64) error {
	// todo implement
	return nil
}

func (handler *sagaFenceWrapperHandler) deleteFenceByDate(datetime time.Time) int32 {
	// todo implement
	return 0
}

func (handler *sagaFenceWrapperHandler) pushCleanChannel(xid string, branchId int64) {
	// todo implement
	fli := &FenceLogIdentity{
		xid:      xid,
		branchId: branchId,
	}
	select {
	case handler.logQueue <- fli:
	// todo add batch delete from log cache.
	default:
		handler.logCache.PushBack(fli)
	}
	log.Infof("add one log to clean queue: %v ", fli)
}

func (handler *sagaFenceWrapperHandler) traversalCleanChannel() {
	handler.logQueue = make(chan *FenceLogIdentity, maxQueueSize)
	for li := range handler.logQueue {
		if err := handler.deleteFence(li.xid, li.branchId); err != nil {
			log.Errorf("delete fence log failed, xid: %s, branchId: &s", li.xid, li.branchId)
		}
	}
}
