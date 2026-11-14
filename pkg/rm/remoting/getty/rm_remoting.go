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
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/util/log"

	"github.com/pkg/errors"
)

var ErrBranchReportResponseFault = errors.New("branch report response fault")

type GettyRMRemoting struct{}

// BranchRegister  Register branch of global transaction
func (r *GettyRMRemoting) BranchRegister(param rm.BranchRegisterParam) (int64, error) {
	request := message.BranchRegisterRequest{
		Xid:             param.Xid,
		LockKey:         param.LockKeys,
		ResourceId:      param.ResourceId,
		BranchType:      param.BranchType,
		ApplicationData: []byte(param.ApplicationData),
	}
	resp, err := getty.GetGettyRemotingClient().SendSyncRequest(request)
	if err != nil {
		log.Errorf("BranchRegister error: %v", err)
		return 0, err
	}
	return decodeBranchRegisterResponse(resp)
}

// decodeBranchRegisterResponse validates a BranchRegisterResponse: nil and
// wrongly typed responses are rejected with an error instead of panicking on
// a bare type assertion (the historic code also logged err.Error() with a
// possibly nil err here).
func decodeBranchRegisterResponse(resp interface{}) (int64, error) {
	if resp == nil {
		return 0, errors.New("BranchRegister received nil response")
	}
	branchResp, ok := resp.(message.BranchRegisterResponse)
	if !ok {
		return 0, fmt.Errorf("BranchRegister received unexpected response type %T", resp)
	}
	if branchResp.ResultCode == message.ResultCodeFailed {
		return 0, fmt.Errorf("Response %s", branchResp.Msg)
	}
	return branchResp.BranchId, nil
}

// BranchReport Report status of transaction branch
func (r *GettyRMRemoting) BranchReport(param rm.BranchReportParam) error {
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
func (r *GettyRMRemoting) LockQuery(param rm.LockQueryParam) (bool, error) {
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

func (r *GettyRMRemoting) RegisterResource(resource rm.Resource) error {
	req := message.RegisterRMRequest{
		AbstractIdentifyRequest: message.AbstractIdentifyRequest{
			Version:                 "1.5.2",
			ApplicationId:           rm.GetRmConfig().ApplicationID,
			TransactionServiceGroup: rm.GetRmConfig().TxServiceGroup,
		},
		ResourceIds: resource.GetResourceId(),
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("RegisterResourceManager error: {%#v}", err)
		return err
	}
	return handleRegisterRMResponse(res, r.onRegisterRMSuccess, r.onRegisterRMFailure)
}

// handleRegisterRMResponse validates a RegisterRMResponse and reports a
// TC-side rejection (Identified=false, e.g. auth SPI or version check) as an
// error: swallowing it made data sources and TCC proxies report a healthy
// startup while every later branch operation failed (aligned with the Java
// client, which throws FrameworkException here).
func handleRegisterRMResponse(res interface{}, onSuccess func(message.RegisterRMResponse), onFailure func(message.RegisterRMResponse)) error {
	if res == nil {
		return errors.New("RegisterResource received nil response")
	}
	registerResp, ok := res.(message.RegisterRMResponse)
	if !ok {
		return fmt.Errorf("RegisterResource received unexpected response type %T", res)
	}
	if !registerResp.Identified {
		onFailure(registerResp)
		return fmt.Errorf("register RM failed: %s", registerResp.Msg)
	}
	onSuccess(registerResp)
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
			return errors.New(res.Msg)
		}
	} else {
		return ErrBranchReportResponseFault
	}

	return nil
}

func (r *GettyRMRemoting) onRegisterRMSuccess(response message.RegisterRMResponse) {
	log.Infof("register RM success. response: %#v", response)
}

func (r *GettyRMRemoting) onRegisterRMFailure(response message.RegisterRMResponse) {
	log.Errorf("register RM failure. response: %#v", response)
}

func (r *GettyRMRemoting) onRegisterTMSuccess(response message.RegisterTMResponse) {
	log.Infof("register TM success. response: %#v", response)
}

func (r *GettyRMRemoting) onRegisterTMFailure(response message.RegisterTMResponse) {
	log.Infof("register TM failure. response: %#v", response)
}
