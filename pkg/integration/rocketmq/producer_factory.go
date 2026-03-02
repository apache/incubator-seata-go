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
)

var (
	globalProducer      *SeataMQProducer
	producerMutex       sync.RWMutex
	producerInitOnce    sync.Once
	producerInitialized bool
)

func InitSeataMQProducer(cfg *SeataMQProducerConfig) error {
	var initErr error
	producerInitOnce.Do(func() {
		producerMutex.Lock()
		defer producerMutex.Unlock()

		producer, err := NewSeataMQProducer(cfg)
		if err != nil {
			initErr = fmt.Errorf("failed to create SeataMQProducer: %w", err)
			return
		}

		globalProducer = producer
		producerInitialized = true

		if err := producer.Start(); err != nil {
			initErr = fmt.Errorf("failed to start SeataMQProducer: %w", err)
			globalProducer = nil
			producerInitialized = false
			return
		}
	})

	return initErr
}

func GetSeataMQProducer() (*SeataMQProducer, error) {
	producerMutex.RLock()
	defer producerMutex.RUnlock()

	if !producerInitialized || globalProducer == nil {
		return nil, fmt.Errorf("SeataMQProducer not initialized, call InitSeataMQProducer first")
	}

	return globalProducer, nil
}

func ShutdownSeataMQProducer() error {
	producerMutex.Lock()
	defer producerMutex.Unlock()

	if !producerInitialized || globalProducer == nil {
		return nil
	}

	err := globalProducer.Shutdown()
	globalProducer = nil
	producerInitialized = false

	return err
}
