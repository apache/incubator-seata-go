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
			createTime: time.Duration(time.Now().Add(-3 * time.Second).Unix()),
			timeout:    time.Second,
		})

		assert.True(t, IsTimeout(ctx))
	})

	t.Run("active time info", func(t *testing.T) {
		ctx := InitSeataContext(context.Background())
		SetTimeInfo(ctx, TimeInfo{
			createTime: time.Duration(time.Now().Unix()),
			timeout:    5 * time.Second,
		})

		assert.False(t, IsTimeout(ctx))
	})
}
