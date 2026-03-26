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
	"errors"
	"testing"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
)

type stubGlobalStatusRequestSender struct {
	response interface{}
	err      error
	lastReq  interface{}
}

func (s *stubGlobalStatusRequestSender) SendSyncRequest(msg interface{}) (interface{}, error) {
	s.lastReq = msg
	if s.err != nil {
		return nil, s.err
	}
	return s.response, nil
}

func TestSeataTransactionListenerExecuteLocalTransaction(t *testing.T) {
	listener := NewSeataTransactionListener(nil)

	msgWithoutXID := primitive.NewMessage("topic-test", []byte("hello"))
	assert.Equal(t, primitive.CommitMessageState, listener.ExecuteLocalTransaction(msgWithoutXID))

	msgWithXID := primitive.NewMessage("topic-test", []byte("hello"))
	msgWithXID.WithProperty(constant.PropertySeataXID, "xid-123")
	assert.Equal(t, primitive.UnknowState, listener.ExecuteLocalTransaction(msgWithXID))
}

func TestSeataTransactionListenerCheckLocalTransaction_MissingXIDRollsBack(t *testing.T) {
	listener := NewSeataTransactionListener(nil)

	state := listener.CheckLocalTransaction(&primitive.MessageExt{})

	assert.Equal(t, primitive.RollbackMessageState, state)
}

func TestSeataTransactionListenerCheckLocalTransaction_StatusMapping(t *testing.T) {
	tests := []struct {
		name         string
		globalStatus message.GlobalStatus
		expected     primitive.LocalTransactionState
	}{
		{name: "committed", globalStatus: message.GlobalStatusCommitted, expected: primitive.CommitMessageState},
		{name: "async committing", globalStatus: message.GlobalStatusAsyncCommitting, expected: primitive.CommitMessageState},
		{name: "commit retrying", globalStatus: message.GlobalStatusCommitRetrying, expected: primitive.UnknowState},
		{name: "rollbacking", globalStatus: message.GlobalStatusRollbacking, expected: primitive.UnknowState},
		{name: "timeout rollback retrying", globalStatus: message.GlobalStatusTimeoutRollbackRetrying, expected: primitive.UnknowState},
		{name: "commit failed", globalStatus: message.GlobalStatusCommitFailed, expected: primitive.RollbackMessageState},
		{name: "timeout rollback failed", globalStatus: message.GlobalStatusTimeoutRollbackFailed, expected: primitive.RollbackMessageState},
		{name: "finished", globalStatus: message.GlobalStatusFinished, expected: primitive.RollbackMessageState},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &stubGlobalStatusRequestSender{
				response: message.GlobalStatusResponse{
					AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
						GlobalStatus: tt.globalStatus,
					},
				},
			}
			listener := &SeataTransactionListener{
				remotingClient: client,
			}

			state := listener.CheckLocalTransaction(newTransactionMessageExt("xid-123", "1001"))

			assert.Equal(t, tt.expected, state)
		})
	}
}

func TestSeataTransactionListenerCheckLocalTransaction_QueryErrorReturnsUnknown(t *testing.T) {
	listener := &SeataTransactionListener{
		remotingClient: &stubGlobalStatusRequestSender{
			err: errors.New("network timeout"),
		},
	}

	state := listener.CheckLocalTransaction(newTransactionMessageExt("xid-123", "1001"))

	assert.Equal(t, primitive.UnknowState, state)
}

func TestSeataTransactionListenerQueryGlobalStatus(t *testing.T) {
	client := &stubGlobalStatusRequestSender{
		response: message.GlobalStatusResponse{
			AbstractGlobalEndResponse: message.AbstractGlobalEndResponse{
				GlobalStatus: message.GlobalStatusRollbacked,
			},
		},
	}
	listener := &SeataTransactionListener{
		remotingClient: client,
	}

	status, err := listener.queryGlobalStatus("xid-123")

	require.NoError(t, err)
	assert.Equal(t, message.GlobalStatusRollbacked, status)
	req, ok := client.lastReq.(message.GlobalStatusRequest)
	require.True(t, ok)
	assert.Equal(t, "xid-123", req.Xid)
}

func TestSeataTransactionListenerQueryGlobalStatus_InvalidResponseType(t *testing.T) {
	listener := &SeataTransactionListener{
		remotingClient: &stubGlobalStatusRequestSender{
			response: "invalid-response",
		},
	}

	status, err := listener.queryGlobalStatus("xid-123")

	require.Error(t, err)
	assert.Equal(t, message.GlobalStatusUnKnown, status)
	assert.Contains(t, err.Error(), "invalid response type")
}

func newTransactionMessageExt(xid, branchID string) *primitive.MessageExt {
	msg := primitive.NewMessage("topic-test", []byte("hello"))
	if xid != "" {
		msg.WithProperty(constant.PropertySeataXID, xid)
	}
	if branchID != "" {
		msg.WithProperty(constant.PropertySeataBranchId, branchID)
	}
	return &primitive.MessageExt{Message: *msg}
}
