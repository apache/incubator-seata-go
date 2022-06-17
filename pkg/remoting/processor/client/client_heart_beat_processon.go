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

	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/getty"
)

func init() {
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_HeartbeatMsg, &clientHeartBeatProcesson{})
}

type clientHeartBeatProcesson struct{}

func (f *clientHeartBeatProcesson) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	if _, ok := rpcMessage.Body.(message.HeartBeatMessage); ok {
		// TODO 如何从context中获取远程服务的信息？
		//log.Infof("received PONG from {}", ctx.channel().remoteAddress())
		log.Infof("received PONG from {}", ctx)
	}
	return nil
}
