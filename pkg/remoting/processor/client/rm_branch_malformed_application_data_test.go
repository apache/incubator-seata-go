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

	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/rm/tcc"
	"seata.apache.org/seata-go/v2/pkg/tm"
)

type malformedApplicationDataAction struct{}

func (*malformedApplicationDataAction) Prepare(context.Context, interface{}) (bool, error) {
	return true, nil
}

func (*malformedApplicationDataAction) Commit(context.Context, *tm.BusinessActionContext) (bool, error) {
	return true, nil
}

func (*malformedApplicationDataAction) Rollback(context.Context, *tm.BusinessActionContext) (bool, error) {
	return true, nil
}

func (*malformedApplicationDataAction) GetActionName() string { return "processor-malformed-action" }

func registerMalformedApplicationDataResource(t *testing.T) string {
	t.Helper()
	resource, err := tcc.ParseTCCResource(&malformedApplicationDataAction{})
	require.NoError(t, err)
	manager := tcc.GetTCCResourceManagerInstance()
	manager.GetCachedResources().Store(resource.GetResourceId(), resource)
	t.Cleanup(func() { manager.GetCachedResources().Delete(resource.GetResourceId()) })
	rm.GetRmCacheInstance().RegisterResourceManager(manager)
	return resource.GetResourceId()
}

func TestBranchCommitProcessorsReturnMalformedDataFailure(t *testing.T) {
	resourceID := registerMalformedApplicationDataResource(t)
	for _, protocol := range []string{"seata", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: protocol})
			var captured interface{}
			processor := &rmBranchCommitProcessor{
				sendGettyResponse: func(_ int32, response interface{}) error { captured = response; return nil },
				sendGrpcResponse:  func(_ int32, response interface{}) error { captured = response; return nil },
			}
			var rpcMessage message.RpcMessage
			if protocol == "grpc" {
				rpcMessage = message.RpcMessage{ID: 1, Body: &pb.BranchCommitRequestProto{AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
					Xid: "xid-commit", BranchId: 11, BranchType: pb.BranchTypeProto_TCC, ResourceId: resourceID, ApplicationData: `{"actionContext":"bad"}`,
				}}}
			} else {
				rpcMessage = message.RpcMessage{ID: 1, Body: message.BranchCommitRequest{AbstractBranchEndRequest: message.AbstractBranchEndRequest{
					Xid: "xid-commit", BranchId: 11, BranchType: branch.BranchTypeTCC, ResourceId: resourceID, ApplicationData: []byte(`{"actionContext":"bad"}`),
				}}}
			}

			require.NoError(t, processor.Process(context.Background(), rpcMessage))
			if protocol == "grpc" {
				response := captured.(*pb.BranchCommitResponseProto).GetAbstractBranchEndResponse()
				result := response.GetAbstractTransactionResponse().GetAbstractResultMessage()
				require.Equal(t, pb.ResultCodeProto_Failed, result.GetResultCode())
				require.Equal(t, int32(branch.BranchStatusPhasetwoCommitFailedUnretryable), int32(response.GetBranchStatus()))
				require.Equal(t, "xid-commit", response.GetXid())
				require.Equal(t, int64(11), response.GetBranchId())
				require.Contains(t, result.GetMsg(), "actionContext")
			} else {
				response := captured.(message.BranchCommitResponse).AbstractBranchEndResponse
				require.Equal(t, message.ResultCodeFailed, response.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
				require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitFailedUnretryable), response.BranchStatus)
				require.Equal(t, "xid-commit", response.Xid)
				require.Equal(t, int64(11), response.BranchId)
				require.Contains(t, response.AbstractTransactionResponse.AbstractResultMessage.Msg, "actionContext")
			}
		})
	}
}

