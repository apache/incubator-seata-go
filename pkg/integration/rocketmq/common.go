package rocketmq

import (
	"seata.apache.org/seata-go/pkg/protocol/branch"
	"seata.apache.org/seata-go/pkg/protocol/message"
)

// StatusMaps groups global status mappings
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
		message.GlobalStatusRollbacked:       {},
		message.GlobalStatusRollbacking:      {},
		message.GlobalStatusRollbackRetrying: {},
	},
}

// Property keys
const (
	PropertySeataXID      = "TX_XID"
	PropertySeataBranchID = "TX_BRANCHID"
)

// Context keys
const (
	RocketMsgKey        = "ROCKET_MSG"
	RocketSendResultKey = "ROCKET_SEND_RESULT"
)

// TCC mode keys
const (
	RocketTccName    = "tccRocketMQ"
	RocketBranchType = branch.BranchTypeTCC
)
