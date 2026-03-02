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
	producerInitialized bool
)

func InitSeataMQProducer(cfg *SeataMQProducerConfig) error {
	producerMutex.Lock()
	defer producerMutex.Unlock()

	// Allow re-initialization if previous attempt failed
	if producerInitialized && globalProducer != nil {
		return nil
	}

	producer, err := NewSeataMQProducer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create SeataMQProducer: %w", err)
	}

	if err := producer.Start(); err != nil {
		return fmt.Errorf("failed to start SeataMQProducer: %w", err)
	}

	globalProducer = producer
	producerInitialized = true

	return nil
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
