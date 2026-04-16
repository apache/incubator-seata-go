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

package sql

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestAsyncWorker_Lifecycle(t *testing.T) {
	cfg := AsyncWorkerConfig{
		BufferLimit:            10,
		BufferCleanInterval:    time.Second,
		ReceiveChanSize:        10,
		CommitWorkerCount:      1,
		CommitWorkerBufferSize: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Pass nil as sourceManager since we don't process tasks in this test
	_ = NewAsyncWorker(ctx, prometheus.NewRegistry(), cfg, nil)

	// Allow it to run for a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop worker
	cancel()

	// Give it some time to exit (though we can't verify it easily without exposing internal state)
	time.Sleep(50 * time.Millisecond)
}
