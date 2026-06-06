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

	"github.com/apache/rocketmq-client-go/v2/primitive"
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
