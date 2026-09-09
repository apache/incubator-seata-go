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
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"seata.apache.org/seata-go/v2/pkg/protocol"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

type testResourceManager struct {
	branchType     branch.BranchType
	commitStatus   branch.BranchStatus
	commitErr      error
	rollbackStatus branch.BranchStatus
	rollbackErr    error
}

func (m *testResourceManager) BranchCommit(context.Context, rm.BranchResource) (branch.BranchStatus, error) {
	return m.commitStatus, m.commitErr
}
func (m *testResourceManager) BranchRollback(context.Context, rm.BranchResource) (branch.BranchStatus, error) {
	return m.rollbackStatus, m.rollbackErr
}
func (*testResourceManager) BranchRegister(context.Context, rm.BranchRegisterParam) (int64, error) {
	return 0, nil
}
func (*testResourceManager) BranchReport(context.Context, rm.BranchReportParam) error { return nil }
func (*testResourceManager) LockQuery(context.Context, rm.LockQueryParam) (bool, error) {
	return false, nil
}
func (*testResourceManager) RegisterResource(rm.Resource) error   { return nil }
func (*testResourceManager) UnregisterResource(rm.Resource) error { return nil }
func (*testResourceManager) GetCachedResources() *sync.Map        { return &sync.Map{} }
func (m *testResourceManager) GetBranchType() branch.BranchType   { return m.branchType }

func TestBranchEndSendResponse(t *testing.T) {
	injectedCalled := false
	fallbackCalled := false
	injected := func(int32, interface{}) error {
		injectedCalled = true
		return nil
	}
	fallback := func(int32, interface{}) error {
		fallbackCalled = true
		return nil
	}

	require.NoError(t, branchEndSendResponse(injected, fallback)(1, "response"))
	require.True(t, injectedCalled)
	require.False(t, fallbackCalled)

	injectedCalled = false
	require.NoError(t, branchEndSendResponse(nil, fallback)(2, "response"))
	require.False(t, injectedCalled)
	require.True(t, fallbackCalled)
}

func TestBranchEndResult(t *testing.T) {
	bizErr := errors.New("operation failed")
	failed := newBranchEndResult(branch.BranchStatusPhasetwoCommitFailedRetryable, bizErr)
	require.Equal(t, message.ResultCodeFailed, failed.resultCode)
	require.Equal(t, pb.ResultCodeProto_Failed, branchEndResultCodeProto(failed.resultCode))
	require.Equal(t, bizErr.Error(), failed.errMsg)

	success := newBranchEndResult(branch.BranchStatusPhasetwoCommitted, nil)
	require.Equal(t, message.ResultCodeSuccess, success.resultCode)
	require.Equal(t, pb.ResultCodeProto_Success, branchEndResultCodeProto(success.resultCode))
	require.Empty(t, success.errMsg)

	sendErr := errors.New("send failed")
	require.NoError(t, branchEndProcessError(bizErr, nil))
	require.ErrorIs(t, branchEndProcessError(nil, sendErr), sendErr)
	combined := branchEndProcessError(bizErr, sendErr)
	require.ErrorIs(t, combined, bizErr)
	require.ErrorIs(t, combined, sendErr)

	injectedManager := &testResourceManager{}
	require.Same(t, injectedManager, branchEndResourceManager(
		func(branch.BranchType) rm.ResourceManager { return injectedManager },
		branch.BranchTypeTCC,
	))

	fallbackType := branch.BranchType(99)
	fallbackManager := &testResourceManager{branchType: fallbackType}
	rmCache := rm.GetRmCacheInstance()
	rmCache.RegisterResourceManager(fallbackManager)
	t.Cleanup(func() { rmCache.UnregisterResourceManager(fallbackType) })
	require.Same(t, fallbackManager, branchEndResourceManager(nil, fallbackType))
}

