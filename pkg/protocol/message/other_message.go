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

type RpcMessage struct {
	ID         int32
	Type       RequestType
	Codec      byte
	Compressor byte
	HeadMap    map[string]string
	Body       interface{}
}

type MessageFuture struct {
	ID       int32
	Err      error
	Response interface{}
	Done     chan struct{}
}

func NewMessageFuture(message RpcMessage) *MessageFuture {
	return &MessageFuture{
		ID:   message.ID,
		Done: make(chan struct{}),
	}
}

type HeartBeatMessage struct {
	Ping bool
}

var (
	HeartBeatMessagePing = HeartBeatMessage{true}
	HeartBeatMessagePong = HeartBeatMessage{false}
)

func (msg HeartBeatMessage) ToString() string {
	if msg.Ping {
		return "services ping"
	} else {
		return "services pong"
	}
}

func (resp HeartBeatMessage) GetTypeCode() MessageType {
	return MessageTypeHeartbeatMsg
}
