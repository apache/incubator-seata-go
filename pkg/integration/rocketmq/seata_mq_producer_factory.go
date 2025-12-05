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
	"fmt"
	"sync"

	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/rm"
	"seata.apache.org/seata-go/pkg/rm/tcc"
)

var (
	defaultProducer *SeataMQProducer
	producerLock    sync.Mutex
)

func CreateSeataMQProducer(producer MQProducer) (*SeataMQProducer, error) {
	producerLock.Lock()
	defer producerLock.Unlock()

	if defaultProducer != nil {
		return nil, fmt.Errorf("only one seata mq producer is permitted")
	}

	seataMQProducer := NewSeataMQProducer(producer)

	tccService := &TCCRocketMQService{
		producer: producer,
	}

	tccResource, err := tcc.ParseTCCResource(tccService)
	if err != nil {
		return nil, fmt.Errorf("parse tcc resource failed: %w", err)
	}

	err = rm.GetRmCacheInstance().
		GetResourceManager(branch.BranchTypeTCC).
		RegisterResource(tccResource)
	if err != nil {
		return nil, fmt.Errorf("register tcc resource failed: %w", err)
	}

	seataMQProducer.SetTCCRocketMQ(tccService)
	defaultProducer = seataMQProducer

	return seataMQProducer, nil
}

func GetSeataMQProducer() *SeataMQProducer {
	return defaultProducer
}
