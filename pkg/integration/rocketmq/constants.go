package rocketmq

import (
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
)

type StatusMaps struct {
	Commit   map[message.GlobalStatus]struct{}
	Rollback map[message.GlobalStatus]struct{}
}

var GlobalStatusMaps = StatusMaps{
	Commit: map[message.GlobalStatus]struct{}{
		message.GlobalStatusCommitted:      {},
		message.GlobalStatusCommitting:     {},
		message.GlobalStatusCommitRetrying: {},
	},
	Rollback: map[message.GlobalStatus]struct{}{
		message.GlobalStatusRollbacked:              {},
		message.GlobalStatusRollbacking:             {},
		message.GlobalStatusRollbackRetrying:        {},
		message.GlobalStatusTimeoutRollbacked:       {},
		message.GlobalStatusTimeoutRollbacking:      {},
		message.GlobalStatusTimeoutRollbackRetrying: {},
		message.GlobalStatusTimeoutRollbackFailed:   {},
	},
}

const (
	PropertySeataXID      = "TX_XID"
	PropertySeataBranchID = "TX_BRANCHID"
	PropertySeataAction   = "TX_ACTION"
)

const (
	RocketMsgKey        = "ROCKET_MSG"
	RocketSendResultKey = "ROCKET_SEND_RESULT"
	RocketTopicKey      = "ROCKET_TOPIC"
	RocketTagsKey       = "ROCKET_TAGS"
)

const (
	RocketTccResourceName = "tccRocketMQ"
	RocketBranchType      = branch.BranchTypeTCC
	RocketTccActionName   = "RocketTccAction"

	TccPrepareMethod  = "Prepare"
	TccCommitMethod   = "Commit"
	TccRollbackMethod = "Rollback"
)

const (
	ErrMsgProducerNotInitialized  = "MQ producer not initialized"
	ErrMsgBusinessContextNotFound = "business action context not found"
	ErrMsgInvalidMessageType      = "invalid message type in context"
	ErrMsgInvalidResultType       = "invalid result type in context"
	ErrMsgInvalidMessage          = "message is nil"
	ErrMsgInvalidResult           = "send result is nil"
	ErrMsgInvalidMessageID        = "invalid message ID"
)
