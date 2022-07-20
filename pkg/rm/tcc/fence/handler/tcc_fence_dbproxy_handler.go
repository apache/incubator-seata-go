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
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/datasource/sql"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/constant"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/dao"
	"github.com/seata/seata-go/pkg/rm/tcc/fence/store/db/model"
	"go.uber.org/atomic"
)

type TCCFenceDbProxyHandler struct {
	tccFenceDao dao.TCCFenceStore
	logQueue    list.List
	done        atomic.Bool
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

func (handler *TCCFenceDbProxyHandler) PrepareFence(txProxy *sql.Tx, proxy tcc.TCCServiceProxy, xid string, branchId int64, actionName string) bool {
	var result = false
	if conn, err := txProxy.GetConnProxy().GetTargetConn(); err == nil {
		result = handler.InsertTCCFenceLog(conn, xid, branchId, actionName, constant.StatusTried)
		if !result {

		} else {
			// this set rollback only is tag only.
			txProxy.SetRollbackOnly()
			panic(fmt.Sprintf("Insert tcc fence record error, prepare fence failed. xid= %s, branchId= %d", xid, branchId))
			handler.AddToLogCleanQueue(xid, branchId)
		}
	} else {
		txProxy.SetRollbackOnly()
		panic("obtain tcc proxy connection failed")
	}
	return result
}

func (handler *TCCFenceDbProxyHandler) CommitFence(txProxy *sql.Tx, proxy tcc.TCCServiceProxy, xid string, branchId int64, args ...interface{}) bool {
	if conn, err := txProxy.GetConnProxy().GetTargetConn(); txProxy.GetRollbackOnly() == false || err == nil {
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
		// do rollback ,if prepare fence have set rollback only or err is nil.
		txProxy.GetTargetTx().Rollback()
		panic("obtain connection failed ")
	}
	return false
}

func (handler *TCCFenceDbProxyHandler) RollbackFence(proxy tcc.TCCServiceProxy, xid string, branchId int64, args ...interface{}) bool {
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

func (handler *TCCFenceDbProxyHandler) InsertTCCFenceLog(conn driver.Conn, xid string, branchId int64, actionName string, status int32) bool {
	tccFenceDo := model.TCCFenceDO{
		Xid:        xid,
		BranchId:   branchId,
		ActionName: actionName,
		Status:     status,
	}
	return handler.tccFenceDao.InsertTCCFenceDO(conn, tccFenceDo)
}

func (handler *TCCFenceDbProxyHandler) updateStatusAndInvokeTargetMethod(conn driver.Conn, proxy tcc.TCCServiceProxy, xid string, branchId int64, status int32, transactionStatus interface{}, args ...interface{}) bool {
	result := handler.tccFenceDao.UpdateTCCFenceDO(conn, xid, branchId, status, constant.StatusTried)
	//if result {
	//if status == constant.StatusCommitted {
	//// todo implement invoke, two phase need return bool value.
	//err := proxy.Commit(context.Background(), tm.BusinessActionContext{})
	//if err != nil {
	//	// todo set rollback only
	//	result = false
	//}
	//}
	//}
	return result
}

func (handler *TCCFenceDbProxyHandler) DeleteFence(xid string, id int64) error {
	return nil
}

func (handler *TCCFenceDbProxyHandler) InitLogCleanExecutor() {
	go handler.FenceLogCleanRunnable()
}

func (handler *TCCFenceDbProxyHandler) DeleteFenceByDate(datetime time.Time) int32 {
	return 0
}

func (handler *TCCFenceDbProxyHandler) AddToLogCleanQueue(xid string, branchId int64) {
	fenceLogIdentity := &FenceLogIdentity{
		xid:      xid,
		branchId: branchId,
	}
	handler.logQueue.PushBack(fenceLogIdentity)
}

func (handler *TCCFenceDbProxyHandler) SetTransactionManager(transactionManager interface{}) {
	// todo
}

func (handler *TCCFenceDbProxyHandler) FenceLogCleanRunnable() {
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
