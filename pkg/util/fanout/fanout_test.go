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
