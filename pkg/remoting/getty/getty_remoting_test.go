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

package getty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
)

func TestGettyRemoting_GetMessageFuture(t *testing.T) {
	tests := []struct {
		name          string
		msgID         int32
		messageFuture *message.MessageFuture
	}{
		{
			name:          "futures is null",
			msgID:         1,
			messageFuture: nil,
		},
		{
			name:  "futures not  null",
			msgID: 1,
			messageFuture: &message.MessageFuture{
				ID:       1,
				Err:      nil,
				Response: nil,
				Done:     nil,
			},
		},
	}
	gettyRemotingClient := GetGettyRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.messageFuture != nil {
				gettyRemotingClient.gettyRemoting.futures.Store(test.msgID, test.messageFuture)
				messageFuture := gettyRemotingClient.gettyRemoting.GetMessageFuture(test.msgID)
				assert.Equal(t, *test.messageFuture, *messageFuture)
			} else {
				messageFuture := gettyRemotingClient.gettyRemoting.GetMessageFuture(test.msgID)
				assert.Empty(t, messageFuture)
			}
		})
	}
}

func TestGettyRemoting_RemoveMessageFuture(t *testing.T) {
	tests := []struct {
		name          string
		msgID         int32
		messageFuture *message.MessageFuture
	}{
		{
			name:  "test remove message future",
			msgID: 1,
			messageFuture: &message.MessageFuture{
				ID:       1,
				Err:      nil,
				Response: nil,
				Done:     nil,
			},
		},
	}
	gettyRemotingClient := GetGettyRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gettyRemotingClient.gettyRemoting.futures.Store(test.msgID, test.messageFuture)
			messageFuture := gettyRemotingClient.gettyRemoting.GetMessageFuture(test.msgID)
			assert.Equal(t, messageFuture, test.messageFuture)
			gettyRemotingClient.gettyRemoting.RemoveMessageFuture(test.msgID)
			messageFuture = gettyRemotingClient.gettyRemoting.GetMessageFuture(test.msgID)
			assert.Empty(t, messageFuture)
		})
	}
}

func TestGettyRemoting_GetMergedMessage(t *testing.T) {
	tests := []struct {
		name              string
		msgID             int32
		mergedWarpMessage *message.MergedWarpMessage
	}{
		{
			name:              "mergeMsgMap is null",
			msgID:             1,
			mergedWarpMessage: nil,
		},
		{
			name:  "mergeMsgMap not  null",
			msgID: 1,
			mergedWarpMessage: &message.MergedWarpMessage{
				Msgs:   []message.MessageTypeAware{},
				MsgIds: []int32{1, 2},
			},
		},
	}
	gettyRemotingClient := GetGettyRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mergedWarpMessage != nil {
				gettyRemotingClient.gettyRemoting.mergeMsgMap.Store(test.msgID, test.mergedWarpMessage)
				mergedWarpMessage := gettyRemotingClient.gettyRemoting.GetMergedMessage(test.msgID)
				assert.Equal(t, *test.mergedWarpMessage, *mergedWarpMessage)
			} else {
				mergedWarpMessage := gettyRemotingClient.gettyRemoting.GetMessageFuture(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			}
		})
	}
}

func TestGettyRemoting_RemoveMergedMessageFuture(t *testing.T) {
	tests := []struct {
		name              string
		msgID             int32
		mergedWarpMessage *message.MergedWarpMessage
	}{
		{
			name:              "mergeMsgMap is null",
			msgID:             1,
			mergedWarpMessage: nil,
		},
		{
			name:  "mergeMsgMap not  null",
			msgID: 1,
			mergedWarpMessage: &message.MergedWarpMessage{
				Msgs:   []message.MessageTypeAware{},
				MsgIds: []int32{1, 2},
			},
		},
	}
	gettyRemotingClient := GetGettyRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mergedWarpMessage != nil {
				gettyRemotingClient.gettyRemoting.mergeMsgMap.Store(test.msgID, test.mergedWarpMessage)
				mergedWarpMessage := gettyRemotingClient.gettyRemoting.GetMergedMessage(test.msgID)
				assert.NotEmpty(t, mergedWarpMessage)
				gettyRemotingClient.gettyRemoting.RemoveMergedMessageFuture(test.msgID)
				mergedWarpMessage = gettyRemotingClient.gettyRemoting.GetMergedMessage(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			} else {
				gettyRemotingClient.gettyRemoting.RemoveMergedMessageFuture(test.msgID)
				mergedWarpMessage := gettyRemotingClient.gettyRemoting.GetMergedMessage(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			}
		})
	}
}

func TestGettyRemoting_NotifyRpcMessageResponseSignalsWaitingFuture(t *testing.T) {
	gettyRemoting := newGettyRemoting()
	request := message.RpcMessage{ID: 1}
	messageFuture := message.NewMessageFuture(request)
	gettyRemoting.futures.Store(request.ID, messageFuture)

	gettyRemoting.NotifyRpcMessageResponse(message.RpcMessage{
		ID:   request.ID,
		Body: "ok",
	})

	assert.Equal(t, "ok", messageFuture.Response)
	select {
	case <-messageFuture.Done:
	default:
		t.Fatal("expected NotifyRpcMessageResponse to signal the waiting future")
	}
}

func TestGettyRemoting_NotifyRpcMessageResponseDoesNotBlockWhenAlreadySignaled(t *testing.T) {
	gettyRemoting := newGettyRemoting()
	request := message.RpcMessage{ID: 1}
	messageFuture := message.NewMessageFuture(request)
	messageFuture.Done <- struct{}{}
	gettyRemoting.futures.Store(request.ID, messageFuture)

	done := make(chan struct{})
	go func() {
		gettyRemoting.NotifyRpcMessageResponse(message.RpcMessage{
			ID:   request.ID,
			Body: "late-response",
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("NotifyRpcMessageResponse blocked when the future was already signaled")
	}

	assert.Equal(t, "late-response", messageFuture.Response)
	assert.Len(t, messageFuture.Done, 1)
}
