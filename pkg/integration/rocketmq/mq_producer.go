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
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/pkg/errors"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/tm"
	"seata.apache.org/seata-go/pkg/util/log"
	"strconv"
	"strings"
)

type MQProducer struct {
	rocketmq.TransactionProducer
	rocketmq.Producer
	transactionListener primitive.TransactionListener
	rocketMQ            TCCRocketMQ
	config              *Config
}

func NewMQProducer(ctx context.Context, config *Config, rocketMQ TCCRocketMQ) (*MQProducer, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	if _, ok := rocketMQ.(*TCCRocketMQImpl); !ok {
		return nil, fmt.Errorf("TCCRocketMQ is invalid")
	}

	tl := newTransactionListener(ctx)

	tp, err := createTransactionProducer(tl, config)
	if err != nil {
		return nil, fmt.Errorf("create transaction producer failed: %w", err)
	}

	p, err := createProducer(config)
	if err != nil {
		return nil, fmt.Errorf("create producer failed: %w", err)
	}

	return &MQProducer{
		TransactionProducer: tp,
		Producer:            p,
		transactionListener: tl,
		rocketMQ:            rocketMQ,
		config:              config,
	}, nil
}

func validateConfig(config *Config) error {
	if config.Addr == "" {
		return errors.New("addr cannot be empty")
	}
	if config.GroupName == "" {
		return errors.New("producer group cannot be empty")
	}
	return nil
}

func createTransactionProducer(tl primitive.TransactionListener, config *Config) (rocketmq.TransactionProducer, error) {
	return rocketmq.NewTransactionProducer(tl,
		producer.WithNameServer([]string{config.Addr}),
		producer.WithNamespace(config.Namespace),
		producer.WithGroupName(config.GroupName),
	)
}

func createProducer(config *Config) (rocketmq.Producer, error) {
	return rocketmq.NewProducer(
		producer.WithNameServer([]string{config.Addr}),
		producer.WithNamespace(config.Namespace),
		producer.WithGroupName(config.GroupName),
	)
}

func (p *MQProducer) Run() error {
	if err := p.TransactionProducer.Start(); err != nil {
		return err
	}

	if err := p.Producer.Start(); err != nil {
		_ = p.TransactionProducer.Shutdown()
		return err
	}

	return nil
}

func (p *MQProducer) SendMsg(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	if tm.IsGlobalTx(ctx) {
		return p.sendTransactionMsg(ctx, msg)
	}
	return p.SendSync(ctx, msg)
}

func (p *MQProducer) sendTransactionMsg(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	if p.rocketMQ.(*TCCRocketMQImpl).producer == nil {
		return nil, errors.New("TCCRocketMQ is not initialized")
	}
	return p.rocketMQ.Prepare(ctx, msg)
}

func (p *MQProducer) SendMsgInTransaction(msg *primitive.Message, xid string, branchID int64) (*primitive.SendResult, error) {
	if err := p.prepareMsgForTransaction(msg, xid, branchID); err != nil {
		return nil, err
	}

	result, err := p.SendMessageInTransaction(context.Background(), msg)
	if err != nil {
		return nil, fmt.Errorf("send transaction message failed: %w", err)
	}

	return result.SendResult, nil
}

func (p *MQProducer) prepareMsgForTransaction(msg *primitive.Message, xid string, branchID int64) error {
	msg.Topic = p.withNamespace(msg.Topic)
	msg.RemoveProperty(primitive.PropertyDelayTimeLevel)

	if len(msg.Topic) == 0 {
		return errors.New("topic cannot be empty")
	}

	msg.WithProperties(map[string]string{
		PropertySeataXID:      xid,
		PropertySeataBranchID: strconv.FormatInt(branchID, 10),
	})

	return nil
}

func (p *MQProducer) SetTCCRocketMQ(rocketMQ TCCRocketMQ) {
	p.rocketMQ = rocketMQ
}

func (p *MQProducer) withNamespace(topic string) string {
	if p.config.Namespace == "" {
		return topic
	}
	return p.config.Namespace + "%" + topic
}

type TransactionListener struct {
	ctx context.Context
}

func newTransactionListener(ctx context.Context) *TransactionListener {
	return &TransactionListener{ctx: ctx}
}

func (l *TransactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	log.Info("ExecuteLocalTransaction: xid = ", msg.GetProperty(PropertySeataXID),
		", branchID = ", msg.GetProperty(PropertySeataBranchID))
	return primitive.UnknowState
}

func (l *TransactionListener) CheckLocalTransaction(msgExt *primitive.MessageExt) primitive.LocalTransactionState {
	xid := msgExt.GetProperty(PropertySeataXID)
	if !isValidXID(xid) {
		return primitive.RollbackMessageState
	}

	globalStatus := *tm.GetTxStatus(l.ctx)
	return determineTransactionState(globalStatus)
}

func isValidXID(xid string) bool {
	return len(xid) > 0 && strings.TrimSpace(xid) != ""
}

func determineTransactionState(status message.GlobalStatus) primitive.LocalTransactionState {
	if _, exists := GlobalStatusMaps.Commit[status]; exists {
		return primitive.CommitMessageState
	}

	if _, exists := GlobalStatusMaps.Rollback[status]; exists || isOnePhaseTimeout(status) {
		return primitive.RollbackMessageState
	}

	if status == message.GlobalStatusFinished {
		return primitive.RollbackMessageState
	}

	return primitive.UnknowState
}

func isOnePhaseTimeout(status message.GlobalStatus) bool {
	timeoutStatuses := []message.GlobalStatus{
		message.GlobalStatusTimeoutRollbacking,
		message.GlobalStatusTimeoutRollbackRetrying,
		message.GlobalStatusTimeoutRollbacked,
		message.GlobalStatusTimeoutRollbackFailed,
	}

	for _, s := range timeoutStatuses {
		if status == s {
			return true
		}
	}
	return false
}
