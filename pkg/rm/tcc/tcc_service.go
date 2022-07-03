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

package tcc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/tm"

	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/common"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/common/net"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/rm"
)

type TCCServiceProxy struct {
	registerResourceOnce sync.Once
	*TCCResource
}

func NewTCCServiceProxy(service interface{}) (*TCCServiceProxy, error) {
	tccResource, err := ParseTCCResource(service)
	if err != nil {
		log.Errorf("invalid tcc service, err %v", err)
		return nil, err
	}
	tccServiceProxy := &TCCServiceProxy{
		TCCResource: tccResource,
	}
	err = tccServiceProxy.registerResource()
	if err != nil {
		log.Errorf("register tcc resource, err %v", err)
		return nil, err
	}
	return tccServiceProxy, err
}

func (t *TCCServiceProxy) registerResource() error {
	var err error
	t.registerResourceOnce.Do(func() {
		err = rm.GetResourceManagerInstance().GetResourceManager(branch.BranchTypeTCC).RegisterResource(t.TCCResource)
		if err != nil {
			log.Errorf("NewTCCServiceProxy registerResource error: %#v", err.Error())
		}
	})
	return err
}

func (t *TCCServiceProxy) Prepare(ctx context.Context, param interface{}) error {
	if tm.HasXID(ctx) {
		err := t.RegisteBranch(ctx, param)
		if err != nil {
			return err
		}
	}
	return t.TCCResource.Prepare(ctx, param)
}

func (t *TCCServiceProxy) RegisteBranch(ctx context.Context, param interface{}) error {
	// register transaction branch
	if !tm.HasXID(ctx) {
		err := errors.New("BranchRegister error, xid should not be nil")
		log.Errorf(err.Error())
		return err
	}
	tccContext := make(map[string]interface{}, 0)
	tccContext[common.StartTime] = time.Now().UnixNano() / 1e6
	tccContext[common.HostName] = net.GetLocalIp()
	tccContextStr, _ := json.Marshal(map[string]interface{}{
		common.ActionContext: tccContext,
	})

	branchId, err := rm.GetRMRemotingInstance().BranchRegister(branch.BranchTypeTCC, t.GetActionName(), "", tm.GetXID(ctx), string(tccContextStr), "")
	if err != nil {
		err = errors.New(fmt.Sprintf("BranchRegister error: %v", err.Error()))
		log.Errorf(err.Error())
		return err
	}

	actionContext := &tm.BusinessActionContext{
		Xid:           tm.GetXID(ctx),
		BranchId:      branchId,
		ActionName:    t.GetActionName(),
		ActionContext: param,
	}
	tm.SetBusinessActionContext(ctx, actionContext)
	return nil
}

func (t *TCCServiceProxy) GetTransactionInfo() tm.TransactionInfo {
	// todo replace with config
	return tm.TransactionInfo{
		TimeOut: 10000,
		Name:    t.GetActionName(),
		//Propagation, Propagation
		//LockRetryInternal, int64
		//LockRetryTimes    int64
	}
}
