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
	"time"

	getty "github.com/apache/dubbo-getty"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/rpc"
	"seata.apache.org/seata-go/pkg/util/log"
)

const (
	RpcRequestTimeout = 20 * time.Second
)

var (
	gettyRemoting     *GettyRemoting
	onceGettyRemoting = &sync.Once{}
)

type (
	callbackMethod func(reqMsg message.RpcMessage, respMsg *message.MessageFuture) (interface{}, error)
	GettyRemoting  struct {
		futures     *sync.Map
		mergeMsgMap *sync.Map
	}
)

func GetGettyRemotingInstance() *GettyRemoting {
	if gettyRemoting == nil {
		onceGettyRemoting.Do(func() {
			gettyRemoting = &GettyRemoting{
				futures:     &sync.Map{},
				mergeMsgMap: &sync.Map{},
			}
		})
	}
	return gettyRemoting
}

func (g *GettyRemoting) SendSync(msg message.RpcMessage, s getty.Session, callback callbackMethod) (interface{}, error) {
	if s == nil {
		s = sessionManager.selectSession(msg)
	}
	rpc.BeginCount(s.RemoteAddr())
	result, err := g.sendAsync(s, msg, callback)
	rpc.EndCount(s.RemoteAddr())
	if err != nil {
		log.Errorf("send message: %#v, session: %s", msg, s.Stat())
		return nil, err
	}
	return result, err
}

func (g *GettyRemoting) SendASync(msg message.RpcMessage, s getty.Session, callback callbackMethod) error {
	if s == nil {
		s = sessionManager.selectSession(msg)
	}
	rpc.BeginCount(s.RemoteAddr())
	_, err := g.sendAsync(s, msg, callback)
	rpc.EndCount(s.RemoteAddr())
	if err != nil {
		log.Errorf("send message: %#v, session: %s", msg, s.Stat())
	}
	return err
}

func (g *GettyRemoting) sendAsync(session getty.Session, msg message.RpcMessage, callback callbackMethod) (interface{}, error) {
	if _, ok := msg.Body.(message.HeartBeatMessage); ok {
		log.Debug("send async message: {%#v}", msg)
	} else {
		log.Infof("send async message: {%#v}", msg)
	}
	var err error
	if session == nil || session.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
		return nil, fmt.Errorf("session is closed")
	}
	resp := message.NewMessageFuture(msg)
	g.futures.Store(msg.ID, resp)
	_, _, err = session.WritePkg(msg, time.Duration(0))
	if err != nil {
		g.futures.Delete(msg.ID)
		log.Errorf("send message: %#v, session: %s", msg, session.Stat())
		return nil, err
	}
	if callback != nil {
		return callback(msg, resp)
	}
	return nil, nil
}

func (g *GettyRemoting) GetMessageFuture(msgID int32) *message.MessageFuture {
	if msg, ok := g.futures.Load(msgID); ok {
		return msg.(*message.MessageFuture)
	}
	return nil
}

func (g *GettyRemoting) RemoveMessageFuture(msgID int32) {
	g.futures.Delete(msgID)
}

func (g *GettyRemoting) RemoveMergedMessageFuture(msgID int32) {
	g.mergeMsgMap.Delete(msgID)
}

func (g *GettyRemoting) GetMergedMessage(msgID int32) *message.MergedWarpMessage {
	if msg, ok := g.mergeMsgMap.Load(msgID); ok {
		return msg.(*message.MergedWarpMessage)
	}
	return nil
}

func (g *GettyRemoting) NotifyRpcMessageResponse(rpcMessage message.RpcMessage) {
	messageFuture := g.GetMessageFuture(rpcMessage.ID)
	if messageFuture != nil {
		messageFuture.Response = rpcMessage.Body
		// todo add messageFuture.Err
		// messageFuture.Err = rpcMessage.Err
		messageFuture.Done <- struct{}{}
		// client.msgFutures.Delete(rpcMessage.RequestID)
	} else {
		log.Infof("msg: {} is not found in msgFutures.", rpcMessage.ID)
	}
}
