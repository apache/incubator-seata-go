package rocketmq

import (
	"context"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/pkg/tm"
)

// TCCRocketMQ defines the interface for TCC transaction operations
type TCCRocketMQ interface {
	// SetProducer sets the MQ producer
	SetProducer(producer *MQProducer)

	// Prepare handles the prepare phase of TCC transaction
	Prepare(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error)

	// Commit handles the commit phase of TCC transaction
	Commit(ctx tm.BusinessActionContext) (bool, error)

	// Rollback handles the rollback phase of TCC transaction
	Rollback(ctx tm.BusinessActionContext) (bool, error)
}
