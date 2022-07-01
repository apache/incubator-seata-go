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

package client

import (
	"context"
	"testing"

	"github.com/seata/seata-go/pkg/protocol/codec"
	"github.com/seata/seata-go/pkg/protocol/message"
)

func TestClientHeartBeatProcessor(t *testing.T) {
	// testcases
	var tests = []struct {
		name string // testcase name

		rpcMsg message.RpcMessage // rpcMessage case

		wantErr bool //want testcase err or not
	}{
		{
			name: "chb-testcase1",
			rpcMsg: message.RpcMessage{
				ID:         123,
				Type:       message.GettyRequestType_HeartbeatRequest,
				Codec:      byte(codec.CodecTypeSeata),
				Compressor: byte(1),
				HeadMap: map[string]string{
					"name":    " Jack",
					"age":     "12",
					"address": "Beijing",
				},
				Body: message.HeartBeatMessage{
					Ping: true,
				},
			},

			wantErr: false,
		},
		{
			name: "chb-testcase2",
			rpcMsg: message.RpcMessage{
				ID:         124,
				Type:       message.GettyRequestType_HeartbeatRequest,
				Codec:      byte(codec.CodecTypeSeata),
				Compressor: byte(1),
				HeadMap: map[string]string{
					"name":    " Mike",
					"age":     "20",
					"address": "Hunan",
				},
				Body: message.HeartBeatMessage{
					Ping: false,
				},
			},

			wantErr: false,
		},
	}

	var ctx context.Context
	var chbProcessor clientHeartBeatProcesson
	// run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := chbProcessor.Process(ctx, tc.rpcMsg)
			if (err != nil) != tc.wantErr {
				t.Errorf("clientHeartBeatProcessor wantErr: %v, got: %v", tc.wantErr, err)
				return
			}
		})
	}

}
