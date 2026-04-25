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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetProducerFactoryForTest() {
	globalProducer = nil
	producerInitialized = false
	newSeataMQProducer = NewSeataMQProducer
}

func newStubFactoryProducer() (*SeataMQProducer, *stubTransactionProducer, *stubNormalProducer) {
	transactionProducer := &stubTransactionProducer{}
	normalProducer := &stubNormalProducer{}
	return &SeataMQProducer{
		transactionProducer: transactionProducer,
		normalProducer:      normalProducer,
	}, transactionProducer, normalProducer
}

func TestGetSeataMQProducer_RequiresInitialization(t *testing.T) {
	resetProducerFactoryForTest()
	t.Cleanup(resetProducerFactoryForTest)

	producer, err := GetSeataMQProducer()

	require.Error(t, err)
	assert.Nil(t, producer)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestInitSeataMQProducer_AllowsRetryAfterCreateFailure(t *testing.T) {
	resetProducerFactoryForTest()
	t.Cleanup(resetProducerFactoryForTest)

	expectedErr := errors.New("create failed")
	healthyProducer, transactionProducer, normalProducer := newStubFactoryProducer()
	createCalls := 0
	newSeataMQProducer = func(*SeataMQProducerConfig) (*SeataMQProducer, error) {
		createCalls++
		if createCalls == 1 {
			return nil, expectedErr
		}
		return healthyProducer, nil
	}

	err := InitSeataMQProducer(&SeataMQProducerConfig{})
	require.ErrorIs(t, err, expectedErr)

	got, getErr := GetSeataMQProducer()
	require.Error(t, getErr)
	assert.Nil(t, got)

	err = InitSeataMQProducer(&SeataMQProducerConfig{})
	require.NoError(t, err)

	got, err = GetSeataMQProducer()
	require.NoError(t, err)
	assert.Same(t, healthyProducer, got)
	assert.Equal(t, 2, createCalls)
	assert.Equal(t, 1, transactionProducer.startCalls)
	assert.Equal(t, 1, normalProducer.startCalls)
}

func TestInitSeataMQProducer_AllowsRetryAfterStartFailure(t *testing.T) {
	resetProducerFactoryForTest()
	t.Cleanup(resetProducerFactoryForTest)

	expectedErr := errors.New("normal start failed")
	failingProducer, failingTransactionProducer, failingNormalProducer := newStubFactoryProducer()
	failingNormalProducer.startErr = expectedErr
	healthyProducer, healthyTransactionProducer, healthyNormalProducer := newStubFactoryProducer()

	createCalls := 0
	newSeataMQProducer = func(*SeataMQProducerConfig) (*SeataMQProducer, error) {
		createCalls++
		if createCalls == 1 {
			return failingProducer, nil
		}
		return healthyProducer, nil
	}

	err := InitSeataMQProducer(&SeataMQProducerConfig{})
	require.ErrorIs(t, err, expectedErr)
	assert.Equal(t, 1, failingTransactionProducer.startCalls)
	assert.Equal(t, 1, failingNormalProducer.startCalls)
	assert.Equal(t, 1, failingTransactionProducer.shutdownCalls)

	got, getErr := GetSeataMQProducer()
	require.Error(t, getErr)
	assert.Nil(t, got)

	err = InitSeataMQProducer(&SeataMQProducerConfig{})
	require.NoError(t, err)

	got, err = GetSeataMQProducer()
	require.NoError(t, err)
	assert.Same(t, healthyProducer, got)
	assert.Equal(t, 2, createCalls)
	assert.Equal(t, 1, healthyTransactionProducer.startCalls)
	assert.Equal(t, 1, healthyNormalProducer.startCalls)
}

func TestInitSeataMQProducer_DoesNotReinitializeAfterSuccess(t *testing.T) {
	resetProducerFactoryForTest()
	t.Cleanup(resetProducerFactoryForTest)

	healthyProducer, transactionProducer, normalProducer := newStubFactoryProducer()
	createCalls := 0
	newSeataMQProducer = func(*SeataMQProducerConfig) (*SeataMQProducer, error) {
		createCalls++
		return healthyProducer, nil
	}

	require.NoError(t, InitSeataMQProducer(&SeataMQProducerConfig{}))
	require.NoError(t, InitSeataMQProducer(&SeataMQProducerConfig{}))

	got, err := GetSeataMQProducer()
	require.NoError(t, err)
	assert.Same(t, healthyProducer, got)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 1, transactionProducer.startCalls)
	assert.Equal(t, 1, normalProducer.startCalls)
}

func TestShutdownSeataMQProducer_ResetsFactoryState(t *testing.T) {
	resetProducerFactoryForTest()
	t.Cleanup(resetProducerFactoryForTest)

	healthyProducer, transactionProducer, normalProducer := newStubFactoryProducer()
	newSeataMQProducer = func(*SeataMQProducerConfig) (*SeataMQProducer, error) {
		return healthyProducer, nil
	}

	require.NoError(t, InitSeataMQProducer(&SeataMQProducerConfig{}))
	require.NoError(t, ShutdownSeataMQProducer())
	require.NoError(t, ShutdownSeataMQProducer())

	assert.Equal(t, 1, transactionProducer.shutdownCalls)
	assert.Equal(t, 1, normalProducer.shutdownCalls)
	assert.True(t, healthyProducer.closed)
	assert.False(t, producerInitialized)
	assert.Nil(t, globalProducer)

	got, err := GetSeataMQProducer()
	require.Error(t, err)
	assert.Nil(t, got)
}
