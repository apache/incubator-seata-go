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
	resolver brokerAddrResolver
	sender   tcpSender
}

func NewTCCRocketMQAction(producer *SeataMQProducer) *TCCRocketMQAction {
	poolSize := defaultConnPoolSize
	if producer != nil && producer.config != nil && producer.config.ConnPoolSize > 0 {
		poolSize = producer.config.ConnPoolSize
	}
	return &TCCRocketMQAction{
		producer: producer,
		resolver: &defaultBrokerAddrResolver{},
		sender:   newDefaultTCPSender(poolSize),
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
	bac.ActionContext[ActionContextKeyTopic] = msg.Topic

	log.Infof("[TCCRocketMQ] Prepare success, xid=%s, branchId=%d, msgId=%s", xid, bac.BranchId, result.MsgID)

	return true, nil
}

func (a *TCCRocketMQAction) Commit(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	if a.producer == nil || a.producer.config == nil {
		log.Warnf("[TCCRocketMQ] Commit skipped, producer or config is nil, fallback to check-back, xid=%s, branchId=%d", bac.Xid, bac.BranchId)
		return true, nil
	}
	topic := getStringFromMap(bac.ActionContext, ActionContextKeyTopic)
	brokerName := getStringFromMap(bac.ActionContext, ActionContextKeyBrokerName)
	if topic == "" || brokerName == "" {
		log.Warnf("[TCCRocketMQ] Commit missing metadata (topic=%s, brokerName=%s), skip active END_TRANSACTION, fallback to check-back, xid=%s, branchId=%d",
			topic, brokerName, bac.Xid, bac.BranchId)
		return true, nil
	}
	header, ok := a.buildEndTransactionHeader(bac, topic, commitOrRollbackCommit)
	if !ok {
		log.Warnf("[TCCRocketMQ] Commit cannot resolve valid commitLogOffset, skip active END_TRANSACTION, fallback to check-back, xid=%s, branchId=%d", bac.Xid, bac.BranchId)
		return true, nil
	}
	err := sendEndTransaction(
		a.producer.config.NameServerAddrs,
		topic,
		brokerName,
		header,
		a.producer.config.SendMsgTimeout,
		a.resolver,
		a.sender,
	)
	if err != nil {
		log.Warnf("[TCCRocketMQ] Commit send END_TRANSACTION failed, fallback to check-back, xid=%s, branchId=%d, err=%v",
			bac.Xid, bac.BranchId, err)
		return true, nil
	}
	log.Infof("[TCCRocketMQ] Commit send END_TRANSACTION success, xid=%s, branchId=%d", bac.Xid, bac.BranchId)
	return true, nil
}

func (a *TCCRocketMQAction) Rollback(ctx context.Context, bac *tm.BusinessActionContext) (bool, error) {
	if a.producer == nil || a.producer.config == nil {
		log.Warnf("[TCCRocketMQ] Rollback skipped, producer or config is nil, fallback to check-back, xid=%s, branchId=%d", bac.Xid, bac.BranchId)
		return true, nil
	}
	topic := getStringFromMap(bac.ActionContext, ActionContextKeyTopic)
	brokerName := getStringFromMap(bac.ActionContext, ActionContextKeyBrokerName)
	if topic == "" || brokerName == "" {
		log.Warnf("[TCCRocketMQ] Rollback missing metadata (topic=%s, brokerName=%s), skip active END_TRANSACTION, fallback to check-back, xid=%s, branchId=%d",
			topic, brokerName, bac.Xid, bac.BranchId)
		return true, nil
	}
	header, ok := a.buildEndTransactionHeader(bac, topic, commitOrRollbackRollback)
	if !ok {
		log.Warnf("[TCCRocketMQ] Rollback cannot resolve valid commitLogOffset, skip active END_TRANSACTION, fallback to check-back, xid=%s, branchId=%d", bac.Xid, bac.BranchId)
		return true, nil
	}
	err := sendEndTransaction(
		a.producer.config.NameServerAddrs,
		topic,
		brokerName,
		header,
		a.producer.config.SendMsgTimeout,
		a.resolver,
		a.sender,
	)
	if err != nil {
		log.Warnf("[TCCRocketMQ] Rollback send END_TRANSACTION failed, fallback to check-back, xid=%s, branchId=%d, err=%v",
			bac.Xid, bac.BranchId, err)
		return true, nil
	}
	log.Infof("[TCCRocketMQ] Rollback send END_TRANSACTION success, xid=%s, branchId=%d", bac.Xid, bac.BranchId)
	return true, nil
}

func (a *TCCRocketMQAction) buildEndTransactionHeader(bac *tm.BusinessActionContext, topic string, commitOrRollback int) (*endTransactionRequestHeader, bool) {
	actionCtx := bac.ActionContext
	if actionCtx == nil {
		actionCtx = make(map[string]interface{})
	}

	offsetMsgID := getStringFromMap(actionCtx, ActionContextKeyOffsetMsgId)
	commitLogOffset := int64(0)
	if offsetMsgID != "" {
		msgID, err := primitive.UnmarshalMsgID([]byte(offsetMsgID))
		if err == nil {
			commitLogOffset = msgID.Offset
		}
	}

	if commitLogOffset == 0 {
		if msgIDStr := getStringFromMap(actionCtx, ActionContextKeyMsgId); msgIDStr != "" {
			if msgID, err := primitive.UnmarshalMsgID([]byte(msgIDStr)); err == nil {
				commitLogOffset = msgID.Offset
			}
		}
	}

	if commitLogOffset == 0 {
		return nil, false
	}

	return &endTransactionRequestHeader{
		Topic:                topic,
		ProducerGroup:        a.producer.config.GroupName,
		TranStateTableOffset: getQueueOffsetFromActionContext(actionCtx),
		CommitLogOffset:      commitLogOffset,
		CommitOrRollback:     commitOrRollback,
		FromTransactionCheck: false,
		MsgID:                getStringFromMap(actionCtx, ActionContextKeyMsgId),
		TransactionId:        getStringFromMap(actionCtx, ActionContextKeyTransactionId),
	}, true
}
