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
	"sync"

	getty "github.com/apache/dubbo-getty"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/config"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/processor"
	"go.uber.org/atomic"
)

var (
	clientHandler     *gettyClientHandler
	onceClientHandler = &sync.Once{}
)

type gettyClientHandler struct {
	conf           *config.ClientConfig
	idGenerator    *atomic.Uint32
	msgFutures     *sync.Map
	mergeMsgMap    *sync.Map
	processorTable map[message.MessageType]processor.RemotingProcessor
}

func GetGettyClientHandlerInstance() *gettyClientHandler {
	if clientHandler == nil {
		onceClientHandler.Do(func() {
			clientHandler = &gettyClientHandler{
				conf:           config.GetDefaultClientConfig("seata-go"),
				idGenerator:    &atomic.Uint32{},
				msgFutures:     &sync.Map{},
				mergeMsgMap:    &sync.Map{},
				processorTable: make(map[message.MessageType]processor.RemotingProcessor, 0),
			}
		})
	}
	return clientHandler
}

func (client *gettyClientHandler) OnOpen(session getty.Session) error {
	sessionManager.RegisterGettySession(session)
	go func() {
		request := message.RegisterTMRequest{AbstractIdentifyRequest: message.AbstractIdentifyRequest{
			Version:                 client.conf.SeataVersion,
			ApplicationId:           client.conf.ApplicationID,
			TransactionServiceGroup: client.conf.TransactionServiceGroup,
		}}
		err := GetGettyRemotingClient().SendAsyncRequest(request)
		//client.sendAsyncRequestWithResponse(session, request, RPC_REQUEST_TIMEOUT)
		if err != nil {
			log.Errorf("OnOpen error: {%#v}", err.Error())
			sessionManager.ReleaseGettySession(session)
			return
		}
		//todo
		//client.GettySessionOnOpenChannel <- session.RemoteAddr()
	}()
	return nil
}

func (client *gettyClientHandler) OnError(session getty.Session, err error) {
	log.Infof("OnError session{%s} got error{%v}, will be closed.", session.Stat(), err)
	sessionManager.ReleaseGettySession(session)
}

func (client *gettyClientHandler) OnClose(session getty.Session) {
	log.Infof("OnClose session{%s} is closing......", session.Stat())
	sessionManager.ReleaseGettySession(session)
}

func (client *gettyClientHandler) OnMessage(session getty.Session, pkg interface{}) {
	ctx := context.Background()
	log.Debugf("received message: {%#v}", pkg)
	rpcMessage, ok := pkg.(message.RpcMessage)
	if !ok {
		log.Errorf("received message is not protocol.RpcMessage. pkg: %#v", pkg)
		return
	}
	if mm, ok := rpcMessage.Body.(message.MessageTypeAware); ok {
		processor := client.processorTable[mm.GetTypeCode()]
		if processor != nil {
			processor.Process(ctx, rpcMessage)
		} else {
			log.Errorf("This message type [%v] has no processor.", mm.GetTypeCode())
		}
	} else {
		log.Errorf("This rpcMessage body %#v is not MessageTypeAware type.", rpcMessage.Body)
	}
}

func (client *gettyClientHandler) OnCron(session getty.Session) {
	//GetGettyRemotingClient().SendAsyncRequest(message.HeartBeatMessagePing)
}

func (client *gettyClientHandler) RegisterProcessor(msgType message.MessageType, processor processor.RemotingProcessor) {
	if nil != processor {
		client.processorTable[msgType] = processor
	}
}
