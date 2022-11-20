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
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/util/log"
)

const DefaultTimeOut = time.Second * 30

type TransactionInfo struct {
	TimeOut           time.Duration
	Name              string
	Propagation       Propagation
	LockRetryInternal int64
	LockRetryTimes    int64
}

// CallbackWithCtx business callback definition
type CallbackWithCtx func(ctx context.Context) error

// WithGlobalTx begin a global transaction and make it step into committed or rollbacked status.
func WithGlobalTx(ctx context.Context, ti *TransactionInfo, business CallbackWithCtx) (re error) {
	if ti == nil {
		return errors.New("global transaction config info is required.")
	}
	if ti.Name == "" {
		return errors.New("global transaction name is required.")
	}

	if ctx, re = begin(ctx, ti); re != nil {
		return
	}
	defer func() {
		// business maybe to throw panic, so need to recover it here.
		err := recover()
		if err != nil {
			log.Errorf("business callback panic:%v", err)
		}
		re = commitOrRollback(ctx, err == nil && re == nil)
		log.Infof("global transaction result %v", re)
	}()

	re = business(ctx)

	return
}

// begin a global transaction, it will obtain a xid from tc in tcp call.
func begin(ctx context.Context, ti *TransactionInfo) (rc context.Context, re error) {
	if ti == nil {
		return nil, errors.New("transaction info is nil")
	}
	if ti.TimeOut == 0 {
		ti.TimeOut = DefaultTimeOut
	}
	if !IsSeataContext(ctx) {
		ctx = InitSeataContext(ctx)
	}

	SetTxName(ctx, ti.Name)
	if GetTransactionRole(ctx) == nil {
		SetTransactionRole(ctx, LAUNCHER)
	}

	var tx *GlobalTransaction
	if IsGlobalTx(ctx) {
		tx = &GlobalTransaction{
			Xid:    GetXID(ctx),
			Status: message.GlobalStatusBegin,
			Role:   PARTICIPANT,
		}
		SetTxStatus(ctx, message.GlobalStatusBegin)
	}

	// todo: Handle the transaction propagation.

	if tx == nil {
		tx = &GlobalTransaction{
			Status: message.GlobalStatusUnKnown,
			Role:   LAUNCHER,
		}
		SetTxStatus(ctx, message.GlobalStatusUnKnown)
	}

	err := GetGlobalTransactionManager().Begin(ctx, tx, ti.TimeOut, ti.Name)
	if err != nil {
		re = fmt.Errorf("transactionTemplate: begin transaction failed, error %v", err)
	}

	return ctx, re
}

// commitOrRollback commit or rollback the global transaction
func commitOrRollback(ctx context.Context, isSuccess bool) (re error) {
	role := *GetTransactionRole(ctx)
	if role == PARTICIPANT {
		// Participant has no responsibility of rollback
		log.Debugf("Ignore Rollback(): just involved in global transaction [%s]", GetXID(ctx))
		return
	}

	tx := &GlobalTransaction{
		Xid:    GetXID(ctx),
		Status: *GetTxStatus(ctx),
		Role:   role,
	}

	if isSuccess {
		if re = GetGlobalTransactionManager().Commit(ctx, tx); re != nil {
			log.Errorf("transactionTemplate: commit transaction failed, error %v", re)
		}
	} else {
		if re = GetGlobalTransactionManager().Rollback(ctx, tx); re != nil {
			log.Errorf("transactionTemplate: Rollback transaction failed, error %v", re)
		}
	}

	// todo unbind xid
	return
}
