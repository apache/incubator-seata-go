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
