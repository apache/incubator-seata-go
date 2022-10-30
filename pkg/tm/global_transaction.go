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

	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/getty"
	"github.com/seata/seata-go/pkg/util/backoff"
	"github.com/seata/seata-go/pkg/util/log"
)

type GlobalTransaction struct {
	Xid    string
	Status message.GlobalStatus
	Role   GlobalTransactionRole
}

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

// Begin a new global transaction with given timeout and given name.
func (g *GlobalTransactionManager) Begin(ctx context.Context, gtr *GlobalTransaction, timeout time.Duration, name string) error {
	if gtr.Role != LAUNCHER {
		log.Infof("Ignore GlobalStatusBegin(): just involved in global transaction %s", gtr.Xid)
		return nil
	}
	if gtr.Xid != "" {
		return fmt.Errorf("Global transaction already "+
			"exists,can't begin a new global transaction, currentXid = %s ", gtr.Xid)
	}

	req := message.GlobalBeginRequest{
		TransactionName: name,
		Timeout:         timeout,
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("GlobalBeginRequest  error %v", err)
		return err
	}
	if res == nil || res.(message.GlobalBeginResponse).ResultCode == message.ResultCodeFailed {
		log.Errorf("GlobalBeginRequest result is empty or result code is failed, res %v", res)
		return errors.New("GlobalBeginRequest result is empty or result code is failed.")
	}
	log.Infof("GlobalBeginRequest success, xid %s, res %v", gtr.Xid, res)

	gtr.Status = message.GlobalStatusBegin
	gtr.Xid = res.(message.GlobalBeginResponse).Xid
	SetXID(ctx, res.(message.GlobalBeginResponse).Xid)
	return nil
}

// Commit the global transaction.
func (g *GlobalTransactionManager) Commit(ctx context.Context, gtr *GlobalTransaction) error {
	if gtr.Role != LAUNCHER {
		log.Infof("Ignore Commit(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return errors.New("Commit xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: 10,
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

	if bf.Err() != nil {
		lastErr := errors.Wrap(err, bf.Err().Error())
		log.Warnf("send global commit request failed, xid %s, error %v", gtr.Xid, lastErr)
		return lastErr
	}

	log.Infof("send global commit request success, xid %s", gtr.Xid)
	gtr.Status = res.(message.GlobalCommitResponse).GlobalStatus
	UnbindXid(ctx)

	return nil
}

// Rollback the global transaction.
func (g *GlobalTransactionManager) Rollback(ctx context.Context, gtr *GlobalTransaction) error {
	if gtr.Role != LAUNCHER {
		log.Infof("Ignore Rollback(): just involved in global gtr %s", gtr.Xid)
		return nil
	}
	if gtr.Xid == "" {
		return errors.New("Rollback xid should not be empty")
	}

	bf := backoff.New(ctx, backoff.Config{
		MaxRetries: 10,
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

	if bf.Err() != nil {
		lastErr := errors.Wrap(err, bf.Err().Error())
		log.Errorf("GlobalRollbackRequest rollback failed, xid %s, error %v", gtr.Xid, lastErr)
		return lastErr
	}

	log.Infof("GlobalRollbackRequest rollback success, xid %s,", gtr.Xid)
	gtr.Status = res.(message.GlobalRollbackResponse).GlobalStatus
	UnbindXid(ctx)

	return nil
}

// Suspend the global transaction.
func (g *GlobalTransactionManager) Suspend() (SuspendedResourcesHolder, error) {
	panic("implement me")
}

// Resume the global transaction.
func (g *GlobalTransactionManager) Resume(suspendedResourcesHolder SuspendedResourcesHolder) error {
	panic("implement me")
}

// GlobalReport report the global transaction status.
func (g *GlobalTransactionManager) GlobalReport(globalStatus message.GlobalStatus) error {
	panic("implement me")
}
