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
	"sync"
	"time"

	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/util/backoff"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	// globalTransactionManager singleton ResourceManagerFacade
	globalTransactionManager     *GlobalTransactionManager
	onceGlobalTransactionManager = &sync.Once{}
)

func GetGlobalTransactionManager() *GlobalTransactionManager {
	if globalTransactionManager == nil {
		onceGlobalTransactionManager.Do(func() {
			globalTransactionManager = &GlobalTransactionManager{}
		})
	}
	return globalTransactionManager
}

type GlobalTransactionManager struct{}

// Begin a global transaction with given timeout and given name.
func (g *GlobalTransactionManager) Begin(ctx context.Context, timeout time.Duration) error {
	req := message.GlobalBeginRequest{
		TransactionName: GetTxName(ctx),
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

	SetXID(ctx, res.(message.GlobalBeginResponse).Xid)
	return nil
}

// Commit the global transaction.
func (g *GlobalTransactionManager) Commit(ctx context.Context, gtr *GlobalTransaction) error {
	if gtr.TxRole != Launcher {
		log.Infof("Ignore Commit(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return fmt.Errorf("Commit xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: config.CommitRetryCount,
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
func (g *GlobalTransactionManager) Rollback(ctx context.Context, gtr *GlobalTransaction) error {
	if gtr.TxRole != Launcher {
		log.Infof("Ignore Rollback(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return fmt.Errorf("Rollback xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: config.RollbackRetryCount,
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
