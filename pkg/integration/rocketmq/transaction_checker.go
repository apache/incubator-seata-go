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
	"context"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"
)

type TransactionChecker struct {
	statusChecker GlobalStatusChecker
}

func NewTransactionChecker(statusChecker GlobalStatusChecker) *TransactionChecker {
	return &TransactionChecker{
		statusChecker: statusChecker,
	}
}

func (c *TransactionChecker) CheckLocalTransaction(ctx context.Context, msgExt interface{}, xid string) TransactionState {
	if xid == "" {
		log.Errorf("message has no xid, will rollback")
		return TransactionRollback
	}

	globalStatus, err := c.statusChecker.GetGlobalStatus(ctx, xid)
	if err != nil {
		log.Errorf("get global status failed, xid=%s, err=%v", xid, err)
		return TransactionUnknown
	}

	switch message.GlobalStatus(globalStatus) {
	case message.GlobalStatusCommitted, message.GlobalStatusCommitting, message.GlobalStatusCommitRetrying:
		return TransactionCommit
	case message.GlobalStatusRollbacked, message.GlobalStatusRollbacking, message.GlobalStatusRollbackRetrying,
		message.GlobalStatusTimeoutRollbacked, message.GlobalStatusTimeoutRollbacking, message.GlobalStatusTimeoutRollbackRetrying:
		return TransactionRollback
	case message.GlobalStatusFinished:
		log.Errorf("global transaction finished, will rollback, xid=%s", xid)
		return TransactionRollback
	default:
		return TransactionUnknown
	}
}
