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
	"github.com/apache/rocketmq-client-go/v2/primitive"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

type SeataTransactionListener struct {
	producer *SeataMQProducer
}

func NewSeataTransactionListener(producer *SeataMQProducer) *SeataTransactionListener {
	return &SeataTransactionListener{producer: producer}
}

func (l *SeataTransactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	xid := msg.GetProperty(constant.PropertySeataXID)
	if xid == "" {
		return primitive.CommitMessageState
	}
	log.Debugf("[SeataTransactionListener] ExecuteLocalTransaction, xid=%s, returning UnknownState", xid)
	return primitive.UnknowState
}

func (l *SeataTransactionListener) CheckLocalTransaction(msgExt *primitive.MessageExt) primitive.LocalTransactionState {
	xid := msgExt.GetProperty(constant.PropertySeataXID)
	if xid == "" {
		log.Warnf("[SeataTransactionListener] CheckLocalTransaction: missing XID, rollback")
		return primitive.RollbackMessageState
	}

	branchIdStr := msgExt.GetProperty(constant.PropertySeataBranchId)
	log.Infof("[SeataTransactionListener] CheckLocalTransaction, xid=%s, branchId=%s", xid, branchIdStr)

	globalStatus, err := l.queryGlobalStatus(xid)
	if err != nil {
		log.Errorf("[SeataTransactionListener] Query global status failed, xid=%s, err=%v", xid, err)
		return primitive.UnknowState
	}

	switch globalStatus {
	case message.GlobalStatusCommitted:
		log.Infof("[SeataTransactionListener] Global tx committed, xid=%s", xid)
		return primitive.CommitMessageState
	case message.GlobalStatusRollbacked, message.GlobalStatusTimeoutRollbacked, message.GlobalStatusRollbackFailed:
		log.Infof("[SeataTransactionListener] Global tx rollbacked, xid=%s, status=%v", xid, globalStatus)
		return primitive.RollbackMessageState
	default:
		log.Infof("[SeataTransactionListener] Global tx in progress, xid=%s, status=%v", xid, globalStatus)
		return primitive.UnknowState
	}
}

func (l *SeataTransactionListener) queryGlobalStatus(xid string) (message.GlobalStatus, error) {
	req := message.GlobalStatusRequest{
		AbstractGlobalEndRequest: message.AbstractGlobalEndRequest{
			Xid: xid,
		},
	}
	res, err := getty.GetGettyRemotingClient().SendSyncRequest(req)
	if err != nil {
		return message.GlobalStatusUnKnown, err
	}
	return res.(message.GlobalStatusResponse).GlobalStatus, nil
}
