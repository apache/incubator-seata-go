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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	remotinggrpc "seata.apache.org/seata-go/v2/pkg/remoting/grpc"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func TestGetGrpcRMRemotingInstance(t *testing.T) {
	rm.SetRMRemotingInstance(&GrpcRMRemoting{})
	tests := []struct {
		name     string
		protocol string
		wantType interface{}
	}{
		{
			name:     "GrpcRMRemoting",
			protocol: "grpc",
			wantType: &GrpcRMRemoting{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{
				Protocol: tt.protocol,
			})

			got := rm.GetRMRemotingInstance()
			assert.NotNil(t, got)
			assert.IsType(t, tt.wantType, got)
		})
	}
}

func TestGrpcRMRemotingBranchReportUsesStatusReportMessageType(t *testing.T) {
	var request *pb.BranchReportRequestProto
	patches := gomonkey.ApplyMethod(reflect.TypeOf(remotinggrpc.GetGrpcRemotingClient()), "SendSyncRequest",
		func(_ *remotinggrpc.GrpcRemotingClient, msg interface{}) (interface{}, error) {
			request = msg.(*pb.BranchReportRequestProto)
			return &pb.BranchReportResponseProto{
				AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
					AbstractResultMessage: &pb.AbstractResultMessageProto{
						ResultCode: pb.ResultCodeProto_Success,
					},
				},
			}, nil
		})
	defer patches.Reset()

	err := (&GrpcRMRemoting{}).BranchReport(rm.BranchReportParam{
		BranchType: branch.BranchTypeAT,
		Xid:        "test-xid",
		BranchId:   1,
	})

	assert.NoError(t, err)
	if assert.NotNil(t, request) {
		assert.Equal(t, pb.MessageTypeProto_TYPE_BRANCH_STATUS_REPORT,
			request.GetAbstractTransactionRequest().GetAbstractMessage().GetMessageType())
	}
}