func TestBranchRollbackProcessorsReturnMalformedDataFailure(t *testing.T) {
	resourceID := registerMalformedApplicationDataResource(t)
	for _, protocol := range []string{"seata", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: protocol})
			var captured interface{}
			processor := &rmBranchRollbackProcessor{
				sendGettyResponse: func(_ int32, response interface{}) error { captured = response; return nil },
				sendGrpcResponse:  func(_ int32, response interface{}) error { captured = response; return nil },
			}
			var rpcMessage message.RpcMessage
			if protocol == "grpc" {
				rpcMessage = message.RpcMessage{ID: 2, Body: &pb.BranchRollbackRequestProto{AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
					Xid: "xid-rollback", BranchId: 12, BranchType: pb.BranchTypeProto_TCC, ResourceId: resourceID, ApplicationData: `{"actionContext":[]}`,
				}}}
			} else {
				rpcMessage = message.RpcMessage{ID: 2, Body: message.BranchRollbackRequest{AbstractBranchEndRequest: message.AbstractBranchEndRequest{
					Xid: "xid-rollback", BranchId: 12, BranchType: branch.BranchTypeTCC, ResourceId: resourceID, ApplicationData: []byte(`{"actionContext":[]}`),
				}}}
			}

			require.NoError(t, processor.Process(context.Background(), rpcMessage))
			if protocol == "grpc" {
				response := captured.(*pb.BranchRollbackResponseProto).GetAbstractBranchEndResponse()
				result := response.GetAbstractTransactionResponse().GetAbstractResultMessage()
				require.Equal(t, pb.ResultCodeProto_Failed, result.GetResultCode())
				require.Equal(t, int32(branch.BranchStatusPhasetwoRollbackFailedUnretryable), int32(response.GetBranchStatus()))
				require.Equal(t, "xid-rollback", response.GetXid())
				require.Equal(t, int64(12), response.GetBranchId())
				require.Contains(t, result.GetMsg(), "actionContext")
			} else {
				response := captured.(message.BranchRollbackResponse).AbstractBranchEndResponse
				require.Equal(t, message.ResultCodeFailed, response.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
				require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoRollbackFailedUnretryable), response.BranchStatus)
				require.Equal(t, "xid-rollback", response.Xid)
				require.Equal(t, int64(12), response.BranchId)
				require.Contains(t, response.AbstractTransactionResponse.AbstractResultMessage.Msg, "actionContext")
			}
		})
	}
}

func TestBranchProcessorsPropagateResponseSendError(t *testing.T) {
	resourceID := registerMalformedApplicationDataResource(t)
	sendErr := errors.New("response send failed")
	tests := []struct {
		name     string
		protocol string
		rollback bool
	}{
		{name: "commit getty", protocol: "seata"},
		{name: "commit grpc", protocol: "grpc"},
		{name: "rollback getty", protocol: "seata", rollback: true},
		{name: "rollback grpc", protocol: "grpc", rollback: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: tc.protocol})
			if tc.rollback {
				processor := &rmBranchRollbackProcessor{
					sendGettyResponse: func(int32, interface{}) error { return sendErr },
					sendGrpcResponse:  func(int32, interface{}) error { return sendErr },
				}
				require.ErrorIs(t, processor.Process(context.Background(), branchRollbackMessage(protocolMessageConfig{protocol: tc.protocol}, resourceID)), sendErr)
				return
			}
			processor := &rmBranchCommitProcessor{
				sendGettyResponse: func(int32, interface{}) error { return sendErr },
				sendGrpcResponse:  func(int32, interface{}) error { return sendErr },
			}
			require.ErrorIs(t, processor.Process(context.Background(), branchCommitMessage(protocolMessageConfig{protocol: tc.protocol}, resourceID)), sendErr)
		})
	}
}

