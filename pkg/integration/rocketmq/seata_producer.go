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
	"sync"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"

	"seata.apache.org/seata-go/v2/pkg/rm/tcc"
	"seata.apache.org/seata-go/v2/pkg/tm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type transactionProducerInterface interface {
	Start() error
	Shutdown() error
	SendMessageInTransaction(context.Context, *primitive.Message) (*primitive.TransactionSendResult, error)
}

type normalProducerInterface interface {
	Start() error
	Shutdown() error
	SendSync(ctx context.Context, msg ...*primitive.Message) (*primitive.SendResult, error)
}

type tccPrepareProxy interface {
	Prepare(context.Context, interface{}) (interface{}, error)
}

type SeataMQProducer struct {
	config              *SeataMQProducerConfig
	transactionProducer transactionProducerInterface
	normalProducer      normalProducerInterface
	tccAction           *TCCRocketMQAction
	tccProxy            tccPrepareProxy

	mu     sync.RWMutex
	closed bool
}

func NewSeataMQProducer(cfg *SeataMQProducerConfig) (*SeataMQProducer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if cfg.NameServerAddrs == nil || len(cfg.NameServerAddrs) == 0 {
		return nil, fmt.Errorf("NameServerAddrs cannot be empty")
	}

	if cfg.GroupName == "" {
		return nil, fmt.Errorf("GroupName cannot be empty")
	}

	p := &SeataMQProducer{
		config: cfg,
	}

	p.tccAction = NewTCCRocketMQAction(p)

	// NewTCCServiceProxy internally calls ParseTCCResource, so we pass the action directly
	var err error
	p.tccProxy, err = tcc.NewTCCServiceProxy(p.tccAction)
	if err != nil {
		return nil, fmt.Errorf("create TCC proxy failed: %w", err)
	}

	listener := NewSeataTransactionListener(p)
	opts := cfg.ToRocketMQProducerOptions()

	p.transactionProducer, err = producer.NewTransactionProducer(listener, opts...)
	if err != nil {
		return nil, fmt.Errorf("create transaction producer failed: %w", err)
	}

	normalCfg := *cfg
	normalCfg.GroupName = cfg.GroupName + "-normal"
	normalOpts := normalCfg.ToRocketMQProducerOptions()
	p.normalProducer, err = producer.NewDefaultProducer(normalOpts...)
	if err != nil {
		return nil, fmt.Errorf("create normal producer failed: %w", err)
	}

	return p, nil
}

func (p *SeataMQProducer) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("producer already closed")
	}

	if err := p.transactionProducer.Start(); err != nil {
		return err
	}

	if err := p.normalProducer.Start(); err != nil {
		p.transactionProducer.Shutdown()
		return err
	}

	return nil
}

func (p *SeataMQProducer) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	var errs []error
	if err := p.transactionProducer.Shutdown(); err != nil {
		errs = append(errs, err)
	}
	if err := p.normalProducer.Shutdown(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

func (p *SeataMQProducer) Send(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, fmt.Errorf("producer is closed")
	}

	if msg == nil {
		return nil, fmt.Errorf("message cannot be nil")
	}

	if !tm.IsGlobalTx(ctx) {
		return p.sendSync(ctx, msg)
	}

	_, err := p.tccProxy.Prepare(ctx, msg)
	if err != nil {
		log.Errorf("[SeataMQProducer] Send in global tx failed, xid=%s, err=%v", tm.GetXID(ctx), err)
		return nil, err
	}

	bac := tm.GetBusinessActionContext(ctx)
	if bac == nil || bac.ActionContext == nil {
		return nil, fmt.Errorf("BusinessActionContext action context not found after Prepare")
	}
	return &primitive.SendResult{
		Status:      primitive.SendOK,
		MsgID:       getStringFromMap(bac.ActionContext, ActionContextKeyMsgId),
		OffsetMsgID: getStringFromMap(bac.ActionContext, ActionContextKeyOffsetMsgId),
	}, nil
}

func (p *SeataMQProducer) sendSync(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	return p.normalProducer.SendSync(ctx, msg)
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
