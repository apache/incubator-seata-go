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
	"sync"

	gxtime "github.com/dubbogo/gost/time"
	"go.uber.org/atomic"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"
)

var (
	gettyRemotingClient     *GettyRemotingClient
	onceGettyRemotingClient = &sync.Once{}
)

type GettyRemotingClient struct {
	idGenerator   *atomic.Uint32
	gettyRemoting *GettyRemoting
}

func GetGettyRemotingClient() *GettyRemotingClient {
	if gettyRemotingClient == nil {
		onceGettyRemotingClient.Do(func() {
			gettyRemotingClient = &GettyRemotingClient{
				idGenerator:   &atomic.Uint32{},
				gettyRemoting: newGettyRemoting(),
			}
		})
	}
	return gettyRemotingClient
}

func (client *GettyRemotingClient) SendAsyncRequest(msg interface{}) error {
	var msgType message.RequestType
	if _, ok := msg.(message.HeartBeatMessage); ok {
		msgType = message.RequestTypeHeartbeatRequest
	} else {
		msgType = message.RequestTypeRequestOneway
	}
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       msgType,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return client.gettyRemoting.SendAsync(rpcMessage, nil, client.asyncCallback)
}

func (client *GettyRemotingClient) SendAsyncResponse(msgID int32, msg interface{}) error {
	rpcMessage := message.RpcMessage{
		ID:         msgID,
		Type:       message.RequestTypeResponse,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return client.gettyRemoting.SendAsync(rpcMessage, nil, nil)
}

func (client *GettyRemotingClient) SendSyncRequest(msg interface{}) (interface{}, error) {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return client.gettyRemoting.SendSync(rpcMessage, nil, client.syncCallback)
}

func (g *GettyRemotingClient) asyncCallback(reqMsg message.RpcMessage, respMsg *message.MessageFuture) (interface{}, error) {
	go g.syncCallback(reqMsg, respMsg)
	return nil, nil
}

func (g *GettyRemotingClient) syncCallback(reqMsg message.RpcMessage, respMsg *message.MessageFuture) (interface{}, error) {
	select {
	case <-gxtime.GetDefaultTimerWheel().After(RpcRequestTimeout):
		g.gettyRemoting.RemoveMergedMessageFuture(reqMsg.ID)
		log.Errorf("wait resp timeout: %#v", reqMsg)
		return nil, fmt.Errorf("wait response timeout, request: %#v", reqMsg)
	case <-respMsg.Done:
		return respMsg.Response, respMsg.Err
	}
}

func (client *GettyRemotingClient) GetMergedMessage(msgID int32) *message.MergedWarpMessage {
	return client.gettyRemoting.GetMergedMessage(msgID)
}

func (client *GettyRemotingClient) GetMessageFuture(msgID int32) *message.MessageFuture {
	return client.gettyRemoting.GetMessageFuture(msgID)
}

func (client *GettyRemotingClient) RemoveMessageFuture(msgID int32) {
	client.gettyRemoting.RemoveMessageFuture(msgID)
}

func (client *GettyRemotingClient) RemoveMergedMessageFuture(msgID int32) {
	client.gettyRemoting.RemoveMergedMessageFuture(msgID)
}

func (client *GettyRemotingClient) NotifyRpcMessageResponse(msg message.RpcMessage) {
	client.gettyRemoting.NotifyRpcMessageResponse(msg)
}
