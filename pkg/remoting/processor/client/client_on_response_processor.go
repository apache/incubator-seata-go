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
)

func initOnResponse() {
	clientOnResponseProcessor := &clientOnResponseProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeSeataMergeResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeBranchRegisterResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeBranchStatusReportResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalLockQueryResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeRegRmResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalBeginResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalCommitResult, clientOnResponseProcessor)

	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalReportResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalRollbackResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalStatusResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeRegCltResult, clientOnResponseProcessor)
}

type clientOnResponseProcessor struct{}

func (f *clientOnResponseProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	log.Infof("the rm client received  clientOnResponse msg %#v from tc server.", rpcMessage)
	gettyRemotingClient := getty.GetGettyRemotingClient()
	if mergedResult, ok := rpcMessage.Body.(message.MergeResultMessage); ok {
		mergedMessage := gettyRemotingClient.GetMergedMessage(rpcMessage.ID)
		if mergedMessage != nil {
			for i := 0; i < len(mergedMessage.Msgs); i++ {
				msgID := mergedMessage.MsgIds[i]
				response := gettyRemotingClient.GetMessageFuture(msgID)
				if response != nil {
					response.Response = mergedResult.Msgs[i]
					response.Done <- struct{}{}
					gettyRemotingClient.RemoveMessageFuture(msgID)
				}
			}
			gettyRemotingClient.RemoveMergedMessageFuture(rpcMessage.ID)
		}
		return nil
	} else {
		msgFuture := gettyRemotingClient.GetMessageFuture(rpcMessage.ID)
		if msgFuture != nil {
			gettyRemotingClient.NotifyRpcMessageResponse(rpcMessage)
			gettyRemotingClient.RemoveMessageFuture(rpcMessage.ID)
		} else {
			if _, ok := rpcMessage.Body.(message.AbstractResultMessage); ok {
				log.Infof("the rm client received response msg [{}] from tc server.", msgFuture)
			}
		}
	}
	return nil
}
