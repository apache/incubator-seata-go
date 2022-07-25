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
	if IsTransactionOpened(ctx) {
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
	err := GetGlobalTransactionManager().Begin(ctx, tx, 60000*30, name)
	if err != nil {
		panic(fmt.Sprintf("transactionTemplate: begin transaction failed, error %v", err))
	}

	return ctx
}

// CommitOrRollback commit global transaction
func CommitOrRollback(ctx context.Context, isSuccess bool) error {
	role := *GetTransactionRole(ctx)
	if role == PARTICIPANT {
		// Participant has no responsibility of rollback
		log.Debugf("Ignore Rollback(): just involved in global transaction [%s]", GetXID(ctx))
		return nil
	}
	tx := &GlobalTransaction{
		Xid:    GetXID(ctx),
		Status: *GetTxStatus(ctx),
		Role:   role,
	}
	var (
		err error
		// todo retry and retryInterval should read from config
		retry         = 10
		retryInterval = 200 * time.Millisecond
	)
	for ; retry > 0; retry-- {
		if isSuccess {
			err = GetGlobalTransactionManager().Commit(ctx, tx)
			if err != nil {
				log.Infof("transactionTemplate: commit transaction failed, error %v", err)
			}
		} else {
			err = GetGlobalTransactionManager().Rollback(ctx, tx)
			if err != nil {
				log.Infof("transactionTemplate: Rollback transaction failed, error %v", err)
			}
		}
		if err == nil {
			break
		} else {
			time.Sleep(retryInterval)
		}
	}
	// todo unbind xid
	return err
}
