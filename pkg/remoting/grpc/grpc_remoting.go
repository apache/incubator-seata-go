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
	"time"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/remoting/rpc"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

const RpcRequestTimeout = 20 * time.Second

// rpcRequestTimeout is the deadline the client actually waits on. It exists so
// tests can shorten the wait without changing the exported constant.
var rpcRequestTimeout = RpcRequestTimeout

type (
	callbackMethod func(reqMsg message.RpcMessage, respMsg *message.MessageFuture) (interface{}, error)

	GrpcRemoting struct {
		futures     *sync.Map
		mergeMsgMap *sync.Map
	}
)

func newGrpcRemoting() *GrpcRemoting {
	return &GrpcRemoting{
		futures:     &sync.Map{},
		mergeMsgMap: &sync.Map{},
	}
}

// SendSync sends a request and waits for the response
func (g *GrpcRemoting) SendSync(msg message.RpcMessage, channel *Channel, callback callbackMethod) (interface{}, error) {
	if channel == nil {
		channel = channelManager.selectChannel(msg)
	}
	if channel == nil || channel.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
		return nil, fmt.Errorf("stream is closed")
	}
	rpc.BeginCount(channel.addr)
	resp, err := g.sendAsync(channel, msg, callback)
	rpc.EndCount(channel.addr)
	if err != nil {
		log.Errorf("send message: %#v", msg)
		return nil, err
	}
	return resp, nil
}

// SendAsync sends a request asynchronously
func (g *GrpcRemoting) SendAsync(msg message.RpcMessage, channel *Channel, callback callbackMethod) error {
	if channel == nil {
		channel = channelManager.selectChannel(msg)
	}
	if channel == nil || channel.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
		return fmt.Errorf("stream is closed")
	}
	rpc.BeginCount(channel.addr)
	_, err := g.sendAsync(channel, msg, callback)
	rpc.EndCount(channel.addr)
	if err != nil {
		log.Errorf("send message: %#v", msg)
	}
	return err
}

func (g *GrpcRemoting) sendAsync(channel *Channel, msg message.RpcMessage, callback callbackMethod) (interface{}, error) {
	if _, ok := msg.Body.(*pb.HeartbeatMessageProto); ok {
		log.Debug("send async message: {%#v}", msg)
	} else {
		log.Infof("send async message: {%#v}", msg)
	}
	if channel == nil || channel.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
		return nil, fmt.Errorf("stream is closed")
	}
	// The future is owned by whoever waits for it. sendAsync clears it on every
	// path that ends here; the callback path is cleared by syncCallback, since
	// an asynchronous callback returns straight away and the wait happens in
	// the goroutine it starts.
	resp := message.NewMessageFuture(msg)
	g.futures.Store(msg.ID, resp)
	request, err := Encode(msg)
	if err != nil {
		g.futures.Delete(msg.ID)
		log.Errorf("encode message: %#v", msg)
		return nil, err
	}

	if err := channel.Send(request); err != nil {
		g.futures.Delete(msg.ID)
		log.Errorf("send message: %#v", msg)
		return nil, err
	}
	if callback != nil {
		return callback(msg, resp)
	}
	// Nothing is going to wait for this one, so it would stay in the map for
	// the lifetime of the process.
	g.futures.Delete(msg.ID)
	return nil, nil
}

func (g *GrpcRemoting) GetMessageFuture(msgID int32) *message.MessageFuture {
	if msg, ok := g.futures.Load(msgID); ok {
		return msg.(*message.MessageFuture)
	}
	return nil
}

func (g *GrpcRemoting) RemoveMessageFuture(msgID int32) {
	g.futures.Delete(msgID)
}

func (g *GrpcRemoting) RemoveMergedMessageFuture(msgID int32) {
	g.mergeMsgMap.Delete(msgID)
}

func (g *GrpcRemoting) GetMergedMessage(msgID int32) *message.MergedWarpMessage {
	if msg, ok := g.mergeMsgMap.Load(msgID); ok {
		return msg.(*message.MergedWarpMessage)
	}
	return nil
}

func (g *GrpcRemoting) NotifyRpcMessageResponse(rpcMessage message.RpcMessage) {
	messageFuture := g.GetMessageFuture(rpcMessage.ID)
	if messageFuture != nil {
		// todo add messageFuture.Err
		// messageFuture.Err = rpcMessage.Err
		if !messageFuture.Complete(rpcMessage.Body) {
			log.Warnf("response notification dropped for msg ID: %d because the future was already signaled", rpcMessage.ID)
		}
		// The waiter removes the future once it stops waiting.
	} else {
		log.Infof("msg: {} is not found in msgFutures.", rpcMessage.ID)
	}
}
