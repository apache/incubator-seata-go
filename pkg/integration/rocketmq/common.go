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
