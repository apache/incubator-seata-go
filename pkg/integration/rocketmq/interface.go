package rocketmq

import (
	"context"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/pkg/tm"
)

// TCCRocketMQ defines the interface for TCC transaction operations with RocketMQ
type TCCRocketMQ interface {
	// Prepare handles the prepare phase of TCC transaction
	// This method will be called in the Try phase
	Prepare(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error)

	// Commit handles the commit phase of TCC transaction
	// This method will be called when global transaction commits
	Commit(ctx tm.BusinessActionContext) (bool, error)

	// Rollback handles the rollback phase of TCC transaction
	// This method will be called when global transaction rollbacks
	Rollback(ctx tm.BusinessActionContext) (bool, error)

	// GetActionName handles to get tcc action name
	GetActionName() string
}

// MessageSender defines the interface for sending messages
type MessageSender interface {
	// SendMessage sends a normal message
	SendMessage(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error)

	// SendTransactionMessage sends a transaction message in TCC mode
	SendTransactionMessage(ctx context.Context, msg *primitive.Message) (*primitive.SendResult, error)

	// Shutdown shuts down the message sender
	Shutdown() error
}
