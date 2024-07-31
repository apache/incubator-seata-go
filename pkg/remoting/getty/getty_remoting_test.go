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

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/message"
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.messageFuture != nil {
				GetGettyRemotingInstance().futures.Store(test.msgID, test.messageFuture)
				messageFuture := GetGettyRemotingInstance().GetMessageFuture(test.msgID)
				assert.Equal(t, *test.messageFuture, *messageFuture)
			} else {
				messageFuture := GetGettyRemotingInstance().GetMessageFuture(test.msgID)
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			GetGettyRemotingInstance().futures.Store(test.msgID, test.messageFuture)
			messageFuture := GetGettyRemotingInstance().GetMessageFuture(test.msgID)
			assert.Equal(t, messageFuture, test.messageFuture)
			GetGettyRemotingInstance().RemoveMessageFuture(test.msgID)
			messageFuture = GetGettyRemotingInstance().GetMessageFuture(test.msgID)
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mergedWarpMessage != nil {
				GetGettyRemotingInstance().mergeMsgMap.Store(test.msgID, test.mergedWarpMessage)
				mergedWarpMessage := GetGettyRemotingInstance().GetMergedMessage(test.msgID)
				assert.Equal(t, *test.mergedWarpMessage, *mergedWarpMessage)
			} else {
				mergedWarpMessage := GetGettyRemotingInstance().GetMessageFuture(test.msgID)
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.mergedWarpMessage != nil {
				GetGettyRemotingInstance().mergeMsgMap.Store(test.msgID, test.mergedWarpMessage)
				mergedWarpMessage := GetGettyRemotingInstance().GetMergedMessage(test.msgID)
				assert.NotEmpty(t, mergedWarpMessage)
				GetGettyRemotingInstance().RemoveMergedMessageFuture(test.msgID)
				mergedWarpMessage = GetGettyRemotingInstance().GetMergedMessage(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			} else {
				GetGettyRemotingInstance().RemoveMergedMessageFuture(test.msgID)
				mergedWarpMessage := GetGettyRemotingInstance().GetMergedMessage(test.msgID)
				assert.Empty(t, mergedWarpMessage)
			}
		})
	}
}
