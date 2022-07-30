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
	"sync"

	"github.com/seata/seata-go/pkg/protocol/codec"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/getty"
	"github.com/seata/seata-go/pkg/remoting/processor/client"
)

var onceInitTmClient sync.Once

// initTmClient init seata tm client
func initTmClient() {
	onceInitTmClient.Do(func() {
		initTmProcessor()
		initConfig()
		initRemoting()
		initCodec()
	})
}

// initTmProcessor register tm processor
func initTmProcessor() {
	clientOnResponseProcessor := &client.ClientOnResponseProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_SeataMergeResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalBeginResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalCommitResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchStatusReportResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalRollbackResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalStatusResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_RegCltResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BatchResultMsg, clientOnResponseProcessor)

	heartBeatProcessor := &client.ClientHeartBeatProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_HeartbeatMsg, heartBeatProcessor)
}

// todo
// initConfig init config processor
func initConfig() {
}

// initRemoting init rpc client
func initRemoting() {
	getty.InitRpcClient()
}

// initCodec register codec
func initCodec() {
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.BranchCommitRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.BranchCommitResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.BranchRegisterRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.BranchRegisterResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.BranchRollbackRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.BranchRollbackResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalBeginRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalBeginResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalCommitRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalCommitResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalReportResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalRollbackRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalRollbackResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalStatusRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalStatusResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.RegisterRMRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.RegisterRMResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.RegisterTMRequestCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.RegisterTMResponseCodec{})
	codec.GetCodecManager().RegisterCodec(codec.CodecTypeSeata, &codec.GlobalReportRequestCodec{})
}
