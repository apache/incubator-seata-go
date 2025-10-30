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

package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/message"
)

func TestGrpcRemoting_GetMessageFuture(t *testing.T) {
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
	grpcRemotingClient := GetGrpcRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.messageFuture != nil {
				grpcRemotingClient.grpcRemoting.futures.Store(test.msgID, test.messageFuture)
				messageFuture := grpcRemotingClient.grpcRemoting.GetMessageFuture(test.msgID)
				assert.Equal(t, *test.messageFuture, *messageFuture)
			} else {
				messageFuture := grpcRemotingClient.grpcRemoting.GetMessageFuture(test.msgID)
				assert.Empty(t, messageFuture)
			}
		})
	}
}

func TestGrpcRemoting_RemoveMessageFuture(t *testing.T) {
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
	grpcRemotingClient := GetGrpcRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			grpcRemotingClient.grpcRemoting.futures.Store(test.msgID, test.messageFuture)
			messageFuture := grpcRemotingClient.grpcRemoting.GetMessageFuture(test.msgID)
			assert.Equal(t, messageFuture, test.messageFuture)
			grpcRemotingClient.grpcRemoting.RemoveMessageFuture(test.msgID)
			messageFuture = grpcRemotingClient.grpcRemoting.GetMessageFuture(test.msgID)
			assert.Empty(t, messageFuture)
		})
	}
}

func TestGrpcRemoting_GetMergedMessage(t *testing.T) {
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
	grpcRemotingClient := GetGrpcRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mergedWarpMessage != nil {
				grpcRemotingClient.grpcRemoting.mergeMsgMap.Store(test.msgID, test.mergedWarpMessage)
				mergedWarpMessage := grpcRemotingClient.grpcRemoting.GetMergedMessage(test.msgID)
				assert.Equal(t, *test.mergedWarpMessage, *mergedWarpMessage)
			} else {
				mergedWarpMessage := grpcRemotingClient.grpcRemoting.GetMessageFuture(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			}
		})
	}
}

func TestGrpcRemoting_RemoveMergedMessageFuture(t *testing.T) {
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
	grpcRemotingClient := GetGrpcRemotingClient()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mergedWarpMessage != nil {
				grpcRemotingClient.grpcRemoting.mergeMsgMap.Store(test.msgID, test.mergedWarpMessage)
				mergedWarpMessage := grpcRemotingClient.grpcRemoting.GetMergedMessage(test.msgID)
				assert.NotEmpty(t, mergedWarpMessage)
				grpcRemotingClient.grpcRemoting.RemoveMergedMessageFuture(test.msgID)
				mergedWarpMessage = grpcRemotingClient.grpcRemoting.GetMergedMessage(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			} else {
				grpcRemotingClient.grpcRemoting.RemoveMergedMessageFuture(test.msgID)
				mergedWarpMessage := grpcRemotingClient.grpcRemoting.GetMergedMessage(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			}
		})
	}
}
