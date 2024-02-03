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
	"fmt"
	"sync"
	"time"

	"seata.apache.org/seata-go/pkg/datasource/sql/datasource"
	"seata.apache.org/seata-go/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/util/backoff"
	"seata.apache.org/seata-go/pkg/util/log"
)

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
	if ctx.TransactionMode.BranchType() == branch.BranchTypeUnknow {
		return nil
	}

	if ctx.TransactionMode.BranchType() == branch.BranchTypeAT {
		if !ctx.HasUndoLog() || !ctx.HasLockKey() {
			return nil
		}
	}

	request := rm.BranchRegisterParam{
		Xid:        ctx.XID,
		BranchType: ctx.TransactionMode.BranchType(),
		ResourceId: ctx.ResourceID,
	}

	var lockKey string
	if ctx.TransactionMode == types.ATMode {
		if !ctx.HasUndoLog() || !ctx.HasLockKey() {
			return nil
		}
		for k, _ := range ctx.LockKeys {
			lockKey += k + ";"
		}
		request.LockKeys = lockKey
	}

	dataSourceManager := datasource.GetDataSourceManager(ctx.TransactionMode.BranchType())
	branchId, err := dataSourceManager.BranchRegister(context.Background(), request)
	if err != nil {
		log.Errorf("Failed to register branch: %s", err.Error())
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
	request := rm.BranchReportParam{
		Xid:      tx.tranCtx.XID,
		BranchId: int64(tx.tranCtx.BranchID),
		Status:   status,
	}
	dataSourceManager := datasource.GetDataSourceManager(tx.tranCtx.TransactionMode.BranchType())
	if dataSourceManager == nil {
		return fmt.Errorf("get dataSourceManager failed")
	}

	retry := backoff.New(context.Background(), backoff.Config{
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: 200 * time.Millisecond,
		MaxRetries: 5,
	})

	var err error
	for retry.Ongoing() {
		if err = dataSourceManager.BranchReport(context.Background(), request); err == nil {
			break
		}
		log.Infof("Failed to report [%s / %s] commit done [%s] Retry Countdown: %s", tx.tranCtx.BranchID, tx.tranCtx.XID, success, retry)
		retry.Wait()
	}
	return err
}

func getStatus(success bool) branch.BranchStatus {
	if success {
		return branch.BranchStatusPhaseoneDone
	} else {
		return branch.BranchStatusPhaseoneFailed
	}
}
