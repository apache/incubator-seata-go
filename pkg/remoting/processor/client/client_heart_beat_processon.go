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

	"seata.apache.org/seata-go/pkg/util/log"
	"seata.apache.org/seata-go/pkg/util/reflectx"

	"seata.apache.org/seata-go/pkg/protocol"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/remoting/grpc"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
)

func initHeartBeat() {
	clientHeartBeatProcessor := &clientHeartBeatProcessor{}
	switch protocol.Protocol(config.GetTransportConfig().Protocol) {
	case protocol.ProtocolGRPC:
		grpc.GetGrpcClientHandlerInstance().RegisterType(reflectx.ProtoMessageName[*pb.HeartbeatMessageProto](), message.MessageTypeHeartbeatMsg)

		grpc.GetGrpcClientHandlerInstance().RegisterProcessor(message.MessageTypeHeartbeatMsg, clientHeartBeatProcessor)
	default:
		getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeHeartbeatMsg, clientHeartBeatProcessor)
	}
}

type clientHeartBeatProcessor struct{}

func (f *clientHeartBeatProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	switch protocol.Protocol(config.GetTransportConfig().Protocol) {
	case protocol.ProtocolGRPC:
		f.handleGrpcHeartbeat(ctx, rpcMessage)
	default:
		f.handleSeataHeartbeat(ctx, rpcMessage)
	}
	return nil
}

func (f *clientHeartBeatProcessor) handleSeataHeartbeat(ctx context.Context, rpcMessage message.RpcMessage) {
	if body, ok := rpcMessage.Body.(message.HeartBeatMessage); ok {
		if !body.Ping {
			log.Debug("received PONG from {}", ctx)
		}
	}
	msgFuture := getty.GetGettyRemotingClient().GetMessageFuture(rpcMessage.ID)
	if msgFuture != nil {
		getty.GetGettyRemotingClient().RemoveMessageFuture(rpcMessage.ID)
	}
}

func (f *clientHeartBeatProcessor) handleGrpcHeartbeat(ctx context.Context, rpcMessage message.RpcMessage) {
	hbMsg := rpcMessage.Body.(*pb.HeartbeatMessageProto)
	if !hbMsg.Ping {
		log.Debug("received PONG from {}", ctx)
	}
	msgFuture := grpc.GetGrpcRemotingClient().GetMessageFuture(rpcMessage.ID)
	if msgFuture != nil {
		grpc.GetGrpcRemotingClient().RemoveMessageFuture(rpcMessage.ID)
	}
}
