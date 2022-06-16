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
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/protocol/codec"
	"github.com/seata/seata-go/pkg/protocol/message"
	"go.uber.org/atomic"
)

var (
	gettyRemotingClient     *GettyRemotingClient
	onceGettyRemotingClient = &sync.Once{}
)

type GettyRemotingClient struct {
	idGenerator *atomic.Uint32
}

func GetGettyRemotingClient() *GettyRemotingClient {
	if gettyRemotingClient == nil {
		onceGettyRemotingClient.Do(func() {
			gettyRemotingClient = &GettyRemotingClient{
				idGenerator: &atomic.Uint32{},
			}
		})
	}
	return gettyRemotingClient
}

func (client *GettyRemotingClient) SendAsyncRequest(msg interface{}) error {
	var msgType message.GettyRequestType
	if _, ok := msg.(message.HeartBeatMessage); ok {
		msgType = message.GettyRequestType_HeartbeatRequest
	} else {
		msgType = message.GettyRequestType_RequestOneway
	}
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       msgType,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendASync(rpcMessage)
}

func (client *GettyRemotingClient) SendAsyncResponse(msg interface{}) error {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.GettyRequestType_Response,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendASync(rpcMessage)
}

func (client *GettyRemotingClient) SendSyncRequest(msg interface{}) (interface{}, error) {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.GettyRequestType_RequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendSync(rpcMessage)
}

func (client *GettyRemotingClient) SendSyncRequestWithTimeout(msg interface{}, timeout time.Duration) (interface{}, error) {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.GettyRequestType_RequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return GetGettyRemotingInstance().SendSyncWithTimeout(rpcMessage, timeout)
}
