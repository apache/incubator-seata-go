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

package tm

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type stubGlobalTransactionManager struct{}

func (s *stubGlobalTransactionManager) Begin(ctx context.Context, timeout time.Duration) error {
	return nil
}

func (s *stubGlobalTransactionManager) Commit(ctx context.Context, gtr *GlobalTransaction) error {
	return nil
}

func (s *stubGlobalTransactionManager) Rollback(ctx context.Context, gtr *GlobalTransaction) error {
	return nil
}

func (s *stubGlobalTransactionManager) GlobalReport(ctx context.Context, gtr *GlobalTransaction) (interface{}, error) {
	return nil, nil
}

func resetGlobalTransactionManagerForTest(t *testing.T) {
	t.Helper()

	originalManager := globalTransactionManager
	originalOnce := onceGlobalTransactionManager

	globalTransactionManager = nil
	onceGlobalTransactionManager = &sync.Once{}

	t.Cleanup(func() {
		globalTransactionManager = originalManager
		onceGlobalTransactionManager = originalOnce
	})
}

func TestSetGlobalTransactionManager(t *testing.T) {
	resetGlobalTransactionManagerForTest(t)

	first := &stubGlobalTransactionManager{}
	second := &stubGlobalTransactionManager{}

	SetGlobalTransactionManager(first)
	SetGlobalTransactionManager(second)

	assert.Same(t, first, GetGlobalTransactionManager())
}

func TestIsTimeout(t *testing.T) {
	t.Run("missing time info", func(t *testing.T) {
		assert.False(t, IsTimeout(context.Background()))

		ctx := InitSeataContext(context.Background())
		assert.False(t, IsTimeout(ctx))
	})

	t.Run("expired time info", func(t *testing.T) {
		ctx := InitSeataContext(context.Background())
		SetTimeInfo(ctx, TimeInfo{
			createTime: now().Add(-3 * time.Second),
			timeout:    time.Second,
		})

		assert.True(t, IsTimeout(ctx))
	})

	t.Run("active time info", func(t *testing.T) {
		ctx := InitSeataContext(context.Background())
		SetTimeInfo(ctx, TimeInfo{
			createTime: now(),
			timeout:    5 * time.Second,
		})

		assert.False(t, IsTimeout(ctx))
	})
}

func TestIsTimeoutWithFakeClock(t *testing.T) {
	restoreNow := now
	t.Cleanup(func() { now = restoreNow })

	// 700ms into a Unix second: the old second-precision path would already count
	// this as expired for any timeout below 1s.
	start := time.Unix(1_700_000_000, 700_000_000)

	tests := []struct {
		name    string
		timeout time.Duration
		elapsed time.Duration
		want    bool
	}{
		{name: "1ms not elapsed", timeout: time.Millisecond, elapsed: 0, want: false},
		{name: "1ms elapsed", timeout: time.Millisecond, elapsed: 2 * time.Millisecond, want: true},
		{name: "500ms not elapsed across second boundary", timeout: 500 * time.Millisecond, elapsed: 0, want: false},
		{name: "500ms still running after crossing the next Unix second", timeout: 500 * time.Millisecond, elapsed: 300 * time.Millisecond, want: false},
		{name: "exactly 500ms is not timeout", timeout: 500 * time.Millisecond, elapsed: 500 * time.Millisecond, want: false},
		{name: "500ms elapsed", timeout: 500 * time.Millisecond, elapsed: 501 * time.Millisecond, want: true},
		{name: "1s not elapsed", timeout: time.Second, elapsed: 0, want: false},
		{name: "1s elapsed", timeout: time.Second, elapsed: time.Second + time.Millisecond, want: true},
		{name: "default zero timeout never expires", timeout: 0, elapsed: time.Hour, want: false},
		{name: "explicit zero timeout never expires", timeout: 0, elapsed: time.Hour, want: false},
		{name: "negative timeout never expires", timeout: -time.Second, elapsed: time.Hour, want: false},
		{name: "very large timeout still running", timeout: 24 * time.Hour, elapsed: time.Second, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			now = func() time.Time { return start }
			ctx := InitSeataContext(context.Background())
			SetTimeInfo(ctx, TimeInfo{createTime: now(), timeout: tc.timeout})
			now = func() time.Time { return start.Add(tc.elapsed) }
			assert.Equal(t, tc.want, IsTimeout(ctx))
		})
	}
}

func TestBeginSubsecondTimeoutIsNotImmediatelyExpired(t *testing.T) {
	restoreNow := now
	t.Cleanup(func() { now = restoreNow })

	frozen := time.Unix(1_700_000_000, 700_000_000)
	now = func() time.Time { return frozen }

	ctx := InitSeataContext(context.Background())
	err := Begin(ctx, &GtxConfig{
		Name:        "subsecond",
		Propagation: Supports,
		Timeout:     500 * time.Millisecond,
	})
	assert.NoError(t, err)

	ti := GetTimeInfo(ctx)
	assert.True(t, ti.createTime.Equal(frozen))
	assert.Equal(t, 700_000_000, ti.createTime.Nanosecond())
	assert.False(t, IsTimeout(ctx))
}
