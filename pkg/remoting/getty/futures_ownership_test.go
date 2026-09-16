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
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"go.uber.org/atomic"

	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/mock"
)

func newTestClient() *GettyRemotingClient {
	return &GettyRemotingClient{idGenerator: &atomic.Uint32{}, gettyRemoting: newGettyRemoting()}
}

// shortenRequestTimeout keeps the tests off the real 20s wait.
func shortenRequestTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	original := rpcRequestTimeout
	rpcRequestTimeout = d
	t.Cleanup(func() { rpcRequestTimeout = original })
}

func countFutures(g *GettyRemoting) int {
	n := 0
	g.futures.Range(func(_, _ interface{}) bool { n++; return true })
	return n
}

// A timed-out request used to delete from mergeMsgMap, leaving its future in
// futures for the lifetime of the process.
func TestSyncCallbackRemovesFutureOnTimeout(t *testing.T) {
	shortenRequestTimeout(t, 50*time.Millisecond)
	client := newTestClient()
	req := message.RpcMessage{ID: 987654321}
	future := message.NewMessageFuture(req)
	client.gettyRemoting.futures.Store(req.ID, future)

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
	client.gettyRemoting.futures.Store(req.ID, future)

	client.gettyRemoting.NotifyRpcMessageResponse(message.RpcMessage{ID: req.ID, Body: "ok"})

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

// SendAsyncResponse sends with no callback, so nothing ever waits on the
// future it used to leave behind.
func TestSendAsyncRemovesFutureWhenNoCallbackWaits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	session := mock.NewMockTestSession(ctrl)
	session.EXPECT().IsClosed().Return(false).AnyTimes()
	session.EXPECT().WritePkg(gomock.Any(), gomock.Any()).Return(0, 0, nil).AnyTimes()

	remoting := newGettyRemoting()
	if _, err := remoting.sendAsync(session, message.RpcMessage{ID: 11}, nil); err != nil {
		t.Fatalf("sendAsync() error = %v, want nil", err)
	}
	if got := countFutures(remoting); got != 0 {
		t.Fatalf("futures holds %d entries after a callback-less send, want 0", got)
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
		client.gettyRemoting.futures.Store(id, future)
		if _, err := client.syncCallback(req, future); err == nil {
			t.Fatalf("round %d: syncCallback() = nil error, want a timeout", i)
		}
	}
	if got := countFutures(client.gettyRemoting); got != 0 {
		t.Fatalf("futures holds %d entries after %d timeouts, want 0", got, rounds)
	}
}

// The tests above drive syncCallback directly. This one goes through the real
// entry point so the store and the delete are exercised by the same code path
// production uses.
func TestSendSyncTimeoutLeavesNoFuture(t *testing.T) {
	shortenRequestTimeout(t, 30*time.Millisecond)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	session := mock.NewMockTestSession(ctrl)
	session.EXPECT().IsClosed().Return(false).AnyTimes()
	session.EXPECT().RemoteAddr().Return("127.0.0.1:8091").AnyTimes()
	session.EXPECT().Stat().Return("mock session").AnyTimes()
	session.EXPECT().WritePkg(gomock.Any(), gomock.Any()).Return(0, 0, nil).AnyTimes()

	client := newTestClient()
	const rounds = 50
	for i := 0; i < rounds; i++ {
		msg := message.RpcMessage{ID: int32(i + 1)}
		if _, err := client.gettyRemoting.SendSync(msg, session, client.syncCallback); err == nil {
			t.Fatalf("round %d: SendSync() = nil error, want a timeout", i)
		}
	}
	if got := countFutures(client.gettyRemoting); got != 0 {
		t.Fatalf("futures holds %d entries after %d timed-out SendSync calls, want 0", got, rounds)
	}
}
