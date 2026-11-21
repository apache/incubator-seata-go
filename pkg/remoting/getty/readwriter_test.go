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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/codec"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
)

func TestRpcPackageHandler(t *testing.T) {
	msg := message.RpcMessage{
		ID:         1123,
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: byte(1),
		HeadMap: map[string]string{
			"name":    " Jack",
			"age":     "12",
			"address": "Beijing",
		},
		Body: message.GlobalBeginRequest{
			Timeout:         2 * time.Second,
			TransactionName: "SeataGoTransaction",
		},
	}

	codec := RpcPackageHandler{}
	bytes, err := codec.Write(nil, msg)
	assert.Nil(t, err)
	msg2, _, _ := codec.Read(nil, bytes)

	assert.Equal(t, msg, msg2)
}

func TestRpcPackageHandler_EmptyHeadMapValue(t *testing.T) {
	msg := message.RpcMessage{
		ID:         1124,
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: byte(1),
		HeadMap: map[string]string{
			"name": "Jack",
			"note": "",
		},
		Body: message.GlobalBeginRequest{
			Timeout:         2 * time.Second,
			TransactionName: "SeataGoTransaction",
		},
	}

	codec := RpcPackageHandler{}
	bytes, err := codec.Write(nil, msg)
	assert.Nil(t, err)
	msg2, _, err := codec.Read(nil, bytes)
	assert.Nil(t, err)

	assert.Equal(t, msg, msg2)
}

func TestRpcPackageHandler_EmptyHeadMapKey(t *testing.T) {
	msg := message.RpcMessage{
		ID:         1125,
		Type:       message.RequestTypeRequestSync,
		Codec:      byte(codec.CodecTypeSeata),
		Compressor: byte(1),
		HeadMap: map[string]string{
			"":     "value",
			"name": "Jack",
		},
		Body: message.GlobalBeginRequest{
			Timeout:         2 * time.Second,
			TransactionName: "SeataGoTransaction",
		},
	}

	codec := RpcPackageHandler{}
	bytes, err := codec.Write(nil, msg)
	assert.Nil(t, err)
	msg2, _, err := codec.Read(nil, bytes)
	assert.Nil(t, err)

	assert.Equal(t, msg, msg2)
}

var benchmarkFrame []byte

func BenchmarkRpcPackageHandlerWrite(b *testing.B) {
	codec.Init()
	handler := RpcPackageHandler{}
	cases := []struct {
		name string
		msg  message.RpcMessage
	}{
		{
			name: "Heartbeat",
			msg: message.RpcMessage{
				Type:  message.RequestTypeHeartbeatRequest,
				Codec: byte(codec.CodecTypeSeata),
				Body:  message.HeartBeatMessagePing,
			},
		},
		{
			name: "GlobalBegin",
			msg: message.RpcMessage{
				Type:  message.RequestTypeRequestSync,
				Codec: byte(codec.CodecTypeSeata),
				Body:  message.GlobalBeginRequest{TransactionName: "checkout", Timeout: 2 * time.Second},
			},
		},
		{
			name: "BranchRegister",
			msg: message.RpcMessage{
				Type:  message.RequestTypeRequestSync,
				Codec: byte(codec.CodecTypeSeata),
				Body: message.BranchRegisterRequest{
					Xid:        "127.0.0.1:8091:1",
					BranchType: branch.BranchTypeAT,
					ResourceId: "inventory",
					LockKey:    "stock:" + strings.Repeat("12345,", 128) + "12346",
				},
			},
		},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			frame, err := handler.Write(nil, tc.msg)
			if err != nil {
				b.Fatal(err)
			}
			if tc.msg.Type != message.RequestTypeHeartbeatRequest && len(frame) <= message.V1HeadLength {
				b.Fatal("request frame contains no message body")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkFrame, err = handler.Write(nil, tc.msg)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
