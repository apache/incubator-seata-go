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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"seata.apache.org/seata-go/v2/pkg/rm/tcc"

	model2 "seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/codec"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/config"
	remotinggrpc "seata.apache.org/seata-go/v2/pkg/remoting/grpc"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
	"seata.apache.org/seata-go/v2/pkg/rm"
)

func TestRmBranchCommitProcessor(t *testing.T) {
	// testcases
	tests := []struct {
		name     string             // testcase name
		protocol string             // protocol:seata/grpc
		rpcMsg   message.RpcMessage // rpcMessage case
		wantErr  bool               // want testcase err or not
	}{
		{
			name:     "rbc-testcase1-failure",
			protocol: "seata",
			rpcMsg: message.RpcMessage{
				ID:         123,
				Type:       message.RequestType(message.MessageTypeBranchCommit),
				Codec:      byte(codec.CodecTypeSeata),
				Compressor: byte(1),
				HeadMap: map[string]string{
					"name":    " Jack",
					"age":     "12",
					"address": "Beijing",
				},
				Body: message.BranchCommitRequest{
					AbstractBranchEndRequest: message.AbstractBranchEndRequest{
						Xid:             "123344",
						BranchId:        56678,
						BranchType:      model2.BranchTypeTCC,
						ResourceId:      "1232323",
						ApplicationData: []byte("TestExtraData"),
					},
				},
			},

			wantErr: true, // need dail to server, so err accured
		},
		{
			name:     "rbc-testcase2-failure",
			protocol: "grpc",
			rpcMsg: message.RpcMessage{
				ID:   123,
				Type: message.RequestType(message.MessageTypeBranchCommit),
				HeadMap: map[string]string{
					"name":    " Jack",
					"age":     "12",
					"address": "Beijing",
				},
				Body: &pb.BranchCommitRequestProto{
					AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
						Xid:             "123345",
						BranchId:        56679,
						BranchType:      pb.BranchTypeProto_TCC,
						ResourceId:      "1232324",
						ApplicationData: "TestExtraData",
					},
				},
			},

			wantErr: true, // need dail to server, so err accured
		},
	}

	var ctx context.Context
	var rbcProcessor rmBranchCommitProcessor

	// run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config.InitTransportConfig(&config.TransportConfig{Protocol: tc.protocol})

			rm.GetRmCacheInstance().RegisterResourceManager(tcc.GetTCCResourceManagerInstance())

			err := rbcProcessor.Process(ctx, tc.rpcMsg)
			if (err != nil) != tc.wantErr {
				t.Errorf("rmBranchCommitProcessor wantErr: %v, got: %v", tc.wantErr, err)
				return
			}
		})
	}
}

func TestRmBranchCommitProcessorResponseUsesCommitResultMessageType(t *testing.T) {
	resourceManager := &testGrpcResourceManager{branchType: model2.BranchTypeTCC}
	rm.GetRmCacheInstance().RegisterResourceManager(resourceManager)
	defer rm.GetRmCacheInstance().UnregisterResourceManager(model2.BranchTypeTCC)
	config.InitTransportConfig(&config.TransportConfig{Protocol: "grpc"})

	var response *pb.BranchCommitResponseProto
	patches := gomonkey.ApplyMethod(reflect.TypeOf(remotinggrpc.GetGrpcRemotingClient()), "SendAsyncResponse",
		func(_ *remotinggrpc.GrpcRemotingClient, _ int32, msg interface{}) error {
			response = msg.(*pb.BranchCommitResponseProto)
			return nil
		})
	defer patches.Reset()

	err := (&rmBranchCommitProcessor{}).Process(context.Background(), message.RpcMessage{
		ID:   1,
		Type: message.RequestType(message.MessageTypeBranchCommit),
		Body: &pb.BranchCommitRequestProto{AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
			Xid:        "test-xid",
			BranchId:   1,
			BranchType: pb.BranchTypeProto_TCC,
		}},
	})

	assert.NoError(t, err)
	if assert.NotNil(t, response) {
		assert.Equal(t, pb.MessageTypeProto_TYPE_BRANCH_COMMIT_RESULT,
			response.GetAbstractBranchEndResponse().GetAbstractTransactionResponse().GetAbstractResultMessage().GetAbstractMessage().GetMessageType())
	}
}

type testGrpcResourceManager struct {
	rm.ResourceManager
	branchType model2.BranchType
}

func (m *testGrpcResourceManager) GetBranchType() model2.BranchType {
	return m.branchType
}

func (m *testGrpcResourceManager) BranchCommit(context.Context, rm.BranchResource) (model2.BranchStatus, error) {
	return model2.BranchStatusPhaseoneDone, nil
}

func (m *testGrpcResourceManager) BranchRollback(context.Context, rm.BranchResource) (model2.BranchStatus, error) {
	return model2.BranchStatusPhaseoneDone, nil
}

