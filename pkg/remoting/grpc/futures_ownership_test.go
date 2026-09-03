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
	"testing"
	"time"

	"go.uber.org/atomic"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/grpc/pb"
)

func newTestClient() *GrpcRemotingClient {
	return &GrpcRemotingClient{idGenerator: &atomic.Uint32{}, grpcRemoting: newGrpcRemoting()}
}

// shortenRequestTimeout keeps the tests off the real 20s wait.
func shortenRequestTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	original := rpcRequestTimeout
	rpcRequestTimeout = d
	t.Cleanup(func() { rpcRequestTimeout = original })
}

func countFutures(g *GrpcRemoting) int {
	n := 0
	g.futures.Range(func(_, _ interface{}) bool { n++; return true })
	return n
}

// openChannel returns a channel whose Send succeeds, by answering the send
// tracker the way sendProcessor would.
func openChannel(t *testing.T) *Channel {
	t.Helper()
	ch := &Channel{sendCh: make(chan *pb.GrpcMessageProto, 1), closeCh: make(chan struct{})}
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-ch.closeCh:
				return
			case req := <-ch.sendCh:
				if tracker, ok := msgSendTrackers.Load(req.Id); ok {
					tracker.(chan error) <- nil
				}
			}
		}
	}()
	t.Cleanup(func() {
		close(ch.closeCh)
		<-stopped
	})
	return ch
}

func commitRequest(id int32) message.RpcMessage {
	return message.RpcMessage{
		ID:      id,
		HeadMap: map[string]string{},
		Body: &pb.GlobalCommitRequestProto{
			AbstractGlobalEndRequest: &pb.AbstractGlobalEndRequestProto{Xid: "xid-1"},
		},
	}
}

// A timed-out request used to delete from mergeMsgMap, leaving its future in
// futures for the lifetime of the process.
func TestSyncCallbackRemovesFutureOnTimeout(t *testing.T) {
	shortenRequestTimeout(t, 50*time.Millisecond)
	client := newTestClient()
	req := message.RpcMessage{ID: 987654321}
	future := message.NewMessageFuture(req)
	client.grpcRemoting.futures.Store(req.ID, future)

	if _, err := client.syncCallback(req, future); err == nil {
		t.Fatal("syncCallback() = nil error, want a timeout")
	}
	if client.GetMessageFuture(req.ID) != nil {
		t.Fatal("the timed-out request is still retained in futures")
	}
}

// On the answered path the on-response processor also removes the future, so
// this is not a leak in production. The waiter still clears it so that cleanup
// has one owner and does not depend on which processor handled the reply.
func TestSyncCallbackRemovesFutureOnResponse(t *testing.T) {
	shortenRequestTimeout(t, 5*time.Second)
	client := newTestClient()
	req := message.RpcMessage{ID: 42}
	future := message.NewMessageFuture(req)
	client.grpcRemoting.futures.Store(req.ID, future)

	client.grpcRemoting.NotifyRpcMessageResponse(message.RpcMessage{ID: req.ID, Body: "ok"})

	res, err := client.syncCallback(req, future)
	if err != nil {
		t.Fatalf("syncCallback() error = %v, want nil", err)
	}
	if res != "ok" {
		t.Fatalf("syncCallback() = %v, want \"ok\"", res)
	}
	if client.GetMessageFuture(req.ID) != nil {
		t.Fatal("the answered request is still retained in futures")
	}
}

func TestSendAsyncRemovesFutureOnEncodeError(t *testing.T) {
	remoting := newGrpcRemoting()
	// A body that is not a proto.Message fails in Encode, before anything is
	// sent. It used to panic there instead, with the future already stored.
	msg := message.RpcMessage{ID: 7, HeadMap: map[string]string{}, Body: "not a proto message"}

	if _, err := remoting.sendAsync(openChannel(t), msg, nil); err == nil {
		t.Fatal("sendAsync() = nil error, want an encode failure")
	}
	if got := countFutures(remoting); got != 0 {
		t.Fatalf("futures holds %d entries after an encode failure, want 0", got)
	}
}

// SendAsyncResponse sends with no callback, so nothing ever waits on the
// future it used to leave behind.
func TestSendAsyncRemovesFutureWhenNoCallbackWaits(t *testing.T) {
	remoting := newGrpcRemoting()

	if _, err := remoting.sendAsync(openChannel(t), commitRequest(11), nil); err != nil {
		t.Fatalf("sendAsync() error = %v, want nil", err)
	}
	if got := countFutures(remoting); got != 0 {
		t.Fatalf("futures holds %d entries after a callback-less send, want 0", got)
	}
}

func TestNotifyRpcMessageResponseSignalsWaitingFuture(t *testing.T) {
	remoting := newGrpcRemoting()
	req := message.RpcMessage{ID: 1}
	future := message.NewMessageFuture(req)
	remoting.futures.Store(req.ID, future)

	remoting.NotifyRpcMessageResponse(message.RpcMessage{ID: req.ID, Body: "ok"})

	if future.Response != "ok" {
		t.Fatalf("Response = %v, want \"ok\"", future.Response)
	}
	select {
	case <-future.Done:
	default:
		t.Fatal("expected NotifyRpcMessageResponse to signal the waiting future")
	}
}

// Done is buffered for one signal, so a duplicate or late response used to
// block the receive loop permanently.
func TestNotifyRpcMessageResponseDoesNotBlockWhenAlreadySignaled(t *testing.T) {
	remoting := newGrpcRemoting()
	req := message.RpcMessage{ID: 1}
	future := message.NewMessageFuture(req)
	future.Done <- struct{}{}
	remoting.futures.Store(req.ID, future)

	returned := make(chan struct{})
	go func() {
		remoting.NotifyRpcMessageResponse(message.RpcMessage{ID: req.ID, Body: "late-response"})
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("NotifyRpcMessageResponse blocked when the future was already signaled")
	}
}

func TestRepeatedTimeoutsDoNotGrowFutures(t *testing.T) {
	shortenRequestTimeout(t, 10*time.Millisecond)
	client := newTestClient()

	const rounds = 200
	for i := 0; i < rounds; i++ {
		id := int32(i + 1)
		req := message.RpcMessage{ID: id}
		future := message.NewMessageFuture(req)
		client.grpcRemoting.futures.Store(id, future)
		if _, err := client.syncCallback(req, future); err == nil {
			t.Fatalf("round %d: syncCallback() = nil error, want a timeout", i)
		}
	}
	if got := countFutures(client.grpcRemoting); got != 0 {
		t.Fatalf("futures holds %d entries after %d timeouts, want 0", got, rounds)
	}
}
