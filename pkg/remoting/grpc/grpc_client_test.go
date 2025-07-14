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
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/util/log"
)

// TestGrpcRemotingClient_SendSyncRequest unit test for SendSyncRequest function
func TestGrpcRemotingClient_SendSyncRequest(t *testing.T) {
	respMsg := &pb.GlobalBeginResponseProto{
		AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
			AbstractResultMessage: &pb.AbstractResultMessageProto{
				AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_GLOBAL_BEGIN_RESULT},
				ResultCode:      pb.ResultCodeProto_Success,
			},
		},
	}
	gomonkey.ApplyMethod(reflect.TypeOf(GetGrpcRemotingClient().grpcRemoting), "SendSync",
		func(_ *GrpcRemoting, msg message.RpcMessage, s *Channel, callback callbackMethod) (interface{},
			error) {
			return respMsg, nil
		})
	resp, err := GetGrpcRemotingClient().SendSyncRequest("message")
	assert.Empty(t, err)
	assert.Equal(t, respMsg, resp.(*pb.GlobalBeginResponseProto))
}

// TestGrpcRemotingClient_SendAsyncResponse unit test for SendAsyncResponse function
func TestGrpcRemotingClient_SendAsyncResponse(t *testing.T) {
	gomonkey.ApplyMethod(reflect.TypeOf(GetGrpcRemotingClient().grpcRemoting), "SendAsync",
		func(_ *GrpcRemoting, msg message.RpcMessage, s *Channel, callback callbackMethod) error {
			return nil
		})
	err := GetGrpcRemotingClient().SendAsyncResponse(1, "message")
	assert.Empty(t, err)
}

// TestGrpcRemotingClient_SendAsyncRequest unit test for SendAsyncRequest function
func TestGrpcRemotingClient_SendAsyncRequest(t *testing.T) {
	tests := []struct {
		name    string
		message interface{}
	}{
		{
			name:    "HeartBeatMessage",
			message: &pb.HeartbeatMessageProto{},
		},
		{
			name:    "not HeartBeatMessage",
			message: "message",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gomonkey.ApplyMethod(reflect.TypeOf(GetGrpcRemotingClient().grpcRemoting), "SendAsync",
				func(_ *GrpcRemoting, msg message.RpcMessage, s *Channel, callback callbackMethod) error {
					return nil
				})
			err := GetGrpcRemotingClient().SendAsyncRequest(test.message)
			assert.Empty(t, err)
		})
	}
}

// Test_syncCallback unit test for syncCallback function
func Test_syncCallback(t *testing.T) {
	codec.Init()
	log.Init()
	tests := []struct {
		name    string
		respMsg *message.MessageFuture
		reqMsg  message.RpcMessage
		wantErr bool
	}{
		{
			name: "timeout",
			respMsg: message.NewMessageFuture(message.RpcMessage{
				ID: 1,
			}),
			reqMsg: message.RpcMessage{
				ID: 2,
			},
			wantErr: true,
		},
		{
			name: "Done",
			respMsg: message.NewMessageFuture(message.RpcMessage{
				ID: 1,
			}),
			reqMsg: message.RpcMessage{
				ID: 2,
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.wantErr {
				response, err := GetGrpcRemotingClient().syncCallback(test.reqMsg, test.respMsg)
				assert.EqualError(t, err, fmt.Sprintf("wait response timeout, request: %#v", test.reqMsg))
				assert.Empty(t, response)
			} else {
				go func() {
					test.respMsg.Done <- struct{}{}
				}()
				response, err := GetGrpcRemotingClient().syncCallback(test.reqMsg, test.respMsg)
				assert.Empty(t, err)
				assert.Empty(t, response)
			}
		})
	}
}

// Test_asyncCallback unit test for asyncCallback function
func Test_asyncCallback(t *testing.T) {
	tests := []struct {
		name    string
		respMsg *message.MessageFuture
		reqMsg  message.RpcMessage
		wantErr bool
	}{
		{
			name: "Done",
			respMsg: message.NewMessageFuture(message.RpcMessage{
				ID: 1,
			}),
			reqMsg: message.RpcMessage{
				ID: 2,
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := GetGrpcRemotingClient().asyncCallback(test.reqMsg, test.respMsg)
			assert.Empty(t, err)
			assert.Empty(t, response)
		})
	}
}
