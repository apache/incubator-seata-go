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
	"sync"

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/getty"
)

var (
	rmRemoting        *RMRemoting
	onceGettyRemoting = &sync.Once{}
)

func GetRMRemotingInstance() *RMRemoting {
	if rmRemoting == nil {
		onceGettyRemoting.Do(func() {
			rmRemoting = &RMRemoting{}
		})
	}
	return rmRemoting
}

type RMRemoting struct {
}

// Branch register long
func (RMRemoting) BranchRegister(branchType branch.BranchType, resourceId, clientId, xid, applicationData, lockKeys string) (int64, error) {
	request := message.BranchRegisterRequest{
		Xid:             xid,
		LockKey:         lockKeys,
		ResourceId:      resourceId,
		BranchType:      branchType,
		ApplicationData: []byte(applicationData),
	}
	resp, err := getty.GetGettyRemotingClient().SendSyncRequest(request)
	if err != nil || resp == nil {
		log.Errorf("BranchRegister error: %v, res %v", err.Error(), resp)
		return 0, err
	}
	return resp.(message.BranchRegisterResponse).BranchId, nil
}

//  Branch report
func (RMRemoting) BranchReport(branchType branch.BranchType, xid string, branchId int64, status branch.BranchStatus, applicationData string) error {
	request := message.BranchReportRequest{
		Xid:             xid,
		BranchId:        branchId,
		Status:          status,
		ApplicationData: []byte(applicationData),
		BranchType:      branch.BranchTypeAT,
	}
	resp, err := getty.GetGettyRemotingClient().SendSyncRequest(request)
	if err != nil || resp == nil || isReportSuccess(resp) == message.ResultCodeFailed {
		log.Errorf("BranchReport error: %v, res %v", err.Error(), resp)
		return err
	}
	return nil
}

// Lock query boolean
func (RMRemoting) LockQuery(branchType branch.BranchType, resourceId, xid, lockKeys string) (bool, error) {
	return false, nil
}

func (r *RMRemoting) RegisterResource(resource Resource) error {
	req := message.RegisterRMRequest{
		AbstractIdentifyRequest: message.AbstractIdentifyRequest{
			//todo replace with config
			Version:                 "1.5.2",
			ApplicationId:           "tcc-sample",
			TransactionServiceGroup: "my_test_tx_group",
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

func isRegisterSuccess(response interface{}) bool {
	if res, ok := response.(message.RegisterRMResponse); ok {
		return res.Identified
	}
	return false
}

func isReportSuccess(response interface{}) message.ResultCode {
	if res, ok := response.(message.BranchReportResponse); ok {
		return res.ResultCode
	}
	return message.ResultCodeFailed
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
