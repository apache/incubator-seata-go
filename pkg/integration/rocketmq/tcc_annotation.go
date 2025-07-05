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
	"seata.apache.org/seata-go/pkg/tm"
)

type TccRocketMQAnnotation struct {
	resourceName string
}

func NewTccRocketMQAnnotation(resourceName string) *TccRocketMQAnnotation {
	if resourceName == "" {
		resourceName = RocketTccResourceName
	}

	return &TccRocketMQAnnotation{
		resourceName: resourceName,
	}
}

func (t *TccRocketMQAnnotation) SendMessageWithTcc(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error) {
	if !tm.IsGlobalTx(ctx) {
		producer, err := GetProducer()
		if err != nil {
			return nil, err
		}
		return producer.SendMessage(ctx, msg)
	}

	tccImpl, err := GetTCCRocketMQ()
	if err != nil {
		return nil, err
	}

	return tccImpl.Prepare(ctx, msg)
}

func (t *TccRocketMQAnnotation) GetResourceName() string {
	return t.resourceName
}
