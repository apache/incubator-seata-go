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

package getty

import (
	"context"
	"fmt"
	"time"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/backoff"
	"seata.apache.org/seata-go/pkg/util/log"

	"github.com/pkg/errors"
)

type GettyGlobalTransactionManager struct{}

// Begin a global transaction with given timeout and given name.
func (g *GettyGlobalTransactionManager) Begin(ctx context.Context, timeout time.Duration) error {
	req := message.GlobalBeginRequest{
		TransactionName: tm.GetTxName(ctx),
		Timeout:         timeout,
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("GlobalBeginRequest  error %v", err)
		return err
	}
	if res == nil || res.(message.GlobalBeginResponse).ResultCode == message.ResultCodeFailed {
		log.Errorf("GlobalBeginRequest result is empty or result code is failed, res %v", res)
		return fmt.Errorf("GlobalBeginRequest result is empty or result code is failed.")
	}
	log.Infof("GlobalBeginRequest success, res %v", res)

	tm.SetXID(ctx, res.(message.GlobalBeginResponse).Xid)
	return nil
}

// Commit the global transaction.
func (g *GettyGlobalTransactionManager) Commit(ctx context.Context, gtr *tm.GlobalTransaction) error {
	if tm.IsTimeout(ctx) {
		log.Infof("Rollback: tm detected timeout in global gtr %s", gtr.Xid)
		if err := tm.GetGlobalTransactionManager().Rollback(ctx, gtr); err != nil {
			log.Errorf("Rollback transaction failed, error: %v in global gtr % s", err, gtr.Xid)
			return err
		}
		return nil
	}
	if gtr.TxRole != tm.Launcher {
		log.Infof("Ignore Commit(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return fmt.Errorf("Commit xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: tm.GetTmConfig().CommitRetryCount,
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: 200 * time.Millisecond,
	})

	req := message.GlobalCommitRequest{
		AbstractGlobalEndRequest: message.AbstractGlobalEndRequest{Xid: gtr.Xid},
	}
	var res interface{}
	var err error
	for bf.Ongoing() {
		if res, err = getty.GetGettyRemotingClient().SendSyncRequest(req); err == nil {
			break
		}
		log.Warnf("send global commit request failed, xid %s, error %v", gtr.Xid, err)
		bf.Wait()
	}

	if err != nil || bf.Err() != nil {
		lastErr := errors.Wrap(err, bf.Err().Error())
		log.Warnf("send global commit request failed, xid %s, error %v", gtr.Xid, lastErr)
		return lastErr
	}

	log.Infof("send global commit request success, xid %s", gtr.Xid)
	gtr.TxStatus = res.(message.GlobalCommitResponse).GlobalStatus

	return nil
}

// Rollback the global transaction.
func (g *GettyGlobalTransactionManager) Rollback(ctx context.Context, gtr *tm.GlobalTransaction) error {
	if gtr.TxRole != tm.Launcher {
		log.Infof("Ignore Rollback(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return fmt.Errorf("Rollback xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: tm.GetTmConfig().RollbackRetryCount,
		MinBackoff: 100 * time.Millisecond,
		MaxBackoff: 200 * time.Millisecond,
	})

	var res interface{}
	req := message.GlobalRollbackRequest{
		AbstractGlobalEndRequest: message.AbstractGlobalEndRequest{Xid: gtr.Xid},
	}

	var err error
	for bf.Ongoing() {
		if res, err = getty.GetGettyRemotingClient().SendSyncRequest(req); err == nil {
			break
		}
		log.Errorf("GlobalRollbackRequest rollback failed, xid %s, error %v", gtr.Xid, err)
		bf.Wait()
	}

	if err != nil && bf.Err() != nil {
		lastErr := errors.Wrap(err, bf.Err().Error())
		log.Errorf("GlobalRollbackRequest rollback failed, xid %s, error %v", gtr.Xid, lastErr)
		return lastErr
	}

	log.Infof("GlobalRollbackRequest rollback success, xid %s,", gtr.Xid)
	gtr.TxStatus = res.(message.GlobalRollbackResponse).GlobalStatus

	return nil
}
