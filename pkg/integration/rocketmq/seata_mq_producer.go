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

	"seata.apache.org/seata-go/pkg/tm"
)

type SeataMQProducer struct {
	producer    MQProducer
	tccRocketMQ *TCCRocketMQService
}

func NewSeataMQProducer(producer MQProducer) *SeataMQProducer {
	return &SeataMQProducer{
		producer: producer,
	}
}

func (p *SeataMQProducer) Send(ctx context.Context, msg interface{}) (interface{}, error) {
	if tm.IsGlobalTx(ctx) {
		if p.tccRocketMQ == nil {
			return nil, fmt.Errorf("tcc rocketmq service not initialized")
		}
		_, err := p.tccRocketMQ.Prepare(ctx, msg)
		if err != nil {
			return nil, err
		}
		return tm.GetBusinessActionContext(ctx).ActionContext[RocketSendResultKey], nil
	}
	return p.producer.SendMessageInTransaction(ctx, msg, "", 0)
}

func (p *SeataMQProducer) SetTCCRocketMQ(tccRocketMQ *TCCRocketMQService) {
	p.tccRocketMQ = tccRocketMQ
}

func (p *SeataMQProducer) GetProducer() MQProducer {
	return p.producer
}