func TestBranchProcessorsRejectUnknownBranchType(t *testing.T) {
	resourceID := registerMalformedApplicationDataResource(t)
	for _, protocol := range []string{"seata", "grpc"} {
		t.Run("commit-"+protocol, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: protocol})
			var captured interface{}
			processor := &rmBranchCommitProcessor{
				sendGettyResponse: func(_ int32, response interface{}) error { captured = response; return nil },
				sendGrpcResponse:  func(_ int32, response interface{}) error { captured = response; return nil },
			}
			request := branchCommitMessage(protocolMessageConfig{protocol: protocol}, resourceID)
			setUnknownBranchType(&request)
			require.NoError(t, processor.Process(context.Background(), request))
			assertCommitFailureResponse(t, captured, protocol, branch.BranchStatusPhasetwoCommitFailedUnretryable)
		})

		t.Run("rollback-"+protocol, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: protocol})
			var captured interface{}
			processor := &rmBranchRollbackProcessor{
				sendGettyResponse: func(_ int32, response interface{}) error { captured = response; return nil },
				sendGrpcResponse:  func(_ int32, response interface{}) error { captured = response; return nil },
			}
			request := branchRollbackMessage(protocolMessageConfig{protocol: protocol}, resourceID)
			setUnknownBranchType(&request)
			require.NoError(t, processor.Process(context.Background(), request))
			assertRollbackFailureResponse(t, captured, protocol, branch.BranchStatusPhasetwoRollbackFailedUnretryable)
		})
	}
}

func TestBranchCommitProcessorsReturnSuccess(t *testing.T) {
	resourceID := registerMalformedApplicationDataResource(t)
	for _, protocol := range []string{"seata", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: protocol})
			var captured interface{}
			processor := &rmBranchCommitProcessor{
				sendGettyResponse: func(_ int32, response interface{}) error { captured = response; return nil },
				sendGrpcResponse:  func(_ int32, response interface{}) error { captured = response; return nil },
			}
			request := branchCommitMessage(protocolMessageConfig{protocol: protocol}, resourceID)
			setValidApplicationData(&request, protocol, true)
			require.NoError(t, processor.Process(context.Background(), request))
			if protocol == "grpc" {
				response := captured.(*pb.BranchCommitResponseProto).GetAbstractBranchEndResponse()
				require.Equal(t, pb.ResultCodeProto_Success, response.GetAbstractTransactionResponse().GetAbstractResultMessage().GetResultCode())
				require.Equal(t, int32(branch.BranchStatusPhasetwoCommitted), int32(response.GetBranchStatus()))
			} else {
				response := captured.(message.BranchCommitResponse).AbstractBranchEndResponse
				require.Equal(t, message.ResultCodeSuccess, response.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
				require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoCommitted), response.BranchStatus)
			}
		})
	}
}

func TestBranchRollbackProcessorsReturnSuccess(t *testing.T) {
	resourceID := registerMalformedApplicationDataResource(t)
	for _, protocol := range []string{"seata", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: protocol})
			var captured interface{}
			processor := &rmBranchRollbackProcessor{
				sendGettyResponse: func(_ int32, response interface{}) error { captured = response; return nil },
				sendGrpcResponse:  func(_ int32, response interface{}) error { captured = response; return nil },
			}
			request := branchRollbackMessage(protocolMessageConfig{protocol: protocol}, resourceID)
			setValidApplicationData(&request, protocol, false)
			require.NoError(t, processor.Process(context.Background(), request))
			if protocol == "grpc" {
				response := captured.(*pb.BranchRollbackResponseProto).GetAbstractBranchEndResponse()
				require.Equal(t, pb.ResultCodeProto_Success, response.GetAbstractTransactionResponse().GetAbstractResultMessage().GetResultCode())
				require.Equal(t, int32(branch.BranchStatusPhasetwoRollbacked), int32(response.GetBranchStatus()))
			} else {
				response := captured.(message.BranchRollbackResponse).AbstractBranchEndResponse
				require.Equal(t, message.ResultCodeSuccess, response.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
				require.Equal(t, branch.BranchStatus(branch.BranchStatusPhasetwoRollbacked), response.BranchStatus)
			}
		})
	}
}

func setValidApplicationData(request *message.RpcMessage, protocol string, commit bool) {
	data := `{"actionContext":{}}`
	if protocol == "grpc" {
		if commit {
			request.Body.(*pb.BranchCommitRequestProto).AbstractBranchEndRequest.ApplicationData = data
		} else {
			request.Body.(*pb.BranchRollbackRequestProto).AbstractBranchEndRequest.ApplicationData = data
		}
		return
	}
	if commit {
		body := request.Body.(message.BranchCommitRequest)
		body.ApplicationData = []byte(data)
		request.Body = body
	} else {
		body := request.Body.(message.BranchRollbackRequest)
		body.ApplicationData = []byte(data)
		request.Body = body
	}
}

