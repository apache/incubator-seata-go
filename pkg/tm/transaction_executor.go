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

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"
)

type GtxConfig struct {
	Timeout           time.Duration
	Name              string
	Propagation       Propagation
	LockRetryInternal time.Duration
	LockRetryTimes    int16
}

// CallbackWithCtx business callback definition
type CallbackWithCtx func(ctx context.Context) error

// WithGlobalTx begin a global transaction and make it step into committed or rollbacked status.
func WithGlobalTx(ctx context.Context, gc *GtxConfig, business CallbackWithCtx) (re error) {
	if gc == nil {
		return fmt.Errorf("global transaction config info is required.")
	}

	if gc.Name == "" {
		return fmt.Errorf("global transaction name is required.")
	}

	// open global transaction for the first time
	if !IsSeataContext(ctx) {
		ctx = InitSeataContext(ctx)
	}

	if IsGlobalTx(ctx) {
		clearTxConf(ctx)
	}

	if re = begin(ctx, gc); re != nil {
		return
	}

	defer func() {
		var err error
		deferErr := recover()
		// no need to do second phase if propagation is some type e.g. NotSupported.
		if IsGlobalTx(ctx) {
			// business maybe to throw panic, so need to recover it here.
			if err = commitOrRollback(ctx, deferErr == nil && re == nil); err != nil {
				log.Errorf("global transaction xid %s, name %s second phase error", GetXID(ctx), GetTxName(ctx), err)
			}
		}

		if re != nil || err != nil {
			re = fmt.Errorf("first phase error: %v, second phase error: %v", re, err)
		}
	}()

	re = business(ctx)

	return
}

// begin a global transaction, it will obtain a xid and put into ctx from tc by tcp rpc.
// it will to call two function beginNewGtx and useExistGtx
// they do these operations on the transaction：
// beginNewGtx:
// start a new transaction, the previous transaction may be suspended or not exist
// useExistGtx:
// use the previous transaction, but the transaction obtained by propagation,
// but will modify the current transaction role to participant.
// in local transaction mode, the transaction role may be overwritten due to sharing a ctx,
// so it is also necessary to provide suspend and resume operations like xid,
// but here I do not use this method, instead, simulate the call in rpc mode,
// construct a new context object and set the xid.
// the advantage of this is that the suspend and resume operations of xid need not to be considered.
func begin(ctx context.Context, gc *GtxConfig) error {
	switch pg := gc.Propagation; pg {
	case NotSupported:
		// If transaction is existing, suspend it
		// return then to execute without transaction
		if IsGlobalTx(ctx) {
			// because each global transaction operation will use a new context,
			// there is no need to implement a suspend operation, just unbind the xid here.
			// the same is true for the following case that needs to be suspended.
			UnbindXid(ctx)
		}
		return nil
	case Supports:
		// if transaction is not existing, return then to execute without transaction
		// else beginNewGtx transaction then return
		if IsGlobalTx(ctx) {
			useExistGtx(ctx, gc)
		}
		return nil
	case RequiresNew:
		// if transaction is existing, suspend it, and then Begin beginNewGtx transaction.
		if IsGlobalTx(ctx) {
			UnbindXid(ctx)
		}
	case Required:
		// default case, If current transaction is existing, execute with current transaction,
		// else continue and execute with beginNewGtx transaction.
		if IsGlobalTx(ctx) {
			useExistGtx(ctx, gc)
			return nil
		}
	case Never:
		// if transaction is existing, throw exception.
		if IsGlobalTx(ctx) {
			return fmt.Errorf("existing transaction found for transaction marked with pg 'never', xid = %s", GetXID(ctx))
		}
		// return then to execute without transaction.
		return nil
	case Mandatory:
		// if transaction is not existing, throw exception.
		// else execute with current transaction.
		if IsGlobalTx(ctx) {
			useExistGtx(ctx, gc)
			return nil
		}
		return fmt.Errorf("no existing transaction found for transaction marked with pg 'mandatory'")
	default:
		return fmt.Errorf("not supported propagation:%d", pg)
	}

	// the follow will to construct a new transaction with xid.
	return beginNewGtx(ctx, gc)
}

// commitOrRollback commit or rollback the global transaction
func commitOrRollback(ctx context.Context, isSuccess bool) (re error) {
	switch *GetTxRole(ctx) {
	case Launcher:
		if tx := GetTx(ctx); isSuccess {
			if re = GetGlobalTransactionManager().Commit(ctx, tx); re != nil {
				log.Errorf("transactionTemplate: commit transaction failed, error %v", re)
			}
		} else {
			if re = GetGlobalTransactionManager().Rollback(ctx, tx); re != nil {
				log.Errorf("transactionTemplate: Rollback transaction failed, error %v", re)
			}
		}
	case Participant:
		// participant has no responsibility of rollback
		log.Infof("ignore second phase(commit or rollback): just involved in global transaction [%s/%s]", GetTxName(ctx), GetXID(ctx))
	case UnKnow:
		re = errors.New("global transaction role is UnKnow.")
	}

	return
}

// beginNewGtx to construct a default global transaction
func beginNewGtx(ctx context.Context, gc *GtxConfig) error {
	timeout := gc.Timeout
	if timeout == 0 {
		timeout = config.DefaultGlobalTransactionTimeout
	}

	SetTxRole(ctx, Launcher)
	SetTxName(ctx, gc.Name)
	SetTxStatus(ctx, message.GlobalStatusBegin)

	if err := GetGlobalTransactionManager().Begin(ctx, timeout); err != nil {
		return fmt.Errorf("transactionTemplate: Begin transaction failed, error %v", err)
	}
	return nil
}

// useExistGtx if xid is not empty, then construct a global transaction
func useExistGtx(ctx context.Context, gc *GtxConfig) {
	if xid := GetXID(ctx); xid != "" {
		SetTx(ctx, &GlobalTransaction{
			Xid:      GetXID(ctx),
			TxStatus: message.GlobalStatusBegin,
			TxRole:   Participant,
			TxName:   gc.Name,
		})
	}
}

// clearTxConf When using global transactions in local mode, you need to clear tx config to use the propagation of global transactions.
func clearTxConf(ctx context.Context) {
	SetTx(ctx, &GlobalTransaction{Xid: GetXID(ctx)})
}
