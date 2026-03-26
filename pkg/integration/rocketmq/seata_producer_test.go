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

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/tm"
)

type stubTransactionProducer struct {
	startErr    error
	shutdownErr error
	sendErr     error
	sendResult  *primitive.TransactionSendResult

	startCalls    int
	shutdownCalls int
	sendCalls     int
	lastCtx       context.Context
	lastMsg       *primitive.Message
}

func (s *stubTransactionProducer) Start() error {
	s.startCalls++
	return s.startErr
}

func (s *stubTransactionProducer) Shutdown() error {
	s.shutdownCalls++
	return s.shutdownErr
}

func (s *stubTransactionProducer) SendMessageInTransaction(ctx context.Context, msg *primitive.Message) (*primitive.TransactionSendResult, error) {
	s.sendCalls++
	s.lastCtx = ctx
	s.lastMsg = msg
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	if s.sendResult == nil {
		s.sendResult = &primitive.TransactionSendResult{SendResult: &primitive.SendResult{}}
	}
	return s.sendResult, nil
}

type stubNormalProducer struct {
	startErr    error
	shutdownErr error
	sendErr     error
	sendResult  *primitive.SendResult

	startCalls    int
	shutdownCalls int
	sendCalls     int
	lastCtx       context.Context
	lastMsgs      []*primitive.Message
}

func (s *stubNormalProducer) Start() error {
	s.startCalls++
	return s.startErr
}

func (s *stubNormalProducer) Shutdown() error {
	s.shutdownCalls++
	return s.shutdownErr
}

func (s *stubNormalProducer) SendSync(ctx context.Context, msg ...*primitive.Message) (*primitive.SendResult, error) {
	s.sendCalls++
	s.lastCtx = ctx
	s.lastMsgs = msg
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	if s.sendResult == nil {
		s.sendResult = &primitive.SendResult{}
	}
	return s.sendResult, nil
}

type stubTCCPrepareProxy struct {
	prepareFn func(context.Context, interface{}) (interface{}, error)

	prepareCalls int
	lastCtx      context.Context
	lastParams   interface{}
}

func (s *stubTCCPrepareProxy) Prepare(ctx context.Context, params interface{}) (interface{}, error) {
	s.prepareCalls++
	s.lastCtx = ctx
	s.lastParams = params
	if s.prepareFn != nil {
		return s.prepareFn(ctx, params)
	}
	return nil, nil
}

func TestSeataMQProducerSend_NonGlobalTransactionUsesNormalProducer(t *testing.T) {
	normalProducer := &stubNormalProducer{
		sendResult: &primitive.SendResult{
			Status: primitive.SendOK,
			MsgID:  "msg-1",
		},
	}
	prepareProxy := &stubTCCPrepareProxy{}
	producer := &SeataMQProducer{
		normalProducer: normalProducer,
		tccProxy:       prepareProxy,
	}
	msg := primitive.NewMessage("topic-test", []byte("hello"))

	result, err := producer.Send(context.Background(), msg)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, primitive.SendOK, result.Status)
	assert.Equal(t, "msg-1", result.MsgID)
	assert.Equal(t, 1, normalProducer.sendCalls)
	assert.Same(t, msg, normalProducer.lastMsgs[0])
	assert.Zero(t, prepareProxy.prepareCalls)
}

func TestSeataMQProducerSend_GlobalTransactionUsesTCCProxy(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "xid-123")
	tm.SetBusinessActionContext(ctx, &tm.BusinessActionContext{
		ActionContext: map[string]interface{}{},
	})

	prepareProxy := &stubTCCPrepareProxy{
		prepareFn: func(ctx context.Context, params interface{}) (interface{}, error) {
			bac := tm.GetBusinessActionContext(ctx)
			bac.ActionContext[ActionContextKeyMsgId] = "msg-2"
			bac.ActionContext[ActionContextKeyOffsetMsgId] = "offset-2"
			return true, nil
		},
	}
	producer := &SeataMQProducer{
		tccProxy: prepareProxy,
	}
	msg := primitive.NewMessage("topic-test", []byte("hello"))

	result, err := producer.Send(ctx, msg)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, primitive.SendOK, result.Status)
	assert.Equal(t, "msg-2", result.MsgID)
	assert.Equal(t, "offset-2", result.OffsetMsgID)
	assert.Equal(t, 1, prepareProxy.prepareCalls)
	assert.Same(t, msg, prepareProxy.lastParams)
}

func TestSeataMQProducerSend_GlobalTransactionPrepareError(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "xid-123")
	tm.SetBusinessActionContext(ctx, &tm.BusinessActionContext{
		ActionContext: map[string]interface{}{},
	})

	expectedErr := errors.New("prepare failed")
	producer := &SeataMQProducer{
		tccProxy: &stubTCCPrepareProxy{
			prepareFn: func(context.Context, interface{}) (interface{}, error) {
				return nil, expectedErr
			},
		},
	}

	result, err := producer.Send(ctx, primitive.NewMessage("topic-test", []byte("hello")))

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, result)
}

func TestSeataMQProducerSend_GlobalTransactionRequiresActionContextAfterPrepare(t *testing.T) {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, "xid-123")

	producer := &SeataMQProducer{
		tccProxy: &stubTCCPrepareProxy{
			prepareFn: func(context.Context, interface{}) (interface{}, error) {
				return true, nil
			},
		},
	}

	result, err := producer.Send(ctx, primitive.NewMessage("topic-test", []byte("hello")))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "BusinessActionContext")
	assert.Nil(t, result)
}

func TestSeataMQProducerSend_RejectsNilMessageAndClosedProducer(t *testing.T) {
	producer := &SeataMQProducer{}

	result, err := producer.Send(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message cannot be nil")
	assert.Nil(t, result)

	producer.closed = true
	result, err = producer.Send(context.Background(), primitive.NewMessage("topic-test", []byte("hello")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "producer is closed")
	assert.Nil(t, result)
}

func TestSeataMQProducerStart_ShutsDownTransactionProducerWhenNormalProducerStartFails(t *testing.T) {
	expectedErr := errors.New("normal start failed")
	transactionProducer := &stubTransactionProducer{}
	normalProducer := &stubNormalProducer{startErr: expectedErr}
	producer := &SeataMQProducer{
		transactionProducer: transactionProducer,
		normalProducer:      normalProducer,
	}

	err := producer.Start()

	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, transactionProducer.startCalls)
	assert.Equal(t, 1, normalProducer.startCalls)
	assert.Equal(t, 1, transactionProducer.shutdownCalls)
}
