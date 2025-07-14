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

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		msg  message.RpcMessage
	}{
		{
			name: "test1",
			msg: message.RpcMessage{
				ID:      1,
				Type:    message.RequestTypeHeartbeatRequest,
				HeadMap: make(map[string]string),
				Body:    &pb.HeartbeatMessageProto{Ping: true},
			},
		}, {
			name: "test2",
			msg: message.RpcMessage{
				ID:      2,
				Type:    message.RequestTypeRequestSync,
				HeadMap: make(map[string]string),
				Body: &pb.RegisterRMRequestProto{
					AbstractIdentifyRequest: &pb.AbstractIdentifyRequestProto{
						AbstractMessage:         &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_REG_RM},
						Version:                 "test",
						ApplicationId:           "1",
						TransactionServiceGroup: "2",
					},
					ResourceIds: "3"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encodeMsg, err := Encode(test.msg)
			assert.NotEmpty(t, encodeMsg)
			assert.Empty(t, err)
			decodeMsg, err := Decode(encodeMsg)
			assert.Empty(t, err)
			assert.Equal(t, decodeMsg.ID, test.msg.ID)
			assert.Equal(t, decodeMsg.Type, test.msg.Type)

			expectedBody := test.msg.Body.(proto.Message)
			actualBody := decodeMsg.Body.(proto.Message)
			assert.True(t, proto.Equal(expectedBody, actualBody), "proto body not equal")
		})
	}
}
