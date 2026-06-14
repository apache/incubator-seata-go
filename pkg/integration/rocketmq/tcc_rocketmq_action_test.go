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
	"errors"
	"testing"
	"time"

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
	assert.Equal(t, "topic-test", bac.ActionContext[ActionContextKeyTopic])
	assert.Equal(t, 1, transactionProducer.sendCalls)
	assert.Same(t, msg, transactionProducer.lastMsg)
}

func TestTCCRocketMQAction_Commit_Success(t *testing.T) {
	resolver := &stubBrokerAddrResolver{addr: "192.168.1.100:10911"}
	sender := &stubTCPSender{}
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			config: &SeataMQProducerConfig{
				NameServerAddrs: []string{"nameserver:9876"},
				GroupName:       "test-group",
				SendMsgTimeout:  3 * time.Second,
			},
		},
		resolver: resolver,
		sender:   sender,
	}
	bac := &tm.BusinessActionContext{
		Xid:      "xid-123",
		BranchId: 1001,
		ActionContext: map[string]interface{}{
			ActionContextKeyMsgId:         "msg-1",
			ActionContextKeyOffsetMsgId:   "offset-1",
			ActionContextKeyQueueOffset:   int64(11),
			ActionContextKeyTransactionId: "tx-1",
			ActionContextKeyQueueId:       3,
			ActionContextKeyBrokerName:    "broker-a",
			ActionContextKeyTopic:         "topic-test",
		},
	}

	ok, err := action.Commit(context.Background(), bac)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, "topic-test", resolver.lastTopic)
	assert.Equal(t, "broker-a", resolver.lastBroker)
	assert.Equal(t, 1, sender.calls)
	assert.Equal(t, "192.168.1.100:10911", sender.lastAddr)
}

func TestTCCRocketMQAction_Commit_ErrorOnResolverError(t *testing.T) {
	resolver := &stubBrokerAddrResolver{err: errors.New("name server unreachable")}
	sender := &stubTCPSender{}
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			config: &SeataMQProducerConfig{
				NameServerAddrs: []string{"nameserver:9876"},
				GroupName:       "test-group",
				SendMsgTimeout:  3 * time.Second,
			},
		},
		resolver: resolver,
		sender:   sender,
	}
	bac := &tm.BusinessActionContext{
		Xid:      "xid-123",
		BranchId: 1001,
		ActionContext: map[string]interface{}{
			ActionContextKeyMsgId:         "msg-1",
			ActionContextKeyQueueOffset:   int64(11),
			ActionContextKeyTransactionId: "tx-1",
			ActionContextKeyBrokerName:    "broker-a",
			ActionContextKeyTopic:         "topic-test",
		},
	}

	ok, err := action.Commit(context.Background(), bac)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 0, sender.calls)
}

func TestTCCRocketMQAction_Rollback_Success(t *testing.T) {
	resolver := &stubBrokerAddrResolver{addr: "192.168.1.100:10911"}
	sender := &stubTCPSender{}
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			config: &SeataMQProducerConfig{
				NameServerAddrs: []string{"nameserver:9876"},
				GroupName:       "test-group",
				SendMsgTimeout:  3 * time.Second,
			},
		},
		resolver: resolver,
		sender:   sender,
	}
	bac := &tm.BusinessActionContext{
		Xid:      "xid-456",
		BranchId: 2002,
		ActionContext: map[string]interface{}{
			ActionContextKeyMsgId:         "msg-2",
			ActionContextKeyOffsetMsgId:   "offset-2",
			ActionContextKeyQueueOffset:   int64(22),
			ActionContextKeyTransactionId: "tx-2",
			ActionContextKeyQueueId:       1,
			ActionContextKeyBrokerName:    "broker-b",
			ActionContextKeyTopic:         "topic-rollback",
		},
	}

	ok, err := action.Rollback(context.Background(), bac)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, "topic-rollback", resolver.lastTopic)
	assert.Equal(t, "broker-b", resolver.lastBroker)
	assert.Equal(t, 1, sender.calls)
	assert.Equal(t, "192.168.1.100:10911", sender.lastAddr)
}

