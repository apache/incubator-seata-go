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

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"

	"github.com/seata/seata-go/pkg/remoting/getty"
)

func init() {
	clientOnResponseProcessor := &clientOnResponseProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_SeataMergeResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchRegisterResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchStatusReportResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalLockQueryResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_RegRmResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalBeginResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalCommitResult, clientOnResponseProcessor)

	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalReportResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalRollbackResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalStatusResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_RegCltResult, clientOnResponseProcessor)
}

type clientOnResponseProcessor struct {
}

func (f *clientOnResponseProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	log.Infof("the rm client received  clientOnResponse msg %#v from tc server.", rpcMessage)
	if mergedResult, ok := rpcMessage.Body.(message.MergeResultMessage); ok {
		mergedMessage := getty.GetGettyRemotingInstance().GetMergedMessage(rpcMessage.ID)
		if mergedMessage != nil {
			for i := 0; i < len(mergedMessage.Msgs); i++ {
				msgID := mergedMessage.MsgIds[i]
				response := getty.GetGettyRemotingInstance().GetMessageFuture(msgID)
				if response != nil {
					response.Response = mergedResult.Msgs[i]
					response.Done <- true
					getty.GetGettyRemotingInstance().RemoveMessageFuture(msgID)
				}
			}
			getty.GetGettyRemotingInstance().RemoveMergedMessageFuture(rpcMessage.ID)
		}
		return nil
	} else {
		// 如果是请求消息，做处理逻辑
		msgFuture := getty.GetGettyRemotingInstance().GetMessageFuture(rpcMessage.ID)
		if msgFuture != nil {
			getty.GetGettyRemotingInstance().NotifyRpcMessageResponse(rpcMessage)
			getty.GetGettyRemotingInstance().RemoveMessageFuture(rpcMessage.ID)
		} else {
			if _, ok := rpcMessage.Body.(message.AbstractResultMessage); ok {
				log.Infof("the rm client received response msg [{}] from tc server.", msgFuture)
			}
		}
	}
	return nil
}
