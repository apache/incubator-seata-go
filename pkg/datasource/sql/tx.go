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

package sql

import (
	"context"
	"database/sql/driver"
	"sync"

	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/util/log"

	"github.com/seata/seata-go/pkg/datasource/sql/types"
)

const REPORT_RETRY_COUNT = 5

var (
	hl      sync.RWMutex
	txHooks []txHook
)

func RegisterTxHook(h txHook) {
	hl.Lock()
	defer hl.Unlock()

	txHooks = append(txHooks, h)
}

func CleanTxHooks() {
	hl.Lock()
	defer hl.Unlock()

	txHooks = make([]txHook, 0, 4)
}

type (
	txOption func(tx *Tx)

	txHook interface {
		BeforeCommit(tx *Tx)

		BeforeRollback(tx *Tx)
	}
)

func newTx(opts ...txOption) (driver.Tx, error) {
	tx := new(Tx)

	for i := range opts {
		opts[i](tx)
	}

	if err := tx.init(); err != nil {
		return nil, err
	}

	return tx, nil
}

// withDriverConn
func withDriverConn(conn *Conn) txOption {
	return func(t *Tx) {
		t.conn = conn
	}
}

// withOriginTx
func withOriginTx(tx driver.Tx) txOption {
	return func(t *Tx) {
		t.target = tx
	}
}

// withTxCtx
func withTxCtx(ctx *types.TransactionContext) txOption {
	return func(t *Tx) {
		t.tranCtx = ctx
	}
}

// Tx
type Tx struct {
	conn    *Conn
	tranCtx *types.TransactionContext
	target  driver.Tx
}

// Commit do commit action
func (tx *Tx) Commit() error {
	tx.beforeCommit()
	return tx.commitOnLocal()
}

func (tx *Tx) beforeCommit() {
	if len(txHooks) != 0 {
		hl.RLock()
		defer hl.RUnlock()

		for i := range txHooks {
			txHooks[i].BeforeCommit(tx)
		}
	}
}

func (tx *Tx) Rollback() error {
	if len(txHooks) != 0 {
		hl.RLock()
		defer hl.RUnlock()

		for i := range txHooks {
			txHooks[i].BeforeRollback(tx)
		}
	}

	return tx.target.Rollback()
}

// init
func (tx *Tx) init() error {
	return nil
}

// commitOnLocal
func (tx *Tx) commitOnLocal() error {
	return tx.target.Commit()
}

// register
func (tx *Tx) register(ctx *types.TransactionContext) error {
	if !ctx.HasUndoLog() || !ctx.HasLockKey() {
		return nil
	}
	lockKey := ""
	for k, _ := range ctx.LockKeys {
		lockKey += k + ";"
	}
	request := rm.BranchRegisterParam{
		Xid:        ctx.XaID,
		BranchType: ctx.TransType.GetBranchType(),
		ResourceId: ctx.ResourceID,
		LockKeys:   lockKey,
	}
	dataSourceManager := datasource.GetDataSourceManager(ctx.TransType.GetBranchType())
	branchId, err := dataSourceManager.BranchRegister(context.Background(), request)
	if err != nil {
		log.Infof("Failed to report branch status: %s", err.Error())
		return err
	}
	ctx.BranchID = uint64(branchId)
	return nil
}

// report
func (tx *Tx) report(success bool) error {
	if tx.tranCtx.BranchID == 0 {
		return nil
	}
	status := getStatus(success)
	request := message.BranchReportRequest{
		Xid:        tx.tranCtx.XaID,
		BranchId:   int64(tx.tranCtx.BranchID),
		ResourceId: tx.tranCtx.ResourceID,
		Status:     status,
	}
	dataSourceManager := datasource.GetDataSourceManager(branch.BranchType(tx.tranCtx.TransType))
	retry := REPORT_RETRY_COUNT
	for retry > 0 {
		err := dataSourceManager.BranchReport(context.Background(), request)
		if err != nil {
			retry--
			log.Infof("Failed to report [%s / %s] commit done [%s] Retry Countdown: %s", tx.tranCtx.BranchID, tx.tranCtx.XaID, success, retry)
			if retry == 0 {
				log.Infof("Failed to report branch status: %s", err.Error())
				return err
			}
		} else {
			return nil
		}
	}
	return nil
}

func getStatus(success bool) branch.BranchStatus {
	if success {
		return branch.BranchStatusPhaseoneDone
	} else {
		return branch.BranchStatusPhaseoneFailed
	}
}