func TestRmBranchCommitProcessorUnknownBranchTypeSendsFailedResponse(t *testing.T) {
	xid := "xid-unknown-commit"
	branchID := int64(1158)

	t.Run("getty", func(t *testing.T) {
		captured := stubAsyncResponses(t)
		config.InitTransportConfig(&config.TransportConfig{Protocol: "seata"})
		err := (&rmBranchCommitProcessor{}).Process(context.Background(), message.RpcMessage{
			ID: 42,
			Body: message.BranchCommitRequest{
				AbstractBranchEndRequest: message.AbstractBranchEndRequest{
					Xid:        xid,
					BranchId:   branchID,
					BranchType: model2.BranchType(99),
				},
			},
		})
		if err != nil {
			t.Fatalf("Process returned error: %v", err)
		}
		if captured.id != 42 {
			t.Fatalf("want msg id 42, got %d", captured.id)
		}
		assertFailedGettyBranchEnd(t, captured.msg, xid, branchID)
	})

	t.Run("grpc", func(t *testing.T) {
		captured := stubAsyncResponses(t)
		config.InitTransportConfig(&config.TransportConfig{Protocol: "grpc"})
		err := (&rmBranchCommitProcessor{}).Process(context.Background(), message.RpcMessage{
			ID: 43,
			Body: &pb.BranchCommitRequestProto{
				AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
					Xid:        xid,
					BranchId:   branchID,
					BranchType: pb.BranchTypeProto(99),
				},
			},
		})
		if err != nil {
			t.Fatalf("Process returned error: %v", err)
		}
		if captured.id != 43 {
			t.Fatalf("want msg id 43, got %d", captured.id)
		}
		assertFailedGrpcBranchEnd(t, captured.msg, xid, branchID)
	})

	t.Run("grpc-truncation", func(t *testing.T) {
		captured := stubAsyncResponses(t)
		config.InitTransportConfig(&config.TransportConfig{Protocol: "grpc"})
		err := (&rmBranchCommitProcessor{}).Process(context.Background(), message.RpcMessage{
			ID: 44,
			Body: &pb.BranchCommitRequestProto{
				AbstractBranchEndRequest: &pb.AbstractBranchEndRequestProto{
					Xid:        xid,
					BranchId:   branchID,
					BranchType: pb.BranchTypeProto(257),
				},
			},
		})
		if err != nil {
			t.Fatalf("Process returned error: %v", err)
		}
		assertFailedGrpcBranchEnd(t, captured.msg, xid, branchID)
		resp := captured.msg.(*pb.BranchCommitResponseProto)
		if resp.AbstractBranchEndResponse.AbstractTransactionResponse.AbstractResultMessage.Msg != "unsupported BranchType: 257" {
			t.Fatalf("want truncation rejected as 257, got %q", resp.AbstractBranchEndResponse.AbstractTransactionResponse.AbstractResultMessage.Msg)
		}
	})
}

func TestParseKnownBranchTypeRejectsTruncatingValues(t *testing.T) {
	if _, ok := parseKnownBranchType(257); ok {
		t.Fatal("257 must not be accepted; int8 conversion would truncate it to TCC")
	}
	if bt, ok := parseKnownBranchType(int32(model2.BranchTypeTCC)); !ok || bt != model2.BranchTypeTCC {
		t.Fatalf("TCC should be accepted, got %v ok=%v", bt, ok)
	}
	if bt, ok := parseKnownBranchType(int32(model2.BranchTypeXA)); !ok || bt != model2.BranchTypeXA {
		t.Fatalf("XA should be accepted, got %v ok=%v", bt, ok)
	}
}

func stubAsyncResponses(t *testing.T) *capturedBranchResponse {
	t.Helper()
	origGetty := sendGettyAsyncResponse
	origGrpc := sendGrpcAsyncResponse
	captured := &capturedBranchResponse{}
	sendGettyAsyncResponse = func(msgID int32, msg interface{}) error {
		captured.id = msgID
		captured.msg = msg
		return nil
	}
	sendGrpcAsyncResponse = func(msgID int32, msg interface{}) error {
		captured.id = msgID
		captured.msg = msg
		return nil
	}
	t.Cleanup(func() {
		sendGettyAsyncResponse = origGetty
		sendGrpcAsyncResponse = origGrpc
	})
	return captured
}

type capturedBranchResponse struct {
	id  int32
	msg interface{}
}

func assertFailedGettyBranchEnd(t *testing.T, sent interface{}, xid string, branchID int64) {
	t.Helper()
	switch resp := sent.(type) {
	case message.BranchCommitResponse:
		if resp.ResultCode != message.ResultCodeFailed {
			t.Fatalf("want ResultCodeFailed, got %v", resp.ResultCode)
		}
		if resp.Xid != xid || resp.BranchId != branchID {
			t.Fatalf("want xid=%s branchId=%d, got xid=%s branchId=%d", xid, branchID, resp.Xid, resp.BranchId)
		}
	case message.BranchRollbackResponse:
		if resp.ResultCode != message.ResultCodeFailed {
			t.Fatalf("want ResultCodeFailed, got %v", resp.ResultCode)
		}
		if resp.Xid != xid || resp.BranchId != branchID {
			t.Fatalf("want xid=%s branchId=%d, got xid=%s branchId=%d", xid, branchID, resp.Xid, resp.BranchId)
		}
	default:
		t.Fatalf("unexpected getty response type %T", sent)
	}
}

func assertFailedGrpcBranchEnd(t *testing.T, sent interface{}, xid string, branchID int64) {
	t.Helper()
	var end *pb.AbstractBranchEndResponseProto
	switch resp := sent.(type) {
	case *pb.BranchCommitResponseProto:
		end = resp.AbstractBranchEndResponse
	case *pb.BranchRollbackResponseProto:
		end = resp.AbstractBranchEndResponse
	default:
		t.Fatalf("unexpected grpc response type %T", sent)
	}
	if end.AbstractTransactionResponse.AbstractResultMessage.ResultCode != pb.ResultCodeProto_Failed {
		t.Fatalf("want grpc Failed, got %v", end.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
	}
	if end.Xid != xid || end.BranchId != branchID {
		t.Fatalf("want xid=%s branchId=%d, got xid=%s branchId=%d", xid, branchID, end.Xid, end.BranchId)
	}
}
