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

package client

import (
	"context"

	"seata.apache.org/seata-go/pkg/protocol"
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/util/log"
	"seata.apache.org/seata-go/pkg/util/reflectx"

	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/remoting/grpc"
	"seata.apache.org/seata-go/pkg/rm"
)

func initBranchRollback() {
	rmBranchRollbackProcessor := &rmBranchRollbackProcessor{}
	switch protocol.Protocol(config.GetTransportConfig().Protocol) {
	case protocol.ProtocolGRPC:
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.BranchRollbackRequestProto](), message.MessageTypeBranchRollback)

		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeBranchRollback, rmBranchRollbackProcessor)
	default:
		getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeBranchRollback, rmBranchRollbackProcessor)
	}
}

type rmBranchRollbackProcessor struct{}

func (f *rmBranchRollbackProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	log.Infof("the rm client received  rmBranchRollback rpcMessage %#v from tc server.", rpcMessage)

	switch protocol.Protocol(config.GetTransportConfig().Protocol) {
	case protocol.ProtocolGRPC:
		return f.handleGrpcBranchRollback(ctx, rpcMessage)
	default:
		return f.handleGettyBranchRollback(ctx, rpcMessage)
	}
}

func (f *rmBranchRollbackProcessor) handleGrpcBranchRollback(ctx context.Context, rpcMessage message.RpcMessage) error {
	request := rpcMessage.Body.(*pb.BranchRollbackRequestProto)
	xid := request.AbstractBranchEndRequest.Xid
	branchID := request.AbstractBranchEndRequest.BranchId
	resourceID := request.AbstractBranchEndRequest.ResourceId
	applicationData := request.AbstractBranchEndRequest.ApplicationData
	log.Infof("Branch rollback request: xid %v, branchID %v, resourceID %v, applicationData %v", xid, branchID, resourceID, applicationData)
	branchType := branch.BranchType(request.AbstractBranchEndRequest.BranchType)
	branchResource := rm.BranchResource{
		BranchType:      branchType,
		Xid:             xid,
		BranchId:        branchID,
		ResourceId:      resourceID,
		ApplicationData: []byte(applicationData),
	}
	status, err := rm.GetRmCacheInstance().GetResourceManager(branchType).BranchRollback(ctx, branchResource)
	if err != nil {
		log.Errorf("branch rollback error: %s", err.Error())
		return err
	}
	log.Infof("branch rollback success: xid %s, branchID %d, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)

	var (
		resultCode pb.ResultCodeProto
		errMsg     string
	)
	if err != nil {
		resultCode = pb.ResultCodeProto_Failed
		errMsg = err.Error()
	} else {
		resultCode = pb.ResultCodeProto_Success
	}
	// reply commit response to tc server
	response := &pb.BranchRollbackResponseProto{
		AbstractBranchEndResponse: &pb.AbstractBranchEndResponseProto{
			AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
				AbstractResultMessage: &pb.AbstractResultMessageProto{
					ResultCode: resultCode,
					Msg:        errMsg,
				},
			},
			Xid:          xid,
			BranchId:     branchID,
			BranchStatus: pb.BranchStatusProto(status),
		},
	}
	err = grpc.GetGrpcRemotingClient().SendAsyncResponse(rpcMessage.ID, response)
	if err != nil {
		log.Errorf("send branch rollback response error: {%#v}", err.Error())
		return err
	}
	log.Infof("send branch rollback response success: xid %s, branchID %d, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)
	return nil
}

func (f *rmBranchRollbackProcessor) handleGettyBranchRollback(ctx context.Context, rpcMessage message.RpcMessage) error {
	request := rpcMessage.Body.(message.BranchRollbackRequest)
	xid := request.Xid
	branchID := request.BranchId
	resourceID := request.ResourceId
	applicationData := request.ApplicationData
	log.Infof("Branch rollback request: xid %v, branchID %v, resourceID %v, applicationData %v", xid, branchID, resourceID, applicationData)
	branchResource := rm.BranchResource{
		BranchType:      request.BranchType,
		Xid:             xid,
		BranchId:        branchID,
		ResourceId:      resourceID,
		ApplicationData: applicationData,
	}
	status, err := rm.GetRmCacheInstance().GetResourceManager(request.BranchType).BranchRollback(ctx, branchResource)
	if err != nil {
		log.Errorf("branch rollback error: %s", err.Error())
		return err
	}
	log.Infof("branch rollback success: xid %s, branchID %d, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)

	var (
		resultCode message.ResultCode
		errMsg     string
	)
	if err != nil {
		resultCode = message.ResultCodeFailed
		errMsg = err.Error()
	} else {
		resultCode = message.ResultCodeSuccess
	}
	// reply commit response to tc server
	response := message.BranchRollbackResponse{
		AbstractBranchEndResponse: message.AbstractBranchEndResponse{
			AbstractTransactionResponse: message.AbstractTransactionResponse{
				AbstractResultMessage: message.AbstractResultMessage{
					ResultCode: resultCode,
					Msg:        errMsg,
				},
			},
			Xid:          xid,
			BranchId:     branchID,
			BranchStatus: status,
		},
	}
	err = getty.GetGettyRemotingClient().SendAsyncResponse(rpcMessage.ID, response)
	if err != nil {
		log.Errorf("send branch rollback response error: {%#v}", err.Error())
		return err
	}
	log.Infof("send branch rollback response success: xid %s, branchID %d, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)
	return nil
}
