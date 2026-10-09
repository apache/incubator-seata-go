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

package message

import (
	"sync"
	"testing"
	"time"
)

func TestMessageFutureCompleteSignalsWaiter(t *testing.T) {
	future := NewMessageFuture(RpcMessage{ID: 1})

	if !future.Complete("ok") {
		t.Fatal("Complete() = false, want true for the first response")
	}
	if future.Response != "ok" {
		t.Fatalf("Response = %v, want \"ok\"", future.Response)
	}
	select {
	case <-future.Done:
	default:
		t.Fatal("Complete() did not signal the waiter")
	}
}

// Done holds a single signal. A duplicate or late response must be dropped
// instead of blocking the transport receive loop.
func TestMessageFutureCompleteDropsDuplicate(t *testing.T) {
	future := NewMessageFuture(RpcMessage{ID: 1})
	future.Complete("first")

	returned := make(chan bool, 1)
	go func() { returned <- future.Complete("second") }()

	select {
	case signaled := <-returned:
		if signaled {
			t.Fatal("Complete() = true for a duplicate response, want false")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Complete() blocked on a duplicate response")
	}
}

// Both transports decode each incoming message in its own goroutine, so two
// responses for the same ID can call Complete concurrently. Run under -race:
// this is what actually proves there is no data race on Response, not just
// that the logical result looks right.
func TestMessageFutureCompleteIsRaceFreeUnderConcurrentCalls(t *testing.T) {
	for i := 0; i < 200; i++ {
		future := NewMessageFuture(RpcMessage{ID: 1})
		var wg sync.WaitGroup
		results := make([]bool, 2)
		wg.Add(2)
		go func() { defer wg.Done(); results[0] = future.Complete("first") }()
		go func() { defer wg.Done(); results[1] = future.Complete("second") }()
		wg.Wait()

		if results[0] == results[1] {
			t.Fatalf("round %d: both Complete calls reported %v, want exactly one true", i, results[0])
		}
		if future.Response != "first" && future.Response != "second" {
			t.Fatalf("round %d: Response = %v, want one of the two calls' payloads", i, future.Response)
		}
	}
}
