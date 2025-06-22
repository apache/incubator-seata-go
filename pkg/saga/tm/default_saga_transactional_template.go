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

package tm

import (
	"context"
	"github.com/seata/seata-go/pkg/rm"
	"time"

	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

type DefaultSagaTransactionalTemplate struct {
	applicationId  string
	txServiceGroup string
}

func (t *DefaultSagaTransactionalTemplate) CommitTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error {
	t.triggerBeforeCommit()
	err := tm.GetGlobalTransactionManager().Commit(ctx, gtr)
	if err != nil {
		return err
	}
	t.triggerAfterCommit()
	return nil
}

func (t *DefaultSagaTransactionalTemplate) RollbackTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error {
	t.triggerBeforeRollback()
	err := tm.GetGlobalTransactionManager().Rollback(ctx, gtr)
	if err != nil {
		return err
	}
	t.triggerAfterRollback()
	return nil
}

func (t *DefaultSagaTransactionalTemplate) BeginTransaction(ctx context.Context, timeout time.Duration, txName string) (*tm.GlobalTransaction, error) {
	t.triggerBeforeBegin()
	tm.SetTxName(ctx, txName)
	err := tm.GetGlobalTransactionManager().Begin(ctx, timeout)
	if err != nil {
		return nil, err
	}
	t.triggerAfterBegin()
	return tm.GetTx(ctx), nil
}

func (t *DefaultSagaTransactionalTemplate) ReloadTransaction(ctx context.Context, xid string) (*tm.GlobalTransaction, error) {
	return &tm.GlobalTransaction{
		Xid:      xid,
		TxStatus: message.GlobalStatusUnKnown,
		TxRole:   tm.Launcher,
	}, nil
}

func (t *DefaultSagaTransactionalTemplate) ReportTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error {
	_, err := tm.GetGlobalTransactionManager().GlobalReport(ctx, gtr)
	if err != nil {
		return err
	}
	t.triggerAfterCommit()
	return nil
}

func (t *DefaultSagaTransactionalTemplate) BranchRegister(ctx context.Context, resourceId string, clientId string, xid string, applicationData string, lockKeys string) (int64, error) {
	//todo Wait implement sagaResource
	return rm.GetRMRemotingInstance().BranchRegister(rm.BranchRegisterParam{
		BranchType:      branch.BranchTypeSAGA,
		ResourceId:      resourceId,
		Xid:             xid,
		ClientId:        clientId,
		ApplicationData: applicationData,
		LockKeys:        lockKeys,
	})
}

func (t *DefaultSagaTransactionalTemplate) BranchReport(ctx context.Context, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	//todo Wait implement sagaResource
	return rm.GetRMRemotingInstance().BranchReport(rm.BranchReportParam{
		BranchType:      branch.BranchTypeSAGA,
		Xid:             xid,
		BranchId:        branchId,
		Status:          status,
		ApplicationData: applicationData,
	})
}

func (t *DefaultSagaTransactionalTemplate) CleanUp(ctx context.Context) {
	tm.GetTransactionalHookManager().Clear()
}

func (t *DefaultSagaTransactionalTemplate) getCurrentHooks() []tm.TransactionalHook {
	return tm.GetTransactionalHookManager().GetHooks()
}

func (t *DefaultSagaTransactionalTemplate) triggerBeforeBegin() {
	for _, hook := range t.getCurrentHooks() {
		err := hook.BeforeBegin()
		if nil != err {
			log.Error(err)
		}
	}
}

func (t *DefaultSagaTransactionalTemplate) triggerAfterBegin() {
	for _, hook := range t.getCurrentHooks() {
		err := hook.AfterBegin()
		if nil != err {
			log.Error(err)
		}
	}
}

func (t *DefaultSagaTransactionalTemplate) triggerBeforeRollback() {
	for _, hook := range t.getCurrentHooks() {
		err := hook.BeforeRollback()
		if nil != err {
			log.Error(err)
		}
	}
}

func (t *DefaultSagaTransactionalTemplate) triggerAfterRollback() {
	for _, hook := range t.getCurrentHooks() {
		err := hook.AfterRollback()
		if nil != err {
			log.Error(err)
		}
	}
}

func (t *DefaultSagaTransactionalTemplate) triggerBeforeCommit() {
	for _, hook := range t.getCurrentHooks() {
		err := hook.BeforeCommit()
		if nil != err {
			log.Error(err)
		}
	}
}

func (t *DefaultSagaTransactionalTemplate) triggerAfterCommit() {
	for _, hook := range t.getCurrentHooks() {
		err := hook.AfterCommit()
		if nil != err {
			log.Error(err)
		}
	}
}

func (t *DefaultSagaTransactionalTemplate) triggerAfterCompletion() {
	for _, hook := range t.getCurrentHooks() {
		err := hook.AfterCompletion()
		if nil != err {
			log.Error(err)
		}
	}
}
