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
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
)

// mockSeataServer implements SeataServiceServer for e2e testing.
// It echoes back a response with the same ID so the client Future can be resolved.
type mockSeataServer struct {
	pb.UnimplementedSeataServiceServer
	received chan *pb.GrpcMessageProto
}

func (s *mockSeataServer) SendRequest(stream pb.SeataService_SendRequestServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		s.received <- msg

		resp := buildResponse(msg)
		_ = stream.Send(resp)
	}
}

func buildResponse(request *pb.GrpcMessageProto) *pb.GrpcMessageProto {
	rpcMessage, err := Decode(request)
	if err != nil {
		return buildGlobalBeginResponse(request.Id)
	}
	if _, ok := rpcMessage.Body.(*pb.RegisterRMRequestProto); ok {
		respMsg := message.RpcMessage{
			ID:   request.Id,
			Type: message.RequestType(pb.MessageTypeProto_TYPE_REG_RM_RESULT),
			Body: &pb.RegisterRMResponseProto{
				AbstractIdentifyResponse: &pb.AbstractIdentifyResponseProto{
					AbstractResultMessage: &pb.AbstractResultMessageProto{
						AbstractMessage: &pb.AbstractMessageProto{
							MessageType: pb.MessageTypeProto_TYPE_REG_RM_RESULT,
						},
						ResultCode: pb.ResultCodeProto_Success,
					},
					Identified: true,
				},
			},
		}
		proto, _ := Encode(respMsg)
		return proto
	}
	return buildGlobalBeginResponse(request.Id)
}

func buildGlobalBeginResponse(id int32) *pb.GrpcMessageProto {
	respMsg := message.RpcMessage{
		ID:   id,
		Type: message.RequestType(pb.MessageTypeProto_TYPE_GLOBAL_BEGIN_RESULT),
		Body: &pb.GlobalBeginResponseProto{
			AbstractTransactionResponse: &pb.AbstractTransactionResponseProto{
				AbstractResultMessage: &pb.AbstractResultMessageProto{
					AbstractMessage: &pb.AbstractMessageProto{
						MessageType: pb.MessageTypeProto_TYPE_GLOBAL_BEGIN_RESULT,
					},
					ResultCode: pb.ResultCodeProto_Success,
				},
			},
		},
	}
	proto, _ := Encode(respMsg)
	return proto
}

// startMockServer starts a real gRPC server and returns its address and the received-messages channel.
func startMockServer(t *testing.T) (addr string, received chan *pb.GrpcMessageProto, stop func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	mock := &mockSeataServer{received: make(chan *pb.GrpcMessageProto, 16)}
	srv := grpc.NewServer()
	pb.RegisterSeataServiceServer(srv, mock)

	go srv.Serve(lis)
	return lis.Addr().String(), mock.received, srv.Stop
}

// newTestChannel dials the mock server and returns a ready Channel.
func newTestChannel(t *testing.T, addr string) *Channel {
	return newTestChannelWithReceiveLoop(t, addr, true)
}

func newTestChannelWithReceiveLoop(t *testing.T, addr string, startReceiveLoop bool) *Channel {
	t.Helper()
	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	require.NoError(t, err)

	client := pb.NewSeataServiceClient(conn)
	stream, err := client.SendRequest(context.Background())
	require.NoError(t, err)

	ch := &Channel{
		addr:       addr,
		conn:       conn,
		client:     client,
		stream:     stream,
		sendCh:     make(chan *pb.GrpcMessageProto, defaultSendChBuffer),
		closeCh:    make(chan struct{}),
		wg:         sync.WaitGroup{},
		mu:         sync.Mutex{},
		regLock:    &sync.Mutex{},
		registered: &atomic.Bool{},
	}
	ch.wg.Add(1)
	if startReceiveLoop {
		ch.wg.Add(1)
	}
	go ch.sendProcessor()
	if startReceiveLoop {
		go GetGrpcClientHandlerInstance().StartReceiveLoop(context.Background(), ch)
	}

	t.Cleanup(func() {
		ch.close()
		conn.Close()
		ch.wg.Wait()
	})
	return ch
}

