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

	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/remoting/getty"
	"github.com/seata/seata-go/pkg/remoting/processor/client"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/rm/tcc"
)

var onceInitRmClient sync.Once

// initRmClient init seata rm client
func initRmClient() {
	onceInitRmClient.Do(func() {
		initRmProcessor()
		initResourceManager()
	})
}

// initRmProcessor register rm processor
func initRmProcessor() {
	clientOnResponseProcessor := &client.ClientOnResponseProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_SeataMergeResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchRegisterResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchStatusReportResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalLockQueryResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_RegRmResult, clientOnResponseProcessor)
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_GlobalReportResult, clientOnResponseProcessor)

	rmBranchCommitProcessor := &client.RmBranchCommitProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchCommit, rmBranchCommitProcessor)

	rmBranchRollbackProcessor := &client.RmBranchRollbackProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_BranchRollback, rmBranchRollbackProcessor)

	// todo rmundolog processor
	heartBeatProcessor := &client.ClientHeartBeatProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageType_HeartbeatMsg, heartBeatProcessor)
}

// initResourceManager register resource manager
func initResourceManager() {
	rm.GetRmCacheInstance().RegisterResourceManager(tcc.GetTCCResourceManagerInstance())
}
