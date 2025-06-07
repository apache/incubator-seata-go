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
	producerMutex   sync.RWMutex
)

var (
	ErrProducerExists         = errors.New("producer already exists")
	ErrProducerNotInitialized = errors.New("producer not initialized")
)

// CreateSingle creates a singleton MQ producer with default namespace
func CreateSingle(nameServer, producerGroup string) (*MQProducer, error) {
	return CreateSingleWithNamespace(context.Background(), nameServer, "", producerGroup)
}

// CreateSingleWithNamespace creates a singleton MQ producer with specified namespace
func CreateSingleWithNamespace(ctx context.Context, nameServer, namespace, groupName string) (*MQProducer, error) {
	producerMutex.Lock()
	defer producerMutex.Unlock()

	if defaultProducer != nil {
		return defaultProducer, nil
	}

	producerOnce.Do(func() {
		config := NewConfigBuilder().
			WithAddr(nameServer).
			WithNamespace(namespace).
			WithGroupName(groupName).
			Build()

		producer, err := NewMQProducer(ctx, config)
		if err != nil {
			initError = err
			return
		}

		if err := producer.Start(); err != nil {
			initError = err
			return
		}

		defaultProducer = producer
	})

	if initError != nil {
		return nil, initError
	}

	return defaultProducer, nil
}

// CreateWithConfig creates a singleton MQ producer with custom configuration
func CreateWithConfig(ctx context.Context, config *Config) (*MQProducer, error) {
	producerMutex.Lock()
	defer producerMutex.Unlock()

	if defaultProducer != nil {
		return defaultProducer, nil
	}

	producerOnce.Do(func() {
		producer, err := NewMQProducer(ctx, config)
		if err != nil {
			initError = err
			return
		}

		if err := producer.Start(); err != nil {
			initError = err
			return
		}

		defaultProducer = producer
	})

	if initError != nil {
		return nil, initError
	}

	return defaultProducer, nil
}

// GetProducer returns the singleton MQ producer
func GetProducer() (*MQProducer, error) {
	producerMutex.RLock()
	defer producerMutex.RUnlock()

	if defaultProducer == nil {
		return nil, ErrProducerNotInitialized
	}

	return defaultProducer, nil
}

// GetTCCRocketMQ returns the TCC implementation from the singleton producer
func GetTCCRocketMQ() (*TCCRocketMQImpl, error) {
	producer, err := GetProducer()
	if err != nil {
		return nil, err
	}

	return producer.GetTCCImpl(), nil
}

// Shutdown shuts down the singleton producer
func Shutdown() error {
	producerMutex.Lock()
	defer producerMutex.Unlock()

	if defaultProducer == nil {
		return nil
	}

	err := defaultProducer.Shutdown()
	defaultProducer = nil

	producerOnce = sync.Once{}
	initError = nil

	return err
}
