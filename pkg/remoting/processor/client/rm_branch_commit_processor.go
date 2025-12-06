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

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"

	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/rm"
)

func initBranchCommit() {
	rmBranchCommitProcessor := &rmBranchCommitProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeBranchCommit, rmBranchCommitProcessor)
}

type rmBranchCommitProcessor struct{}

func (f *rmBranchCommitProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	log.Infof("the rm client received  rmBranchCommit msg %#v from tc server.", rpcMessage)
	request := rpcMessage.Body.(message.BranchCommitRequest)
	xid := request.Xid
	branchID := request.BranchId
	resourceID := request.ResourceId
	applicationData := request.ApplicationData
	log.Infof("Branch committing: xid %s, branchID %d, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)
	branchResource := rm.BranchResource{
		ResourceId:      resourceID,
		BranchId:        branchID,
		ApplicationData: applicationData,
		Xid:             xid,
	}

	status, err := rm.GetRmCacheInstance().GetResourceManager(request.BranchType).BranchCommit(ctx, branchResource)
	if err != nil {
		log.Errorf("branch commit error: %s", err.Error())
		return err
	}
	log.Infof("branch commit success: xid %s, branchID %d, resourceID %s, applicationData %s", xid, branchID, resourceID, applicationData)

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
	// todo add TransactionErrorCode
	response := message.BranchCommitResponse{
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
		log.Errorf("send branch commit response error: {%#v}", err.Error())
		return err
	}
	log.Infof("send branch commit success: xid %v, branchID %v, resourceID %v, applicationData %v", xid, branchID, resourceID, applicationData)
	return nil
}
