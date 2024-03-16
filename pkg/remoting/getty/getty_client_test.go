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
	"fmt"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	getty "github.com/apache/dubbo-getty"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/pkg/protocol/codec"
	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/util/log"
)

// TestGettyRemotingClient_SendSyncRequest unit test for SendSyncRequest function
func TestGettyRemotingClient_SendSyncRequest(t *testing.T) {
	respMsg := message.GlobalBeginResponse{
		AbstractTransactionResponse: message.AbstractTransactionResponse{
			AbstractResultMessage: message.AbstractResultMessage{
				ResultCode: message.ResultCodeSuccess,
			},
		},
	}
	gomonkey.ApplyMethod(reflect.TypeOf(GetGettyRemotingInstance()), "SendSync",
		func(_ *GettyRemoting, msg message.RpcMessage, s getty.Session, callback callbackMethod) (interface{},
			error) {
			return respMsg, nil
		})
	resp, err := GetGettyRemotingClient().SendSyncRequest("message")
	assert.Empty(t, err)
	assert.Equal(t, respMsg, resp.(message.GlobalBeginResponse))
}

// TestGettyRemotingClient_SendAsyncResponse unit test for SendAsyncResponse function
func TestGettyRemotingClient_SendAsyncResponse(t *testing.T) {
	gomonkey.ApplyMethod(reflect.TypeOf(GetGettyRemotingInstance()), "SendASync",
		func(_ *GettyRemoting, msg message.RpcMessage, s getty.Session, callback callbackMethod) error {
			return nil
		})
	err := GetGettyRemotingClient().SendAsyncResponse(1, "message")
	assert.Empty(t, err)
}

// TestGettyRemotingClient_SendAsyncRequest unit test for SendAsyncRequest function
func TestGettyRemotingClient_SendAsyncRequest(t *testing.T) {
	tests := []struct {
		name    string
		message interface{}
	}{
		{
			name:    "HeartBeatMessage",
			message: message.HeartBeatMessage{},
		},
		{
			name:    "not HeartBeatMessage",
			message: "message",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gomonkey.ApplyMethod(reflect.TypeOf(GetGettyRemotingInstance()), "SendASync",
				func(_ *GettyRemoting, msg message.RpcMessage, s getty.Session, callback callbackMethod) error {
					return nil
				})
			err := GetGettyRemotingClient().SendAsyncRequest(test.message)
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
				response, err := GetGettyRemotingClient().syncCallback(test.reqMsg, test.respMsg)
				assert.EqualError(t, err, fmt.Sprintf("wait response timeout, request: %#v", test.reqMsg))
				assert.Empty(t, response)
			} else {
				go func() {
					test.respMsg.Done <- struct{}{}
				}()
				response, err := GetGettyRemotingClient().syncCallback(test.reqMsg, test.respMsg)
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
			response, err := GetGettyRemotingClient().asyncCallback(test.reqMsg, test.respMsg)
			assert.Empty(t, err)
			assert.Empty(t, response)
		})
	}
}
