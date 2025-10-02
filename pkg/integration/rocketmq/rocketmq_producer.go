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
	"strconv"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"
)

type RocketMQProducer struct {
	transProducer rocketmq.TransactionProducer
}

func NewRocketMQProducer(nameServers []string, groupName string, checker *TransactionChecker) (*RocketMQProducer, error) {
	listener := &transactionListener{
		checker: checker,
	}

	p, err := rocketmq.NewTransactionProducer(
		listener,
		producer.WithNameServer(nameServers),
		producer.WithGroupName(groupName),
		producer.WithRetry(2),
	)
	if err != nil {
		return nil, fmt.Errorf("create transaction producer failed: %w", err)
	}

	if err := p.Start(); err != nil {
		return nil, fmt.Errorf("start transaction producer failed: %w", err)
	}

	return &RocketMQProducer{
		transProducer: p,
	}, nil
}

func (p *RocketMQProducer) SendMessageInTransaction(ctx context.Context, msg interface{}, xid string, branchId int64) (interface{}, error) {
	mqMsg, ok := msg.(*primitive.Message)
	if !ok {
		return nil, fmt.Errorf("message must be *primitive.Message, got %T", msg)
	}

	if xid != "" {
		mqMsg.WithProperty(PropertySeataXID, xid)
		mqMsg.WithProperty(PropertySeataBranchID, strconv.FormatInt(branchId, 10))
	}

	result, err := p.transProducer.SendMessageInTransaction(ctx, mqMsg)
	if err != nil {
		return nil, fmt.Errorf("send message in transaction failed: %w", err)
	}

	if result.Status != primitive.SendOK {
		return nil, fmt.Errorf("send message failed, status: %v", result.Status)
	}

	return result, nil
}

func (p *RocketMQProducer) EndTransaction(ctx context.Context, sendResult interface{}, state TransactionState) error {
	result, ok := sendResult.(*primitive.SendResult)
	if !ok {
		return fmt.Errorf("sendResult must be *primitive.SendResult, got %T", sendResult)
	}

	var localState primitive.LocalTransactionState
	switch state {
	case TransactionCommit:
		localState = primitive.CommitMessageState
		log.Infof("commit rocketmq transaction, msgId=%s", result.MsgID)
	case TransactionRollback:
		localState = primitive.RollbackMessageState
		log.Infof("rollback rocketmq transaction, msgId=%s", result.MsgID)
	default:
		localState = primitive.UnknowState
	}

	if prodImpl, ok := p.transProducer.(interface {
		EndTransaction(context.Context, *primitive.SendResult, primitive.LocalTransactionState) error
	}); ok {
		return prodImpl.EndTransaction(ctx, result, localState)
	}

	log.Warnf("transaction producer does not support EndTransaction, state=%v will be ignored", state)
	return nil
}

func (p *RocketMQProducer) Shutdown() error {
	return p.transProducer.Shutdown()
}

type transactionListener struct {
	checker *TransactionChecker
}

func (l *transactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	return primitive.UnknowState
}

func (l *transactionListener) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
	if l.checker == nil {
		log.Warnf("transaction checker is nil, return unknown state")
		return primitive.UnknowState
	}

	xid := msg.GetProperty(PropertySeataXID)
	if xid == "" {
		log.Errorf("message has no seata xid, msgId=%s", msg.MsgId)
		return primitive.RollbackMessageState
	}

	ctx := context.Background()
	state := l.checker.CheckLocalTransaction(ctx, msg, xid)

	switch state {
	case TransactionCommit:
		return primitive.CommitMessageState
	case TransactionRollback:
		return primitive.RollbackMessageState
	default:
		return primitive.UnknowState
	}
}

type DefaultGlobalStatusChecker struct{}

func (c *DefaultGlobalStatusChecker) GetGlobalStatus(ctx context.Context, xid string) (int32, error) {
	log.Warnf("using default global status checker, always return committed for xid=%s", xid)
	return int32(message.GlobalStatusCommitted), nil
}
