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
	"sync"

	gxtime "github.com/dubbogo/gost/time"
	"go.uber.org/atomic"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	grpcRemotingClient     *GrpcRemotingClient
	onceGrpcRemotingClient = &sync.Once{}
)

type GrpcRemotingClient struct {
	idGenerator  *atomic.Uint32
	grpcRemoting *GrpcRemoting
}

func GetGrpcRemotingClient() *GrpcRemotingClient {
	if grpcRemotingClient == nil {
		onceGrpcRemotingClient.Do(func() {
			grpcRemotingClient = &GrpcRemotingClient{
				idGenerator:  &atomic.Uint32{},
				grpcRemoting: newGrpcRemoting(),
			}
		})
	}
	return grpcRemotingClient
}

func (client *GrpcRemotingClient) SendAsyncRequest(msg interface{}) error {
	var msgType message.RequestType
	if _, ok := msg.(message.HeartBeatMessage); ok {
		msgType = message.RequestTypeHeartbeatRequest
	} else {
		msgType = message.RequestTypeRequestOneway
	}
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       msgType,
		Codec:      byte(codec.CodecTypeGRPC),
		Compressor: 0,
		HeadMap:    make(map[string]string),
		Body:       msg,
	}
	return client.grpcRemoting.SendAsync(rpcMessage, nil, client.asyncCallback)
}

func (client *GrpcRemotingClient) SendAsyncResponse(msgID int32, msg interface{}) error {
	rpcMessage := message.RpcMessage{
		ID:         msgID,
		Type:       message.RequestTypeResponse,
		Codec:      byte(codec.CodecTypeGRPC),
		Compressor: 0,
		HeadMap:    make(map[string]string),
		Body:       msg,
	}
	return client.grpcRemoting.SendAsync(rpcMessage, nil, nil)
}

func (client *GrpcRemotingClient) SendSyncRequest(msg interface{}) (interface{}, error) {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeGRPC),
		Compressor: 0,
		HeadMap:    make(map[string]string),
		Body:       msg,
	}
	return client.grpcRemoting.SendSync(rpcMessage, nil, client.syncCallback)
}

func (client *GrpcRemotingClient) asyncCallback(reqMsg message.RpcMessage, respMsg *message.MessageFuture) (interface{}, error) {
	go client.syncCallback(reqMsg, respMsg)
	return nil, nil
}

func (client GrpcRemotingClient) syncCallback(reqMsg message.RpcMessage, respMsg *message.MessageFuture) (interface{}, error) {
	select {
	case <-gxtime.GetDefaultTimerWheel().After(RpcRequestTimeout):
		client.grpcRemoting.RemoveMergedMessageFuture(reqMsg.ID)
		log.Errorf("wait resp timeout: %#v", reqMsg)
		return nil, fmt.Errorf("wait response timeout, request: %#v", reqMsg)
	case <-respMsg.Done:
		return respMsg.Response, respMsg.Err
	}
}

func (client *GrpcRemotingClient) GetMergedMessage(msgID int32) *message.MergedWarpMessage {
	return client.grpcRemoting.GetMergedMessage(msgID)
}

func (client *GrpcRemotingClient) GetMessageFuture(msgID int32) *message.MessageFuture {
	return client.grpcRemoting.GetMessageFuture(msgID)
}

func (client *GrpcRemotingClient) RemoveMessageFuture(msgID int32) {
	client.grpcRemoting.RemoveMessageFuture(msgID)
}

func (client *GrpcRemotingClient) RemoveMergedMessageFuture(msgID int32) {
	client.grpcRemoting.RemoveMergedMessageFuture(msgID)
}

func (client *GrpcRemotingClient) NotifyRpcMessageResponse(msg message.RpcMessage) {
	client.grpcRemoting.NotifyRpcMessageResponse(msg)
}
