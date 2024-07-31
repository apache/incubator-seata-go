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

package rm

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	rmRemoting        *RMRemoting
	onceGettyRemoting = &sync.Once{}
)

var ErrBranchReportResponseFault = errors.New("branch report response fault")

func GetRMRemotingInstance() *RMRemoting {
	if rmRemoting == nil {
		onceGettyRemoting.Do(func() {
			rmRemoting = &RMRemoting{}
		})
	}
	return rmRemoting
}

type RMRemoting struct{}

// BranchRegister  Register branch of global transaction
func (r *RMRemoting) BranchRegister(param BranchRegisterParam) (int64, error) {
	request := message.BranchRegisterRequest{
		Xid:             param.Xid,
		LockKey:         param.LockKeys,
		ResourceId:      param.ResourceId,
		BranchType:      param.BranchType,
		ApplicationData: []byte(param.ApplicationData),
	}
	resp, err := getty.GetGettyRemotingClient().SendSyncRequest(request)
	if err != nil || resp == nil {
		log.Errorf("BranchRegister error: %v, res %v", err.Error(), resp)
		return 0, err
	}
	branchResp := resp.(message.BranchRegisterResponse)
	if branchResp.ResultCode == message.ResultCodeFailed {
		return 0, fmt.Errorf("Response %s", branchResp.Msg)
	}
	return branchResp.BranchId, nil
}

// BranchReport Report status of transaction branch
func (r *RMRemoting) BranchReport(param BranchReportParam) error {
	request := message.BranchReportRequest{
		Xid:             param.Xid,
		BranchId:        param.BranchId,
		Status:          param.Status,
		ApplicationData: []byte(param.ApplicationData),
		BranchType:      param.BranchType,
	}

	resp, err := getty.GetGettyRemotingClient().SendSyncRequest(request)
	if err != nil {
		log.Errorf("branch report request error: %+v", err)
		return err
	}

	if err = isReportSuccess(resp); err != nil {
		log.Errorf("BranchReport response error: %v, res %v", err.Error(), resp)
		return err
	}

	return nil
}

// LockQuery Query lock status of transaction branch
func (r *RMRemoting) LockQuery(param LockQueryParam) (bool, error) {
	req := message.GlobalLockQueryRequest{
		BranchRegisterRequest: message.BranchRegisterRequest{
			Xid:        param.Xid,
			LockKey:    param.LockKeys,
			ResourceId: param.ResourceId,
			BranchType: param.BranchType,
		},
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("send lock query request error: {%#v}", err.Error())
		return false, err
	}

	if isQueryLockSuccess(res) {
		log.Infof("lock is lockable, lock %s", param.LockKeys)
		return true, nil
	}
	log.Infof("lock is unlockable, lock %s", param.LockKeys)
	return false, nil
}

func (r *RMRemoting) RegisterResource(resource Resource) error {
	req := message.RegisterRMRequest{
		AbstractIdentifyRequest: message.AbstractIdentifyRequest{
			Version:                 "1.5.2",
			ApplicationId:           rmConfig.ApplicationID,
			TransactionServiceGroup: rmConfig.TxServiceGroup,
		},
		ResourceIds: resource.GetResourceId(),
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("RegisterResourceManager error: {%#v}", err.Error())
		return err
	}

	if isRegisterSuccess(res) {
		r.onRegisterRMSuccess(res.(message.RegisterRMResponse))
	} else {
		r.onRegisterRMFailure(res.(message.RegisterRMResponse))
	}

	return nil
}

func isQueryLockSuccess(response interface{}) bool {
	if res, ok := response.(message.GlobalLockQueryResponse); ok {
		return res.Lockable
	}
	return false
}

func isRegisterSuccess(response interface{}) bool {
	if res, ok := response.(message.RegisterRMResponse); ok {
		return res.Identified
	}
	return false
}

func isReportSuccess(response interface{}) error {
	if res, ok := response.(message.BranchReportResponse); ok {
		if res.ResultCode == message.ResultCodeFailed {
			return fmt.Errorf(res.Msg)
		}
	} else {
		return ErrBranchReportResponseFault
	}

	return nil
}

func (r *RMRemoting) onRegisterRMSuccess(response message.RegisterRMResponse) {
	log.Infof("register RM success. response: %#v", response)
}

func (r *RMRemoting) onRegisterRMFailure(response message.RegisterRMResponse) {
	log.Infof("register RM failure. response: %#v", response)
}

func (r *RMRemoting) onRegisterTMSuccess(response message.RegisterTMResponse) {
	log.Infof("register TM success. response: %#v", response)
}

func (r *RMRemoting) onRegisterTMFailure(response message.RegisterTMResponse) {
	log.Infof("register TM failure. response: %#v", response)
}
