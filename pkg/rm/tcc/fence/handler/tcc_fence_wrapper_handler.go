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
	"container/list"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"

	"seata.apache.org/seata-go/pkg/rm/tcc/fence/enum"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/dao"
	"seata.apache.org/seata-go/pkg/rm/tcc/fence/store/db/model"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
)

type tccFenceWrapperHandler struct {
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

const (
	maxQueueSize = 500
)

var (
	fenceHandler *tccFenceWrapperHandler
	fenceOnce    sync.Once
)

func GetFenceHandler() *tccFenceWrapperHandler {
	if fenceHandler == nil {
		fenceOnce.Do(func() {
			fenceHandler = &tccFenceWrapperHandler{
				tccFenceDao: dao.GetTccFenceStoreDatabaseMapper(),
			}
		})
	}
	return fenceHandler
}

func (handler *tccFenceWrapperHandler) PrepareFence(ctx context.Context, tx *sql.Tx) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName

	err := handler.insertTCCFenceLog(tx, xid, branchId, actionName, enum.StatusTried)
	if err != nil {
		if mysqlError, ok := errors.Unwrap(err).(*mysql.MySQLError); ok && mysqlError.Number == 1062 {
			// todo add clean command to channel.
			handler.pushCleanChannel(xid, branchId)
		}
		return fmt.Errorf("insert tcc fence record errors, prepare fence failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
	}

	return nil
}

func (handler *tccFenceWrapperHandler) CommitFence(ctx context.Context, tx *sql.Tx) error {
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

func (handler *tccFenceWrapperHandler) RollbackFence(ctx context.Context, tx *sql.Tx) error {
	xid := tm.GetBusinessActionContext(ctx).Xid
	branchId := tm.GetBusinessActionContext(ctx).BranchId
	actionName := tm.GetBusinessActionContext(ctx).ActionName
	fenceDo, err := handler.tccFenceDao.QueryTCCFenceDO(tx, xid, branchId)
	if err != nil {
		return fmt.Errorf("rollback fence method failed. xid= %s, branchId= %d, [%w]", xid, branchId, err)
	}

	// record is null, mean the need suspend
	if fenceDo == nil {
		err = handler.insertTCCFenceLog(tx, xid, branchId, actionName, enum.StatusSuspended)
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

func (handler *tccFenceWrapperHandler) insertTCCFenceLog(tx *sql.Tx, xid string, branchId int64, actionName string, status enum.FenceStatus) error {
	tccFenceDo := model.TCCFenceDO{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
		Status:     status,
	}
	return handler.tccFenceDao.InsertTCCFenceDO(tx, &tccFenceDo)
}

func (handler *tccFenceWrapperHandler) updateFenceStatus(tx *sql.Tx, xid string, branchId int64, status enum.FenceStatus) error {
	return handler.tccFenceDao.UpdateTCCFenceDO(tx, xid, branchId, enum.StatusTried, status)
}

func (handler *tccFenceWrapperHandler) InitLogCleanChannel() {
	handler.logQueueOnce.Do(func() {
		go handler.traversalCleanChannel()
	})
}

func (handler *tccFenceWrapperHandler) DestroyLogCleanChannel() {
	handler.logQueueCloseOnce.Do(func() {
		close(handler.logQueue)
	})
}

func (handler *tccFenceWrapperHandler) deleteFence(xid string, id int64) error {
	// todo implement
	return nil
}

func (handler *tccFenceWrapperHandler) deleteFenceByDate(datetime time.Time) int32 {
	// todo implement
	return 0
}

func (handler *tccFenceWrapperHandler) pushCleanChannel(xid string, branchId int64) {
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

func (handler *tccFenceWrapperHandler) traversalCleanChannel() {
	handler.logQueue = make(chan *FenceLogIdentity, maxQueueSize)
	for li := range handler.logQueue {
		if err := handler.deleteFence(li.xid, li.branchId); err != nil {
			log.Errorf("delete fence log failed, xid: %s, branchId: &s", li.xid, li.branchId)
		}
	}
}
