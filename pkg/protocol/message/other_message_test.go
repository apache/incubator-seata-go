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

package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMessageFuture(t *testing.T) {
	rpcMessage := RpcMessage{ID: 0}
	assert.Equal(t, int32(0), NewMessageFuture(rpcMessage).ID)
}

func TestHeartBeatMessage_ToString(t *testing.T) {
	assert.Equal(t, "services ping", HeartBeatMessagePing.ToString())
	assert.Equal(t, "services pong", HeartBeatMessagePong.ToString())
}

func TestHeartBeatMessage_GetTypeCode(t *testing.T) {
	assert.Equal(t, MessageType_HeartbeatMsg, HeartBeatMessage{}.GetTypeCode())
}
