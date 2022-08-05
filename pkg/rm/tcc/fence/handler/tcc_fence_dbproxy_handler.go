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

package handler

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	seataErrors "github.com/seata/seata-go/pkg/common/errors"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/dao"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"
	"github.com/seata/seata-go/pkg/tm"
)

type tccFenceDbProxyHandler struct {
	tccFenceDao       dao.TCCFenceStore
	logQueue          chan interface{}
	logQueueOnce      sync.Once
	logQueueCloseOnce sync.Once
}

type FenceLogIdentity struct {
	xid      string
	branchId int64
}

const (
	MaxTreadClean = 1
	MaxQueueSize  = 500
)

var (
	fenceHandlerSingleton *tccFenceDbProxyHandler
	fenceOnce             sync.Once
)

func init() {

}

func GetFenceHandlerSingleton() *tccFenceDbProxyHandler {
	if fenceHandlerSingleton == nil {
		fenceOnce.Do(func() {
			fenceHandlerSingleton = &tccFenceDbProxyHandler{
				tccFenceDao: dao.GetTccFenceStoreDatabaseMapperSingleton(),
			}
		})
	}
	return fenceHandlerSingleton
}

func (handler *tccFenceDbProxyHandler) PrepareFence(ctx context.Context, tx *sql.Tx, callback func() error) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName

	defer func() {
		rec := recover()
		err, ok := rec.(seataErrors.TccFenceError)
		if ok {
			if err.Code == seataErrors.FenceErrorCodeDuplicateKey {
				handler.AddToLogCleanQueue(xid, branchId)
			}
		}
		// panic uniform process in outside method
		panic(rec)
	}()

	ok := handler.InsertTCCFenceLog(tx, xid, branchId, actionName, constant.StatusTried)
	log.Infof("tcc fence prepare result: %b. xid: %s, branchId: %d", ok, xid, branchId)
	if ok {
		err := callback()
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return errors.New(err.Error() + rollbackErr.Error())
			}
			return err
		}
		return tx.Commit()
	} else {
		return seataErrors.NewTccFenceError(seataErrors.InsertRecordError, fmt.Sprintf("insert tcc fence record errors, prepare fence failed. xid= %s, branchId= %d", xid, branchId))
	}
}

func (handler *tccFenceDbProxyHandler) CommitFence(ctx context.Context, tx *sql.Tx, callback func() error) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId

	fenceDo := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	if fenceDo == nil {
		return seataErrors.NewTccFenceError(seataErrors.RecordNotExists, fmt.Sprintf("tcc fence record not exists, commit fence method failed. xid= %s, branchId= %d ", xid, branchId))
	}
	if fenceDo.Status == constant.StatusCommitted {
		log.Infof("branch transaction has already committed before. idempotency rejected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == constant.StatusRollbacked || fenceDo.Status == constant.StatusSuspended {
		// enable warn level
		log.Warnf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return fmt.Errorf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
	}
	return handler.updateStatusAndInvokeTargetMethod(tx, callback, xid, branchId, constant.StatusCommitted)
}

func (handler *tccFenceDbProxyHandler) RollbackFence(ctx context.Context, tx *sql.Tx, callback func() error) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName

	fenceDo := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	// record is null, mean the need suspend
	if fenceDo == nil {
		ok := handler.InsertTCCFenceLog(tx, xid, branchId, actionName, constant.StatusSuspended)
		log.Infof("Insert tcc fence record ok: %v. xid: %s, branchId: %d", ok, xid, branchId)
		if ok {
			log.Infof("TCC fence record not exists, rollback fence method failed. xid= %s, branchId= %d ", xid, branchId)
			return nil
		} else {
			return seataErrors.NewTccFenceError(seataErrors.InsertRecordError, fmt.Sprintf("insert tcc fence record error, rollback fence method failed. xid= %s, branchId= %d", xid, branchId))
		}
	}

	// have rollback or suspend
	if fenceDo.Status == constant.StatusRollbacked || fenceDo.Status == constant.StatusSuspended {
		// enable warn level
		log.Infof("Branch transaction had already rollbacked before, idempotency rejected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == constant.StatusCommitted {
		log.Warnf("Branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return fmt.Errorf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
	}
	return handler.updateStatusAndInvokeTargetMethod(tx, callback, xid, branchId, constant.StatusRollbacked)
}

func (handler *tccFenceDbProxyHandler) InsertTCCFenceLog(tx *sql.Tx, xid string, branchId int64, actionName string, status int32) bool {
	tccFenceDo := model.TCCFenceDO{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
		Status:     status,
	}
	return handler.tccFenceDao.InsertTCCFenceDO(tx, &tccFenceDo)
}

func (handler *tccFenceDbProxyHandler) updateStatusAndInvokeTargetMethod(tx *sql.Tx, callback func() error, xid string, branchId int64, status int32) error {
	ok := handler.tccFenceDao.UpdateTCCFenceDO(tx, xid, branchId, constant.StatusTried, status)
	if ok {
		err := callback()
		if err != nil {
			return tx.Rollback()
		}
		return tx.Commit()
	} else {
		return seataErrors.NewTccFenceError(seataErrors.UpdateRecordError, fmt.Sprintf("update record error xid %s, branch id %d, target status %d", xid, branchId, status))
	}
}

func (handler *tccFenceDbProxyHandler) DeleteFence(xid string, id int64) error {
	return nil
}

func (handler *tccFenceDbProxyHandler) InitLogCleanExecutor() {
	handler.logQueueOnce.Do(func() {
		go handler.fenceLogCleanRunnable()
	})
}

func (handler *tccFenceDbProxyHandler) DestroyLogCleanExecutor() {
	handler.logQueueCloseOnce.Do(func() {
		close(handler.logQueue)
	})
}

func (handler *tccFenceDbProxyHandler) DeleteFenceByDate(datetime time.Time) int32 {
	return 0
}

func (handler *tccFenceDbProxyHandler) AddToLogCleanQueue(xid string, branchId int64) {
	fenceLogIdentity := &FenceLogIdentity{
		xid:      xid,
		branchId: branchId,
	}
	handler.logQueue <- fenceLogIdentity
}

func (handler *tccFenceDbProxyHandler) fenceLogCleanRunnable() {
	handler.logQueue = make(chan interface{}, MaxQueueSize)
	for fenceLog := range handler.logQueue {
		logIdentity := fenceLog.(FenceLogIdentity)
		if err := handler.DeleteFence(logIdentity.xid, logIdentity.branchId); err != nil {
			log.Errorf("delete fence log failed, xid: %s, branchId: &s", logIdentity.xid, logIdentity.branchId)
		}
	}
}
