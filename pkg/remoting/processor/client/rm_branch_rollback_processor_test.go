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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"seata.apache.org/seata-go/v2/pkg/protocol"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func TestRmBranchRollbackProcessor_SendsFailureResponse(t *testing.T) {
	bizErr := errors.New("rollback failed")
	manager := &testResourceManager{rollbackStatus: branch.BranchStatusPhasetwoRollbackFailedRetryable, rollbackErr: bizErr}

	t.Run("getty", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchRollbackProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGettyBranchRollback(context.Background(), message.RpcMessage{ID: 1, Body: message.BranchRollbackRequest{
			AbstractBranchEndRequest: message.AbstractBranchEndRequest{Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(message.BranchRollbackResponse)
		require.Equal(t, message.ResultCodeFailed, got.ResultCode)
		require.Equal(t, bizErr.Error(), got.Msg)
		require.Equal(t, manager.rollbackStatus, got.BranchStatus)
		require.Equal(t, "xid", got.Xid)
		require.Equal(t, int64(7), got.BranchId)
	})

	t.Run("grpc", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchRollbackProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGrpcBranchRollback(context.Background(), message.RpcMessage{ID: 1, Body: &pb.BranchRollbackRequestProto{
			AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(*pb.BranchRollbackResponseProto)
		result := got.AbstractBranchEndResponse.AbstractTransactionResponse.AbstractResultMessage
		require.Equal(t, pb.ResultCodeProto_Failed, result.ResultCode)
		require.Equal(t, bizErr.Error(), result.Msg)
		require.Equal(t, pb.BranchStatusProto(manager.rollbackStatus), got.AbstractBranchEndResponse.BranchStatus)
		require.Equal(t, "xid", got.AbstractBranchEndResponse.Xid)
		require.Equal(t, int64(7), got.AbstractBranchEndResponse.BranchId)
	})
}

func TestRmBranchRollbackProcessor_ObservesBusinessAndSendErrors(t *testing.T) {
	bizErr := errors.New("rollback failed")
	sendErr := errors.New("send failed")
	manager := &testResourceManager{rollbackStatus: branch.BranchStatusPhasetwoRollbackFailedRetryable, rollbackErr: bizErr}

	t.Run("getty", func(t *testing.T) {
		processor := rmBranchRollbackProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(int32, interface{}) error { return sendErr },
		}
		err := processor.handleGettyBranchRollback(context.Background(), message.RpcMessage{ID: 1, Body: message.BranchRollbackRequest{
			AbstractBranchEndRequest: message.AbstractBranchEndRequest{Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource"},
		}})
		require.ErrorIs(t, err, bizErr)
		require.ErrorIs(t, err, sendErr)
	})

	t.Run("grpc", func(t *testing.T) {
		processor := rmBranchRollbackProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(int32, interface{}) error { return sendErr },
		}
		err := processor.handleGrpcBranchRollback(context.Background(), message.RpcMessage{ID: 1, Body: &pb.BranchRollbackRequestProto{
			AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource"},
		}})
		require.ErrorIs(t, err, bizErr)
		require.ErrorIs(t, err, sendErr)
	})
}

func TestRmBranchRollbackProcessor_SendsSuccessResponse(t *testing.T) {
	manager := &testResourceManager{rollbackStatus: branch.BranchStatusPhasetwoRollbacked}

	t.Run("getty", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchRollbackProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGettyBranchRollback(context.Background(), message.RpcMessage{ID: 1, Body: message.BranchRollbackRequest{
			AbstractBranchEndRequest: message.AbstractBranchEndRequest{Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(message.BranchRollbackResponse)
		require.Equal(t, message.ResultCodeSuccess, got.ResultCode)
		require.Empty(t, got.Msg)
		require.Equal(t, manager.rollbackStatus, got.BranchStatus)
		require.Equal(t, "xid", got.Xid)
		require.Equal(t, int64(7), got.BranchId)
	})

	t.Run("grpc", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchRollbackProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGrpcBranchRollback(context.Background(), message.RpcMessage{ID: 1, Body: &pb.BranchRollbackRequestProto{
			AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(*pb.BranchRollbackResponseProto)
		result := got.AbstractBranchEndResponse.AbstractTransactionResponse.AbstractResultMessage
		require.Equal(t, pb.ResultCodeProto_Success, result.ResultCode)
		require.Empty(t, result.Msg)
		require.Equal(t, pb.BranchStatusProto(manager.rollbackStatus), got.AbstractBranchEndResponse.BranchStatus)
		require.Equal(t, "xid", got.AbstractBranchEndResponse.Xid)
		require.Equal(t, int64(7), got.AbstractBranchEndResponse.BranchId)
	})
}

func TestRmBranchRollbackProcessor_ProcessRoutesByProtocol(t *testing.T) {
	previous := config.GetTransportConfig()
	t.Cleanup(func() { config.InitTransportConfig(previous) })

	manager := &testResourceManager{rollbackStatus: branch.BranchStatusPhasetwoRollbacked}
	tests := []struct {
		name     string
		protocol protocol.Protocol
		body     interface{}
		wantType interface{}
	}{
		{
			name:     "seata",
			protocol: protocol.ProtocolSEATA,
			body: message.BranchRollbackRequest{AbstractBranchEndRequest: message.AbstractBranchEndRequest{
				Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource",
			}},
			wantType: message.BranchRollbackResponse{},
		},
		{
			name:     "grpc",
			protocol: protocol.ProtocolGRPC,
			body: &pb.BranchRollbackRequestProto{AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
				Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource",
			}},
			wantType: (*pb.BranchRollbackResponseProto)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: tt.protocol.String()})
			var sent interface{}
			processor := rmBranchRollbackProcessor{
				getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
				sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
			}

			require.NoError(t, processor.Process(context.Background(), message.RpcMessage{ID: 1, Body: tt.body}))
			require.IsType(t, tt.wantType, sent)
		})
	}
}
