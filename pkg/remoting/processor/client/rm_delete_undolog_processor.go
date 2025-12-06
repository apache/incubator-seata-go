/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.   You may obtain a copy of the License at
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

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/getty"
	"seata.apache.org/seata-go/pkg/util/log"
)

func initDeleteUndoLog() {
	rmDeleteUndoLogProcessor := &rmDeleteUndoLogProcessor{}
	getty.GetGettyClientHandlerInstance().RegisterProcessor(message.MessageTypeRmDeleteUndolog, rmDeleteUndoLogProcessor)
}

type rmDeleteUndoLogProcessor struct{}

func (r *rmDeleteUndoLogProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	req := rpcMessage.Body.(message.UndoLogDeleteRequest)

	log.Infof("Received undo log delete request: resourceId=%s, saveDays=%d, branchType=%v",
		req.ResourceId, req.SaveDays, req.BranchType)

	// TODO: Implement actual undo log deletion logic here
	// This should delete undo log records older than saveDays for the specified resourceId and branchType

	// Note: The TC server sends this message as a one-way notification, no response is required
	return nil
}
