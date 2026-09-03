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
