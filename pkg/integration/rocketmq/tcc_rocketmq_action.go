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

package rocketmq

import (
	"context"
	"fmt"

	"github.com/apache/rocketmq-client-go/v2/primitive"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type TCCRocketMQAction struct {
	producer *SeataMQProducer
}

func NewTCCRocketMQAction(producer *SeataMQProducer) *TCCRocketMQAction {
	return &TCCRocketMQAction{
		producer: producer,
	}
}

func (a *TCCRocketMQAction) GetActionName() string {
	return ResourceIDTCCRocketMQ
}

func (a *TCCRocketMQAction) Prepare(ctx context.Context, params interface{}) (bool, error) {
	msg, ok := params.(*primitive.Message)
	if !ok {
		return false, fmt.Errorf("params must be *primitive.Message, got %T", params)
	}

	bac := tm.GetBusinessActionContext(ctx)
	if bac == nil {
		return false, fmt.Errorf("BusinessActionContext not found in context")
	}

	xid := tm.GetXID(ctx)
	if xid == "" {
		return false, fmt.Errorf("XID not found in context")
	}
	if bac.ActionContext == nil {
		bac.ActionContext = make(map[string]interface{}, 6)
	}

	msg.WithProperty(constant.PropertySeataXID, xid)
	msg.WithProperty(constant.PropertySeataBranchId, fmt.Sprintf("%d", bac.BranchId))

	result, err := a.producer.transactionProducer.SendMessageInTransaction(ctx, msg)
	if err != nil {
		log.Errorf("[TCCRocketMQ] Prepare failed, xid=%s, err=%v", xid, err)
		return false, err
	}

	bac.ActionContext[ActionContextKeyMsgId] = result.MsgID
	bac.ActionContext[ActionContextKeyOffsetMsgId] = result.OffsetMsgID
	bac.ActionContext[ActionContextKeyQueueOffset] = result.QueueOffset
	bac.ActionContext[ActionContextKeyTransactionId] = result.TransactionID
	if result.MessageQueue != nil {
		bac.ActionContext[ActionContextKeyQueueId] = result.MessageQueue.QueueId
		bac.ActionContext[ActionContextKeyBrokerName] = result.MessageQueue.BrokerName
	}

	log.Infof("[TCCRocketMQ] Prepare success, xid=%s, branchId=%d, msgId=%s", xid, bac.BranchId, result.MsgID)

	return true, nil
}

func (a *TCCRocketMQAction) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	// Commit is a no-op because RocketMQ transactional messages use a check-back mechanism.
	// When the global transaction commits, RocketMQ will invoke CheckLocalTransaction
	// via SeataTransactionListener to determine the final message disposition.
	// The message has already been sent to the broker during Prepare phase with an
	// initial state of UnknowState, pending the check-back resolution.
	log.Infof("[TCCRocketMQ] Commit (no-op, rely on check-back), xid=%s, branchId=%d", bac.Xid, bac.BranchId)
	return true, nil
}

func (a *TCCRocketMQAction) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	// Rollback is a no-op because RocketMQ transactional messages use a check-back mechanism.
	// When the global transaction rolls back, RocketMQ will invoke CheckLocalTransaction
	// via SeataTransactionListener, which queries the TC for the global status and returns
	// RollbackMessageState, causing the broker to discard the message.
	log.Infof("[TCCRocketMQ] Rollback (no-op, rely on check-back), xid=%s, branchId=%d", bac.Xid, bac.BranchId)
	return true, nil
}
