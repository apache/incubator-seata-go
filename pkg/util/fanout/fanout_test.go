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

package fanout

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestFanout_Do(t *testing.T) {
	ca := New("cache", WithWorker(1), WithBuffer(1024))
	var run bool
	var mtx sync.Mutex

	ca.Do(context.Background(), func(c context.Context) {
		mtx.Lock()
		run = true
		mtx.Unlock()
		//panic("error")
	})

	time.Sleep(time.Millisecond * 50)
	t.Log("not panic")
	mtx.Lock()
	defer mtx.Unlock()
	if !run {
		t.Fatal("expect run be true")
	}
}

func TestFanout_Close(t *testing.T) {
	ca := New("cache", WithWorker(1), WithBuffer(1024))
	ca.Close()
	err := ca.Do(context.Background(), func(c context.Context) {})
	if err == nil {
		t.Fatal("expect get err")
	}
}

// enqueueContext signals after Do passes its closed check and evaluates the
// caller cancellation case, so shutdown cannot win before Do starts waiting.
type enqueueContext struct {
	context.Context
	waiting chan struct{}
	once    sync.Once
}

func (ctx *enqueueContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.waiting) })
	return ctx.Context.Done()
}

func TestFanoutDoUnblocksOnCancellation(t *testing.T) {
	for _, buffer := range []int{0, 1} {
		for _, shutdown := range []bool{false, true} {
			name := "caller"
			if shutdown {
				name = "fanout"
			}
			t.Run(fmt.Sprintf("%s/buffer_%d", name, buffer), func(t *testing.T) {
				pool := New("cancel-test", WithWorker(1), WithBuffer(buffer))
				running := make(chan struct{})
				release := make(chan struct{})
				if err := pool.Do(context.Background(), func(context.Context) {
					close(running)
					<-release
				}); err != nil {
					t.Fatal(err)
				}
				<-running
				if buffer > 0 {
					if err := pool.Do(context.Background(), func(context.Context) {}); err != nil {
						t.Fatal(err)
					}
				}

				caller, cancel := context.WithCancel(context.Background())
				ctx := &enqueueContext{Context: caller, waiting: make(chan struct{})}
				result := make(chan error, 1)
				closed := make(chan error, 1)
				go func() { result <- pool.Do(ctx, func(context.Context) {}) }()
				<-ctx.waiting

				defer func() {
					cancel()
					close(release)
					if !shutdown {
						go func() { closed <- pool.Close() }()
					}
					select {
					case err := <-closed:
						if err != nil {
							t.Errorf("Close() = %v", err)
						}
					case <-time.After(time.Second):
						t.Error("Close did not finish after releasing the active callback")
					}
				}()

				if shutdown {
					go func() { closed <- pool.Close() }()
					<-pool.ctx.Done()
				} else {
					cancel()
				}
				select {
				case err := <-result:
					if !errors.Is(err, context.Canceled) {
						t.Errorf("Do() = %v, want context.Canceled", err)
					}
				case <-time.After(time.Second):
					t.Error("Do stayed blocked after cancellation with no queue capacity")
				}
			})
		}
	}
}