func setUnknownBranchType(request *message.RpcMessage) {
	switch body := request.Body.(type) {
	case *pb.BranchCommitRequestProto:
		body.AbstractBranchEndRequest.BranchType = pb.BranchTypeProto(99)
	case *pb.BranchRollbackRequestProto:
		body.AbstractBranchEndRequest.BranchType = pb.BranchTypeProto(99)
	case message.BranchCommitRequest:
		body.BranchType = branch.BranchType(99)
		request.Body = body
	case message.BranchRollbackRequest:
		body.BranchType = branch.BranchType(99)
		request.Body = body
	}
}

func assertCommitFailureResponse(t *testing.T, captured interface{}, protocol string, want branch.BranchStatus) {
	t.Helper()
	if protocol == "grpc" {
		response := captured.(*pb.BranchCommitResponseProto).GetAbstractBranchEndResponse()
		require.Equal(t, pb.ResultCodeProto_Failed, response.GetAbstractTransactionResponse().GetAbstractResultMessage().GetResultCode())
		require.Equal(t, int32(want), int32(response.GetBranchStatus()))
		return
	}
	response := captured.(message.BranchCommitResponse).AbstractBranchEndResponse
	require.Equal(t, message.ResultCodeFailed, response.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
	require.Equal(t, want, response.BranchStatus)
}

func assertRollbackFailureResponse(t *testing.T, captured interface{}, protocol string, want branch.BranchStatus) {
	t.Helper()
	if protocol == "grpc" {
		response := captured.(*pb.BranchRollbackResponseProto).GetAbstractBranchEndResponse()
		require.Equal(t, pb.ResultCodeProto_Failed, response.GetAbstractTransactionResponse().GetAbstractResultMessage().GetResultCode())
		require.Equal(t, int32(want), int32(response.GetBranchStatus()))
		return
	}
	response := captured.(message.BranchRollbackResponse).AbstractBranchEndResponse
	require.Equal(t, message.ResultCodeFailed, response.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
	require.Equal(t, want, response.BranchStatus)
}

type protocolMessageConfig struct{ protocol string }

func branchCommitMessage(cfg protocolMessageConfig, resourceID string) message.RpcMessage {
	if cfg.protocol == "grpc" {
		return message.RpcMessage{ID: 3, Body: &pb.BranchCommitRequestProto{AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
			Xid: "xid-send-error", BranchId: 13, BranchType: pb.BranchTypeProto_TCC, ResourceId: resourceID, ApplicationData: `{"actionContext":"bad"}`,
		}}}
	}
	return message.RpcMessage{ID: 3, Body: message.BranchCommitRequest{AbstractBranchEndRequest: message.AbstractBranchEndRequest{
		Xid: "xid-send-error", BranchId: 13, BranchType: branch.BranchTypeTCC, ResourceId: resourceID, ApplicationData: []byte(`{"actionContext":"bad"}`),
	}}}
}

func branchRollbackMessage(cfg protocolMessageConfig, resourceID string) message.RpcMessage {
	if cfg.protocol == "grpc" {
		return message.RpcMessage{ID: 4, Body: &pb.BranchRollbackRequestProto{AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
			Xid: "xid-send-error", BranchId: 14, BranchType: pb.BranchTypeProto_TCC, ResourceId: resourceID, ApplicationData: `{"actionContext":[]}`,
		}}}
	}
	return message.RpcMessage{ID: 4, Body: message.BranchRollbackRequest{AbstractBranchEndRequest: message.AbstractBranchEndRequest{
		Xid: "xid-send-error", BranchId: 14, BranchType: branch.BranchTypeTCC, ResourceId: resourceID, ApplicationData: []byte(`{"actionContext":[]}`),
	}}}
}
