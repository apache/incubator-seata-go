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
	"sync"
)

var (
	defaultProducer *MQProducer
	producerOnce    sync.Once
	initError       error
)

var (
	ErrProducerExists = errors.New("producer already exists")
)

func CreateSingle(nameServer, producerGroup string) (*MQProducer, error) {
	return CreateSingleWithNamespace(context.Background(), nameServer, "", producerGroup)
}

func CreateSingleWithNamespace(ctx context.Context, nameServer, namespace, groupName string) (*MQProducer, error) {
	if defaultProducer != nil {
		return nil, ErrProducerExists
	}

	producerOnce.Do(func() {
		config := NewConfigBuilder().
			WithAddr(nameServer).
			WithNamespace(namespace).
			WithGroupName(groupName).
			Build()

		tccRocketMQ := NewTCCRocketMQImpl()

		producer, err := NewMQProducer(ctx, &config, tccRocketMQ)
		if err != nil {
			initError = err
			return
		}

		tccRocketMQ.SetProducer(producer)
		producer.SetTCCRocketMQ(tccRocketMQ)

		if err := producer.Run(); err != nil {
			initError = err
			return
		}

		defaultProducer = producer
	})

	if initError != nil {
		return nil, initError
	}

	if defaultProducer == nil {
		return nil, errors.New("create producer failed")
	}

	return defaultProducer, nil
}

func GetProducer() (*MQProducer, error) {
	if defaultProducer == nil {
		return nil, errors.New("create producer first")
	}

	return defaultProducer, nil
}
