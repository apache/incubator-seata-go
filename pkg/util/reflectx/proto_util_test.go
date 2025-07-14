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

package reflectx_test

import (
	"testing"

	"seata.apache.org/seata-go/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/pkg/util/reflectx"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestProtoMessageName(t *testing.T) {
	tests := []struct {
		name          string
		protoName     string
		wantprotoName string
	}{
		{
			name:          "test1",
			protoName:     string(proto.MessageName(&pb.HeartbeatMessageProto{}).Name()),
			wantprotoName: reflectx.ProtoMessageName[*pb.HeartbeatMessageProto](),
		}, {
			name:          "test2",
			protoName:     string(proto.MessageName(&pb.MergedResultMessageProto{}).Name()),
			wantprotoName: reflectx.ProtoMessageName[*pb.MergedResultMessageProto](),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.protoName, tt.wantprotoName)
		})
	}
}