// TestE2E_HeartbeatSentToServer verifies that a heartbeat message travels over
// the real gRPC stream and is received by the mock server.
func TestE2E_HeartbeatSentToServer(t *testing.T) {
	addr, received, stop := startMockServer(t)
	defer stop()

	ch := newTestChannel(t, addr)

	hbMsg := message.RpcMessage{
		ID:   42,
		Type: message.RequestTypeHeartbeatRequest,
		Body: &pb.HeartbeatMessageProto{Ping: true},
	}
	proto, err := Encode(hbMsg)
	require.NoError(t, err)

	require.NoError(t, ch.Send(proto))

	select {
	case got := <-received:
		assert.Equal(t, int32(42), got.Id)
		assert.Equal(t, int32(message.RequestTypeHeartbeatRequest), got.MessageType)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: mock server did not receive heartbeat")
	}
}

// TestE2E_SyncRequestResponseFuture verifies the full round-trip:
// client sends a GlobalBeginRequest, mock server echoes a GlobalBeginResponse,
// and the client's Future resolves with the correct response.
func TestE2E_SyncRequestResponseFuture(t *testing.T) {
	addr, _, stop := startMockServer(t)
	defer stop()

	// Register typeMap so StartReceiveLoop can route the response to the processor.
	handler := GetGrpcClientHandlerInstance()
	handler.typeMap["GlobalBeginResponseProto"] = message.MessageTypeGlobalBeginResult
	handler.processorMap[message.MessageTypeGlobalBeginResult] = &testResponseProcessor{
		remoting: GetGrpcRemotingClient().grpcRemoting,
	}

	ch := newTestChannel(t, addr)

	reqMsg := message.RpcMessage{
		ID:   99,
		Type: message.RequestType(pb.MessageTypeProto_TYPE_GLOBAL_BEGIN),
		Body: &pb.GlobalBeginRequestProto{
			TransactionName: "test-tx",
			Timeout:         60000,
		},
	}

	future := message.NewMessageFuture(reqMsg)
	GetGrpcRemotingClient().grpcRemoting.futures.Store(reqMsg.ID, future)

	proto, err := Encode(reqMsg)
	require.NoError(t, err)
	require.NoError(t, ch.Send(proto))

	select {
	case <-future.Done:
		resp, ok := future.Response.(*pb.GlobalBeginResponseProto)
		require.True(t, ok)
		assert.Equal(t, pb.ResultCodeProto_Success,
			resp.AbstractTransactionResponse.AbstractResultMessage.ResultCode)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout: Future was not resolved")
	}
}

func TestE2E_SyncRequestUsesExplicitChannel(t *testing.T) {
	handler := GetGrpcClientHandlerInstance()
	handler.typeMap["RegisterRMResponseProto"] = message.MessageTypeRegRmResult
	handler.processorMap[message.MessageTypeRegRmResult] = &testResponseProcessor{
		remoting: GetGrpcRemotingClient().grpcRemoting,
	}

	explicitAddr, explicitReceived, stopExplicit := startMockServer(t)
	defer stopExplicit()
	selectedAddr, selectedReceived, stopSelected := startMockServer(t)
	defer stopSelected()

	explicit := newTestChannel(t, explicitAddr)
	newTestChannel(t, selectedAddr)

	req := &pb.RegisterRMRequestProto{
		AbstractIdentifyRequest: &pb.AbstractIdentifyRequestProto{
			AbstractMessage: &pb.AbstractMessageProto{MessageType: pb.MessageTypeProto_TYPE_REG_RM},
		},
		ResourceIds: "issue-1203",
	}
	client := GetGrpcRemotingClient()
	_, err := client.SendSyncRequestOnChannel(req, explicit)
	require.NoError(t, err)

	select {
	case got := <-explicitReceived:
		decoded, err := Decode(got)
		require.NoError(t, err)
		request, ok := decoded.Body.(*pb.RegisterRMRequestProto)
		require.True(t, ok)
		assert.Equal(t, "issue-1203", request.GetResourceIds())
	case <-time.After(3 * time.Second):
		t.Fatal("explicit channel did not receive the request")
	}
	select {
	case got := <-selectedReceived:
		t.Fatalf("load-balanced channel unexpectedly received request ID %d", got.Id)
	case <-time.After(200 * time.Millisecond):
	}
}

// testResponseProcessor calls NotifyRpcMessageResponse to resolve the Future.
type testResponseProcessor struct {
	remoting *GrpcRemoting
}

func (p *testResponseProcessor) Process(_ context.Context, rpcMessage message.RpcMessage) error {
	p.remoting.NotifyRpcMessageResponse(rpcMessage)
	p.remoting.RemoveMessageFuture(rpcMessage.ID)
	return nil
}
