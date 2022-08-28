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

	seataErrors "github.com/seata/seata-go/pkg/common/errors"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/dao"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"
	"github.com/seata/seata-go/pkg/tm"
)

type tccFenceDbProxyHandler struct {
	tccFenceDao       dao.TCCFenceStore
	logQueue          chan *FenceLogIdentity
	logQueueOnce      sync.Once
	logQueueCloseOnce sync.Once
}

type FenceLogIdentity struct {
	xid      string
	branchId int64
}

const (
	MaxQueueSize = 500
)

var (
	fenceHandlerSingleton *tccFenceDbProxyHandler
	fenceOnce             sync.Once
)

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

	err := handler.insertTCCFenceLog(tx, xid, branchId, actionName, constant.StatusTried)
	if err != nil {
		dbError, ok := err.(seataErrors.TccFenceError)
		if ok && dbError.Code == seataErrors.TccFenceDbDuplicateKeyError {
			handler.addToLogCleanQueue(xid, branchId)
		}

		return seataErrors.NewTccFenceError(
			seataErrors.PrepareFenceError,
			fmt.Sprintf("insert tcc fence record errors, prepare fence failed. xid= %s, branchId= %d", xid, branchId),
			err,
		)
	}

	log.Infof("to call the business method: %p", callback)
	err = callback()
	if err != nil {
		return seataErrors.NewTccFenceError(
			seataErrors.FenceBusinessError,
			fmt.Sprintf("the business method error msg of: %p", callback),
			err,
		)
	}

	return nil
}

func (handler *tccFenceDbProxyHandler) CommitFence(ctx context.Context, tx *sql.Tx, callback func() error) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId

	fenceDo, err := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	if err != nil {
		return seataErrors.NewTccFenceError(seataErrors.CommitFenceError,
			fmt.Sprintf(" commit fence method failed. xid= %s, branchId= %d ", xid, branchId),
			err,
		)
	}
	if fenceDo == nil {
		return seataErrors.NewTccFenceError(seataErrors.CommitFenceError,
			fmt.Sprintf("tcc fence record not exists, commit fence method failed. xid= %s, branchId= %d ", xid, branchId),
			err,
		)
	}

	if fenceDo.Status == constant.StatusCommitted {
		log.Infof("branch transaction has already committed before. idempotency rejected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == constant.StatusRollbacked || fenceDo.Status == constant.StatusSuspended {
		// enable warn level
		log.Warnf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return seataErrors.NewTccFenceError(seataErrors.CommitFenceError,
			fmt.Sprintf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status),
			nil,
		)
	}

	return handler.updateStatusAndInvokeTargetMethod(tx, callback, xid, branchId, constant.StatusCommitted)
}

func (handler *tccFenceDbProxyHandler) RollbackFence(ctx context.Context, tx *sql.Tx, callback func() error) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName

	fenceDo, err := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)

	if err != nil {
		return seataErrors.NewTccFenceError(seataErrors.RollbackFenceError,
			fmt.Sprintf(" commit fence method failed. xid= %s, branchId= %d ", xid, branchId),
			err,
		)
	}

	// record is null, mean the need suspend
	if fenceDo == nil {
		err = handler.insertTCCFenceLog(tx, xid, branchId, actionName, constant.StatusSuspended)
		if err != nil {
			return seataErrors.NewTccFenceError(seataErrors.RollbackFenceError,
				fmt.Sprintf("insert tcc fence suspend record error, rollback fence method failed. xid= %s, branchId= %d", xid, branchId),
				err,
			)
		}
		log.Infof("Insert tcc fence suspend record xid: %s, branchId: %d", xid, branchId)
		return nil
	}

	// have rollback or suspend
	if fenceDo.Status == constant.StatusRollbacked || fenceDo.Status == constant.StatusSuspended {
		// enable warn level
		log.Infof("Branch transaction had already rollbacked before, idempotency rejected. xid: %s, branchId: %d, status: %s", xid, branchId, fenceDo.Status)
		return nil
	}
	if fenceDo.Status == constant.StatusCommitted {
		log.Warnf("Branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status)
		return seataErrors.NewTccFenceError(seataErrors.RollbackFenceError,
			fmt.Sprintf("branch transaction status is unexpected. xid: %s, branchId: %d, status: %d", xid, branchId, fenceDo.Status),
			err,
		)
	}

	return handler.updateStatusAndInvokeTargetMethod(tx, callback, xid, branchId, constant.StatusRollbacked)
}

func (handler *tccFenceDbProxyHandler) insertTCCFenceLog(tx *sql.Tx, xid string, branchId int64, actionName string, status constant.FenceStatus) error {
	tccFenceDo := model.TCCFenceDO{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
		Status:     status,
	}
	return handler.tccFenceDao.InsertTCCFenceDO(tx, &tccFenceDo)
}

func (handler *tccFenceDbProxyHandler) updateStatusAndInvokeTargetMethod(tx *sql.Tx, callback func() error, xid string, branchId int64, status constant.FenceStatus) error {
	err := handler.tccFenceDao.UpdateTCCFenceDO(tx, xid, branchId, constant.StatusTried, status)
	if err != nil {
		return err
	}

	log.Infof("to call the business method: %p", callback)
	err = callback()
	if err != nil {
		return seataErrors.NewTccFenceError(
			seataErrors.FenceBusinessError,
			fmt.Sprintf("the business method error msg of: %p", callback),
			err,
		)
	}

	return nil
}

func (handler *tccFenceDbProxyHandler) InitLogCleanExecutor() {
	handler.logQueueOnce.Do(func() {
		go handler.startFenceLogCleanRunnable()
	})
}

func (handler *tccFenceDbProxyHandler) DestroyLogCleanExecutor() {
	handler.logQueueCloseOnce.Do(func() {
		close(handler.logQueue)
	})
}

func (handler *tccFenceDbProxyHandler) deleteFence(xid string, id int64) error {
	// todo implement
	return nil
}

func (handler *tccFenceDbProxyHandler) deleteFenceByDate(datetime time.Time) int32 {
	// todo implement
	return 0
}

func (handler *tccFenceDbProxyHandler) addToLogCleanQueue(xid string, branchId int64) {
	// todo implement
	fenceLogIdentity := &FenceLogIdentity{
		xid:      xid,
		branchId: branchId,
	}
	handler.logQueue <- fenceLogIdentity
	log.Infof("add one log to clean queue: %v ", fenceLogIdentity)
}

func (handler *tccFenceDbProxyHandler) startFenceLogCleanRunnable() {
	handler.logQueue = make(chan *FenceLogIdentity, MaxQueueSize)
	for logIdentity := range handler.logQueue {
		if err := handler.deleteFence(logIdentity.xid, logIdentity.branchId); err != nil {
			log.Errorf("delete fence log failed, xid: %s, branchId: &s", logIdentity.xid, logIdentity.branchId)
		}
	}
}
