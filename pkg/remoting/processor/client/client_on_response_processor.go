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
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/util/log"
	"seata.apache.org/seata-go/pkg/util/reflectx"

	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/remoting/grpc"
)

func initOnResponse() {
	clientOnResponseProcessor := &clientOnResponseProcessor{}
	switch protocol.Protocol(config.GetTransportConfig().Protocol) {
	case protocol.ProtocolGRPC:
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.MergedResultMessageProto](), message.MessageTypeSeataMergeResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.BranchRegisterResponseProto](), message.MessageTypeBranchRegisterResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.BranchReportResponseProto](), message.MessageTypeBranchStatusReportResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.GlobalLockQueryResponseProto](), message.MessageTypeGlobalLockQueryResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.RegisterRMResponseProto](), message.MessageTypeRegRmResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.GlobalBeginResponseProto](), message.MessageTypeGlobalBeginResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.GlobalCommitResponseProto](), message.MessageTypeGlobalCommitResult)

		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.GlobalReportResponseProto](), message.MessageTypeGlobalReportResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.GlobalRollbackResponseProto](), message.MessageTypeGlobalRollbackResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.GlobalStatusResponseProto](), message.MessageTypeGlobalStatusResult)
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.RegisterTMResponseProto](), message.MessageTypeRegCltResult)

		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeSeataMergeResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeBranchRegisterResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeBranchStatusReportResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalLockQueryResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeRegRmResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalBeginResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalCommitResult, clientOnResponseProcessor)

		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalReportResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalRollbackResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeGlobalStatusResult, clientOnResponseProcessor)
		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeRegCltResult, clientOnResponseProcessor)
	default:
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
}

type clientOnResponseProcessor struct{}

func (f *clientOnResponseProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	log.Infof("the rm client received  clientOnResponse msg %#v from tc server.", rpcMessage)
	switch protocol.Protocol(config.GetTransportConfig().Protocol) {
	case protocol.ProtocolGRPC:
		return f.handleGrpcOnResponse(rpcMessage)
	default:
		return f.handleSeataOnResponse(rpcMessage)
	}
}

func (f *clientOnResponseProcessor) handleGrpcOnResponse(rpcMessage message.RpcMessage) error {
	grpcRemotingClient := grpc.GetGrpcRemotingClient()
	if mergedResult, ok := rpcMessage.Body.(*pb.MergedResultMessageProto); ok {
		mergedMessage := grpcRemotingClient.GetMergedMessage(rpcMessage.ID)
		if mergedMessage != nil {
			for i := 0; i < len(mergedMessage.Msgs); i++ {
				msgID := mergedMessage.MsgIds[i]
				response := grpcRemotingClient.GetMessageFuture(msgID)
				if response != nil {
					response.Response = mergedResult.Msgs[i]
					response.Done <- struct{}{}
					grpcRemotingClient.RemoveMessageFuture(msgID)
				}
			}
			grpcRemotingClient.RemoveMergedMessageFuture(rpcMessage.ID)
		}
		return nil
	} else {
		// 如果是请求消息，做处理逻辑
		msgFuture := grpcRemotingClient.GetMessageFuture(rpcMessage.ID)
		if msgFuture != nil {
			grpcRemotingClient.NotifyRpcMessageResponse(rpcMessage)
			grpcRemotingClient.RemoveMessageFuture(rpcMessage.ID)
		} else {
			if _, ok := rpcMessage.Body.(*pb.AbstractResultMessageProto); ok {
				log.Infof("the rm client received response rpcMessage [{%v}] from tc server.", msgFuture)
			}
		}
	}
	return nil
}

func (f *clientOnResponseProcessor) handleSeataOnResponse(rpcMessage message.RpcMessage) error {
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
		// 如果是请求消息，做处理逻辑
		msgFuture := gettyRemotingClient.GetMessageFuture(rpcMessage.ID)
		if msgFuture != nil {
			gettyRemotingClient.NotifyRpcMessageResponse(rpcMessage)
			gettyRemotingClient.RemoveMessageFuture(rpcMessage.ID)
		} else {
			if _, ok := rpcMessage.Body.(message.AbstractResultMessage); ok {
				log.Infof("the rm client received response msg [{%v}] from tc server.", msgFuture)
			}
		}
	}
	return nil
}
