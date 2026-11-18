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

package grpc

import (
	"fmt"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/util/log"

	"github.com/pkg/errors"
)

var ErrBranchReportResponseFault = errors.New("branch report response fault")

type GrpcRMRemoting struct{}

// BranchRegister  Register branch of global transaction
func (r *GrpcRMRemoting) BranchRegister(param rm.BranchRegisterParam) (int64, error) {
	request := &pb.BranchRegisterRequestProto{
		AbstractTransactionRequest: &pb.AbstractTransactionRequestProto{
			AbstractMessage: &pb.AbstractMessageProto{
				MessageType: pb.MessageTypeProto_TYPE_BRANCH_REGISTER},
		},
		Xid:             param.Xid,
		LockKey:         param.LockKeys,
		ResourceId:      param.ResourceId,
		BranchType:      pb.BranchTypeProto(param.BranchType),
		ApplicationData: param.ApplicationData,
	}
	resp, err := grpc.GetGrpcRemotingClient().SendSyncRequest(request)
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
	branchResp, ok := resp.(*pb.BranchRegisterResponseProto)
	if !ok {
		return 0, fmt.Errorf("BranchRegister received unexpected response type %T", resp)
	}
	if branchResp.AbstractTransactionResponse.AbstractResultMessage.ResultCode == pb.ResultCodeProto_Failed {
		return 0, fmt.Errorf("Response %s", branchResp.AbstractTransactionResponse.AbstractResultMessage.Msg)
	}
	return branchResp.BranchId, nil
}

// BranchReport Report status of transaction branch
func (r *GrpcRMRemoting) BranchReport(param rm.BranchReportParam) error {
	request := &pb.BranchReportRequestProto{
		AbstractTransactionRequest: &pb.AbstractTransactionRequestProto{
			AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_BRANCH_STATUS_REPORT},
		},
		Xid:             param.Xid,
		BranchId:        param.BranchId,
		Status:          pb.BranchStatusProto(param.Status),
		ApplicationData: param.ApplicationData,
		BranchType:      pb.BranchTypeProto(param.BranchType),
	}

	resp, err := grpc.GetGrpcRemotingClient().SendSyncRequest(request)
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
func (r *GrpcRMRemoting) LockQuery(param rm.LockQueryParam) (bool, error) {
	req := &pb.GlobalLockQueryRequestProto{
		BranchRegisterRequest: &pb.BranchRegisterRequestProto{
			AbstractTransactionRequest: &pb.AbstractTransactionRequestProto{
				AbstractMessage: &pb.AbstractMessageProto{
					MessageType: pb.MessageTypeProto_TYPE_GLOBAL_LOCK_QUERY},
			},
			Xid:        param.Xid,
			LockKey:    param.LockKeys,
			ResourceId: param.ResourceId,
			BranchType: pb.BranchTypeProto(param.BranchType),
		},
	}
	res, err := grpc.GetGrpcRemotingClient().SendSyncRequest(req)
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

func (r *GrpcRMRemoting) RegisterResource(resource rm.Resource) error {
	req := &pb.RegisterRMRequestProto{
		AbstractIdentifyRequest: &pb.AbstractIdentifyRequestProto{
			AbstractMessage:         &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_REG_RM},
			Version:                 "1.5.2",
			ApplicationId:           rm.GetRmConfig().ApplicationID,
			TransactionServiceGroup: rm.GetRmConfig().TxServiceGroup,
		},
		ResourceIds: resource.GetResourceId(),
	}
	res, err := grpc.GetGrpcRemotingClient().SendSyncRequest(req)
	if err != nil {
		log.Errorf("RegisterResourceManager error: {%#v}", err)
		return err
	}
	registerResp, err := decodeRegisterRMResponse(res)
	if err != nil {
		return err
	}
	if !registerResp.AbstractIdentifyResponse.Identified {
		r.onRegisterRMFailure(registerResp)
		return fmt.Errorf("register RM failed: %s", registerResp.AbstractIdentifyResponse.AbstractResultMessage.Msg)
	}
	r.onRegisterRMSuccess(registerResp)
	return nil
}

// decodeRegisterRMResponse validates a RegisterRMResponse: nil and wrongly
// typed responses are rejected with an error instead of panicking on a bare
// type assertion. A TC-side rejection (Identified=false, e.g. auth SPI or
// version check) must reach the caller: swallowing it made data sources and
// TCC proxies report a healthy startup while every later branch operation
// failed (aligned with the Java client, which throws FrameworkException).
func decodeRegisterRMResponse(res interface{}) (*pb.RegisterRMResponseProto, error) {
	if res == nil {
		return nil, errors.New("RegisterResource received nil response")
	}
	registerResp, ok := res.(*pb.RegisterRMResponseProto)
	if !ok {
		return nil, fmt.Errorf("RegisterResource received unexpected response type %T", res)
	}
	return registerResp, nil
}

func isQueryLockSuccess(response interface{}) bool {
	if res, ok := response.(*pb.GlobalLockQueryResponseProto); ok {
		return res.Lockable
	}
	return false
}

func isRegisterSuccess(response interface{}) bool {
	if res, ok := response.(*pb.RegisterRMResponseProto); ok {
		return res.AbstractIdentifyResponse.Identified
	}
	return false
}

func isReportSuccess(response interface{}) error {
	if res, ok := response.(*pb.BranchReportResponseProto); ok {
		if res.AbstractTransactionResponse.AbstractResultMessage.ResultCode == pb.ResultCodeProto(message.ResultCodeFailed) {
			return errors.New(res.AbstractTransactionResponse.AbstractResultMessage.Msg)
		}
	} else {
		return ErrBranchReportResponseFault
	}

	return nil
}

func (r *GrpcRMRemoting) onRegisterRMSuccess(response *pb.RegisterRMResponseProto) {
	log.Infof("register RM success. response: %#v", response)
}

func (r *GrpcRMRemoting) onRegisterRMFailure(response *pb.RegisterRMResponseProto) {
	log.Errorf("register RM failure. response: %#v", response)
}

func (r *GrpcRMRemoting) onRegisterTMSuccess(response *pb.RegisterTMResponseProto) {
	log.Infof("register TM success. response: %#v", response)
}

func (r *GrpcRMRemoting) onRegisterTMFailure(response *pb.RegisterTMResponseProto) {
	log.Infof("register TM failure. response: %#v", response)
}
