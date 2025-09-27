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
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	getty "github.com/apache/dubbo-getty"
	gxsync "github.com/dubbogo/gost/sync"
	gxtime "github.com/dubbogo/gost/time"
	"go.uber.org/atomic"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/config"
	"seata.apache.org/seata-go/pkg/util/backoff"
	"seata.apache.org/seata-go/pkg/util/log"
)

func init() {
	c := config.GetGettyConfig()
	initSessionManager(c)
}

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
	var msgType message.GettyRequestType
	if _, ok := msg.(message.HeartBeatMessage); ok {
		msgType = message.GettyRequestTypeHeartbeatRequest
	} else {
		msgType = message.GettyRequestTypeRequestOneway
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
		Type:       message.GettyRequestTypeResponse,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: 0,
		Body:       msg,
	}
	return client.gettyRemoting.SendAsync(rpcMessage, nil, nil)
}

func (client *GettyRemotingClient) SendSyncRequest(msg interface{}) (interface{}, error) {
	rpcMessage := message.RpcMessage{
		ID:         int32(client.idGenerator.Inc()),
		Type:       message.GettyRequestTypeRequestSync,
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

func (client *GettyRemotingClient) reconnectWithBackoff(session getty.Session, maxRetries int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := backoff.Config{
		MinBackoff: 1 * time.Second,
		MaxBackoff: 60 * time.Second,
		MaxRetries: maxRetries,
	}
	backoffInstance := backoff.New(ctx, cfg)

	log.Infof("Starting reconnection attempts with backoff for session: %s", session.Stat())

	for backoffInstance.Ongoing() {
		backoffInstance.Wait()

		log.Infof("Reconnection attempt %d for session: %s", backoffInstance.NumRetries(), session.Stat())
		//retry
		if err := client.reconnectToServer(session); err == nil {
			return
		}
	}

	if err := backoffInstance.Err(); err != nil {
		log.Errorf("Reconnection failed after %d attempts: %v", backoffInstance.NumRetries(), err)
	}
}

func (client *GettyRemotingClient) reconnectToServer(session getty.Session) error {

	remoteAddr := session.RemoteAddr()

	host, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		log.Errorf("Failed to parse remote address %s: %v", remoteAddr, err)
		return err
	}

	_, err = strconv.Atoi(port)
	if err != nil {
		log.Errorf("Failed to parse port %s: %v", port, err)
		return err
	}

	session.Close()

	gettyClient := getty.NewTCPClient(
		getty.WithServerAddress(net.JoinHostPort(host, port)),
		getty.WithConnectionNumber(1),
		getty.WithReconnectInterval(GetSessionManager().gettyConf.ReconnectInterval),
		getty.WithClientTaskPool(gxsync.NewTaskPoolSimple(0)),
	)

	go gettyClient.RunEventLoop(GetSessionManager().newSession)

	log.Infof("Reconnection initiated for server %s", remoteAddr)
	return nil
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