func TestBuildEndTransactionHeader_ParseCommitLogOffset(t *testing.T) {
	offsetMsgID := primitive.CreateMessageId([]byte{10, 93, 233, 58}, 10911, 42)
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			config: &SeataMQProducerConfig{
				GroupName: "test-group",
			},
		},
	}
	bac := &tm.BusinessActionContext{
		ActionContext: map[string]interface{}{
			ActionContextKeyOffsetMsgId:   offsetMsgID,
			ActionContextKeyQueueOffset:   int64(11),
			ActionContextKeyMsgId:         "msg-1",
			ActionContextKeyTransactionId: "tx-1",
		},
	}

	header := action.buildEndTransactionHeader(bac, "test-topic", commitOrRollbackCommit)

	assert.Equal(t, "test-topic", header.Topic)
	assert.Equal(t, "test-group", header.ProducerGroup)
	assert.Equal(t, int64(11), header.TranStateTableOffset)
	assert.Equal(t, int64(42), header.CommitLogOffset)
	assert.Equal(t, commitOrRollbackCommit, header.CommitOrRollback)
	assert.Equal(t, "msg-1", header.MsgID)
	assert.Equal(t, "tx-1", header.TransactionId)
	assert.False(t, header.FromTransactionCheck)
}

func TestBuildEndTransactionHeader_InvalidOffsetMsgID(t *testing.T) {
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			config: &SeataMQProducerConfig{
				GroupName: "test-group",
			},
		},
	}
	bac := &tm.BusinessActionContext{
		ActionContext: map[string]interface{}{
			ActionContextKeyOffsetMsgId: "invalid-id",
			ActionContextKeyQueueOffset: int64(11),
		},
	}

	header := action.buildEndTransactionHeader(bac, "test-topic", commitOrRollbackRollback)

	assert.Equal(t, "test-topic", header.Topic)
	assert.Equal(t, int64(0), header.CommitLogOffset)
	assert.Equal(t, commitOrRollbackRollback, header.CommitOrRollback)
}

func TestBuildEndTransactionHeader_MissingOffsetMsgID(t *testing.T) {
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			config: &SeataMQProducerConfig{
				GroupName: "test-group",
			},
		},
	}
	bac := &tm.BusinessActionContext{
		ActionContext: map[string]interface{}{
			ActionContextKeyQueueOffset: int64(11),
		},
	}

	header := action.buildEndTransactionHeader(bac, "test-topic", commitOrRollbackCommit)

	assert.Equal(t, "test-topic", header.Topic)
	assert.Equal(t, int64(0), header.CommitLogOffset)
}

func TestTCCRocketMQAction_Rollback_ErrorOnSendError(t *testing.T) {
	resolver := &stubBrokerAddrResolver{addr: "192.168.1.100:10911"}
	sender := &stubTCPSender{err: errors.New("connection refused")}
	action := &TCCRocketMQAction{
		producer: &SeataMQProducer{
			config: &SeataMQProducerConfig{
				NameServerAddrs: []string{"nameserver:9876"},
				GroupName:       "test-group",
				SendMsgTimeout:  3 * time.Second,
			},
		},
		resolver: resolver,
		sender:   sender,
	}
	bac := &tm.BusinessActionContext{
		Xid:      "xid-789",
		BranchId: 3003,
		ActionContext: map[string]interface{}{
			ActionContextKeyMsgId:         "msg-3",
			ActionContextKeyQueueOffset:   int64(33),
			ActionContextKeyTransactionId: "tx-3",
			ActionContextKeyBrokerName:    "broker-c",
			ActionContextKeyTopic:         "topic-send-err",
		},
	}

	ok, err := action.Rollback(context.Background(), bac)

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, 1, resolver.calls)
	assert.Equal(t, 1, sender.calls)
}
