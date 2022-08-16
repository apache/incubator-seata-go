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

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/datasource/sql/datasource"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/datasource/sql/types"
	"github.com/seata/seata-go/pkg/datasource/sql/undo"
)

const REPORT_RETRY_COUNT = 5

type txOption func(tx *Tx)

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
		t.ctx = ctx
	}
}

// Tx
type Tx struct {
	conn   *Conn
	ctx    *types.TransactionContext
	target driver.Tx
}

// Commit do commit action
// case 1. no open global-transaction, just do local transaction commit
// case 2. not need flush undolog, is XA mode, do local transaction commit
// case 3. need run AT transaction
func (tx *Tx) Commit() error {
	if tx.ctx.TransType == types.Local {
		return tx.commitOnLocal()
	}

	// flush undo log if need, is XA mode
	if tx.ctx.TransType == types.XAMode {
		return tx.commitOnXA()
	}

	return tx.commitOnAT()
}

func (tx *Tx) Rollback() error {
	err := tx.target.Rollback()
	if err != nil {
		if tx.ctx.OpenGlobalTrsnaction() && tx.ctx.IsBranchRegistered() {
			tx.report(false)
		}
	}

	return err
}

// init
func (tx *Tx) init() error {
	return nil
}

// commitOnLocal
func (tx *Tx) commitOnLocal() error {
	return tx.target.Commit()
}

// commitOnXA
func (tx *Tx) commitOnXA() error {
	return nil
}

// commitOnAT
func (tx *Tx) commitOnAT() error {
	// if TX-Mode is AT, run regis this transaction branch
	if err := tx.regis(tx.ctx); err != nil {
		return err
	}

	undoLogMgr, err := undo.GetUndoLogManager(tx.ctx.DBType)
	if err != nil {
		return err
	}

	if err := undoLogMgr.FlushUndoLog(tx.ctx, nil); err != nil {
		if rerr := tx.report(false); rerr != nil {
			return errors.WithStack(rerr)
		}
		return errors.WithStack(err)
	}

	if err := tx.commitOnLocal(); err != nil {
		if rerr := tx.report(false); rerr != nil {
			return errors.WithStack(rerr)
		}
		return errors.WithStack(err)
	}

	tx.report(true)
	return nil
}

// regis
func (tx *Tx) regis(ctx *types.TransactionContext) error {
	if !ctx.HasUndoLog() || !ctx.HasLockKey() {
		return nil
	}
	lockKey := ""
	for _, v := range ctx.LockKeys {
		lockKey += v + ";"
	}
	request := message.BranchRegisterRequest{
		Xid:             ctx.XaID,
		BranchType:      branch.BranchType(ctx.TransType),
		ResourceId:      ctx.ResourceID,
		LockKey:         lockKey,
		ApplicationData: nil,
	}
	dataSourceManager := datasource.GetDataSourceManager(branch.BranchType(ctx.TransType))
	branchId, err := dataSourceManager.BranchRegister(context.Background(), "", request)
	if err != nil {
		log.Infof("Failed to report branch status: %s", err.Error())
		return err
	}
	ctx.BranchID = uint64(branchId)
	return nil
}

// report
func (tx *Tx) report(success bool) error {
	if tx.ctx.BranchID == 0 {
		return nil
	}
	status := getStatus(success)
	request := message.BranchReportRequest{
		Xid:        tx.ctx.XaID,
		BranchId:   int64(tx.ctx.BranchID),
		ResourceId: tx.ctx.ResourceID,
		Status:     status,
	}
	dataSourceManager := datasource.GetDataSourceManager(branch.BranchType(tx.ctx.TransType))
	retry := REPORT_RETRY_COUNT
	for retry > 0 {
		err := dataSourceManager.BranchReport(context.Background(), request)
		if err != nil {
			retry--
			log.Infof("Failed to report [%s / %s] commit done [%s] Retry Countdown: %s", tx.ctx.BranchID, tx.ctx.XaID, success, retry)
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
