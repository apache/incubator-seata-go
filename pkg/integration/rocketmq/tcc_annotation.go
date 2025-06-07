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
