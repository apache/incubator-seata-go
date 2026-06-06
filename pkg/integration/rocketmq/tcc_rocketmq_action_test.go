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
	"testing"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/rm/tcc"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

func TestTCCRocketMQAction_ParseTCCResource(t *testing.T) {
	action := &TCCRocketMQAction{}

	resource, err := tcc.ParseTCCResource(action)
	assert.NoError(t, err)
	assert.NotNil(t, resource)

	assert.Equal(t, ResourceIDTCCRocketMQ, resource.GetResourceId())
	assert.Equal(t, ResourceIDTCCRocketMQ, action.GetActionName())
}

func TestTCCRocketMQAction_GetActionName(t *testing.T) {
	action := &TCCRocketMQAction{}
	assert.Equal(t, ResourceIDTCCRocketMQ, action.GetActionName())
}

func TestTCCRocketMQActionPrepare_SetsMessagePropertiesAndActionContext(t *testing.T) {
	transactionProducer := &stubTransactionProducer{
		sendResult: &primitive.TransactionSendResult{
			SendResult: &primitive.SendResult{
				MsgID:         "msg-1",
				OffsetMsgID:   "offset-1",
				QueueOffset:   11,
				TransactionID: "tx-1",
				MessageQueue: &primitive.MessageQueue{
					BrokerName: "broker-a",
					QueueId:    3,
				},
			},
		},
	}
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			transactionProducer: transactionProducer,
		},
	}
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "xid-123")
	tm.SetBusinessActionContext(ctx, &tm.BusinessActionContext{
		BranchId:      1001,
		ActionContext: map[string]interface{}{},
	})
	msg := primitive.NewMessage("topic-test", []byte("hello"))

	ok, err := action.Prepare(ctx, msg)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "xid-123", msg.GetProperty(constant.PropertySeataXID))
	assert.Equal(t, "1001", msg.GetProperty(constant.PropertySeataBranchId))
	bac := tm.GetBusinessActionContext(ctx)
	require.NotNil(t, bac)
	assert.Equal(t, "msg-1", bac.ActionContext[ActionContextKeyMsgId])
	assert.Equal(t, "offset-1", bac.ActionContext[ActionContextKeyOffsetMsgId])
	assert.EqualValues(t, int64(11), bac.ActionContext[ActionContextKeyQueueOffset])
	assert.Equal(t, "tx-1", bac.ActionContext[ActionContextKeyTransactionId])
	assert.Equal(t, 3, bac.ActionContext[ActionContextKeyQueueId])
	assert.Equal(t, "broker-a", bac.ActionContext[ActionContextKeyBrokerName])
	assert.Equal(t, 1, transactionProducer.sendCalls)
	assert.Same(t, msg, transactionProducer.lastMsg)
}