func TestRmBranchCommitProcessor_SendsFailureResponse(t *testing.T) {
	bizErr := errors.New("commit failed")
	manager := &testResourceManager{commitStatus: branch.BranchStatusPhasetwoCommitFailedRetryable, commitErr: bizErr}

	t.Run("getty", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchCommitProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGettyBranchCommit(context.Background(), message.RpcMessage{ID: 1, Body: message.BranchCommitRequest{
			AbstractBranchEndRequest: message.AbstractBranchEndRequest{Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(message.BranchCommitResponse)
		require.Equal(t, message.ResultCodeFailed, got.ResultCode)
		require.Equal(t, bizErr.Error(), got.Msg)
		require.Equal(t, manager.commitStatus, got.BranchStatus)
		require.Equal(t, "xid", got.Xid)
		require.Equal(t, int64(7), got.BranchId)
	})

	t.Run("grpc", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchCommitProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGrpcBranchCommit(context.Background(), message.RpcMessage{ID: 1, Body: &pb.BranchCommitRequestProto{
			AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(*pb.BranchCommitResponseProto)
		result := got.AbstractBranchEndResponse.AbstractTransactionResponse.AbstractResultMessage
		require.Equal(t, pb.ResultCodeProto_Failed, result.ResultCode)
		require.Equal(t, bizErr.Error(), result.Msg)
		require.Equal(t, pb.BranchStatusProto(manager.commitStatus), got.AbstractBranchEndResponse.BranchStatus)
		require.Equal(t, "xid", got.AbstractBranchEndResponse.Xid)
		require.Equal(t, int64(7), got.AbstractBranchEndResponse.BranchId)
	})
}

func TestRmBranchCommitProcessor_ObservesBusinessAndSendErrors(t *testing.T) {
	bizErr := errors.New("commit failed")
	sendErr := errors.New("send failed")
	manager := &testResourceManager{commitStatus: branch.BranchStatusPhasetwoCommitFailedRetryable, commitErr: bizErr}

	t.Run("getty", func(t *testing.T) {
		processor := rmBranchCommitProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(int32, interface{}) error { return sendErr },
		}
		err := processor.handleGettyBranchCommit(context.Background(), message.RpcMessage{ID: 1, Body: message.BranchCommitRequest{
			AbstractBranchEndRequest: message.AbstractBranchEndRequest{Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource"},
		}})
		require.ErrorIs(t, err, bizErr)
		require.ErrorIs(t, err, sendErr)
	})

	t.Run("grpc", func(t *testing.T) {
		processor := rmBranchCommitProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(int32, interface{}) error { return sendErr },
		}
		err := processor.handleGrpcBranchCommit(context.Background(), message.RpcMessage{ID: 1, Body: &pb.BranchCommitRequestProto{
			AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource"},
		}})
		require.ErrorIs(t, err, bizErr)
		require.ErrorIs(t, err, sendErr)
	})
}

func TestRmBranchCommitProcessor_SendsSuccessResponse(t *testing.T) {
	manager := &testResourceManager{commitStatus: branch.BranchStatusPhasetwoCommitted}

	t.Run("getty", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchCommitProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGettyBranchCommit(context.Background(), message.RpcMessage{ID: 1, Body: message.BranchCommitRequest{
			AbstractBranchEndRequest: message.AbstractBranchEndRequest{Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(message.BranchCommitResponse)
		require.Equal(t, message.ResultCodeSuccess, got.ResultCode)
		require.Empty(t, got.Msg)
		require.Equal(t, manager.commitStatus, got.BranchStatus)
		require.Equal(t, "xid", got.Xid)
		require.Equal(t, int64(7), got.BranchId)
	})

	t.Run("grpc", func(t *testing.T) {
		var sent interface{}
		processor := rmBranchCommitProcessor{
			getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
			sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
		}
		err := processor.handleGrpcBranchCommit(context.Background(), message.RpcMessage{ID: 1, Body: &pb.BranchCommitRequestProto{
			AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource"},
		}})
		require.NoError(t, err)
		got := sent.(*pb.BranchCommitResponseProto)
		result := got.AbstractBranchEndResponse.AbstractTransactionResponse.AbstractResultMessage
		require.Equal(t, pb.ResultCodeProto_Success, result.ResultCode)
		require.Empty(t, result.Msg)
		require.Equal(t, pb.BranchStatusProto(manager.commitStatus), got.AbstractBranchEndResponse.BranchStatus)
		require.Equal(t, "xid", got.AbstractBranchEndResponse.Xid)
		require.Equal(t, int64(7), got.AbstractBranchEndResponse.BranchId)
	})
}

func TestRmBranchCommitProcessor_ProcessRoutesByProtocol(t *testing.T) {
	previous := config.GetTransportConfig()
	t.Cleanup(func() { config.InitTransportConfig(previous) })

	manager := &testResourceManager{commitStatus: branch.BranchStatusPhasetwoCommitted}
	tests := []struct {
		name     string
		protocol protocol.Protocol
		body     interface{}
		wantType interface{}
	}{
		{
			name:     "seata",
			protocol: protocol.ProtocolSEATA,
			body: message.BranchCommitRequest{AbstractBranchEndRequest: message.AbstractBranchEndRequest{
				Xid: "xid", BranchId: 7, BranchType: branch.BranchTypeTCC, ResourceId: "resource",
			}},
			wantType: message.BranchCommitResponse{},
		},
		{
			name:     "grpc",
			protocol: protocol.ProtocolGRPC,
			body: &pb.BranchCommitRequestProto{AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
				Xid: "xid", BranchId: 7, BranchType: pb.BranchTypeProto_TCC, ResourceId: "resource",
			}},
			wantType: (*pb.BranchCommitResponseProto)(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: tt.protocol.String()})
			var sent interface{}
			processor := rmBranchCommitProcessor{
				getResourceManager: func(branch.BranchType) rm.ResourceManager { return manager },
				sendResponse:       func(_ int32, response interface{}) error { sent = response; return nil },
			}

			require.NoError(t, processor.Process(context.Background(), message.RpcMessage{ID: 1, Body: tt.body}))
			require.IsType(t, tt.wantType, sent)
		})
	}
}
