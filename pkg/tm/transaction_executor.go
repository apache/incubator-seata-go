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

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
)

type TransactionInfo struct {
	TimeOut           int32
	Name              string
	Propagation       Propagation
	LockRetryInternal int64
	LockRetryTimes    int64
}

type TransactionalExecutor interface {
	Execute(ctx context.Context, param interface{}) (interface{}, error)
	GetTransactionInfo() TransactionInfo
}

func Begin(ctx context.Context, name string) context.Context {
	if !IsSeataContext(ctx) {
		ctx = InitSeataContext(ctx)
	}

	SetTxName(ctx, name)
	if GetTransactionRole(ctx) == nil {
		SetTransactionRole(ctx, LAUNCHER)
	}

	var tx *GlobalTransaction
	if HasXID(ctx) {
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
			Xid:    GetXID(ctx),
			Status: message.GlobalStatusUnKnown,
			Role:   LAUNCHER,
		}
		SetTxStatus(ctx, message.GlobalStatusUnKnown)
	}

	// todo timeout should read from config
	err := GetGlobalTransactionManager().Begin(ctx, tx, 50, name)
	if err != nil {
		panic(fmt.Sprintf("transactionTemplate: begin transaction failed, error %v", err))
	}

	return ctx
}

// commit global transaction
func CommitOrRollback(ctx context.Context, err *error) error {
	tx := &GlobalTransaction{
		Xid:    GetXID(ctx),
		Status: *GetTxStatus(ctx),
		Role:   *GetTransactionRole(ctx),
	}

	var resp error
	if *err == nil {
		resp = GetGlobalTransactionManager().Commit(ctx, tx)
		if resp != nil {
			log.Infof("transactionTemplate: commit transaction failed, error %v", err)
		}
	} else {
		resp = GetGlobalTransactionManager().Rollback(ctx, tx)
		if resp != nil {
			log.Infof("transactionTemplate: Rollback transaction failed, error %v", err)
		}
	}
	return resp
}
