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
)

import (
	getty "github.com/apache/dubbo-getty"

	gxtime "github.com/dubbogo/gost/time"

	"github.com/pkg/errors"
)

import (
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
)

const (
	RPC_REQUEST_TIMEOUT = 30 * time.Second
)

var (
	gettyRemoting     *GettyRemoting
	onceGettyRemoting = &sync.Once{}
)

type GettyRemoting struct {
	futures     *sync.Map
	mergeMsgMap *sync.Map
}

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

func (client *GettyRemoting) SendSync(msg message.RpcMessage) (interface{}, error) {
	ss := sessionManager.AcquireGettySession()
	return client.sendAsync(ss, msg, RPC_REQUEST_TIMEOUT)
}

func (client *GettyRemoting) SendSyncWithTimeout(msg message.RpcMessage, timeout time.Duration) (interface{}, error) {
	ss := sessionManager.AcquireGettySession()
	return client.sendAsync(ss, msg, timeout)
}

func (client *GettyRemoting) SendASync(msg message.RpcMessage) error {
	ss := sessionManager.AcquireGettySession()
	_, err := client.sendAsync(ss, msg, 0*time.Second)
	return err
}

func (client *GettyRemoting) sendAsync(session getty.Session, msg message.RpcMessage, timeout time.Duration) (interface{}, error) {
	var err error
	if session == nil || session.IsClosed() {
		log.Warn("sendAsyncRequestWithResponse nothing, caused by null channel.")
	}
	resp := message.NewMessageFuture(msg)
	client.futures.Store(msg.ID, resp)
	_, _, err = session.WritePkg(msg, time.Duration(0))
	if err != nil {
		client.futures.Delete(msg.ID)
		log.Errorf("send message: %#v, session: %s", msg, session.Stat())
		return nil, err
	}

	log.Debugf("send message: %#v, session: %s", msg, session.Stat())

	actualTimeOut := timeout
	if timeout <= time.Duration(0) {
		// todo timeoue use config
		actualTimeOut = time.Duration(200)
	}

	wait := func() (interface{}, error) {
		select {
		case <-gxtime.GetDefaultTimerWheel().After(actualTimeOut):
			client.futures.Delete(msg.ID)
			if session != nil {
				return nil, errors.Errorf("wait response timeout, ip: %s, request: %#v", session.RemoteAddr(), msg)
			} else {
				return nil, errors.Errorf("wait response timeout and session is nil, request: %#v", msg)
			}
		case <-resp.Done:
			err = resp.Err
			return resp.Response, err
		}
	}

	if timeout > time.Duration(0) {
		return wait()
	} else {
		go wait()
	}
	return nil, err
}

func (client *GettyRemoting) GetMessageFuture(msgID int32) *message.MessageFuture {
	if msg, ok := client.futures.Load(msgID); ok {
		return msg.(*message.MessageFuture)
	}
	return nil
}

func (client *GettyRemoting) RemoveMessageFuture(msgID int32) {
	client.futures.Delete(msgID)
}

func (client *GettyRemoting) RemoveMergedMessageFuture(msgID int32) {
	client.mergeMsgMap.Delete(msgID)
}

func (client *GettyRemoting) GetMergedMessage(msgID int32) *message.MergedWarpMessage {
	if msg, ok := client.mergeMsgMap.Load(msgID); ok {
		return msg.(*message.MergedWarpMessage)
	}
	return nil
}

func (client *GettyRemoting) NotifyRpcMessageResponse(rpcMessage message.RpcMessage) {
	messageFuture := client.GetMessageFuture(rpcMessage.ID)
	if messageFuture != nil {
		messageFuture.Response = rpcMessage.Body
		// todo add messageFuture.Err
		//messageFuture.Err = rpcMessage.Err
		messageFuture.Done <- true
		//client.msgFutures.Delete(rpcMessage.RequestID)
	} else {
		log.Infof("msg: {} is not found in msgFutures.", rpcMessage.ID)
	}
}
