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

	getty2 "github.com/seata/seata-go/pkg/remoting/getty"
)

func init() {
	clientOnResponseProcessor := &clientOnResponseProcessor{}
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_SeataMergeResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchRegisterResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchStatusReportResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalLockQueryResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_RegRmResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalBeginResult, clientOnResponseProcessor)
	getty2.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalCommitResult, clientOnResponseProcessor)
}

type clientOnResponseProcessor struct {
}

func (f *clientOnResponseProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	// 如果是合并的结果消息，直接通知已经处理完成
	if mergedResult, ok := rpcMessage.Body.(message.MergeResultMessage); ok {

		mergedMessage := getty2.GetGettyRemotingInstance().GetMergedMessage(rpcMessage.ID)
		if mergedMessage != nil {
			for i := 0; i < len(mergedMessage.Msgs); i++ {
				msgID := mergedMessage.MsgIds[i]
				response := getty2.GetGettyRemotingInstance().GetMessageFuture(msgID)
				if response != nil {
					response.Response = mergedResult.Msgs[i]
					response.Done <- true
					getty2.GetGettyRemotingInstance().RemoveMessageFuture(msgID)
				}
			}
			getty2.GetGettyRemotingInstance().RemoveMergedMessageFuture(rpcMessage.ID)
		}
		return nil
	} else {
		// 如果是请求消息，做处理逻辑
		msgFuture := getty2.GetGettyRemotingInstance().GetMessageFuture(rpcMessage.ID)
		if msgFuture != nil {
			getty2.GetGettyRemotingInstance().NotifyRpcMessageResponse(rpcMessage)
			getty2.GetGettyRemotingInstance().RemoveMessageFuture(rpcMessage.ID)
		} else {
			if _, ok := rpcMessage.Body.(message.AbstractResultMessage); ok {
				log.Infof("the rm client received response msg [{}] from tc server.", msgFuture)
			}
		}
	}
	return nil
}
