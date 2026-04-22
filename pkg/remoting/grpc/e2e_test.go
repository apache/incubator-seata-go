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

		// Echo back a GlobalBeginResponse with the same ID so the client Future resolves.
		resp := buildGlobalBeginResponse(msg.Id)
		_ = stream.Send(resp)
	}
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
	t.Helper()
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(),
		grpc.WithTimeout(3*time.Second))
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
	ch.wg.Add(2)
	go ch.sendProcessor()
	go GetGrpcClientHandlerInstance().StartReceiveLoop(context.Background(), ch)

	t.Cleanup(func() { ch.close(); conn.Close() })
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

// testResponseProcessor calls NotifyRpcMessageResponse to resolve the Future.
type testResponseProcessor struct {
	remoting *GrpcRemoting
}

func (p *testResponseProcessor) Process(_ context.Context, rpcMessage message.RpcMessage) error {
	p.remoting.NotifyRpcMessageResponse(rpcMessage)
	p.remoting.RemoveMessageFuture(rpcMessage.ID)
	return nil
}
